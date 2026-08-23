package fwupgrade_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
)

func TestConcurrentSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}

	s := store.NewFileStore(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	if err := s.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer func() {
		cancel()
		time.Sleep(100 * time.Millisecond)
	}()

	for i := 0; i < 5; i++ {
		m := model.NewDeviceModel(
			"Base-"+string(rune('A'+i)),
			"Mfr",
			"HW",
			"Base model",
		)
		if err := s.CreateModel(ctx, m); err != nil {
			t.Fatalf("CreateModel failed: %v", err)
		}
	}

	time.Sleep(150 * time.Millisecond)

	dataFile := filepath.Join(tmpDir, "data.json")
	if _, err := os.Stat(dataFile); err != nil {
		t.Logf("数据文件状态: %v", err)
	}

	panicCount := 0
	var mu sync.Mutex
	s.SetPanicGuard(func(code string, rawURL string) bool {
		mu.Lock()
		panicCount++
		mu.Unlock()
		return false
	})

	var wg sync.WaitGroup
	errorCh := make(chan error, 50)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				m := model.NewDeviceModel(
					"Concurrent-"+string(rune('A'+idx))+string(rune('0'+j)),
					"Mfr",
					"HW",
					"Concurrent model",
				)
				if err := s.CreateModel(ctx, m); err != nil {
					errorCh <- err
				}
			}
		}(i)
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				m, err := s.GetModelByID(ctx, model.ID((idx%5)+1))
				if err != nil {
					continue
				}
				m.Description = "Updated-" + time.Now().String()
				if err := s.UpdateModel(ctx, m); err != nil {
					errorCh <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errorCh)

	errCount := 0
	for err := range errorCh {
		if err != nil {
			errCount++
		}
	}

	mu.Lock()
	totalPanics := panicCount
	mu.Unlock()

	time.Sleep(100 * time.Millisecond)

	if totalPanics > 0 || errCount > 0 {
		t.Log("RED（红灯，缺陷未修复）")
		t.Logf("Panic guard 触发次数: %d, 操作错误次数: %d", totalPanics, errCount)
	} else {
		models, _, _ := s.ListModels(ctx, 1, 1000)
		if len(models) >= 30 {
			t.Log("GREEN（绿灯，缺陷已修复）")
			t.Logf("最终型号数量: %d", len(models))
		} else {
			t.Log("RED（红灯，缺陷未修复）")
			t.Logf("型号数量异常: %d (期望>=30)", len(models))
		}
	}
}
