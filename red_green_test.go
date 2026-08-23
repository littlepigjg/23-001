package main

import (
	"context"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

// TestRedGreen 验证版本处理逻辑的正确性
// RED: 存在缺陷时测试失败（panic）
// GREEN: 缺陷修复后测试通过
func TestRedGreen(t *testing.T) {
	memStore := store.NewMemoryStore()
	cfg := config.DefaultConfig()
	ctx := context.Background()

	err := memStore.Init(ctx)
	if err != nil {
		t.Fatalf("初始化存储失败: %v", err)
	}

	// 准备测试数据
	deviceModel := model.NewDeviceModel("TestModel", "TestMfg", "HW-v1", "测试型号")
	err = memStore.CreateModel(ctx, deviceModel)
	if err != nil {
		t.Fatalf("创建设备型号失败: %v", err)
	}

	// 测试路径1: 设备轮询时空版本导致panic
	t.Run("PollDevice空版本处理", func(t *testing.T) {
		// 注册一个空版本的设备
		regReq := &model.RegisterDeviceRequest{
			DeviceID:    "DEVICE-EMPTY-VER",
			ModelID:     deviceModel.ID,
			ModelName:   deviceModel.Name,
			Name:        "空版本设备",
			FirmwareVer: "",
			IPAddress:   "192.168.1.100",
			SerialNumber: "SN-EMPTY-001",
			Status:      "online",
		}

		deviceSvc := service.NewDeviceService(memStore, memStore, cfg)
		device, err := deviceSvc.RegisterDevice(ctx, regReq)
		if err != nil {
			t.Fatalf("注册设备失败: %v", err)
		}

		if device.CurrentFWVer != "" {
			t.Log("GREEN: 空版本被正确处理")
		}

		// 创建一个活跃任务
		fw := model.NewFirmware(deviceModel.ID, deviceModel.Name, "v2.0.0", "abc123def456", 1024*1024, "/tmp/test.bin", time.Now(), "测试固件")
		err = memStore.CreateFirmware(ctx, fw)
		if err != nil {
			t.Fatalf("创建固件失败: %v", err)
		}

		task := model.NewUpgradeTask(
			"测试任务", "描述",
			deviceModel.ID, deviceModel.Name,
			fw.ID, fw.Version,
			model.TaskTypeFull, 100.0, nil, "admin",
		)
		err = memStore.CreateTask(ctx, task)
		if err != nil {
			t.Fatalf("创建任务失败: %v", err)
		}

		// 将任务设为运行状态以触发 shouldSkipUpgrade 路径
		err = memStore.UpdateTaskStatus(ctx, task.ID, model.TaskRunning)
		if err != nil {
			t.Fatalf("更新任务状态失败: %v", err)
		}

		// 调用 PollDevice - 如果版本处理有缺陷会panic
		pollSvc := service.NewPollService(memStore, memStore, memStore, memStore, service.NewGrayscaleService(memStore, cfg))

		pollReq := &model.PollUpgradeRequest{
			DeviceID:   device.DeviceID,
			ModelID:    device.ModelID,
			CurrentVer: device.CurrentFWVer,
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RED: PollDevice处理空版本时发生panic - %v", r)
				}
			}()
			resp, err := pollSvc.PollDevice(ctx, pollReq)
			if err != nil {
				t.Errorf("PollDevice返回错误: %v", err)
			}
			t.Logf("GREEN: PollDevice正常返回 ShouldUpgrade=%v", resp.ShouldUpgrade)
		}()
	})

	// 测试路径2: 固件列表排序时空版本导致panic
	t.Run("ListFirmwares空版本排序", func(t *testing.T) {
		fwSvc := service.NewFirmwareService(memStore, memStore, cfg)

		// 创建多个固件，其中一个版本为空
		fw1 := model.NewFirmware(deviceModel.ID, deviceModel.Name, "v1.0.0", "aaa111", 512*1024, "/tmp/fw1.bin", time.Now().Add(-48*time.Hour), "旧版本")
		err = memStore.CreateFirmware(ctx, fw1)
		if err != nil {
			t.Fatalf("创建固件1失败: %v", err)
		}

		// 创建一个空版本的固件用于触发排序缺陷
		fwEmpty := model.NewFirmware(deviceModel.ID, deviceModel.Name, "", "bbb222", 256*1024, "/tmp/fw_empty.bin", time.Now(), "空版本固件")
		err = memStore.CreateFirmware(ctx, fwEmpty)
		if err != nil {
			t.Fatalf("创建空版本固件失败: %v", err)
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RED: ListFirmwares排序空版本时发生panic - %v", r)
				}
			}()
			_, _, err := fwSvc.ListFirmwares(ctx, 1, 100, deviceModel.ID)
			if err != nil {
				t.Errorf("ListFirmwares返回错误: %v", err)
			}
			t.Log("GREEN: ListFirmwares排序正常完成")
		}()
	})

	// 测试路径3: 设备注册空版本后再次触发版本比较
	t.Run("shouldSkipUpgrade空版本比较", func(t *testing.T) {
		pollSvc := service.NewPollService(memStore, memStore, memStore, memStore, service.NewGrayscaleService(memStore, cfg))

		// shouldSkipUpgrade 中两个空版本的比较
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RED: shouldSkipUpgrade空版本比较时发生panic - %v", r)
				}
			}()
			result := pollSvc.ShouldSkipUpgrade("", "v1.0.0")
			t.Logf("GREEN: shouldSkipUpgrade正常返回 %v", result)
		}()

		// 两个都为空
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RED: shouldSkipUpgrade双空版本比较时发生panic - %v", r)
				}
			}()
			result := pollSvc.ShouldSkipUpgrade("", "")
			t.Logf("GREEN: shouldSkipUpgrade双空版本正常返回 %v", result)
		}()
	})
}
