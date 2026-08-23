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
	ctx := context.Background()
	cfg := config.DefaultConfig()
	memStore := store.NewMemoryStore()

	// 创建设备型号
	modelName := "TestModel"
	manufacturer := "TestMfg"
	hardwareVer := "v1.0"
	deviceModel := model.NewDeviceModel(modelName, manufacturer, hardwareVer, "Test model for grayscale")
	err := memStore.CreateModel(ctx, deviceModel)
	if err != nil {
		t.Fatalf("RED（红灯，缺陷未修复）\n创建设备型号失败: %v", err)
	}

	// 创建设备（100台）
	deviceCount := 100
	for i := 1; i <= deviceCount; i++ {
		deviceID := fmt.Sprintf("device-%03d", i)
		device := model.NewDevice(deviceID, deviceModel.ID, modelName, fmt.Sprintf("Device %03d", i), fmt.Sprintf("192.168.1.%d", i+100), fmt.Sprintf("SN%03d", i))
		device.CurrentFWVer = "v1.0.0"
		err := memStore.CreateDevice(ctx, device)
		if err != nil {
			t.Fatalf("RED（红灯，缺陷未修复）\n创建设备失败: %v", err)
		}
	}

	// 创建固件
	firmware := model.NewFirmware(deviceModel.ID, modelName, "v2.0.0", "abcdef123456", 1024*1024, "/tmp/test_firmware.bin", time.Now(), "Test firmware")
	err = memStore.CreateFirmware(ctx, firmware)
	if err != nil {
		t.Fatalf("RED（红灯，缺陷未修复）\n创建固件失败: %v", err)
	}

	// 测试1：使用 GrayscaleService.GenerateDeviceGroup 测试灰度分组
	grayService := service.NewGrayscaleService(memStore, cfg)

	deviceIDs := make([]string, deviceCount)
	for i := 0; i < deviceCount; i++ {
		deviceIDs[i] = fmt.Sprintf("device-%03d", i+1)
	}

	// 测试 10% 灰度比例
	grayGroup10, waitGroup10 := grayService.GenerateDeviceGroup(deviceIDs, 10.0)

	// 验证 10% 灰度应该能选出约 10% 的设备
	grayRatio10 := float64(len(grayGroup10)) / float64(deviceCount) * 100

	if len(grayGroup10) == 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Printf("测试1失败：10%% 灰度比例只分到 %d 台设备（预期约 %d 台）\n", len(grayGroup10), deviceCount/10)
		fmt.Printf("灰度组大小: %d, 等待组大小: %d\n", len(grayGroup10), len(waitGroup10))
		fmt.Printf("实际灰度比例: %.1f%%（预期约 10%%）\n", grayRatio10)
		os.Exit(1)
	}

	// 测试 20% 灰度比例应该能选出约 20% 的设备
	grayGroup20, _ := grayService.GenerateDeviceGroup(deviceIDs, 20.0)
	grayRatio20 := float64(len(grayGroup20)) / float64(deviceCount) * 100

	if len(grayGroup20) == 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Printf("测试2失败：20%% 灰度比例只分到 %d 台设备\n", len(grayGroup20))
		fmt.Printf("实际灰度比例: %.1f%%（预期约 20%%）\n", grayRatio20)
		os.Exit(1)
	}

	// 测试 5% 灰度比例（小于10的倍数）应该也能正常工作
	grayGroup5, _ := grayService.GenerateDeviceGroup(deviceIDs, 5.0)
	grayRatio5 := float64(len(grayGroup5)) / float64(deviceCount) * 100

	if len(grayGroup5) == 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Printf("测试3失败：5%% 灰度比例只分到 %d 台设备\n", len(grayGroup5))
		fmt.Printf("实际灰度比例: %.1f%%（预期约 5%%）\n", grayRatio5)
		os.Exit(1)
	}

	// 测试2：使用 TaskService.StartTask 测试目标设备计算
	taskService := service.NewTaskService(memStore, memStore, memStore, memStore, memStore, cfg)

	createReq := &model.CreateTaskRequest{
		Name:           "Test Grayscale Task",
		Description:    "Test task for grayscale ratio",
		ModelID:        deviceModel.ID,
		FirmwareID:     firmware.ID,
		TaskType:       model.TaskTypeGrayscale,
		GrayscaleRatio: 10.0,
		CreatedBy:      "test",
	}

	task, err := taskService.CreateTask(ctx, createReq)
	if err != nil {
		t.Fatalf("RED（红灯，缺陷未修复）\n创建任务失败: %v", err)
	}

	err = taskService.StartTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("RED（红灯，缺陷未修复）\n启动任务失败: %v", err)
	}

	// 重新获取任务查看结果
	startedTask, err := taskService.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("RED（红灯，缺陷未修复）\n获取任务失败: %v", err)
	}

	if startedTask.TotalDevices == 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Printf("测试4失败：10%% 灰度任务总共分配了 0 台设备\n")
		fmt.Printf("任务状态: %s\n", startedTask.Status)
		fmt.Printf("TotalDevices: %d\n", startedTask.TotalDevices)
		os.Exit(1)
	}

	// 验证 TotalDevices 应该约为 10（100台的10%）
	if startedTask.TotalDevices < 5 || startedTask.TotalDevices > 20 {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Printf("测试4失败：10%% 灰度任务分配了 %d 台设备（预期约 10 台）\n", startedTask.TotalDevices)
		os.Exit(1)
	}

	// 所有测试通过
	fmt.Println("GREEN（绿灯，缺陷已修复）")
	fmt.Println("所有灰度比例测试通过！")
	fmt.Printf("  10%% 灰度: %d 台设备 (%.1f%%)\n", len(grayGroup10), grayRatio10)
	fmt.Printf("  20%% 灰度: %d 台设备 (%.1f%%)\n", len(grayGroup20), grayRatio20)
	fmt.Printf("  5%% 灰度: %d 台设备 (%.1f%%)\n", len(grayGroup5), grayRatio5)
	fmt.Printf("  TaskService 10%%: %d 台设备\n", startedTask.TotalDevices)
}
