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

func TestRedGreen(t *testing.T) {
	t.Run("firmware分页越界", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("RED（红灯，缺陷未修复）: 固件分页page=0触发slice越界 panic: %v", r)
				t.FailNow()
			}
		}()

		memStore := store.NewMemoryStore()
		ctx := context.Background()

		m := model.NewDeviceModel("TestModel", "TestMfr", "v1.0", "Test device model")
		if err := memStore.CreateModel(ctx, m); err != nil {
			t.Fatal(err)
		}

		for i := 0; i < 3; i++ {
			fw := model.NewFirmware(m.ID, m.Name, fmt.Sprintf("v%d", i), "abc123def456", 2048, "/tmp/fw.bin", time.Now(), "test changelog")
			if err := memStore.CreateFirmware(ctx, fw); err != nil {
				t.Fatal(err)
			}
		}

		cfg := config.DefaultConfig()
		fwService := service.NewFirmwareService(memStore, memStore, cfg)

		_, _, err := fwService.ListFirmwares(ctx, 0, 10, m.ID)
		if err != nil {
			t.Logf("RED（红灯，缺陷未修复）: 固件分页page=0返回错误: %v", err)
			t.FailNow()
		}

		_, _, err = fwService.ListFirmwares(ctx, 100, 10, m.ID)
		if err != nil {
			t.Logf("RED（红灯，缺陷未修复）: 固件分页page=100返回错误: %v", err)
			t.FailNow()
		}

		t.Log("GREEN（绿灯，缺陷已修复）")
	})

	t.Run("record分页越界", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("RED（红灯，缺陷未修复）: 记录分页page=0触发slice越界 panic: %v", r)
				t.FailNow()
			}
		}()

		memStore := store.NewMemoryStore()
		ctx := context.Background()

		for i := 0; i < 3; i++ {
			rec := model.NewUpgradeRecord(fmt.Sprintf("dev-%d", i), fmt.Sprintf("Device %d", i), model.ID(i+1), fmt.Sprintf("Task-%d", i), "v1.0", "v2.0")
			if err := memStore.CreateRecord(ctx, rec); err != nil {
				t.Fatal(err)
			}
		}

		historyService := service.NewHistoryService(memStore)

		_, _, err := historyService.ListRecords(ctx, 0, 10, "")
		if err != nil {
			t.Logf("RED（红灯，缺陷未修复）: 记录分页page=0返回错误: %v", err)
			t.FailNow()
		}

		_, _, err = historyService.ListRecords(ctx, 100, 10, "")
		if err != nil {
			t.Logf("RED（红灯，缺陷未修复）: 记录分页page=100返回错误: %v", err)
			t.FailNow()
		}

		t.Log("GREEN（绿灯，缺陷已修复）")
	})

	t.Run("store直接分页调用", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("RED（红灯，缺陷未修复）: store层分页触发slice越界 panic: %v", r)
				t.FailNow()
			}
		}()

		memStore := store.NewMemoryStore()
		ctx := context.Background()

		m := model.NewDeviceModel("DirectModel", "Mfr", "v1", "desc")
		memStore.CreateModel(ctx, m)

		for i := 0; i < 3; i++ {
			fw := model.NewFirmware(m.ID, m.Name, fmt.Sprintf("v%d", i), "md5", 100, "/tmp/fw", time.Now(), "changelog")
			memStore.CreateFirmware(ctx, fw)
		}

		_, _, err := memStore.ListFirmwares(ctx, 0, 10, m.ID)
		if err != nil {
			t.Logf("RED（红灯，缺陷未修复）: store层page=0返回错误: %v", err)
			t.FailNow()
		}

		_, _, err = memStore.ListFirmwares(ctx, 100, 10, m.ID)
		if err != nil {
			t.Logf("RED（红灯，缺陷未修复）: store层page=100返回错误: %v", err)
			t.FailNow()
		}

		t.Log("GREEN（绿灯，缺陷已修复）")
	})

	t.Run("正常分页不受影响", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("RED（红灯，缺陷未修复）: 正常分页触发panic: %v", r)
				t.FailNow()
			}
		}()

		memStore := store.NewMemoryStore()
		ctx := context.Background()

		m := model.NewDeviceModel("NormalModel", "Mfr", "v1", "desc")
		memStore.CreateModel(ctx, m)

		for i := 0; i < 5; i++ {
			fw := model.NewFirmware(m.ID, m.Name, fmt.Sprintf("v%d", i), "md5", 100, "/tmp/fw", time.Now(), "changelog")
			memStore.CreateFirmware(ctx, fw)
		}

		cfg := config.DefaultConfig()
		fwService := service.NewFirmwareService(memStore, memStore, cfg)

		results, total, err := fwService.ListFirmwares(ctx, 1, 2, m.ID)
		if err != nil {
			t.Logf("RED（红灯，缺陷未修复）: 正常分页page=1返回错误: %v", err)
			t.FailNow()
		}
		if total != 5 {
			t.Logf("RED（红灯，缺陷未修复）: 总数不匹配，期望5，实际%d", total)
			t.FailNow()
		}
		if len(results) != 2 {
			t.Logf("RED（红灯，缺陷未修复）: 每页数量不匹配，期望2，实际%d", len(results))
			t.FailNow()
		}

		results, total, err = fwService.ListFirmwares(ctx, 3, 2, m.ID)
		if err != nil {
			t.Logf("RED（红灯，缺陷未修复）: 正常分页page=3返回错误: %v", err)
			t.FailNow()
		}
		if len(results) != 1 {
			t.Logf("RED（红灯，缺陷未修复）: 最后一页数量不匹配，期望1，实际%d", len(results))
			t.FailNow()
		}

		t.Log("GREEN（绿灯，缺陷已修复）")
	})
}
