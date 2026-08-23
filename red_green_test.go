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

// TestRedGreen 缺陷验证测试
// 当缺陷存在时输出 RED（红灯），缺陷修复后输出 GREEN（绿灯）
func TestRedGreen(t *testing.T) {
	pollOK := testPollServiceLifecycle(t)
	taskOK := testTaskServiceLifecycle(t)

	if pollOK && taskOK {
		fmt.Println("=== GREEN（绿灯，缺陷已修复）===")
	} else {
		fmt.Println("=== RED（红灯，缺陷未修复）===")
		t.Fail()
	}
}

// testPollServiceLifecycle 测试 PollService 的生命周期管理是否有 goroutine 泄漏
func testPollServiceLifecycle(t *testing.T) bool {
	lc := service.NewServiceLifecycle()
	ctx := lc.Start(context.Background())

	memStore := store.NewMemoryStore()
	cfg := config.DefaultConfig()

	gs := service.NewGrayscaleService(memStore, cfg)
	ps := service.NewPollService(memStore, memStore, memStore, memStore, gs)
	ps.SetLifecycle(lc)

	// 创建足够多的设备以触发多个批次
	for i := 0; i < 200; i++ {
		device := &model.Device{
			DeviceID: fmt.Sprintf("test-device-%d", i),
			Name:     fmt.Sprintf("Test Device %d", i),
			ModelID:  1,
			Status:   model.DeviceOnline,
		}
		_ = memStore.CreateDevice(ctx, device)
	}

	// 启动轮询，短间隔以快速产生goroutine
	go ps.SchedulePolling(ctx, 10*time.Millisecond)

	// 等待足够的tick来产生多个批次goroutine
	time.Sleep(15 * time.Millisecond)

	// 停止生命周期：取消context并等待goroutine清理
	// 如果goroutine未正确调用Done()，Stop将超时
	err := lc.Stop(3 * time.Second)
	if err != nil {
		t.Logf("RED: PollService goroutine leak detected - %v", err)
		return false
	}
	t.Log("GREEN: PollService lifecycle shutdown clean")
	return true
}

// testTaskServiceLifecycle 测试 TaskService 的生命周期管理是否有 goroutine 泄漏
func testTaskServiceLifecycle(t *testing.T) bool {
	lc := service.NewServiceLifecycle()
	ctx := lc.Start(context.Background())

	memStore := store.NewMemoryStore()
	cfg := config.DefaultConfig()

	ts := service.NewTaskService(memStore, memStore, memStore, memStore, memStore, cfg)
	ts.SetLifecycle(lc)

	// 创建型号
	m := &model.DeviceModel{
		Name:         "Test Model",
		Manufacturer: "Test Mfg",
		IsActive:     true,
	}
	_ = memStore.CreateModel(ctx, m)

	// 创建固件
	fw := &model.Firmware{
		ModelID:     m.ID,
		Version:     "1.0.0",
		IsActive:    true,
		ReleaseDate: time.Now(),
	}
	_ = memStore.CreateFirmware(ctx, fw)

	// 创建设备
	for i := 0; i < 5; i++ {
		device := &model.Device{
			DeviceID: fmt.Sprintf("task-test-device-%d", i),
			Name:     fmt.Sprintf("Task Device %d", i),
			ModelID:  m.ID,
			Status:   model.DeviceOnline,
		}
		_ = memStore.CreateDevice(ctx, device)
	}

	// 创建多个已到期的计划任务
	now := time.Now()
	for i := 0; i < 3; i++ {
		task := model.NewUpgradeTask(
			fmt.Sprintf("Test Task %d", i),
			fmt.Sprintf("Test task %d for leak detection", i),
			m.ID, m.Name, fw.ID, fw.Version,
			model.TaskTypeFull, 100, nil, "tester",
		)
		task.ScheduledAt = now.Add(-1 * time.Hour)
		task.Status = model.TaskPending
		_ = memStore.CreateTask(ctx, task)
	}

	// 启动计划任务处理
	_ = ts.ProcessScheduledTasks(ctx)

	// 等待goroutine启动并进入sleep窗口
	time.Sleep(10 * time.Millisecond)

	// 停止生命周期：取消context并等待goroutine清理
	err := lc.Stop(3 * time.Second)
	if err != nil {
		t.Logf("RED: TaskService goroutine leak detected - %v", err)
		return false
	}
	t.Log("GREEN: TaskService lifecycle shutdown clean")
	return true
}
