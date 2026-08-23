package fwupgrade

import (
	"context"
	"fmt"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	ms := store.NewMemoryStore()
	ctx := context.Background()

	_ = ms.Init(ctx)

	m := model.NewDeviceModel("TestModel", "TestMfg", "v1", "test model")
	_ = ms.CreateModel(ctx, m)

	fw := &model.Firmware{
		ModelID:  m.ID,
		Version:  "1.0.1",
		Md5:      "abc123",
		Size:     1024,
		FilePath: "/tmp/fw.bin",
	}
	_ = ms.CreateFirmware(ctx, fw)

	deviceIDs := []string{"dev001", "dev002", "dev003", "dev004", "dev005", "dev006", "dev007", "dev008", "dev009", "dev010"}
	for _, did := range deviceIDs {
		d := model.NewDevice(did, m.ID, m.Name, "Device-"+did, "192.168.1.1", "sn-"+did)
		_ = ms.CreateDevice(ctx, d)
	}

	defectExists := false
	output := ""

	gs := service.NewGrayscaleService(ms, cfg)
	gs.SetRatioFn(func(deviceID string) float64 {
		for i := 0; i < 5; i++ {
			if deviceIDs[i] == deviceID {
				return 100.0
			}
		}
		return 0.0
	})

	ids := make([]string, len(deviceIDs))
	copy(ids, deviceIDs)
	original := []string{"dev001", "dev002", "dev003", "dev004", "dev005", "dev006", "dev007", "dev008", "dev009", "dev010"}

	grayGroup, waitGroup := gs.GenerateDeviceGroup(ids, 50.0)
	output += fmt.Sprintf("grayGroup=%v waitGroup=%v ", grayGroup, waitGroup)

	corrupted := false
	for i := range ids {
		if ids[i] != original[i] {
			corrupted = true
			break
		}
	}
	if corrupted {
		output += fmt.Sprintf("input_corrupted ids=%v | ", ids)
		defectExists = true
	} else {
		output += "input_preserved | "
	}

	gs2 := service.NewGrayscaleService(ms, cfg)
	gs2.SetGrayscaleGuard(func(deviceID string, ratio float64) bool {
		return true
	})
	gs2.SetRatioFn(func(deviceID string) float64 {
		for i := 0; i < 3; i++ {
			if deviceIDs[i] == deviceID {
				return 100.0
			}
		}
		return 0.0
	})

	testIDs := []string{"a", "b", "c", "d", "e"}
	snapshot := make([]string, len(testIDs))
	copy(snapshot, testIDs)

	gray, wait := gs2.GenerateDeviceGroup(testIDs, 60.0)
	output += fmt.Sprintf("test2: input=%v gray=%v wait=%v ", snapshot, gray, wait)

	corrupted2 := false
	for i := range testIDs {
		if testIDs[i] != snapshot[i] {
			corrupted2 = true
			break
		}
	}
	if corrupted2 {
		output += fmt.Sprintf("input_corrupted testIDs=%v | ", testIDs)
		defectExists = true
	} else {
		output += "input_preserved | "
	}

	if len(gray)+len(wait) != len(snapshot) {
		output += fmt.Sprintf("count_mismatch gray=%d+wait=%d!=%d | ", len(gray), len(wait), len(snapshot))
		defectExists = true
	}

	ts := service.NewTaskService(ms, ms, ms, ms, ms, cfg)
	ts.SetGrayscaleService(gs)

	task := model.NewUpgradeTask(
		"TestGrayTask", "grayscale test task",
		m.ID, m.Name, fw.ID, fw.Version,
		model.TaskTypeGrayscale, 50.0, nil, "admin",
	)
	_ = ms.CreateTask(ctx, task)

	err := ts.StartTask(ctx, task.ID)
	if err != nil {
		output += fmt.Sprintf("StartTask error=%v | ", err)
		defectExists = true
	} else {
		output += "StartTask_ok | "
	}

	if defectExists {
		fmt.Printf("RED（红灯，缺陷未修复）: %s\n", output)
		t.Fail()
	} else {
		fmt.Printf("GREEN（绿灯，缺陷已修复）: %s\n", output)
	}
}
