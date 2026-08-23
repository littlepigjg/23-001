package service

import (
	"context"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
)

// seedStoreWithDevices 构造一个内存存储，预置一个型号、一个固件以及 n 个在线设备。
func seedStoreWithDevices(t *testing.T, n int) *store.MemoryStore {
	t.Helper()

	s := store.NewMemoryStore()
	ctx := context.Background()

	m := model.NewDeviceModel("TestModel", "TestMfg", "1.0", "测试型号")
	if err := s.CreateModel(ctx, m); err != nil {
		t.Fatalf("create model: %v", err)
	}

	fw := model.NewFirmware(m.ID, m.Name, "1.1.0", "d41d8cd98f00b204e9800998ecf8427e", 1024, "/tmp/fake.bin", time.Now(), "changelog")
	if err := s.CreateFirmware(ctx, fw); err != nil {
		t.Fatalf("create firmware: %v", err)
	}

	for i := 0; i < n; i++ {
		d := model.NewDevice(deviceID(i), m.ID, m.Name, deviceName(i), "10.0.0.1", "SN")
		d.CurrentFWVer = "1.0.0"
		d.Status = model.DeviceOnline
		if err := s.CreateDevice(ctx, d); err != nil {
			t.Fatalf("create device %d: %v", i, err)
		}
	}

	return s
}

func deviceID(i int) string {
	// 固定格式的设备ID，便于断言
	return devPrefix(i) + "-0000"
}

func deviceName(i int) string { return devPrefix(i) + " #0000" }

func devPrefix(i int) string {
	return "sensor-" + string(rune('a'+(i%26))) + string(rune('a'+((i/26)%26)))
}

// TestStartTask_FullManyDevices 不再因折半缓冲区而 panic。
// 回归场景：十几个设备执行全量升级任务启动时，旧代码会 index out of range。
func TestStartTask_FullManyDevices(t *testing.T) {
	const n = 12 // 用户报告"十几个设备"触发 panic

	s := seedStoreWithDevices(t, n)
	ctx := context.Background()

	cfg := &config.Config{}

	ts := NewTaskService(s, s, s, s, s, cfg)

	task := model.NewUpgradeTask(
		"全量升级", "回归测试", 1, "TestModel", 1, "1.1.0",
		model.TaskTypeFull, 0, nil, "tester",
	)
	if err := ts.store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 关键断言：旧实现在此处 panic（index out of range）。
	if err := ts.StartTask(ctx, task.ID); err != nil {
		t.Fatalf("StartTask failed: %v", err)
	}

	got, err := ts.store.GetTaskByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.TotalDevices != n {
		t.Fatalf("TotalDevices = %d, want %d", got.TotalDevices, n)
	}
	if got.PendingCount != n {
		t.Fatalf("PendingCount = %d, want %d", got.PendingCount, n)
	}

	records, err := s.ListRecordsByTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("list records: %v", err)
	}
	if len(records) != n {
		t.Fatalf("records count = %d, want %d", len(records), n)
	}
}

// TestStartTask_FullOddDeviceCount 确保 odd 设备数（bufSize = n/2 的极端情形）也不 panic。
func TestStartTask_FullOddDeviceCount(t *testing.T) {
	for _, n := range []int{1, 3, 5, 7, 13, 25} {
		t.Run(devPrefix(n), func(t *testing.T) {
			s := seedStoreWithDevices(t, n)
			ctx := context.Background()
			ts := NewTaskService(s, s, s, s, s, &config.Config{})

			task := model.NewUpgradeTask("t", "d", 1, "TestModel", 1, "1.1.0",
				model.TaskTypeFull, 0, nil, "tester")
			if err := ts.store.CreateTask(ctx, task); err != nil {
				t.Fatalf("create task: %v", err)
			}
			if err := ts.StartTask(ctx, task.ID); err != nil {
				t.Fatalf("StartTask failed for n=%d: %v", n, err)
			}
			got, _ := ts.store.GetTaskByID(ctx, task.ID)
			if got.TotalDevices != n {
				t.Fatalf("TotalDevices = %d, want %d", got.TotalDevices, n)
			}
		})
	}
}

// TestBatchReportProgress_ManyDevices 不再因折半缓冲区而 panic。
// 回归场景：一次上报十来个设备的进度时，旧代码在 updateTaskProgress 中 index out of range。
func TestBatchReportProgress_ManyDevices(t *testing.T) {
	const n = 12

	s := seedStoreWithDevices(t, n)
	ctx := context.Background()
	cfg := &config.Config{}

	ts := NewTaskService(s, s, s, s, s, cfg)
	ps := NewProgressService(s, s, s)

	task := model.NewUpgradeTask("全量升级", "回归测试", 1, "TestModel", 1, "1.1.0",
		model.TaskTypeFull, 0, nil, "tester")
	if err := ts.store.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := ts.StartTask(ctx, task.ID); err != nil {
		t.Fatalf("StartTask: %v", err)
	}

	// 收集已注册的设备ID，构造一批进度上报（部分成功、部分失败、部分进行中）。
	devices, err := s.ListDevicesByModel(ctx, task.ModelID)
	if err != nil {
		t.Fatalf("list devices: %v", err)
	}
	if len(devices) != n {
		t.Fatalf("devices = %d, want %d", len(devices), n)
	}

	reports := make([]*model.ReportProgressRequest, 0, n)
	for i, d := range devices {
		status := model.UpgradeInProgress
		progress := 50
		switch i % 3 {
		case 0:
			status = model.UpgradeSuccess
			progress = 100
		case 1:
			status = model.UpgradeFailed
			progress = 100
		}
		reports = append(reports, &model.ReportProgressRequest{
			DeviceID:     d.DeviceID,
			TaskID:       task.ID,
			Progress:     progress,
			Status:       string(status),
			ErrorMessage: "",
		})
	}

	// 关键断言：旧实现在此处 panic。
	results := ps.BatchReportProgress(ctx, reports)
	for _, err := range results {
		if err != nil {
			t.Fatalf("BatchReportProgress error: %v", err)
		}
	}

	// 验证任务进度统计正确（4 成功 + 4 失败 + 4 进行中 = 12）。
	got, _ := ts.store.GetTaskByID(ctx, task.ID)
	if got.SuccessCount != 4 {
		t.Fatalf("SuccessCount = %d, want 4", got.SuccessCount)
	}
	if got.FailCount != 4 {
		t.Fatalf("FailCount = %d, want 4", got.FailCount)
	}
}
