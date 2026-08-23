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

// ============================================================
// 存储层缺陷测试: MemoryStore.DeleteDevice / DeleteRecord
// ============================================================

// TestDeleteDevice_NonExistent_Store 验证存储层删除不存在设备时返回错误
func TestDeleteDevice_NonExistent_Store(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()

	err := memStore.DeleteDevice(ctx, 99999)
	if err == nil {
		t.Errorf("RED（红灯，缺陷未修复）: 存储层 DeleteDevice 对不存在的设备应返回错误，但返回了 nil")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	}
}

// TestDeleteRecord_NonExistent_Store 验证存储层删除不存在记录时返回错误
func TestDeleteRecord_NonExistent_Store(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()

	err := memStore.DeleteRecord(ctx, 99999)
	if err == nil {
		t.Errorf("RED（红灯，缺陷未修复）: 存储层 DeleteRecord 对不存在的记录应返回错误，但返回了 nil")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	}
}

// TestDeleteDevice_Existing_Store 验证存储层删除存在设备成功
func TestDeleteDevice_Existing_Store(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()

	devModel := &model.DeviceModel{
		Name:         "test-model",
		Manufacturer: "test-mfr",
		HardwareVer:  "v1.0",
		Description:  "test",
		IsActive:     true,
	}
	if err := memStore.CreateModel(ctx, devModel); err != nil {
		t.Fatalf("创建型号失败: %v", err)
	}

	device := &model.Device{
		DeviceID: "DEV-001",
		ModelID:  devModel.ID,
		Name:     "test-device",
		Status:   model.DeviceOnline,
	}
	if err := memStore.CreateDevice(ctx, device); err != nil {
		t.Fatalf("创建设备失败: %v", err)
	}

	err := memStore.DeleteDevice(ctx, device.ID)
	if err != nil {
		t.Errorf("RED（红灯，缺陷未修复）: 存储层 DeleteDevice 对存在的设备应返回 nil，但返回了: %v", err)
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）: 删除存在设备成功")
	}
}

// TestDeleteRecord_Existing_Store 验证存储层删除存在记录成功
func TestDeleteRecord_Existing_Store(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()

	record := &model.UpgradeRecord{
		DeviceID:   "DEV-001",
		TaskID:     1,
		FromVersion: "v1.0",
		ToVersion:   "v2.0",
		Status:     model.UpgradeSuccess,
		Progress:   100,
	}
	if err := memStore.CreateRecord(ctx, record); err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}

	err := memStore.DeleteRecord(ctx, record.ID)
	if err != nil {
		t.Errorf("RED（红灯，缺陷未修复）: 存储层 DeleteRecord 对存在的记录应返回 nil，但返回了: %v", err)
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）: 删除存在记录成功")
	}
}

// ============================================================
// 服务层缺陷测试: DeviceService / HistoryService
// ============================================================

// TestServiceDeleteDevice_NonExistent 验证服务层删除不存在设备时返回错误
func TestServiceDeleteDevice_NonExistent(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := &config.Config{}
	deviceService := service.NewDeviceService(memStore, memStore, cfg)
	ctx := context.Background()

	err := deviceService.DeleteDevice(ctx, 99999)
	if err == nil {
		t.Errorf("RED（红灯，缺陷未修复）: 服务层 DeleteDevice 对不存在的设备应返回错误，但返回了 nil")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	}
}

// TestServiceDeleteRecord_NonExistent 验证服务层删除不存在记录时返回错误
func TestServiceDeleteRecord_NonExistent(t *testing.T) {
	memStore := store.NewMemoryStore()
	historyService := service.NewHistoryService(memStore)
	ctx := context.Background()

	err := historyService.DeleteRecord(ctx, 99999)
	if err == nil {
		t.Errorf("RED（红灯，缺陷未修复）: 服务层 DeleteRecord 对不存在的记录应返回错误，但返回了 nil")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	}
}

// TestServiceDeleteDevice_Existing 验证服务层删除存在设备成功
func TestServiceDeleteDevice_Existing(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := &config.Config{}
	deviceService := service.NewDeviceService(memStore, memStore, cfg)
	ctx := context.Background()

	m := &model.DeviceModel{
		Name:         "svc-test-model",
		Manufacturer: "test-mfr",
		HardwareVer:  "v1.0",
		Description:  "test",
		IsActive:     true,
	}
	if err := memStore.CreateModel(ctx, m); err != nil {
		t.Fatalf("创建型号失败: %v", err)
	}

	device := &model.Device{
		DeviceID: "DEV-SVC-001",
		ModelID:  m.ID,
		Name:     "svc-test-device",
		Status:   model.DeviceOnline,
	}
	if err := memStore.CreateDevice(ctx, device); err != nil {
		t.Fatalf("创建设备失败: %v", err)
	}

	err := deviceService.DeleteDevice(ctx, device.ID)
	if err != nil {
		t.Errorf("RED（红灯，缺陷未修复）: 服务层 DeleteDevice 对存在的设备应返回 nil，但返回了: %v", err)
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）: 服务层删除存在设备成功")
	}
}

// TestServiceDeleteRecord_Existing 验证服务层删除存在记录成功
func TestServiceDeleteRecord_Existing(t *testing.T) {
	memStore := store.NewMemoryStore()
	historyService := service.NewHistoryService(memStore)
	ctx := context.Background()

	now := time.Now()
	record := &model.UpgradeRecord{
		DeviceID:    "DEV-SVC-001",
		TaskID:      1,
		FromVersion: "v1.0",
		ToVersion:   "v2.0",
		Status:      model.UpgradeSuccess,
		Progress:    100,
		CompletedAt: &now,
	}
	if err := memStore.CreateRecord(ctx, record); err != nil {
		t.Fatalf("创建记录失败: %v", err)
	}

	err := historyService.DeleteRecord(ctx, record.ID)
	if err != nil {
		t.Errorf("RED（红灯，缺陷未修复）: 服务层 DeleteRecord 对存在的记录应返回 nil，但返回了: %v", err)
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）: 服务层删除存在记录成功")
	}
}

// ============================================================
// 边界条件测试
// ============================================================

// TestDeleteDevice_ZeroID 验证删除ID为0的设备返回错误
func TestDeleteDevice_ZeroID(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()

	err := memStore.DeleteDevice(ctx, 0)
	if err == nil {
		t.Errorf("RED（红灯，缺陷未修复）: 删除ID为0的设备应返回错误，但返回了 nil")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	}
}

// TestDeleteRecord_ZeroID 验证删除ID为0的记录返回错误
func TestDeleteRecord_ZeroID(t *testing.T) {
	memStore := store.NewMemoryStore()
	ctx := context.Background()

	err := memStore.DeleteRecord(ctx, 0)
	if err == nil {
		t.Errorf("RED（红灯，缺陷未修复）: 删除ID为0的记录应返回错误，但返回了 nil")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	}
}
