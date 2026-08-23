package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/internal/service"
)

func TestRedGreen(t *testing.T) {
	memStore := store.NewMemoryStore()
	memStore.Init(context.Background())

	nonExistentPath := "/non-existent/dir/config_should_not_exist.txt"

	_, err := service.NewFirmwareServiceFromConfig(memStore, memStore, nonExistentPath)

	if err == nil {
		t.Fatal("expected error for non-existent config file, got nil")
	}

	isConfigNotFound := errors.Is(err, config.ErrConfigFileNotFound)
	isFileNotFound := errors.Is(err, os.ErrNotExist)

	if isConfigNotFound && isFileNotFound {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
		t.Log("PASS: errors.Is correctly identifies both ErrConfigFileNotFound and os.ErrNotExist")
		t.Logf("Error chain preserved: %v", err)
	} else {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Errorf("FAIL: errors.Is failed to identify error types")
		t.Errorf("  errors.Is(err, config.ErrConfigFileNotFound) = %v", isConfigNotFound)
		t.Errorf("  errors.Is(err, os.ErrNotExist) = %v", isFileNotFound)
		t.Errorf("  actual error: %v", err)
	}
}

func TestRedGreenUploadPath(t *testing.T) {
	memStore := store.NewMemoryStore()
	memStore.Init(context.Background())

	svc := service.NewFirmwareService(memStore, memStore, config.DefaultConfig())

	req := &model.UploadFirmwareRequest{
		ModelID:   1,
		Version:   "1.0.0",
		Md5:       "abc123",
		Changelog: "test",
	}

	nonExistentPath := "/non-existent/dir/config_should_not_exist.txt"

	_, err := svc.UploadFirmwareWithConfig(
		context.Background(),
		nonExistentPath,
		req,
		[]byte("test"),
		"test.bin",
	)

	if err == nil {
		t.Fatal("expected error for non-existent config file, got nil")
	}

	isConfigNotFound := errors.Is(err, config.ErrConfigFileNotFound)
	isFileNotFound := errors.Is(err, os.ErrNotExist)

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
