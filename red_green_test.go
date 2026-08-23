package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/internal/service"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Storage.Type = "memory"
	cfg.Firmware.RequireMD5 = false

	memStore := store.NewMemoryStore()
	ctx := context.Background()

	if err := memStore.Init(ctx); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer memStore.Close()

	modelStore := memStore
	firmwareStore := memStore

	modelSvc := service.NewDeviceModelService(modelStore, cfg)
	firmwareSvc := service.NewFirmwareService(firmwareStore, modelStore, cfg)

	createReq := &model.CreateModelRequest{
		Name:        "TestModel",
		Manufacturer: "TestManufacturer",
		HardwareVer: "v1.0",
		Description: "Test Model Description",
	}

	testModel, err := modelSvc.CreateModel(ctx, createReq)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	tests := []struct {
		name        string
		version     string
		expectError bool
		description string
	}{
		{
			name:        "Valid version",
			version:     "1.0.0",
			expectError: false,
			description: "Normal semantic version should pass validation",
		},
		{
			name:        "Valid version with dashes",
			version:     "1.0.0-beta",
			expectError: false,
			description: "Version with dashes should pass validation",
		},
		{
			name:        "Malicious version with path traversal",
			version:     "1.0.0/../../../etc/passwd",
			expectError: true,
			description: "Version with path traversal should be rejected",
		},
		{
			name:        "Malicious version with dotslashes",
			version:     "../etc/shadow",
			expectError: true,
			description: "Version with ../ should be rejected",
		},
		{
			name:        "Version with absolute path",
			version:     "/etc/passwd",
			expectError: true,
			description: "Version with absolute path should be rejected",
		},
	}

	allPassed := true
	var testResults []string

	for _, tc := range tests {
		req := &model.UploadFirmwareRequest{
			ModelID:   testModel.ID,
			Version:   tc.version,
			Md5:       "",
			Changelog: "Test changelog",
		}

		fileData := []byte("test firmware data")
		originalFilename := "test_firmware.bin"

		_, err := firmwareSvc.UploadFirmware(ctx, req, fileData, originalFilename)

		if tc.expectError && err == nil {
			testResults = append(testResults, fmt.Sprintf("FAIL: %s - %s (version '%s' should have been rejected but was accepted)", tc.name, tc.description, tc.version))
			allPassed = false
		} else if !tc.expectError && err != nil {
			testResults = append(testResults, fmt.Sprintf("FAIL: %s - %s (version '%s' should have been accepted but got error: %v)", tc.name, tc.description, tc.version, err))
			allPassed = false
		} else {
			testResults = append(testResults, fmt.Sprintf("PASS: %s - %s", tc.name, tc.description))
		}
	}

	// Also test ValidateVersionFormat directly
	t.Run("ValidateVersionFormat_Direct", func(t *testing.T) {
		maliciousVersions := []string{
			"1.0.0/../../../etc/passwd",
			"../etc/shadow",
			"/etc/passwd",
		}

		for _, ver := range maliciousVersions {
			err := model.ValidateVersionFormat(ver)
			if err == nil {
				testResults = append(testResults, fmt.Sprintf("FAIL: ValidateVersionFormat accepted malicious version '%s'", ver))
				allPassed = false
			} else {
				testResults = append(testResults, fmt.Sprintf("PASS: ValidateVersionFormat rejected malicious version '%s'", ver))
			}
		}
	})

	// Test SanitizeVersion with path traversal characters
	t.Run("SanitizeVersion_Security", func(t *testing.T) {
		maliciousVersion := "1.0.0/../../../etc/passwd"
		sanitized := model.SanitizeVersion(maliciousVersion)

		if sanitized == maliciousVersion {
			testResults = append(testResults, "FAIL: SanitizeVersion did not remove path traversal characters")
			allPassed = false
		} else {
			testResults = append(testResults, "PASS: SanitizeVersion removed path traversal characters")
		}
	})

	fmt.Println("\n=== Test Results ===")
	for _, result := range testResults {
		fmt.Println(result)
	}

	// Cleanup uploaded files
	uploadDir := filepath.Join(".", "uploads", fmt.Sprintf("model_%d", testModel.ID))
	os.RemoveAll(uploadDir)

	if allPassed {
		fmt.Println("\nGREEN（绿灯，缺陷已修复）")
	} else {
		fmt.Println("\nRED（红灯，缺陷未修复）")
	}

	if !allPassed {
		t.Errorf("Tests failed - RED (defect not fixed). Some malicious versions were not properly rejected.")
	}
}

func TestFirmwareValidate(t *testing.T) {
	// Test that Firmware.Validate properly checks version format
	validFirmware := &model.Firmware{
		ModelID:  1,
		Version:  "1.0.0",
		Md5:      fmt.Sprintf("%x", md5.Sum([]byte("test"))),
		Size:     100,
		FilePath: "/tmp/test.bin",
	}

	if err := validFirmware.Validate(); err != nil {
		t.Errorf("Valid firmware should pass validation: %v", err)
	}

	maliciousVersions := []string{
		"1.0.0/../../../etc/passwd",
		"../etc/shadow",
		"/etc/passwd",
	}

	allRejected := true
	for _, ver := range maliciousVersions {
		fw := &model.Firmware{
			ModelID:  1,
			Version:  ver,
			Md5:      fmt.Sprintf("%x", md5.Sum([]byte("test"))),
			Size:     100,
			FilePath: "/tmp/test.bin",
		}

		if err := fw.Validate(); err == nil {
			allRejected = false
			t.Logf("BUG: Malicious version '%s' was accepted", ver)
		}
	}

	if !allRejected {
		fmt.Println("RED（红灯，缺陷未修复）- Firmware.Validate accepted malicious versions")
		t.Error("Firmware.Validate did not reject malicious versions with path traversal characters")
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）- Firmware.Validate correctly rejected all malicious versions")
	}

	_ = time.Now() // ensure time import is used
}
