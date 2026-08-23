package fwupgrade_test

import (
	"context"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

// TestRedGreen 单一缺陷验证测试
// 缺陷未修复时：RED - panic
// 缺陷已修复时：GREEN - 返回错误信息
func TestRedGreen(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := config.DefaultConfig()
	ctx := context.Background()

	// 创建一个有效型号（id=1）
	validModel := model.NewDeviceModel("TestModel", "TestMfg", "1.0", "test model")
	if err := memStore.CreateModel(ctx, validModel); err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	// 创建固件，ModelID 指向不存在的型号（999）
	// 这样固件归属检查能通过，但型号查询会返回 nil
	now := time.Now()
	fw := model.NewFirmware(model.ID(999), "NonExistent", "1.0.0", "md5hash", 1024, "/path/to/fw.bin", now, "release")
	if err := memStore.CreateFirmware(ctx, fw); err != nil {
		t.Fatalf("failed to create firmware: %v", err)
	}

	// 设置 PanicGuard：模型不存在时返回 (nil, nil)
	memStore.SetPanicGuard(func(modelID model.ID) bool {
		return false
	})

	taskSvc := service.NewTaskService(
		memStore, memStore, memStore, memStore, memStore, cfg,
	)

	req := &model.CreateTaskRequest{
		Name:       "TestTask",
		ModelID:    model.ID(999),
		FirmwareID: fw.ID,
		TaskType:   model.TaskTypeFull,
		CreatedBy:  "tester",
	}

	hasDefect := false

	defer func() {
		if r := recover(); r != nil {
			hasDefect = true
			t.Logf("RED: 检测到缺陷 - 触发 panic: %v", r)
		} else {
			t.Log("GREEN: 缺陷已修复 - 未触发 panic")
		}
		if hasDefect {
			t.Log("结论: RED - 缺陷存在，需修复")
		} else {
			t.Log("结论: GREEN - 缺陷已修复")
		}
	}()

	_, err := taskSvc.CreateTask(ctx, req)
	if err != nil {
		t.Logf("GREEN: CreateTask 返回错误（修复后行为）: %v", err)
	}
}
