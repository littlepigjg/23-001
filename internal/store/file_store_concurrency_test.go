package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
)

// newTestFileStore 构造一个使用临时数据目录的 FileStore 并启动 autoSave 协程。
func newTestFileStore(t *testing.T) (*FileStore, context.CancelFunc) {
	t.Helper()
	dataDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Storage.DataDir = dataDir

	fs := NewFileStore(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	if err := fs.Init(ctx); err != nil {
		cancel()
		t.Fatalf("Init failed: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		_ = fs.Close()
	})
	return fs, cancel
}

// countPersisted 读取 data.json 中已持久化的型号与固件数量。
func countPersisted(t *testing.T, fs *FileStore) (models, firmwares int) {
	t.Helper()
	dataDir := fs.cfg.Storage.DataDir
	raw, err := os.ReadFile(filepath.Join(dataDir, "data.json"))
	if err != nil {
		t.Fatalf("read data.json: %v", err)
	}
	var persisted struct {
		Models    []*model.DeviceModel `json:"models"`
		Firmwares []*model.Firmware    `json:"firmwares"`
	}
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatalf("unmarshal data.json: %v", err)
	}
	return len(persisted.Models), len(persisted.Firmwares)
}

// TestFileStoreConcurrentModelsAndFirmware 复现用户报告的场景：
//   - 先创建 5 个基础型号
//   - 10 个协程并发各创建 5 个新型号，同时 5 个协程更新已有型号的描述字段
//   - 与此同时 autoSave 每 100ms 落盘一次
//
// 修复后期望：无 panic、无 data race、落盘数量与内存一致、重启后可恢复全部数据。
func TestFileStoreConcurrentModelsAndFirmware(t *testing.T) {
	fs, _ := newTestFileStore(t)
	ctx := context.Background()

	// 1. 创建 5 个基础型号
	baseIDs := make([]model.ID, 0, 5)
	for i := 0; i < 5; i++ {
		m := model.NewDeviceModel(
			fmt.Sprintf("base-model-%d", i),
			"Acme",
			"hw-1.0",
			"base model",
		)
		if err := fs.CreateModel(ctx, m); err != nil {
			t.Fatalf("create base model %d: %v", i, err)
		}
		baseIDs = append(baseIDs, m.ID)
	}

	// 2. 并发创建 + 更新，运行约 200ms
	const (
		creators   = 10
		modelsEach = 5
		updaters   = 5
	)
	var wg sync.WaitGroup
	stop := make(chan struct{})

	// 创建型号 + 对应固件（协程同时创建型号和固件，复现“同时创建设备型号和固件”）
	var nameCounter uint64
	for c := 0; c < creators; c++ {
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				for k := 0; k < modelsEach; k++ {
					// 用全局自增计数保证名称唯一，避免重复创建报错干扰
					seq := atomic.AddUint64(&nameCounter, 1)
					name := fmt.Sprintf("concurrent-%d-%d", c, seq)
					m := model.NewDeviceModel(name, "Acme", "hw-1.0", "desc")
					if err := fs.CreateModel(ctx, m); err != nil {
						t.Logf("create model %s: %v", name, err)
						continue
					}
					// 为该型号创建一个固件
					fw := model.NewFirmware(
						m.ID, m.Name,
						fmt.Sprintf("v-%d", seq),
						"d41d8cd98f00b204e9800998ecf8427e",
						1,
						filepath.Join(fs.cfg.Storage.UploadDir, name+".bin"),
						time.Now(),
						"changelog",
					)
					if err := fs.CreateFirmware(ctx, fw); err != nil {
						t.Logf("create firmware for %s: %v", name, err)
					}
				}
			}
		}(c)
	}

	// 更新已有型号的描述字段
	for u := 0; u < updaters; u++ {
		wg.Add(1)
		go func(u int) {
			defer wg.Done()
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
				}
				id := baseIDs[i%len(baseIDs)]
				// GetModelByID 返回的是内存中对象的指针，这里拷贝后再修改，
				// 避免测试自身在并发读快照时修改同一对象引入伪竞态。
				got, err := fs.GetModelByID(ctx, id)
				if err != nil {
					continue
				}
				cp := *got
				cp.Description = fmt.Sprintf("updated-by-%d-iter-%d", u, i)
				_ = fs.UpdateModel(ctx, &cp)
				i++
			}
		}(u)
	}

	// 让并发运行约 200ms（与用户报告的触发窗口一致），同时触发多次 autoSave。
	time.Sleep(200 * time.Millisecond)
	close(stop)
	wg.Wait()

	// 3. 显式保存并校验：落盘数量必须与内存一致
	if err := fs.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	memModels, _, _ := fs.ListModels(ctx, 1, 100000)
	memFirmwares, _, _ := fs.ListFirmwares(ctx, 1, 100000, 0)
	persistedModels, persistedFirmwares := countPersisted(t, fs)

	if persistedModels != len(memModels) {
		t.Errorf("model count mismatch: persisted=%d in-memory=%d (data loss)",
			persistedModels, len(memModels))
	}
	if persistedFirmwares != len(memFirmwares) {
		t.Errorf("firmware count mismatch: persisted=%d in-memory=%d (data loss)",
			persistedFirmwares, len(memFirmwares))
	}
	t.Logf("final counts: models=%d firmwares=%d", persistedModels, persistedFirmwares)
}

// TestFileStoreRestartRecoversAllData 验证“重启后数据丢失”：用一个新的 FileStore
// 重新加载同一 data.json，型号数量必须与写入数量一致。
func TestFileStoreRestartRecoversAllData(t *testing.T) {
	fs1, cancel1 := newTestFileStore(t)
	ctx := context.Background()

	const want = 50
	for i := 0; i < want; i++ {
		m := model.NewDeviceModel(fmt.Sprintf("r-model-%d", i), "Acme", "hw-1.0", "d")
		if err := fs1.CreateModel(ctx, m); err != nil {
			t.Fatalf("create model %d: %v", i, err)
		}
	}
	if err := fs1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	cancel1()

	// 用相同数据目录重新加载
	cfg2 := config.DefaultConfig()
	cfg2.Storage.DataDir = fs1.cfg.Storage.DataDir
	fs2 := NewFileStore(cfg2)
	ctx2, cancel2 := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel2()
		_ = fs2.Close()
	})
	if err := fs2.Init(ctx2); err != nil {
		t.Fatalf("re-Init: %v", err)
	}

	got, _, err := fs2.ListModels(ctx2, 1, 100000)
	if err != nil {
		t.Fatalf("ListModels after restart: %v", err)
	}
	if len(got) != want {
		t.Fatalf("restart data loss: want %d models, got %d", want, len(got))
	}
}

// TestSnapshotIsRaceFree 直接对 Snapshot 在并发读写下进行压测，
// 配合 `go test -race` 确保读快照路径不会与写路径竞争。
func TestSnapshotIsRaceFree(t *testing.T) {
	fs, _ := newTestFileStore(t)
	ctx := context.Background()

	// 预置一些数据
	for i := 0; i < 20; i++ {
		m := model.NewDeviceModel(fmt.Sprintf("race-%d", i), "Acme", "hw-1.0", "d")
		_ = fs.CreateModel(ctx, m)
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// 并发写
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
				}
				m := model.NewDeviceModel(fmt.Sprintf("rw-%d-%d", w, i), "Acme", "hw-1.0", "d")
				_ = fs.CreateModel(ctx, m)
				i++
			}
		}(w)
	}

	// 并发读快照
	for r := 0; r < 8; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				snap := fs.memStore.Snapshot()
				// 触发对快照数据的读取，确保拷贝被实际使用
				_ = len(snap.Models)
				runtime.Gosched()
			}
		}()
	}

	time.Sleep(150 * time.Millisecond)
	close(stop)
	wg.Wait()
}
