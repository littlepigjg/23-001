package service

import (
	"context"
	"strings"
	"sync"
	"testing"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
)

// newProgressServiceWithDevice 构造真实的内存存储 + 单台设备，返回服务与设备存储 ID。
// 使用真实 MemoryStore（而非 mock），以便 -race 能捕获到真实的并发数据竞争。
func newProgressServiceWithDevice(t *testing.T) (*ProgressService, store.Store, model.ID) {
	t.Helper()
	ctx := context.Background()
	s := store.NewMemoryStore()

	m := model.NewDeviceModel("M1", "vendor", "hw1", "")
	if err := s.CreateModel(ctx, m); err != nil {
		t.Fatalf("create model: %v", err)
	}
	d := model.NewDevice("dev-1", m.ID, m.Name, "device1", "10.0.0.1", "SN1")
	if err := s.CreateDevice(ctx, d); err != nil {
		t.Fatalf("create device: %v", err)
	}

	ps := NewProgressService(s, s, s)
	return ps, s, d.ID
}

// TestReportProgress_ConcurrentSameDevice_FinalMax 复刻用户场景：同一设备并发
// 上报 1..100，断言最终进度恒为 100，且全程不返回 regression/rollback 错误。
func TestReportProgress_ConcurrentSameDevice_FinalMax(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for round := 0; round < 20; round++ {
		ps, st, _ := newProgressServiceWithDevice(t)

		const n = 100
		var wg sync.WaitGroup
		wg.Add(n)
		for i := 1; i <= n; i++ {
			go func(progress int) {
				defer wg.Done()
				req := &model.ReportProgressRequest{
					DeviceID: "dev-1",
					Progress: progress,
				}
				if err := ps.ReportProgress(ctx, req); err != nil {
					// max 语义下，乱序到达的旧进度静默保留，不应报错。
					t.Errorf("report %d: %v", progress, err)
				}
			}(i)
		}
		wg.Wait()

		device, err := ps.GetDeviceProgress(ctx, "dev-1")
		if err != nil {
			t.Fatalf("round %d get progress: %v", round, err)
		}
		if device.UpgradeProgress != n {
			t.Fatalf("round %d: final progress = %d, want %d", round, device.UpgradeProgress, n)
		}
		_ = st // keep store referenced
	}
}

// TestReportProgress_NoRegressionErrors 确保并发上报的错误信息里
// 不出现 regression / rollback 字样（即使乱序）。
func TestReportProgress_NoRegressionErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ps, _, _ := newProgressServiceWithDevice(t)

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	errs := make(chan error, n)
	for i := 1; i <= n; i++ {
		go func(progress int) {
			defer wg.Done()
			req := &model.ReportProgressRequest{DeviceID: "dev-1", Progress: progress}
			errs <- ps.ReportProgress(ctx, req)
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err == nil {
			continue
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "regression") || strings.Contains(msg, "rollback") {
			t.Fatalf("unexpected regression/rollback error: %v", err)
		}
	}
}

// TestGetDeviceProgress_ReturnsConsistentSnapshot 并发写 + 并发读，
// 断言读到的进度始终在合法区间 [0,100] 且非撕裂值（-race 下通过）。
func TestGetDeviceProgress_ReturnsConsistentSnapshot(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ps, _, _ := newProgressServiceWithDevice(t)

	const writers = 50
	const readers = 50
	var wg sync.WaitGroup
	wg.Add(writers + readers)

	// writers: 并发上报，整体覆盖 1..100，保证最终最大值为 100。
	for w := 0; w < writers; w++ {
		go func(seed int) {
			defer wg.Done()
			for r := 0; r < 5; r++ {
				progress := ((seed + r*13) % 100) + 1
				_ = ps.ReportProgress(ctx, &model.ReportProgressRequest{
					DeviceID: "dev-1",
					Progress: progress,
				})
			}
		}(w)
	}

	// readers: 并发读进度
	for r := 0; r < readers; r++ {
		go func() {
			defer wg.Done()
			device, err := ps.GetDeviceProgress(ctx, "dev-1")
			if err != nil {
				t.Errorf("get progress: %v", err)
				return
			}
			if device.UpgradeProgress < 0 || device.UpgradeProgress > 100 {
				t.Errorf("torn/inconsistent progress read: %d", device.UpgradeProgress)
			}
		}()
	}
	wg.Wait()

	device, _ := ps.GetDeviceProgress(ctx, "dev-1")
	if device.UpgradeProgress != 100 {
		t.Fatalf("final progress = %d, want 100", device.UpgradeProgress)
	}
}
