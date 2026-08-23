package fwupgrade_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func setupTestEnv() (*service.TaskService, *service.ProgressService, *service.PollService, *store.MemoryStore) {
	memStore := store.NewMemoryStore()
	memStore.Init(context.Background())

	cfg := &config.Config{}

	taskService := service.NewTaskService(
		memStore,
		memStore,
		memStore,
		memStore,
		memStore,
		cfg,
	)

	progressService := service.NewProgressService(
		memStore,
		memStore,
		memStore,
	)

	grayscaleService := service.NewGrayscaleService(memStore, cfg)

	pollService := service.NewPollService(
		memStore,
		memStore,
		memStore,
		memStore,
		grayscaleService,
	)

	return taskService, progressService, pollService, memStore
}

func createTestDevice(t *testing.T, s store.DeviceStore, deviceID string, modelID model.ID, currentVer string) *model.Device {
	device := &model.Device{
		DeviceID:     deviceID,
		Name:         "Test Device " + deviceID,
		ModelID:      modelID,
		CurrentFWVer: currentVer,
		Status:       model.DeviceOnline,
		LastSeenAt:   time.Now(),
	}
	err := s.CreateDevice(context.Background(), device)
	if err != nil {
		t.Fatalf("Failed to create test device: %v", err)
	}
	return device
}

func createTestModel(t *testing.T, s store.DeviceModelStore, name string) *model.DeviceModel {
	m := &model.DeviceModel{
		Name:        name,
		Description: "Test Model " + name,
	}
	err := s.CreateModel(context.Background(), m)
	if err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}
	return m
}

func createTestFirmware(t *testing.T, s store.FirmwareStore, modelID model.ID, version string) *model.Firmware {
	fw := &model.Firmware{
		ModelID:   modelID,
		Version:   version,
		Changelog: "Test Firmware " + version,
	}
	err := s.CreateFirmware(context.Background(), fw)
	if err != nil {
		t.Fatalf("Failed to create test firmware: %v", err)
	}
	return fw
}

// TestRedGreen 上下文取消状态污染验证
func TestRedGreen(t *testing.T) {
	hasDefect := false
	var defectDetails []string

	taskService, progressService, pollService, memStore := setupTestEnv()

	testModel := createTestModel(t, memStore, "TestModel")
	device := createTestDevice(t, memStore, "device-001", testModel.ID, "v1.0.0")
	fw := createTestFirmware(t, memStore, testModel.ID, "v2.0.0")

	createReq := &model.CreateTaskRequest{
		Name:           "Test Upgrade Task",
		Description:    "Test task for context cancellation",
		ModelID:        testModel.ID,
		FirmwareID:     fw.ID,
		TaskType:       model.TaskTypeFull,
		GrayscaleRatio: 100,
		TargetDevices:   []string{},
		CreatedBy:      "test",
	}

	task, err := taskService.CreateTask(context.Background(), createReq)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	if task.ID == 0 {
		t.Fatal("Task ID should not be zero")
	}

	// 测试1: 用已取消的上下文启动任务
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = taskService.StartTask(cancelCtx, task.ID)
	if err == nil {
		hasDefect = true
		defectDetails = append(defectDetails, "StartTask: 上下文已取消但任务仍被启动")
	}

	updatedTask, _ := memStore.GetTaskByID(context.Background(), task.ID)
	if updatedTask.Status == model.TaskRunning {
		hasDefect = true
		defectDetails = append(defectDetails, "StartTask: 任务状态被污染为 Running")
	}

	// 测试2: 用已超时的上下文启动任务
	expiredCtx, cancelExp := context.WithTimeout(context.Background(), 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	defer cancelExp()

	err = taskService.StartTask(expiredCtx, task.ID)
	if err == nil {
		hasDefect = true
		defectDetails = append(defectDetails, "StartTask: 上下文超时但任务仍被启动")
	}

	// 验证任务状态未被超时上下文改变
	currentTask, _ := memStore.GetTaskByID(context.Background(), task.ID)
	if currentTask.Status == model.TaskRunning {
		hasDefect = true
		defectDetails = append(defectDetails, "StartTask: 超时上下文污染任务状态")
	}

	// 重新启动任务以便后续测试
	_ = taskService.StartTask(context.Background(), task.ID)
	deviceAfterStart, _ := memStore.GetDeviceByDeviceID(context.Background(), device.DeviceID)
	if deviceAfterStart == nil {
		t.Fatal("Device not found after task start")
	}

	records, _ := memStore.ListRecordsByTask(context.Background(), task.ID)
	if len(records) == 0 {
		t.Fatal("No records found for task")
	}

	// 测试3: 用已取消的上下文上报进度
	cancelCtx2, cancel2 := context.WithCancel(context.Background())
	cancel2()

	progressReq := &model.ReportProgressRequest{
		DeviceID: records[0].DeviceID,
		TaskID:   task.ID,
		Progress: 50,
		Status:   string(model.UpgradeInProgress),
	}

	err = progressService.ReportProgress(cancelCtx2, progressReq)
	if err == nil {
		hasDefect = true
		defectDetails = append(defectDetails, "ReportProgress: 上下文已取消但进度仍被上报")
	}

	deviceAfterProgress, _ := memStore.GetDeviceByDeviceID(context.Background(), device.DeviceID)
	if deviceAfterProgress.UpgradeProgress == 50 {
		hasDefect = true
		defectDetails = append(defectDetails, "ReportProgress: 设备进度被取消上下文更新")
	}

	// 测试4: 用已取消的上下文轮询设备
	pollReq := &model.PollUpgradeRequest{
		DeviceID:   device.DeviceID,
		ModelID:    testModel.ID,
		CurrentVer: "v1.0.0",
	}

	deviceBeforePoll, _ := memStore.GetDeviceByDeviceID(context.Background(), device.DeviceID)
	lastSeenBeforePoll := deviceBeforePoll.LastSeenAt

	cancelCtx3, cancel3 := context.WithCancel(context.Background())
	cancel3()

	_, err = pollService.PollDevice(cancelCtx3, pollReq)
	if err != nil {
		t.Logf("PollDevice returned error: %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	deviceAfterPoll, _ := memStore.GetDeviceByDeviceID(context.Background(), device.DeviceID)
	if !deviceAfterPoll.LastSeenAt.Equal(lastSeenBeforePoll) {
		hasDefect = true
		defectDetails = append(defectDetails, "PollDevice: 取消上下文更新了心跳时间")
	}

	// 测试5: ProcessScheduledTasks 上下文取消传播
	cancelCtx4, cancel4 := context.WithCancel(context.Background())
	cancel4()

	err = taskService.ProcessScheduledTasks(cancelCtx4)
	if err != nil {
		hasDefect = true
		defectDetails = append(defectDetails, "ProcessScheduledTasks: 上下文已取消但仍处理任务")
	}

	// 最终判定
	if hasDefect {
		t.Error("RED（红灯，缺陷未修复）")
		for _, d := range defectDetails {
			t.Logf("  - %s", d)
		}
		fmt.Println("RESULT: RED（红灯，缺陷未修复）- 上下文取消未传播，状态被污染")
	} else {
		fmt.Println("RESULT: GREEN（绿灯，缺陷已修复）- 上下文取消正确传播，状态未被污染")
	}
}