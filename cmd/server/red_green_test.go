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

// TestRedGreen 测试 context 取消时服务的行为
func TestRedGreen(t *testing.T) {
	fmt.Println("=== 开始红灯/绿灯测试 ===")

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

	// 测试场景 2: context 被取消时 PollDevice 应该返回错误
	t.Run("context取消时PollDevice", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// 设置 context 验证器来模拟 context 错误
		pollService.SetContextValidator(func(ctx context.Context) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return nil
		})

		// 取消 context
		cancel()

		req := &model.PollUpgradeRequest{
			DeviceID:   "device-001",
			ModelID:    1,
			CurrentVer: "1.0.0",
		}

		resp, err := pollService.PollDevice(ctx, req)

		// 缺陷存在时：服务忽略 context 错误，仍然返回数据
		// 缺陷修复后：服务应该返回错误
		if err == nil && resp != nil {
			fmt.Println("  [RED] context取消时PollDevice忽略了错误，仍然返回数据")
			t.Log("缺陷存在：服务在context取消时仍然继续执行")
		} else if err != nil {
			fmt.Println("  [GREEN] context取消时PollDevice正确返回错误")
			t.Log("缺陷已修复：服务在context取消时正确返回错误")
		}

		// 恢复默认验证器
		pollService.SetContextValidator(func(ctx context.Context) error {
			return nil
		})
	})

	// 测试场景 3: context 被取消时 GetDeviceHistory 应该返回错误
	t.Run("context取消时GetDeviceHistory", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// 设置 context 验证器来模拟 context 错误
		historyService.SetContextValidator(func(ctx context.Context) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return nil
		})

		// 取消 context
		cancel()

		records, err := historyService.GetDeviceHistory(ctx, "device-001")

		// 缺陷存在时：服务忽略 context 错误，仍然返回数据
		// 缺陷修复后：服务应该返回错误
		if err == nil && records != nil {
			fmt.Println("  [RED] context取消时GetDeviceHistory忽略了错误，仍然返回数据")
			t.Log("缺陷存在：服务在context取消时仍然继续执行")
		} else if err != nil {
			fmt.Println("  [GREEN] context取消时GetDeviceHistory正确返回错误")
			t.Log("缺陷已修复：服务在context取消时正确返回错误")
		}

		// 恢复默认验证器
		historyService.SetContextValidator(func(ctx context.Context) error {
			return nil
		})
	})

	// 测试场景 4: context 超时 PollDevice 应该返回错误
	t.Run("context超时时PollDevice", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// 等待 context 超时
		time.Sleep(10 * time.Millisecond)

		// 设置 context 验证器
		pollService.SetContextValidator(func(ctx context.Context) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return nil
		})

		req := &model.PollUpgradeRequest{
			DeviceID:   "device-001",
			ModelID:    1,
			CurrentVer: "1.0.0",
		}

		resp, err := pollService.PollDevice(ctx, req)

		// 缺陷存在时：服务忽略 context 错误
		if err == nil && resp != nil {
			fmt.Println("  [RED] context超时时PollDevice忽略了错误")
			t.Log("缺陷存在：服务在context超时时仍然继续执行")
		} else if err != nil {
			fmt.Println("  [GREEN] context超时时PollDevice正确返回错误")
			t.Log("缺陷已修复：服务在context超时时正确返回错误")
		}

		// 恢复默认验证器
		pollService.SetContextValidator(func(ctx context.Context) error {
			return nil
		})
	})

	// 测试场景 5: context 超时时 GetDeviceHistory 应该返回错误
	t.Run("context超时时GetDeviceHistory", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// 等待 context 超时
		time.Sleep(10 * time.Millisecond)

		// 设置 context 验证器
		historyService.SetContextValidator(func(ctx context.Context) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return nil
		})

		records, err := historyService.GetDeviceHistory(ctx, "device-001")

		// 缺陷存在时：服务忽略 context 错误
		if err == nil && records != nil {
			fmt.Println("  [RED] context超时时GetDeviceHistory忽略了错误")
			t.Log("缺陷存在：服务在context超时时仍然继续执行")
		} else if err != nil {
			fmt.Println("  [GREEN] context超时时GetDeviceHistory正确返回错误")
			t.Log("缺陷已修复：服务在context超时时正确返回错误")
		}

		// 恢复默认验证器
		historyService.SetContextValidator(func(ctx context.Context) error {
			return nil
		})
	})

	// 测试场景 6: PollDevice 多次 context 取消场景
	t.Run("多次context取消PollDevice", func(t *testing.T) {
		redCount := 0
		greenCount := 0

		for i := 0; i < 5; i++ {
			ctx, cancel := context.WithCancel(context.Background())

			pollService.SetContextValidator(func(ctx context.Context) error {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return nil
			})

			cancel()

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
			fmt.Printf("  [RED] 多次测试中 %d/%d 次忽略了context错误\n", redCount, 5)
			t.Logf("缺陷存在：服务在context取消时仍然继续执行")
		} else if greenCount == 5 {
			fmt.Printf("  [GREEN] 多次测试中 %d/%d 次正确返回了错误\n", greenCount, 5)
			t.Log("缺陷已修复：服务在context取消时正确返回错误")
		}
	})

	// 测试场景 7: GetDeviceHistory 多次 context 取消场景
	t.Run("多次context取消GetDeviceHistory", func(t *testing.T) {
		redCount := 0
		greenCount := 0

		for i := 0; i < 5; i++ {
			ctx, cancel := context.WithCancel(context.Background())

			historyService.SetContextValidator(func(ctx context.Context) error {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				return nil
			})

			cancel()

			records, err := historyService.GetDeviceHistory(ctx, "device-001")

			if err == nil && records != nil {
				redCount++
			} else if err != nil {
				greenCount++
			}
		}

		historyService.SetContextValidator(func(ctx context.Context) error {
			return nil
		})

		if redCount > 0 {
			fmt.Printf("  [RED] 多次测试中 %d/%d 次忽略了context错误\n", redCount, 5)
			t.Logf("缺陷存在：服务在context取消时仍然继续执行")
		} else if greenCount == 5 {
			fmt.Printf("  [GREEN] 多次测试中 %d/%d 次正确返回了错误\n", greenCount, 5)
			t.Log("缺陷已修复：服务在context取消时正确返回错误")
		}
	})

	fmt.Println("=== 测试完成 ===")
}

// TestContextCancellationInPollService 专门测试 PollService 中的 context 取消处理
func TestContextCancellationInPollService(t *testing.T) {
	fmt.Println("=== 测试 PollService 中 context 取消处理 ===")

	memStore := store.NewMemoryStore()
	if err := memStore.Init(context.Background()); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	defer memStore.Close()

	setupTestData(t, memStore)

	pollService := service.NewPollService(memStore, memStore, memStore, memStore, nil)

	// 设置 context 验证器模拟 context 取消
	pollService.SetContextValidator(func(ctx context.Context) error {
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		}
		return nil
	})

	// 创建已取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &model.PollUpgradeRequest{
		DeviceID:   "device-001",
		ModelID:    1,
		CurrentVer: "1.0.0",
	}

	resp, err := pollService.PollDevice(ctx, req)

	if err == nil && resp != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Log("缺陷未修复：PollDevice 在 context 取消时没有返回错误")
		t.Log("服务应该在 context 无效时停止执行并返回错误")
	} else if err != nil {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		t.Log("缺陷已修复：PollDevice 在 context 取消时正确返回了错误")
	}
}

// TestContextCancellationInHistoryService 专门测试 HistoryService 中的 context 取消处理
func TestContextCancellationInHistoryService(t *testing.T) {
	fmt.Println("=== 测试 HistoryService 中 context 取消处理 ===")

	memStore := store.NewMemoryStore()
	if err := memStore.Init(context.Background()); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	defer memStore.Close()

	setupTestData(t, memStore)

	historyService := service.NewHistoryService(memStore)

	// 设置 context 验证器模拟 context 取消
	historyService.SetContextValidator(func(ctx context.Context) error {
		if ctx.Err() != nil {
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		}
		return nil
	})

	// 创建已取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	records, err := historyService.GetDeviceHistory(ctx, "device-001")

	if err == nil && records != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Log("缺陷未修复：GetDeviceHistory 在 context 取消时没有返回错误")
		t.Log("服务应该在 context 无效时停止执行并返回错误")
	} else if err != nil {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		t.Log("缺陷已修复：GetDeviceHistory 在 context 取消时正确返回了错误")
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

// TestContextErrorPropagation 测试 context 错误传播
func TestContextErrorPropagation(t *testing.T) {
	fmt.Println("=== 测试 context 错误传播 ===")

	memStore := store.NewMemoryStore()
	if err := memStore.Init(context.Background()); err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}
	defer memStore.Close()

	setupTestData(t, memStore)

	pollService := service.NewPollService(memStore, memStore, memStore, memStore, nil)
	historyService := service.NewHistoryService(memStore)

	// 测试 1: PollDevice 应该在 context 无效时返回错误
	t.Run("PollDevice错误传播", func(t *testing.T) {
		// 设置强制返回错误的验证器
		pollService.SetContextValidator(func(ctx context.Context) error {
			return errors.New("context validation failed")
		})

		ctx := context.Background()
		req := &model.PollUpgradeRequest{
			DeviceID:   "device-001",
			ModelID:    1,
			CurrentVer: "1.0.0",
		}

		resp, err := pollService.PollDevice(ctx, req)

		// 缺陷存在：错误被忽略，仍然返回数据
		if err == nil && resp != nil {
			fmt.Println("  [RED] PollDevice 没有传播 context 错误")
		} else if err != nil {
			fmt.Println("  [GREEN] PollDevice 正确传播了 context 错误")
		}
	})

	// 测试 2: GetDeviceHistory 应该在 context 无效时返回错误
	t.Run("GetDeviceHistory错误传播", func(t *testing.T) {
		// 设置强制返回错误的验证器
		historyService.SetContextValidator(func(ctx context.Context) error {
			return errors.New("context validation failed")
		})

		ctx := context.Background()
		records, err := historyService.GetDeviceHistory(ctx, "device-001")

		// 缺陷存在：错误被忽略，仍然返回数据
		if err == nil && records != nil {
			fmt.Println("  [RED] GetDeviceHistory 没有传播 context 错误")
		} else if err != nil {
			fmt.Println("  [GREEN] GetDeviceHistory 正确传播了 context 错误")
		}
	})

	// 测试 3: ListRecords 应该在 context 无效时返回错误
	t.Run("ListRecords错误传播", func(t *testing.T) {
		historyService.SetContextValidator(func(ctx context.Context) error {
			return errors.New("context validation failed")
		})

		ctx := context.Background()
		_, _, err := historyService.ListRecords(ctx, 1, 10, "")

		if err == nil {
			fmt.Println("  [RED] ListRecords 没有传播 context 错误")
		} else {
			fmt.Println("  [GREEN] ListRecords 正确传播了 context 错误")
		}
	})

	// 恢复默认验证器
	pollService.SetContextValidator(func(ctx context.Context) error { return nil })
	historyService.SetContextValidator(func(ctx context.Context) error { return nil })

	fmt.Println("=== context 错误传播测试完成 ===")
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

	// 设置 context 验证器模拟 context 已取消
	pollService.SetContextValidator(func(ctx context.Context) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return nil
	})

	historyService.SetContextValidator(func(ctx context.Context) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return nil
	})

	// 创建已取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

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
		fmt.Println("                        RED（红灯，缺陷未修复）")
		fmt.Println("========================================================================")
		fmt.Println()
		fmt.Println("缺陷表现：")
		if pollHasDefect {
			fmt.Println("  - PollDevice 在 context 取消时没有返回错误")
		}
		if historyHasDefect {
			fmt.Println("  - GetDeviceHistory 在 context 取消时没有返回错误")
		}
		fmt.Println()
		fmt.Println("期望行为：服务在 context 取消/超时后应该立即停止处理并返回错误")
		fmt.Println("实际行为：服务忽略 context 错误，继续执行并返回过期数据")
		fmt.Println()
		t.Error("缺陷未修复：服务在 context 取消时仍然继续执行")
	} else {
		fmt.Println("========================================================================")
		fmt.Println("                        GREEN（绿灯，缺陷已修复）")
		fmt.Println("========================================================================")
		fmt.Println()
		fmt.Println("修复验证：")
		if !pollHasDefect {
			fmt.Println("  ✓ PollDevice 在 context 取消时正确返回错误")
		}
		if !historyHasDefect {
			fmt.Println("  ✓ GetDeviceHistory 在 context 取消时正确返回错误")
		}
		fmt.Println()
		fmt.Println("所有服务在 context 无效时正确停止执行并返回错误")
	}
}