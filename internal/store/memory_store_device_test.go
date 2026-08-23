package store

import (
	"context"
	"sync"
	"testing"

	"fwupgrade/internal/model"
)

// newTestStoreWithDevice 创建一个含单台设备的内存存储，供进度并发测试使用。
func newTestStoreWithDevice(t *testing.T) (*MemoryStore, model.ID) {
	t.Helper()
	ctx := context.Background()
	s := NewMemoryStore()

	m := model.NewDeviceModel("M1", "vendor", "hw1", "")
	if err := s.CreateModel(ctx, m); err != nil {
		t.Fatalf("create model: %v", err)
	}

	d := model.NewDevice("dev-1", m.ID, m.Name, "device1", "10.0.0.1", "SN1")
	if err := s.CreateDevice(ctx, d); err != nil {
		t.Fatalf("create device: %v", err)
	}
	return s, d.ID
}

// TestUpdateDeviceProgress_ConcurrentMonotonic_FinalIsMax 复刻用户场景：
// 同一设备并发上报 1..100，最终进度必须等于最大上报值 100。
func TestUpdateDeviceProgress_ConcurrentMonotonic_FinalIsMax(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	for round := 0; round < 20; round++ {
		s, deviceID := newTestStoreWithDevice(t)

		const n = 100
		var wg sync.WaitGroup
		wg.Add(n)
		// 通过闭包捕获 i，使每个 goroutine 上报一个不同的进度值。
		for i := 1; i <= n; i++ {
			go func(progress int) {
				defer wg.Done()
				if err := s.UpdateDeviceProgress(ctx, deviceID, progress); err != nil {
					t.Errorf("update progress %d: %v", progress, err)
				}
			}(i)
		}
		wg.Wait()

		d, err := s.GetDeviceByID(ctx, deviceID)
		if err != nil {
			t.Fatalf("round %d get device: %v", round, err)
		}
		if d.UpgradeProgress != n {
			t.Fatalf("round %d: final progress = %d, want %d", round, d.UpgradeProgress, n)
		}
	}
}

// TestUpdateDeviceProgress_NeverRegresses 先把进度推到 100，再并发上报 1..99，
// 断言进度始终为 100（永不回退），且 max 语义下旧值上报不应返回错误。
func TestUpdateDeviceProgress_NeverRegresses(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s, deviceID := newTestStoreWithDevice(t)
	if err := s.UpdateDeviceProgress(ctx, deviceID, 100); err != nil {
		t.Fatalf("seed 100: %v", err)
	}

	const n = 99
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go func(progress int) {
			defer wg.Done()
			if err := s.UpdateDeviceProgress(ctx, deviceID, progress); err != nil {
				t.Errorf("update stale progress %d: %v", progress, err)
			}
		}(i)
	}
	wg.Wait()

	d, err := s.GetDeviceByID(ctx, deviceID)
	if err != nil {
		t.Fatalf("get device: %v", err)
	}
	if d.UpgradeProgress != 100 {
		t.Fatalf("progress regressed to %d, want 100", d.UpgradeProgress)
	}
}

// TestUpdateDeviceProgress_AtomicCompareAndSet 验证达到 100 时状态与目标版本同步更新。
func TestUpdateDeviceProgress_AtomicCompareAndSet(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s, deviceID := newTestStoreWithDevice(t)

	// 先设一个中间进度，状态不应被置为 online。
	if err := s.UpdateDeviceProgress(ctx, deviceID, 50); err != nil {
		t.Fatalf("update 50: %v", err)
	}
	d, _ := s.GetDeviceByID(ctx, deviceID)
	if d.UpgradeProgress != 50 {
		t.Fatalf("progress = %d, want 50", d.UpgradeProgress)
	}

	// 推到 100：状态应置 online，TargetFWVer 应等于 CurrentFWVer。
	if err := s.UpdateDeviceProgress(ctx, deviceID, 100); err != nil {
		t.Fatalf("update 100: %v", err)
	}
	d, _ = s.GetDeviceByID(ctx, deviceID)
	if d.UpgradeProgress != 100 {
		t.Fatalf("progress = %d, want 100", d.UpgradeProgress)
	}
	if d.Status != model.DeviceOnline {
		t.Fatalf("status = %s, want online", d.Status)
	}
	if d.TargetFWVer != d.CurrentFWVer {
		t.Fatalf("target fw = %q, want %q", d.TargetFWVer, d.CurrentFWVer)
	}
}

// TestReadPathsReturnSnapshots 验证读路径返回独立快照：修改返回值不应影响存储内部状态。
func TestReadPathsReturnSnapshots(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s, deviceID := newTestStoreWithDevice(t)
	if err := s.UpdateDeviceProgress(ctx, deviceID, 42); err != nil {
		t.Fatalf("update 42: %v", err)
	}

	// GetDeviceByID 返回快照，外部修改不应污染存储。
	got, err := s.GetDeviceByID(ctx, deviceID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	got.UpgradeProgress = 999
	got.Status = model.DeviceError

	again, _ := s.GetDeviceByID(ctx, deviceID)
	if again.UpgradeProgress != 42 {
		t.Fatalf("snapshot leaked external mutation: progress = %d, want 42", again.UpgradeProgress)
	}
	if again.Status == model.DeviceError {
		t.Fatalf("snapshot leaked external mutation: status = error")
	}

	// GetDeviceByDeviceID 同理。
	byDev, _ := s.GetDeviceByDeviceID(ctx, "dev-1")
	if byDev.UpgradeProgress != 42 {
		t.Fatalf("get by device id progress = %d, want 42", byDev.UpgradeProgress)
	}
}
