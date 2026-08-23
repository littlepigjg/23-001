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
	fmt.Println("=== 开始缺陷检测测试 ===")

	// 创建配置
	cfg := config.DefaultConfig()
	cfg.Storage.DataDir = "/tmp/test_fwupgrade_data"
	cfg.Storage.UploadDir = "/tmp/test_fwupgrade_uploads"

	// 清理测试数据
	os.RemoveAll("/tmp/test_fwupgrade_data")
	os.RemoveAll("/tmp/test_fwupgrade_uploads")

	// 创建内存存储
	memStore := store.NewMemoryStore()

	// 初始化上下文
	ctx := context.Background()

	// 创建设备型号
	modelID := model.ID(1)
	deviceModel := model.NewDeviceModel("TestModel", "TestManufacturer", "V1.0", "Test Model")
	deviceModel.ID = modelID
	memStore.CreateModel(ctx, deviceModel)

	// 创建固件
	firmware := model.NewFirmware(modelID, "TestModel", "1.0.0", "abc123def456", 1024, "/tmp/test_firmware.bin", time.Now(), "Initial version")
	firmware.ID = model.ID(1)
	memStore.CreateFirmware(ctx, firmware)

	// 创建固件服务
	firmwareService := service.NewFirmwareService(memStore, memStore, cfg)

	// 测试1：获取存在的固件应该正常返回
	fmt.Println("--- 测试1：获取存在的固件 ---")
	existingFw, err := firmwareService.GetFirmware(ctx, model.ID(1))
	if err != nil {
		fmt.Println("RED（红灯，缺陷未修复）：获取存在的固件返回了错误", err)
		t.Error("获取存在的固件应该正常返回")
		return
	}
	if existingFw == nil {
		fmt.Println("RED（红灯，缺陷未修复）：获取存在的固件返回了 nil")
		t.Error("获取存在的固件不应该返回 nil")
		return
	}
	fmt.Println("测试1通过：获取存在的固件正常返回")

	// 测试2：获取不存在的固件应该返回错误
	fmt.Println("--- 测试2：获取不存在的固件 ---")
	nonExistentID := model.ID(9999)
	_, err = firmwareService.GetFirmware(ctx, nonExistentID)
	if err == nil {
		fmt.Println("RED（红灯，缺陷未修复）：获取不存在的固件应该返回错误，但返回了 nil 错误")
		t.Error("获取不存在的固件应该返回错误")
		return
	}
	fmt.Println("测试2通过：获取不存在的固件返回了错误")

	// 测试3：获取固件文件 - 不存在的固件 ID
	fmt.Println("--- 测试3：获取不存在固件的文件 ---")
	_, _, err = firmwareService.GetFirmwareFile(ctx, nonExistentID)
	if err == nil {
		fmt.Println("RED（红灯，缺陷未修复）：获取不存在固件的文件应该返回错误，但返回了 nil 错误")
		t.Error("获取不存在固件的文件应该返回错误")
		return
	}
	fmt.Println("测试3通过：获取不存在固件的文件返回了错误")

	// 测试4：创建设备服务并测试获取不存在的设备
	fmt.Println("--- 测试4：获取不存在的设备 ---")
	deviceService := service.NewDeviceService(memStore, memStore, cfg)
	nonExistentDeviceID := model.ID(9999)
	_, err = deviceService.GetDevice(ctx, nonExistentDeviceID)
	if err == nil {
		fmt.Println("RED（红灯，缺陷未修复）：获取不存在的设备应该返回错误，但返回了 nil 错误")
		t.Error("获取不存在的设备应该返回错误")
		return
	}
	fmt.Println("测试4通过：获取不存在的设备返回了错误")

	// 测试5：更新不存在的设备
	fmt.Println("--- 测试5：更新不存在的设备 ---")
	updateReq := &model.UpdateDeviceRequest{
		Name: "UpdatedName",
	}
	_, err = deviceService.UpdateDevice(ctx, nonExistentDeviceID, updateReq)
	if err == nil {
		fmt.Println("RED（红灯，缺陷未修复）：更新不存在的设备应该返回错误，但返回了 nil 错误")
		t.Error("更新不存在的设备应该返回错误")
		return
	}
	fmt.Println("测试5通过：更新不存在的设备返回了错误")

	// 测试6：进度服务 - 完成不存在设备的升级
	fmt.Println("--- 测试6：完成不存在设备的升级 ---")
	progressService := service.NewProgressService(memStore, memStore, memStore)
	err = progressService.CompleteDeviceUpgrade(ctx, "non_existent_device_9999", true, "")
	if err == nil {
		fmt.Println("RED（红灯，缺陷未修复）：完成不存在设备的升级应该返回错误，但返回了 nil 错误")
		t.Error("完成不存在设备的升级应该返回错误")
		return
	}
	fmt.Println("测试6通过：完成不存在设备的升级返回了错误")

	fmt.Println("=== GREEN（绿灯，缺陷已修复）：所有测试通过 ===")
}