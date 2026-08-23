package fwupgrade_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fwupgrade-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		t.Fatal(err)
	}
	origLimit := rLimit
	defer syscall.Setrlimit(syscall.RLIMIT_NOFILE, &origLimit)

	rLimit.Cur = 256
	rLimit.Max = 256
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Storage.Type = "file"
	cfg.Storage.DataDir = filepath.Join(tmpDir, "data")
	cfg.Storage.UploadDir = filepath.Join(tmpDir, "uploads")
	cfg.Firmware.RequireMD5 = false
	cfg.Firmware.AllowedExts = ".bin,.hex,.img"

	fs := store.NewFileStore(cfg)
	ctx := context.Background()
	if err := fs.Init(ctx); err != nil {
		t.Fatal(err)
	}
	defer fs.Close()

	m := model.NewDeviceModel("TestModel", "TestMfr", "v1.0", "test model")
	if err := fs.CreateModel(ctx, m); err != nil {
		t.Fatal(err)
	}

	firmwareDir := filepath.Join(tmpDir, "firmware")
	if err := os.MkdirAll(firmwareDir, 0755); err != nil {
		t.Fatal(err)
	}

	itemCount := 300
	items := make([]model.FirmwareBatchItem, 0, itemCount)
	for i := 0; i < itemCount; i++ {
		fwPath := filepath.Join(firmwareDir, fmt.Sprintf("fw_%04d.bin", i))
		data := []byte(fmt.Sprintf("firmware_binary_data_%d", i))
		if err := os.WriteFile(fwPath, data, 0644); err != nil {
			t.Fatal(err)
		}
		items = append(items, model.FirmwareBatchItem{
			ModelID:  m.ID,
			Version:  fmt.Sprintf("v1.%d", i),
			FilePath: fwPath,
		})
	}

	svc := service.NewFirmwareService(fs, fs, cfg)
	result, err := svc.BatchUploadFirmware(ctx, items)
	if err != nil {
		fmt.Println("RED（红灯，缺陷未修复）- batch upload returned error:", err)
		return
	}

	if result != nil && result.FailCount > 0 {
		fmt.Printf("RED（红灯，缺陷未修复）- success=%d, fail=%d\n", result.SuccessCount, result.FailCount)
		return
	}

	if result == nil || result.SuccessCount != itemCount {
		fmt.Printf("RED（红灯，缺陷未修复）- unexpected result: %+v\n", result)
		return
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）- all", itemCount, "items uploaded successfully")
}
