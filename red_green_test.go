package fwupgrade

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	ctx := context.Background()
	hasDefect := false

	// ====== 测试1: 设备活跃状态检测 ======
	t.Run("IsActive with zero LastSeenAt", func(t *testing.T) {
		// 直接创建设备，LastSeenAt 保持零值
		d := &model.Device{
			DeviceID: "device-001",
			ModelID:  1,
			ModelName: "Router",
			Name:     "LivingRoom",
			Status:   model.DeviceOnline,
		}
		if d.LastSeenAt.IsZero() {
			// 设备从未被看到过
			if d.IsActive() {
				fmt.Println("RED（红灯，缺陷未修复）: IsActive 对零值 LastSeenAt 返回 true")
				hasDefect = true
			} else {
				fmt.Println("GREEN: IsActive 正确地对零值 LastSeenAt 返回 false")
			}
		} else {
			t.Fatal("设备应具有零值 LastSeenAt")
		}
	})

	// ====== 测试2: 统计服务成功率计算 ======
	t.Run("Success rate with all in-progress records", func(t *testing.T) {
		memStore := store.NewMemoryStore()
		memStore.Init(ctx)

		// 创建一些升级记录，全部为进行中状态
		for i := 0; i < 5; i++ {
			rec := model.NewUpgradeRecord(
				fmt.Sprintf("device-%03d", i),
				fmt.Sprintf("Device %03d", i),
				1,
				"TestTask",
				"v1.0.0",
				"v2.0.0",
			)
			rec.Status = model.UpgradeInProgress
			memStore.CreateRecord(ctx, rec)
		}

		statsSvc := service.NewStatsService(memStore, memStore, memStore, memStore, memStore)
		dashboard, err := statsSvc.GetDashboard(ctx)
		if err != nil {
			t.Fatalf("GetDashboard 失败: %v", err)
		}

		if dashboard.SuccessRate != dashboard.SuccessRate {
			// NaN != NaN 为 true
			fmt.Println("RED（红灯，缺陷未修复）: 成功率计算产生 NaN（除零错误）")
			hasDefect = true
		} else if math.IsNaN(dashboard.SuccessRate) {
			fmt.Println("RED（红灯，缺陷未修复）: 成功率为 NaN")
			hasDefect = true
		} else {
			// 当所有记录都进行中时，成功率应该返回 0 而不是 NaN
			if dashboard.SuccessRate == 0 {
				fmt.Println("GREEN: 所有记录进行中时正确返回 0 成功率")
			} else {
				fmt.Printf("成功率: %f\n", dashboard.SuccessRate)
				fmt.Println("RED（红灯，缺陷未修复）: 异常的成功率值")
				hasDefect = true
			}
		}
	})

	// ====== 测试3: 超时检测逻辑 ======
	t.Run("Timeout detection with recent records", func(t *testing.T) {
		memStore := store.NewMemoryStore()
		memStore.Init(ctx)

		// 创建一个刚刚开始的升级记录（应该还没超时）
		recentRec := model.NewUpgradeRecord("device-recent", "Recent Device", 1, "Task", "v1.0", "v2.0")
		recentRec.Status = model.UpgradeInProgress
		recentRec.StartedAt = time.Now() // 刚刚开始
		memStore.CreateRecord(ctx, recentRec)

		// 创建一个早已开始的升级记录（应该已超时）
		oldRec := model.NewUpgradeRecord("device-old", "Old Device", 1, "Task", "v1.0", "v2.0")
		oldRec.Status = model.UpgradeInProgress
		oldRec.StartedAt = time.Now().Add(-2 * time.Hour) // 2小时前开始
		memStore.CreateRecord(ctx, oldRec)

		progressSvc := service.NewProgressService(memStore, memStore, memStore)
		timeout := 30 * time.Minute
		timedOut := progressSvc.TimeOutCheck(ctx, timeout)

		// 只有 oldRec 应该超时
		if len(timedOut) == 1 && timedOut[0].DeviceID == "device-old" {
			fmt.Println("GREEN: 超时检测正确识别了超时记录")
		} else if len(timedOut) == 2 {
			// 缺陷：recentRec 也被标记为超时
			fmt.Println("RED（红灯，缺陷未修复）: 新近记录也被错误标记为超时")
			hasDefect = true
		} else if len(timedOut) == 0 {
			// 缺陷：没有记录被标记为超时
			fmt.Println("RED（红灯，缺陷未修复）: 真正超时的记录未被检测到")
			hasDefect = true
		} else {
			fmt.Printf("RED（红灯，缺陷未修复）: 超时检测结果异常，共 %d 条记录被标记为超时\n", len(timedOut))
			hasDefect = true
		}
	})

	// ====== 最终判定 ======
	fmt.Println()
	if hasDefect {
		fmt.Println("========================================")
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("========================================")
		t.FailNow()
	} else {
		fmt.Println("========================================")
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		fmt.Println("========================================")
	}
}
