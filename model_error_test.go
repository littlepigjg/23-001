package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestModelServiceErrorChain(t *testing.T) {
	memStore := store.NewMemoryStore()
	memStore.Init(context.Background())

	svc := service.NewDeviceModelService(memStore, config.DefaultConfig())

	req := &model.CreateModelRequest{
		Name:        "TestModel",
		Manufacturer: "TestManufacturer",
		HardwareVer: "v1.0",
	}

	nonExistentPath := "/non-existent/dir/config_should_not_exist.txt"

	_, err := svc.CreateModelWithConfig(context.Background(), nonExistentPath, req)

	if err == nil {
		t.Fatal("expected error for non-existent config file, got nil")
	}

	isConfigNotFound := errors.Is(err, config.ErrConfigFileNotFound)
	isFileNotFound := errors.Is(err, os.ErrNotExist)

	fmt.Printf("Error: %v\n", err)
	fmt.Printf("errors.Is(err, config.ErrConfigFileNotFound) = %v\n", isConfigNotFound)
	fmt.Printf("errors.Is(err, os.ErrNotExist) = %v\n", isFileNotFound)

	if isConfigNotFound && isFileNotFound {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
		t.Log("PASS: errors.Is correctly identifies both ErrConfigFileNotFound and os.ErrNotExist")
	} else {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Errorf("FAIL: errors.Is failed to identify error types")
		t.Errorf("  errors.Is(err, config.ErrConfigFileNotFound) = %v", isConfigNotFound)
		t.Errorf("  errors.Is(err, os.ErrNotExist) = %v", isFileNotFound)
		t.Errorf("  actual error: %v", err)
	}
}
