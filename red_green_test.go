package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

// TestRedGreen 测试 nil 缺陷场景：服务关闭时 ctx.Err() 返回 nil 被忽视
func TestRedGreen(t *testing.T) {
	fmt.Println("=== 开始红灯/绿灯测试（nil 缺陷） ===")

	// 创建内存存储
	memStore := store.NewMemoryStore()
	if err := memStore.Init(context.Background()); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	defer memStore.Close()

	// 准备测试数据
	setupTestData(t, memStore)

	// 创建服务
	pollService := service.NewPollService(memStore, memStore, memStore, memStore, nil)
	historyService := service.NewHistoryService(memStore)

	// 测试场景 1: 正常 context 下 PollDevice 应该正常工作
	t.Run("正常context下PollDevice", func(t *testing.T) {
		ctx := context.Background()
		req := &model.PollUpgradeRequest{
			DeviceID:   "device-001",
			ModelID:    1,
			CurrentVer: "1.0.0",
		}

		resp, err := pollService.PollDevice(ctx, req)
		if err != nil {
			t.Errorf("PollDevice should not return error with valid context: %v", err)
		}
		if resp == nil {
			t.Error("PollDevice should return response")
		}
		fmt.Println("  [PASS] 正常context下PollDevice正常工作")
	})

	// 测试场景 2: 服务关闭模拟 - ctx.Err() 返回 nil 但 validateContext 返回错误
	t.Run("服务关闭场景PollDevice", func(t *testing.T) {
		// 使用 context.Background() - ctx.Err() 返回 nil
		ctx := context.Background()

		// 设置 context 验证器模拟服务关闭
		// 当服务关闭时，即使 ctx.Err() 返回 nil，validateContext 也应该返回错误
		pollService.SetContextValidator(func(ctx context.Context) error {
			if ctx.Err() == nil {
				// 模拟服务关闭场景：ctx.Err() 返回 nil 但服务实际已关闭
				return errors.New("service is shutting down")
			}
			return ctx.Err()
		})

		req := &model.PollUpgradeRequest{
			DeviceID:   "device-001",
			ModelID:    1,
			CurrentVer: "1.0.0",
		}

		resp, err := pollService.PollDevice(ctx, req)

		// 缺陷存在时：ctx.Err() 返回 nil，代码忽略 validateContext 的错误
		// 缺陷修复后：代码应该返回 validateContext 的错误
		if err == nil && resp != nil {
			fmt.Println("  [RED] 服务关闭时PollDevice忽略了nil异常，仍然返回数据")
			t.Log("nil缺陷存在：ctx.Err()返回nil时，validateContext的返回值被忽视")
		} else if err != nil {
			fmt.Println("  [GREEN] 服务关闭时PollDevice正确返回错误")
			t.Log("nil缺陷已修复：正确处理了ctx.Err()为nil的异常情况")
		}

		// 恢复默认验证器
		pollService.SetContextValidator(func(ctx context.Context) error {
			return nil
		})
	})

	// 测试场景 3: 服务关闭模拟 - ctx.Err() 返回 nil 但 validateContext 返回错误
	t.Run("服务关闭场景GetDeviceHistory", func(t *testing.T) {
		// 使用 context.Background() - ctx.Err() 返回 nil
		ctx := context.Background()

		// 设置 context 验证器模拟服务关闭
		historyService.SetContextValidator(func(ctx context.Context) error {
			if ctx.Err() == nil {
				// 模拟服务关闭场景：ctx.Err() 返回 nil 但服务实际已关闭
				return errors.New("service is shutting down")
			}
			return ctx.Err()
		})

		records, err := historyService.GetDeviceHistory(ctx, "device-001")

		// 缺陷存在时：ctx.Err() 返回 nil，代码忽略 validateContext 的错误
		if err == nil && records != nil {
			fmt.Println("  [RED] 服务关闭时GetDeviceHistory忽略了nil异常，仍然返回数据")
			t.Log("nil缺陷存在：ctx.Err()返回nil时，validateContext的返回值被忽视")
		} else if err != nil {
			fmt.Println("  [GREEN] 服务关闭时GetDeviceHistory正确返回错误")
			t.Log("nil缺陷已修复：正确处理了ctx.Err()为nil的异常情况")
		}

		// 恢复默认验证器
		historyService.SetContextValidator(func(ctx context.Context) error {
			return nil
		})
	})

	// 测试场景 4: context 已取消场景（验证正常取消处理）
	t.Run("context取消时PollDevice", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 取消 context

		pollService.SetContextValidator(func(ctx context.Context) error {
			return ctx.Err()
		})

		req := &model.PollUpgradeRequest{
			DeviceID:   "device-001",
			ModelID:    1,
			CurrentVer: "1.0.0",
		}

		resp, err := pollService.PollDevice(ctx, req)

		if err == nil && resp != nil {
			fmt.Println("  [RED] context取消时PollDevice仍然返回数据")
		} else if err != nil {
			fmt.Println("  [GREEN] context取消时PollDevice正确返回错误")
		}

		pollService.SetContextValidator(func(ctx context.Context) error {
			return nil
		})
	})

	// 测试场景 5: 服务关闭场景多次测试
	t.Run("服务关闭多次测试PollDevice", func(t *testing.T) {
		redCount := 0
		greenCount := 0

		for i := 0; i < 5; i++ {
			ctx := context.Background() // ctx.Err() 返回 nil

			pollService.SetContextValidator(func(ctx context.Context) error {
				if ctx.Err() == nil {
					return errors.New("service is shutting down")
				}
				return ctx.Err()
			})

			req := &model.PollUpgradeRequest{
				DeviceID:   "device-001",
				ModelID:    1,
				CurrentVer: "1.0.0",
			}

			resp, err := pollService.PollDevice(ctx, req)

			if err == nil && resp != nil {
				redCount++
			} else if err != nil {
				greenCount++
			}
		}

		pollService.SetContextValidator(func(ctx context.Context) error {
			return nil
		})

		if redCount > 0 {
			fmt.Printf("  [RED] 多次测试中 %d/%d 次忽略了nil异常\n", redCount, 5)
			t.Logf("nil缺陷存在：ctx.Err()为nil时validateContext的返回值被忽视")
		} else if greenCount == 5 {
			fmt.Printf("  [GREEN] 多次测试中 %d/%d 次正确返回了错误\n", greenCount, 5)
			t.Log("nil缺陷已修复")
		}
	})

	fmt.Println("=== 测试完成 ===")
}

// TestNilDefectInPollService 专门测试 PollService 中的 nil 缺陷
func TestNilDefectInPollService(t *testing.T) {
	fmt.Println("=== 测试 PollService 中 nil 缺陷 ===")

	memStore := store.NewMemoryStore()
	if err := memStore.Init(context.Background()); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	defer memStore.Close()

	setupTestData(t, memStore)

	pollService := service.NewPollService(memStore, memStore, memStore, memStore, nil)

	// nil 缺陷场景：ctx.Err() 返回 nil，但服务应被视为关闭
	// validateContext 返回错误，但代码忽略了这个返回值
	ctx := context.Background() // ctx.Err() 返回 nil

	pollService.SetContextValidator(func(ctx context.Context) error {
		if ctx.Err() == nil {
			// 模拟服务关闭时，ctx.Err() 返回 nil 的异常场景
			return fmt.Errorf("service shutting down: context is invalid")
		}
		return ctx.Err()
	})

	req := &model.PollUpgradeRequest{
		DeviceID:   "device-001",
		ModelID:    1,
		CurrentVer: "1.0.0",
	}

	resp, err := pollService.PollDevice(ctx, req)

	if err == nil && resp != nil {
		fmt.Println("RED（红灯，nil缺陷未修复）")
		t.Log("nil缺陷未修复：ctx.Err()为nil时validateContext返回值被忽视")
		t.Log("服务应该在ctx.Err()返回nil但validateContext返回错误时中断执行")
	} else if err != nil {
		fmt.Println("GREEN（绿灯，nil缺陷已修复）")
		t.Log("nil缺陷已修复：正确处理了ctx.Err()为nil的异常情况")
	}
}

// TestNilDefectInHistoryService 专门测试 HistoryService 中的 nil 缺陷
func TestNilDefectInHistoryService(t *testing.T) {
	fmt.Println("=== 测试 HistoryService 中 nil 缺陷 ===")

	memStore := store.NewMemoryStore()
	if err := memStore.Init(context.Background()); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	defer memStore.Close()

	setupTestData(t, memStore)

	historyService := service.NewHistoryService(memStore)

	// nil 缺陷场景：ctx.Err() 返回 nil，但服务应被视为关闭
	ctx := context.Background() // ctx.Err() 返回 nil

	historyService.SetContextValidator(func(ctx context.Context) error {
		if ctx.Err() == nil {
			// 模拟服务关闭时，ctx.Err() 返回 nil 的异常场景
			return fmt.Errorf("service shutting down: context is invalid")
		}
		return ctx.Err()
	})

	records, err := historyService.GetDeviceHistory(ctx, "device-001")

	if err == nil && records != nil {
		fmt.Println("RED（红灯，nil缺陷未修复）")
		t.Log("nil缺陷未修复：ctx.Err()为nil时validateContext返回值被忽视")
		t.Log("服务应该在ctx.Err()返回nil但validateContext返回错误时中断执行")
	} else if err != nil {
		fmt.Println("GREEN（绿灯，nil缺陷已修复）")
		t.Log("nil缺陷已修复：正确处理了ctx.Err()为nil的异常情况")
	}
}

// TestNilErrorPropagation 测试 nil 错误传播
func TestNilErrorPropagation(t *testing.T) {
	fmt.Println("=== 测试 nil 错误传播 ===")

	memStore := store.NewMemoryStore()
	if err := memStore.Init(context.Background()); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	defer memStore.Close()

	setupTestData(t, memStore)

	pollService := service.NewPollService(memStore, memStore, memStore, memStore, nil)
	historyService := service.NewHistoryService(memStore)

	// 测试 1: PollDevice 在 nil 场景下是否正确传播错误
	t.Run("PollDevice nil错误传播", func(t *testing.T) {
		ctx := context.Background() // ctx.Err() 返回 nil

		// 设置在 nil 场景下返回错误的验证器
		pollService.SetContextValidator(func(ctx context.Context) error {
			if ctx.Err() == nil {
				return errors.New("nil context error should be propagated")
			}
			return ctx.Err()
		})

		req := &model.PollUpgradeRequest{
			DeviceID:   "device-001",
			ModelID:    1,
			CurrentVer: "1.0.0",
		}

		resp, err := pollService.PollDevice(ctx, req)

		if err == nil && resp != nil {
			fmt.Println("  [RED] PollDevice 没有传播 nil 错误")
		} else if err != nil {
			fmt.Println("  [GREEN] PollDevice 正确传播了 nil 错误")
		}
	})

	// 测试 2: GetDeviceHistory 在 nil 场景下是否正确传播错误
	t.Run("GetDeviceHistory nil错误传播", func(t *testing.T) {
		ctx := context.Background() // ctx.Err() 返回 nil

		historyService.SetContextValidator(func(ctx context.Context) error {
			if ctx.Err() == nil {
				return errors.New("nil context error should be propagated")
			}
			return ctx.Err()
		})

		records, err := historyService.GetDeviceHistory(ctx, "device-001")

		if err == nil && records != nil {
			fmt.Println("  [RED] GetDeviceHistory 没有传播 nil 错误")
		} else if err != nil {
			fmt.Println("  [GREEN] GetDeviceHistory 正确传播了 nil 错误")
		}
	})

	// 测试 3: ListRecords 在 nil 场景下是否正确传播错误
	t.Run("ListRecords nil错误传播", func(t *testing.T) {
		ctx := context.Background() // ctx.Err() 返回 nil

		historyService.SetContextValidator(func(ctx context.Context) error {
			if ctx.Err() == nil {
				return errors.New("nil context error should be propagated")
			}
			return ctx.Err()
		})

		_, _, err := historyService.ListRecords(ctx, 1, 10, "")

		if err == nil {
			fmt.Println("  [RED] ListRecords 没有传播 nil 错误")
		} else {
			fmt.Println("  [GREEN] ListRecords 正确传播了 nil 错误")
		}
	})

	// 恢复默认验证器
	pollService.SetContextValidator(func(ctx context.Context) error { return nil })
	historyService.SetContextValidator(func(ctx context.Context) error { return nil })

	fmt.Println("=== nil 错误传播测试完成 ===")
}

// TestFinalGreenRed 最终的红绿灯判定测试
func TestFinalGreenRed(t *testing.T) {
	memStore := store.NewMemoryStore()
	if err := memStore.Init(context.Background()); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	defer memStore.Close()

	setupTestData(t, memStore)

	pollService := service.NewPollService(memStore, memStore, memStore, memStore, nil)
	historyService := service.NewHistoryService(memStore)

	// nil 缺陷场景：ctx.Err() 返回 nil，但服务应被视为关闭
	// 验证器检测到这种异常情况并返回错误
	pollService.SetContextValidator(func(ctx context.Context) error {
		if ctx.Err() == nil {
			return context.Canceled
		}
		return ctx.Err()
	})

	historyService.SetContextValidator(func(ctx context.Context) error {
		if ctx.Err() == nil {
			return context.Canceled
		}
		return ctx.Err()
	})

	// 使用 context.Background() - ctx.Err() 返回 nil
	ctx := context.Background()

	// 测试 PollDevice
	req := &model.PollUpgradeRequest{
		DeviceID:   "device-001",
		ModelID:    1,
		CurrentVer: "1.0.0",
	}

	pollResp, pollErr := pollService.PollDevice(ctx, req)
	historyRecords, historyErr := historyService.GetDeviceHistory(ctx, "device-001")

	// 判定
	pollHasDefect := (pollErr == nil && pollResp != nil)
	historyHasDefect := (historyErr == nil && historyRecords != nil)

	if pollHasDefect || historyHasDefect {
		fmt.Println("========================================================================")
		fmt.Println("                        RED（红灯，nil缺陷未修复）")
		fmt.Println("========================================================================")
		fmt.Println()
		fmt.Println("nil缺陷表现：")
		if pollHasDefect {
			fmt.Println("  - PollDevice 在 ctx.Err() 返回 nil 时没有正确处理 validateContext 错误")
		}
		if historyHasDefect {
			fmt.Println("  - GetDeviceHistory 在 ctx.Err() 返回 nil 时没有正确处理 validateContext 错误")
		}
		fmt.Println()
		fmt.Println("期望行为：服务在 ctx.Err() 返回 nil 但 validateContext 返回错误时应中断执行")
		fmt.Println("实际行为：服务忽略 nil 异常，继续执行并返回数据")
		fmt.Println()
		t.Error("nil缺陷未修复：ctx.Err()为nil时validateContext的返回值被忽视")
	} else {
		fmt.Println("========================================================================")
		fmt.Println("                        GREEN（绿灯，nil缺陷已修复）")
		fmt.Println("========================================================================")
		fmt.Println()
		fmt.Println("修复验证：")
		if !pollHasDefect {
			fmt.Println("  ✓ PollDevice 正确处理了 ctx.Err() 为 nil 的异常情况")
		}
		if !historyHasDefect {
			fmt.Println("  ✓ GetDeviceHistory 正确处理了 ctx.Err() 为 nil 的异常情况")
		}
		fmt.Println()
		fmt.Println("所有服务正确处理了 ctx.Err() 返回 nil 的异常场景")
	}
}

// setupTestData 设置测试数据
func setupTestData(t *testing.T, s *store.MemoryStore) {
	t.Helper()

	// 创建设备型号
	deviceModel := model.NewDeviceModel("test-model", "test-manufacturer", "v1.0", "test model")
	if err := s.CreateModel(context.Background(), deviceModel); err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// 创建设备
	device := model.NewDevice("device-001", deviceModel.ID, "test-model", "Test Device", "192.168.1.1", "SN001")
	device.CurrentFWVer = "1.0.0"
	if err := s.CreateDevice(context.Background(), device); err != nil {
		t.Fatalf("Failed to create device: %v", err)
	}

	// 创建固件
	fw := model.NewFirmware(deviceModel.ID, "test-model", "2.0.0", "abc123def456", 1024, "/test/path", time.Now(), "test firmware")
	if err := s.CreateFirmware(context.Background(), fw); err != nil {
		t.Fatalf("Failed to create firmware: %v", err)
	}

	// 创建升级记录
	record := model.NewUpgradeRecord("device-001", "Test Device", 1, "Test Task", "1.0.0", "2.0.0")
	if err := s.CreateRecord(context.Background(), record); err != nil {
		t.Fatalf("Failed to create record: %v", err)
	}
}