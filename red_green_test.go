package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	allPassed := true

	// 测试 1: StartTask 缓冲区容量估算
	if !testStartTaskBuffer() {
		allPassed = false
		t.Log("StartTask 缓冲区容量不足，触发切片越界")
	} else {
		t.Log("StartTask 缓冲区容量正常")
	}

	// 测试 2: BatchReportProgress 缓冲区容量估算
	if !testBatchReportBuffer() {
		allPassed = false
		t.Log("BatchReportProgress 缓冲区容量不足，触发切片越界")
	} else {
		t.Log("BatchReportProgress 缓冲区容量正常")
	}

	if allPassed {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		os.Exit(0)
	} else {
		fmt.Println("RED（红灯，缺陷未修复）")
		os.Exit(1)
	}
}

// testStartTaskBuffer 测试 StartTask 的缓冲区容量是否足够
func testStartTaskBuffer() (passed bool) {
	defer func() {
		if r := recover(); r != nil {
			passed = false
		}
	}()

	ctx := context.Background()
	memStore := store.NewMemoryStore()
	cfg := config.DefaultConfig()

	taskSvc := service.NewTaskService(memStore, memStore, memStore, memStore, memStore, cfg)

	// 创建设备型号
	m := model.NewDeviceModel("TestModel", "TestMfr", "v1", "Test model")
	memStore.CreateModel(ctx, m)

	// 创建固件
	fw := model.NewFirmware(m.ID, m.Name, "1.0.0", "abc123def456", 1024, "/path/fw.bin", time.Now(), "Initial version")
	memStore.CreateFirmware(ctx, fw)

	// 创建 12 个设备（超过 50% 预分配容量）
	deviceIDs := make([]string, 12)
	for i := 0; i < 12; i++ {
		dev := model.NewDevice(fmt.Sprintf("DEV-%03d", i), m.ID, m.Name, fmt.Sprintf("Device %d", i),
			fmt.Sprintf("192.168.1.%d", i+1), fmt.Sprintf("SN-%03d", i))
		memStore.CreateDevice(ctx, dev)
		deviceIDs[i] = dev.DeviceID
	}

	// 创建指定设备升级任务
	req := &model.CreateTaskRequest{
		Name:           "Test Task",
		Description:    "Test buffer estimation",
		ModelID:        m.ID,
		FirmwareID:     fw.ID,
		TaskType:       model.TaskTypeTargeted,
		GrayscaleRatio: 0,
		TargetDevices:  deviceIDs,
		ScheduledAt:    time.Now(),
		CreatedBy:      "tester",
	}

	task, err := taskSvc.CreateTask(ctx, req)
	if err != nil {
		return false
	}

	// 启动任务 - 这里会触发 calculateTargetDevices 中的缓冲区溢出
	err = taskSvc.StartTask(ctx, task.ID)
	if err != nil {
		return false
	}

	return true
}

// testBatchReportBuffer 测试 BatchReportProgress 的缓冲区容量是否足够
func testBatchReportBuffer() (passed bool) {
	defer func() {
		if r := recover(); r != nil {
			passed = false
		}
	}()

	ctx := context.Background()
	memStore := store.NewMemoryStore()
	cfg := config.DefaultConfig()

	taskSvc := service.NewTaskService(memStore, memStore, memStore, memStore, memStore, cfg)
	progressSvc := service.NewProgressService(memStore, memStore, memStore)

	// 创建设备型号
	m := model.NewDeviceModel("TestModel2", "TestMfr2", "v1", "Test model 2")
	memStore.CreateModel(ctx, m)

	// 创建固件
	fw := model.NewFirmware(m.ID, m.Name, "2.0.0", "abc123def456", 2048, "/path/fw2.bin", time.Now(), "Second version")
	memStore.CreateFirmware(ctx, fw)

	// 创建 10 个设备
	deviceIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		dev := model.NewDevice(fmt.Sprintf("DEV2-%03d", i), m.ID, m.Name, fmt.Sprintf("Device2 %d", i),
			fmt.Sprintf("192.168.2.%d", i+1), fmt.Sprintf("SN2-%03d", i))
		memStore.CreateDevice(ctx, dev)
		deviceIDs[i] = dev.DeviceID
	}

	// 创建并启动任务
	req := &model.CreateTaskRequest{
		Name:           "Test Task 2",
		Description:    "Test batch buffer estimation",
		ModelID:        m.ID,
		FirmwareID:     fw.ID,
		TaskType:       model.TaskTypeTargeted,
		GrayscaleRatio: 0,
		TargetDevices:  deviceIDs,
		ScheduledAt:    time.Now(),
		CreatedBy:      "tester",
	}

	task, err := taskSvc.CreateTask(ctx, req)
	if err != nil {
		return false
	}

	err = taskSvc.StartTask(ctx, task.ID)
	if err != nil {
		return false
	}

	// 生成 10 个进度报告
	reports := make([]*model.ReportProgressRequest, 10)
	for i := 0; i < 10; i++ {
		reports[i] = &model.ReportProgressRequest{
			DeviceID:    deviceIDs[i],
			TaskID:      task.ID,
			Progress:    100,
			Status:      string(model.UpgradeSuccess),
			ErrorMessage: "",
		}
	}

	// 批量上报进度
	progressSvc.BatchReportProgress(ctx, reports)

	return true
}