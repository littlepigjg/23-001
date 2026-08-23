package fwupgrade_test

import (
	"context"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

// setupTestEnv 创建测试环境
func setupTestEnv() (*store.MemoryStore, *service.StatsService, *service.TaskService, *service.HistoryService) {
	memStore := store.NewMemoryStore()
	cfg := config.DefaultConfig()

	statsSvc := service.NewStatsService(memStore, memStore, memStore, memStore, memStore)
	taskSvc := service.NewTaskService(memStore, memStore, memStore, memStore, memStore, cfg)
	historySvc := service.NewHistoryService(memStore)

	ctx := context.Background()

	memStore.CreateModel(ctx, &model.DeviceModel{Name: "Model-A", Manufacturer: "MFG-A"})
	memStore.CreateModel(ctx, &model.DeviceModel{Name: "Model-B", Manufacturer: "MFG-B"})

	memStore.CreateFirmware(ctx, &model.Firmware{ModelID: 1, Version: "v1.0.0", Md5: "abc123", Size: 1024})
	memStore.CreateFirmware(ctx, &model.Firmware{ModelID: 1, Version: "v2.0.0", Md5: "def456", Size: 2048})

	memStore.CreateDevice(ctx, &model.Device{DeviceID: "DEV-001", ModelID: 1, Name: "Device-1", CurrentFWVer: "v1.0.0", Status: model.DeviceOnline})
	memStore.CreateDevice(ctx, &model.Device{DeviceID: "DEV-002", ModelID: 1, Name: "Device-2", CurrentFWVer: "v2.0.0", Status: model.DeviceOnline})
	memStore.CreateDevice(ctx, &model.Device{DeviceID: "DEV-003", ModelID: 2, Name: "Device-3", CurrentFWVer: "v1.0.0", Status: model.DeviceOffline})

	memStore.CreateTask(ctx, &model.UpgradeTask{Name: "Task-1", ModelID: 1, FirmwareVer: "v2.0.0", Status: model.TaskPending, TotalDevices: 2})

	memStore.CreateRecord(ctx, &model.UpgradeRecord{DeviceID: "DEV-001", TaskID: 1, Status: model.UpgradeSuccess, Progress: 100})
	memStore.CreateRecord(ctx, &model.UpgradeRecord{DeviceID: "DEV-002", TaskID: 1, Status: model.UpgradeFailed, Progress: 50})
	memStore.CreateRecord(ctx, &model.UpgradeRecord{DeviceID: "DEV-003", TaskID: 1, Status: model.UpgradeInProgress, Progress: 75})

	return memStore, statsSvc, taskSvc, historySvc
}

func printRED(t *testing.T, msg string) {
	t.Logf("RED（红灯，缺陷未修复）: %s", msg)
}

func printGREEN(t *testing.T, msg string) {
	t.Logf("GREEN（绿灯，缺陷已修复）: %s", msg)
}

// TestCancelledContextGetDashboard 验证取消的上下文仍返回数据
func TestCancelledContextGetDashboard(t *testing.T) {
	_, statsSvc, _, _ := setupTestEnv()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dashboard, err := statsSvc.GetDashboard(ctx)

	if err == nil && dashboard != nil {
		printRED(t, "已取消的上下文仍然返回了数据")
		t.Logf("  TotalDevices=%d, ActiveTasks=%d, TodayRecords=%d",
			dashboard.TotalDevices, dashboard.ActiveTasks, dashboard.TodayRecords)
		return
	}

	printGREEN(t, "已取消的上下文正确返回了错误")
}

// TestCancelledContextGetStatistics 验证取消的上下文在 GetStatistics 中的行为
func TestCancelledContextGetStatistics(t *testing.T) {
	_, statsSvc, _, _ := setupTestEnv()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	stats, err := statsSvc.GetStatistics(ctx)

	if err == nil && stats != nil {
		printRED(t, "GetStatistics 使用已取消的上下文仍返回数据")
		t.Logf("  TotalDevices=%d, TotalModels=%d, TotalFirmware=%d",
			stats.TotalDevices, stats.TotalModels, stats.TotalFirmware)
		return
	}

	printGREEN(t, "GetStatistics 正确返回了错误")
}

// TestCancelledContextListTasks 验证取消的上下文在任务列表中的行为
func TestCancelledContextListTasks(t *testing.T) {
	_, _, taskSvc, _ := setupTestEnv()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tasks, total, err := taskSvc.ListTasks(ctx, 1, 20, "")

	if err == nil && len(tasks) > 0 {
		printRED(t, "ListTasks 使用已取消的上下文仍返回数据")
		t.Logf("  返回 %d 个任务，总数 %d", len(tasks), total)
		return
	}

	printGREEN(t, "ListTasks 正确返回了错误")
}

// TestCancelledContextGetActiveTasks 验证取消的上下文在获取活跃任务中的行为
func TestCancelledContextGetActiveTasks(t *testing.T) {
	_, _, taskSvc, _ := setupTestEnv()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tasks, err := taskSvc.GetActiveTasks(ctx)

	if err == nil && len(tasks) > 0 {
		printRED(t, "GetActiveTasks 使用已取消的上下文仍返回数据")
		t.Logf("  返回 %d 个活跃任务", len(tasks))
		return
	}

	printGREEN(t, "GetActiveTasks 正确返回了错误")
}

// TestCancelledContextGetRecentRecords 验证取消的上下文在记录查询中的行为
func TestCancelledContextGetRecentRecords(t *testing.T) {
	_, _, _, historySvc := setupTestEnv()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	records, err := historySvc.GetRecentRecords(ctx, 10)

	if err == nil && len(records) > 0 {
		printRED(t, "GetRecentRecords 使用已取消的上下文仍返回数据")
		t.Logf("  返回 %d 条记录", len(records))
		return
	}

	printGREEN(t, "GetRecentRecords 正确返回了错误")
}

// TestCancelledContextCountTodayRecords 验证取消的上下文在今日统计中的行为
func TestCancelledContextCountTodayRecords(t *testing.T) {
	_, _, _, historySvc := setupTestEnv()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	count, err := historySvc.CountTodayRecords(ctx)

	if err == nil && count >= 0 {
		printRED(t, "CountTodayRecords 使用已取消的上下文仍返回数据")
		t.Logf("  返回的今日记录数: %d", count)
		return
	}

	printGREEN(t, "CountTodayRecords 正确返回了错误")
}

// TestCancelledContextGetModel 验证取消的上下文在型号查询中的行为
func TestCancelledContextGetModel(t *testing.T) {
	memStore := store.NewMemoryStore()
	memStore.CreateModel(context.Background(), &model.DeviceModel{Name: "Test", Manufacturer: "T"})

	modelSvc := service.NewDeviceModelService(memStore, config.DefaultConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	m, err := modelSvc.GetModel(ctx, 1)

	if err == nil && m != nil {
		printRED(t, "GetModel 使用已取消的上下文仍返回数据")
		t.Logf("  返回的型号: %s", m.Name)
		return
	}

	printGREEN(t, "GetModel 正确返回了错误")
}

// TestCancelledContextGetDevice 验证取消的上下文在设备查询中的行为
func TestCancelledContextGetDevice(t *testing.T) {
	memStore := store.NewMemoryStore()
	memStore.CreateDevice(context.Background(), &model.Device{DeviceID: "D-1", ModelID: 1, Name: "Dev-1", CurrentFWVer: "v1", Status: model.DeviceOnline})

	deviceSvc := service.NewDeviceService(memStore, memStore, config.DefaultConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d, err := deviceSvc.GetDevice(ctx, 1)

	if err == nil && d != nil {
		printRED(t, "GetDevice 使用已取消的上下文仍返回数据")
		t.Logf("  返回的设备: %s", d.Name)
		return
	}

	printGREEN(t, "GetDevice 正确返回了错误")
}

// TestCancelledContextGetFirmware 验证取消的上下文在固件查询中的行为
func TestCancelledContextGetFirmware(t *testing.T) {
	memStore := store.NewMemoryStore()
	memStore.CreateFirmware(context.Background(), &model.Firmware{ModelID: 1, Version: "v1.0", Md5: "abc", Size: 100})

	fwSvc := service.NewFirmwareService(memStore, memStore, config.DefaultConfig())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fw, err := fwSvc.GetFirmware(ctx, 1)

	if err == nil && fw != nil {
		printRED(t, "GetFirmware 使用已取消的上下文仍返回数据")
		t.Logf("  返回的固件: %s", fw.Version)
		return
	}

	printGREEN(t, "GetFirmware 正确返回了错误")
}

// TestCancelledContextVersionDistribution 验证取消的上下文在版本分布统计中的行为
func TestCancelledContextVersionDistribution(t *testing.T) {
	memStore := store.NewMemoryStore()
	memStore.CreateDevice(context.Background(), &model.Device{DeviceID: "D-1", ModelID: 1, Name: "Dev-1", CurrentFWVer: "v1", Status: model.DeviceOnline})

	statsSvc := service.NewStatsService(memStore, memStore, memStore, memStore, memStore)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dist, err := statsSvc.GetVersionDistribution(ctx)

	if err == nil && len(dist) > 0 {
		printRED(t, "GetVersionDistribution 使用已取消的上下文仍返回数据")
		t.Logf("  返回的版本分布: %v", dist)
		return
	}

	printGREEN(t, "GetVersionDistribution 正确返回了错误")
}