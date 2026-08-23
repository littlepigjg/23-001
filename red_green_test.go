package main

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

// TestRedGreen 测试 context 生命周期管理缺陷
func TestRedGreen(t *testing.T) {
	fmt.Println("==========================================")
	fmt.Println("开始测试 Context 生命周期管理缺陷")
	fmt.Println("==========================================")

	// 创建内存存储
	memStore := store.NewMemoryStore()
	ctx := context.Background()

	// 初始化存储
	if err := memStore.Init(ctx); err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}

	// 创建配置
	cfg := config.DefaultConfig()

	// 创建服务
	taskService := service.NewTaskService(memStore, memStore, memStore, memStore, memStore, cfg)
	progressService := service.NewProgressService(memStore, memStore, memStore)

	// 创建型号
	deviceModel := model.NewDeviceModel("TestModel", "TestManufacturer", "V1.0", "Test Description")
	if err := memStore.CreateModel(ctx, deviceModel); err != nil {
		t.Fatalf("创建设备型号失败: %v", err)
	}

	// 创建固件
	firmware := model.NewFirmware(deviceModel.ID, deviceModel.Name, "V2.0", "abcdef123456", 1024*1024, "/path/to/firmware.bin", time.Now(), "Test changelog")
	if err := memStore.CreateFirmware(ctx, firmware); err != nil {
		t.Fatalf("创建固件失败: %v", err)
	}

	// 创建设备
	device := model.NewDevice("DEV001", deviceModel.ID, deviceModel.Name, "TestDevice", "192.168.1.1", "SN001")
	if err := memStore.CreateDevice(ctx, device); err != nil {
		t.Fatalf("创建设备失败: %v", err)
	}

	// ========== 测试 1: 正常 context 启动任务 ==========
	fmt.Println("\n--- 测试 1: 使用正常 context 启动任务 ---")
	taskReq := &model.CreateTaskRequest{
		Name:          "Test Task Normal",
		Description:   "Test task for normal start",
		ModelID:       deviceModel.ID,
		FirmwareID:    firmware.ID,
		TaskType:      model.TaskTypeTargeted,
		TargetDevices: []string{"DEV001"},
		CreatedBy:     "test-user",
	}

	task, err := taskService.CreateTask(ctx, taskReq)
	if err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}

	err = taskService.StartTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("启动任务失败: %v", err)
	}

	// 验证任务状态已更新为 running
	updatedTask, err := memStore.GetTaskByID(ctx, task.ID)
	if err != nil {
		t.Fatalf("获取任务失败: %v", err)
	}

	if updatedTask.Status != model.TaskRunning {
		fmt.Printf("✗ 测试 1 失败: 任务状态应为 running，但实际为 %s\n", updatedTask.Status)
	} else {
		fmt.Println("✓ 测试 1 通过: 正常 context 下任务状态正确更新为 running")
	}

	// ========== 测试 2: 已取消 context 启动任务 ==========
	fmt.Println("\n--- 测试 2: 使用已取消的 context 启动任务 ---")
	taskReq2 := &model.CreateTaskRequest{
		Name:          "Test Task Canceled",
		Description:   "Test task for canceled context",
		ModelID:       deviceModel.ID,
		FirmwareID:    firmware.ID,
		TaskType:      model.TaskTypeTargeted,
		TargetDevices: []string{"DEV001"},
		CreatedBy:     "test-user",
	}

	task2, err := taskService.CreateTask(ctx, taskReq2)
	if err != nil {
		t.Fatalf("创建第二个任务失败: %v", err)
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()

	err = taskService.StartTask(cancelCtx, task2.ID)
	if err != nil {
		// 修复后：context 取消时应该返回错误
		fmt.Println("✓ 测试 2 通过: context 取消时正确返回错误")
	} else {
		// 缺陷：context 取消时返回 nil（成功），跳过状态更新
		updatedTask2, _ := memStore.GetTaskByID(ctx, task2.ID)
		if updatedTask2.Status == model.TaskPending {
			fmt.Println("✗ 测试 2 失败: context 取消时任务状态应保持 pending，但缺陷导致函数返回成功")
		}
	}

	// ========== 测试 3: 已取消 context 上报进度 ==========
	fmt.Println("\n--- 测试 3: 使用已取消的 context 上报进度 ---")
	progressReq := &model.ReportProgressRequest{
		DeviceID: "DEV001",
		TaskID:   task.ID,
		Progress: 50,
		Status:   string(model.UpgradeInProgress),
	}

	cancelCtx3, cancel3 := context.WithCancel(ctx)
	cancel3()

	err = progressService.ReportProgress(cancelCtx3, progressReq)
	if err != nil {
		// 修复后：context 取消时应该返回错误
		fmt.Println("✓ 测试 3 通过: context 取消时正确返回错误")
	} else {
		// 缺陷：context 取消时返回 nil（成功），使用默认值继续执行
		fmt.Println("✗ 测试 3 失败: context 取消时应报错，但缺陷导致使用默认值继续执行")
	}

	// ========== 测试 4: 已取消 context 完成进度上报 ==========
	fmt.Println("\n--- 测试 4: 使用已取消的 context 完成进度上报 ---")
	completeProgressReq := &model.ReportProgressRequest{
		DeviceID: "DEV001",
		TaskID:   task.ID,
		Progress: 100,
		Status:   string(model.UpgradeSuccess),
	}

	// 先使用正常 context 上报一次，建立正确的进度
	_ = progressService.ReportProgress(ctx, completeProgressReq)

	// 获取正常上报后的任务状态
	taskBefore, _ := memStore.GetTaskByID(ctx, task.ID)
	successBefore := taskBefore.SuccessCount

	// 然后用取消的 context 再上报一次
	cancelCtx4, cancel4 := context.WithCancel(ctx)
	cancel4()

	err = progressService.ReportProgress(cancelCtx4, completeProgressReq)
	if err != nil {
		// 修复后：context 取消时应该返回错误
		fmt.Println("✓ 测试 4 通过: context 取消时正确返回错误")
		fmt.Println("\n==========================================")
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		fmt.Println("==========================================")
		return
	}

	// 缺陷：context 取消时使用零值更新进度
	taskAfter, _ := memStore.GetTaskByID(ctx, task.ID)
	if taskAfter.SuccessCount == 0 && successBefore > 0 {
		fmt.Println("✗ 测试 4 失败: context 取消时任务进度被错误重置为零值")
		fmt.Println("\n==========================================")
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("缺陷表现: context 生命周期管理不当，导致取消的 context 被错误处理")
		fmt.Println("==========================================")
		t.Fail()
	} else if taskAfter.SuccessCount == successBefore {
		fmt.Println("✓ 测试 4 通过: 任务进度保持正确值")
		fmt.Println("\n==========================================")
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		fmt.Println("==========================================")
	}
}
