package fwupgrade_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	tmpDir := t.TempDir()

	cfgPath1 := filepath.Join(tmpDir, "config1.conf")
	cfgPath2 := filepath.Join(tmpDir, "config2.conf")

	cfgContent1 := fmt.Sprintf(
		"storage.type=file\n"+
			"storage.data_dir=%s\n"+
			"storage.upload_dir=%s\n"+
			"firmware.max_file_size=104857600\n"+
			"firmware.allowed_exts=.bin,.hex,.img\n"+
			"firmware.require_md5=true\n"+
			"grayscale.default_ratio=10.0\n"+
			"grayscale.max_ratio=100.0\n"+
			"grayscale.min_ratio=0.0\n"+
			"grayscale.rollback_threshold=20\n"+
			"grayscale.max_concurrent_tasks=10\n",
		tmpDir, filepath.Join(tmpDir, "uploads1"),
	)

	cfgContent2 := fmt.Sprintf(
		"storage.type=memory\n"+
			"storage.data_dir=%s\n"+
			"storage.upload_dir=%s\n"+
			"firmware.max_file_size=52428800\n"+
			"firmware.allowed_exts=.bin,.hex\n"+
			"firmware.require_md5=false\n"+
			"grayscale.default_ratio=25.0\n"+
			"grayscale.max_ratio=75.0\n"+
			"grayscale.min_ratio=5.0\n"+
			"grayscale.rollback_threshold=35\n"+
			"grayscale.max_concurrent_tasks=5\n",
		tmpDir, filepath.Join(tmpDir, "uploads2"),
	)

	if err := os.WriteFile(cfgPath1, []byte(cfgContent1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath2, []byte(cfgContent2), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()

	memStore := store.NewMemoryStore()
	fileStore := store.NewFileStore(cfg)

	fwSvc := service.NewFirmwareService(fileStore, memStore, cfg)
	graySvc := service.NewGrayscaleService(memStore, cfg)

	ctx := context.Background()

	modelID := model.ID(1)
	version := "1.0.0-test"

	_ = memStore.CreateModel(ctx, &model.DeviceModel{
		ID:           modelID,
		Name:         "test-model",
		Manufacturer: "test-mfr",
		HardwareVer:  "v1",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})

	_ = fileStore.CreateModel(ctx, &model.DeviceModel{
		ID:           modelID,
		Name:         "test-model-fs",
		Manufacturer: "test-mfr",
		HardwareVer:  "v1",
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})

	testFirmware := &model.Firmware{
		ID:            model.ID(1),
		ModelID:       modelID,
		ModelName:     "test-model",
		Version:       version,
		Md5:           "abcdef1234567890abcdef1234567890",
		Size:          1024,
		FilePath:      filepath.Join(tmpDir, "test.bin"),
		ReleaseDate:   time.Now(),
		IsActive:      true,
		DownloadCount: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	_ = memStore.CreateFirmware(ctx, testFirmware)
	_ = fileStore.CreateFirmware(ctx, testFirmware)

	// Pre-load config to avoid initial state issues
	_ = cfg.ReloadConfig(cfgPath1)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Config reloader goroutines (concurrent writes without locks)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					if id%2 == 0 {
						_ = cfg.ReloadConfig(cfgPath1)
					} else {
						_ = cfg.ReloadConfig(cfgPath2)
					}
				}
			}
		}(i)
	}

	// Service reader goroutines (concurrent reads without locks)
	// These check for config INCONSISTENCY - the key symptom of the race
	var inconsistentCount int64
	var mu sync.Mutex

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					maxSize, reqMD5, uploadDir := fwSvc.GetConfigSnapshot()

					isCfg1 := (maxSize == 104857600 && reqMD5 == true &&
						containsUpload(uploadDir, "uploads1"))
					isCfg2 := (maxSize == 52428800 && reqMD5 == false &&
						containsUpload(uploadDir, "uploads2"))

					if !isCfg1 && !isCfg2 {
						mu.Lock()
						inconsistentCount++
						mu.Unlock()
					}

					_ = fwSvc.ValidateConfigConsistency()
					_, _ = fwSvc.GetLatestFirmware(ctx, modelID)
					_, _, _ = graySvc.GetConfigSnapshot()
					_ = graySvc.ValidateConfigRange()
					_ = graySvc.CalculateNextRatio(10.0)
				}
			}
		}()
	}

	time.Sleep(3 * time.Second)
	close(stop)
	wg.Wait()

	mu.Lock()
	ic := inconsistentCount
	mu.Unlock()

	if ic > 0 {
		ms, md5, ud := fwSvc.GetConfigSnapshot()
		t.Errorf("RED (红灯，检测到缺陷) - 检测到 %d 次配置不一致读写，存在数据竞争！", ic)
		t.Errorf("  当前配置: max_size=%d, require_md5=%v, upload_dir=%s - 来自不同配置源，违反一致性",
			ms, md5, ud)
	} else {
		t.Log("GREEN (绿灯，缺陷已修复) - 无配置不一致，无数据竞争")
	}

	maxSize, _, uploadDir := fwSvc.GetConfigSnapshot()
	if maxSize <= 0 {
		t.Errorf("config max file size is invalid: %d", maxSize)
	}
	if uploadDir == "" {
		t.Errorf("config upload dir is empty")
	}

	if err := cfg.Validate(); err != nil {
		t.Logf("config validation error: %v", err)
	}
}

func containsUpload(dir, suffix string) bool {
	return len(dir) > 0 && len(suffix) > 0 && len(dir) >= len(suffix) && dir[len(dir)-len(suffix):] == suffix
}