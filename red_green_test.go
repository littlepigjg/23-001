package fwupgrade

import (
	"context"
	"fmt"
	"os"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	pass := true
	var failures []string

	memStore := store.NewMemoryStore()
	cfg := config.DefaultConfig()
	svc := service.NewDeviceService(memStore, memStore, cfg)
	ctx := context.Background()

	devModel := model.NewDeviceModel("TestModel", "TestManufacturer", "v1.0", "Test device model")
	if err := memStore.CreateModel(ctx, devModel); err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	t.Run("CaseInsensitiveCreatePreventsDuplicate", func(t *testing.T) {
		_, err := svc.CreateDevice(ctx, &model.CreateDeviceRequest{
			DeviceID:     "DEVICE001",
			ModelID:      devModel.ID,
			Name:         "Device One",
			IPAddress:    "192.168.1.1",
			SerialNumber: "SN001",
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("CaseInsensitiveCreate: first create failed: %v", err))
			pass = false
			return
		}

		_, err = svc.CreateDevice(ctx, &model.CreateDeviceRequest{
			DeviceID:     "device001",
			ModelID:      devModel.ID,
			Name:         "Device One Lowercase",
			IPAddress:    "192.168.1.2",
			SerialNumber: "SN001_L",
		})
		if err == nil {
			failures = append(failures, "CaseInsensitiveCreate: second create with different case should have failed but succeeded")
			pass = false
		}
	})

	t.Run("CaseInsensitiveGetByDeviceID", func(t *testing.T) {
		_, err := svc.GetDeviceByDeviceID(ctx, "device001")
		if err != nil {
			failures = append(failures, fmt.Sprintf("CaseInsensitiveGet: lookup with lowercase failed: %v", err))
			pass = false
		}

		_, err = svc.GetDeviceByDeviceID(ctx, "DEVICE001")
		if err != nil {
			failures = append(failures, fmt.Sprintf("CaseInsensitiveGet: lookup with original case failed: %v", err))
			pass = false
		}

		_, err = svc.GetDeviceByDeviceID(ctx, "Device001")
		if err != nil {
			failures = append(failures, fmt.Sprintf("CaseInsensitiveGet: lookup with mixed case failed: %v", err))
			pass = false
		}
	})

	t.Run("CaseInsensitiveRegisterPreventsDuplicate", func(t *testing.T) {
		memStore2 := store.NewMemoryStore()
		svc2 := service.NewDeviceService(memStore2, memStore2, cfg)
		model2 := model.NewDeviceModel("TestModel2", "TestManufacturer", "v1.0", "Test device model 2")
		if err := memStore2.CreateModel(ctx, model2); err != nil {
			t.Fatalf("Failed to create model: %v", err)
		}

		_, err := svc2.RegisterDevice(ctx, &model.RegisterDeviceRequest{
			DeviceID:  "SENSOR001",
			ModelID:   model2.ID,
			Name:      "Sensor One",
			ModelName: "TestModel2",
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("CaseInsensitiveRegister: first register failed: %v", err))
			pass = false
			return
		}

		_, err = svc2.RegisterDevice(ctx, &model.RegisterDeviceRequest{
			DeviceID:  "sensor001",
			ModelID:   model2.ID,
			Name:      "Sensor One Updated",
			ModelName: "TestModel2",
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("CaseInsensitiveRegister: second register failed: %v", err))
			pass = false
			return
		}

		_, total, err := memStore2.ListDevices(ctx, 1, 100, 0, "")
		if err != nil {
			failures = append(failures, fmt.Sprintf("CaseInsensitiveRegister: list devices failed: %v", err))
			pass = false
			return
		}
		if total != 1 {
			failures = append(failures, fmt.Sprintf("CaseInsensitiveRegister: expected 1 device, got %d (duplicate created)", total))
			pass = false
		}
	})

	t.Run("CaseInsensitiveDelete", func(t *testing.T) {
		memStore3 := store.NewMemoryStore()
		svc3 := service.NewDeviceService(memStore3, memStore3, cfg)
		model3 := model.NewDeviceModel("TestModel3", "TestManufacturer", "v1.0", "Test device model 3")
		if err := memStore3.CreateModel(ctx, model3); err != nil {
			t.Fatalf("Failed to create model: %v", err)
		}

		dev, err := svc3.CreateDevice(ctx, &model.CreateDeviceRequest{
			DeviceID:     "TempDevice",
			ModelID:      model3.ID,
			Name:         "Temp Device",
			IPAddress:    "10.0.0.1",
			SerialNumber: "TEMP001",
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("CaseInsensitiveDelete: create failed: %v", err))
			pass = false
			return
		}

		err = memStore3.DeleteDevice(ctx, dev.ID)
		if err != nil {
			failures = append(failures, fmt.Sprintf("CaseInsensitiveDelete: delete failed: %v", err))
			pass = false
			return
		}

		_, err = svc3.GetDeviceByDeviceID(ctx, "TempDevice")
		if err == nil {
			failures = append(failures, "CaseInsensitiveDelete: device should have been deleted but was found")
			pass = false
		}
	})

	if pass {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		os.Exit(0)
	} else {
		fmt.Println("RED（红灯，缺陷未修复）")
		fmt.Println("Failures:")
		for _, f := range failures {
			fmt.Println("  - " + f)
		}
		os.Exit(1)
	}
}
