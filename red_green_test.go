package fwupgrade

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/md5util"
)

func TestRedGreen(t *testing.T) {
	tmpDir := t.TempDir()
	uploadDir := filepath.Join(tmpDir, "uploads")

	cfg := config.DefaultConfig()
	cfg.Storage.UploadDir = uploadDir
	cfg.Storage.DataDir = filepath.Join(tmpDir, "data")

	memStore := store.NewMemoryStore()

	svc := service.NewFirmwareService(memStore, memStore, cfg)

	ctx := context.Background()

	testModel := model.NewDeviceModel("test_device", "TestMfg", "HW-v1", "Test model for firmware upload")
	err := memStore.CreateModel(ctx, testModel)
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	modelID := testModel.ID

	testData := []byte("test_firmware_binary_data_v1")
	correctMD5 := md5util.ComputeMD5(testData)
	wrongMD5 := "00000000000000000000000000000000"

	allPassed := true
	var failReasons []string

	t.Run("MD5校验失败后文件清理", func(t *testing.T) {
		req := &model.UploadFirmwareRequest{
			ModelID:  modelID,
			Version:  "1.0.0",
			Md5:      wrongMD5,
			Changelog: "test",
		}

		_, err := svc.UploadFirmware(ctx, req, testData, "firmware.bin")
		if err == nil {
			t.Errorf("expected MD5 mismatch error, got nil")
			allPassed = false
			failReasons = append(failReasons, "MD5校验未正确返回错误")
			return
		}

		modelDir := filepath.Join(uploadDir, fmt.Sprintf("model_%d", modelID))
		if _, statErr := os.Stat(modelDir); statErr == nil {
			files, _ := os.ReadDir(modelDir)
			if len(files) > 0 {
				t.Errorf("file not cleaned up after MD5 failure: %d files remain in %s", len(files), modelDir)
				allPassed = false
				failReasons = append(failReasons, "MD5校验失败后文件未清理")
				return
			}
		}

		t.Log("PASS: MD5校验失败后文件已正确清理")
	})

	t.Run("版本冲突ID计数器污染", func(t *testing.T) {
		req1 := &model.UploadFirmwareRequest{
			ModelID:  modelID,
			Version:  "2.0.0",
			Md5:      correctMD5,
			Changelog: "first upload",
		}

		fw1, err := svc.UploadFirmware(ctx, req1, testData, "firmware.bin")
		if err != nil {
			t.Fatalf("first upload failed: %v", err)
		}
		firstID := fw1.ID

		req2 := &model.UploadFirmwareRequest{
			ModelID:  modelID,
			Version:  "2.0.0",
			Md5:      correctMD5,
			Changelog: "duplicate upload",
		}

		_, err2 := svc.UploadFirmware(ctx, req2, testData, "firmware.bin")
		if err2 == nil {
			t.Errorf("expected version conflict error, got nil")
			allPassed = false
			failReasons = append(failReasons, "版本冲突未正确返回错误")
			return
		}

		req3 := &model.UploadFirmwareRequest{
			ModelID:  modelID,
			Version:  "2.0.1",
			Md5:      correctMD5,
			Changelog: "second upload",
		}

		fw3, err3 := svc.UploadFirmware(ctx, req3, testData, "firmware.bin")
		if err3 != nil {
			t.Fatalf("third upload failed: %v", err3)
		}
		thirdID := fw3.ID

		expectedSecondID := firstID + 1
		if thirdID != expectedSecondID {
			t.Errorf("ID gap detected: first=%d, expected_second=%d, got_third=%d. ID counter was consumed by failed upload",
				firstID, expectedSecondID, thirdID)
			allPassed = false
			failReasons = append(failReasons, "版本冲突时ID计数器被消耗导致ID不连续")
			return
		}

		t.Log("PASS: ID计数器未被失败上传消耗")
	})

	if allPassed {
		t.Log("GREEN（绿灯，缺陷已修复）")
	} else {
		t.Logf("RED（红灯，缺陷未修复）: %v", failReasons)
		t.Fail()
	}
}

func TestIDSequence(t *testing.T) {
	tmpDir := t.TempDir()
	uploadDir := filepath.Join(tmpDir, "uploads")

	cfg := config.DefaultConfig()
	cfg.Storage.UploadDir = uploadDir
	cfg.Storage.DataDir = filepath.Join(tmpDir, "data")

	memStore := store.NewMemoryStore()
	svc := service.NewFirmwareService(memStore, memStore, cfg)
	ctx := context.Background()

	testModel := model.NewDeviceModel("seq_test", "TestMfg", "HW-v1", "ID sequence test")
	memStore.CreateModel(ctx, testModel)
	modelID := testModel.ID

	testData := []byte("sequence_test_data")
	correctMD5 := md5util.ComputeMD5(testData)

	allPassed := true
	var failReasons []string

	t.Run("连续上传ID连续性", func(t *testing.T) {
		versions := []string{"3.0.0", "3.0.1", "3.0.2"}
		ids := make([]model.ID, 0)

		for _, ver := range versions {
			req := &model.UploadFirmwareRequest{
				ModelID: modelID,
				Version: ver,
				Md5:     correctMD5,
			}
			fw, err := svc.UploadFirmware(ctx, req, testData, "fw.bin")
			if err != nil {
				t.Fatalf("upload %s failed: %v", ver, err)
			}
			ids = append(ids, fw.ID)
		}

		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

		for i := 1; i < len(ids); i++ {
			if ids[i] != ids[i-1]+1 {
				t.Errorf("IDs not sequential: %v, gap between %d and %d", ids, ids[i-1], ids[i])
				allPassed = false
				failReasons = append(failReasons, fmt.Sprintf("ID不连续: %v", ids))
				break
			}
		}

		if allPassed {
			t.Logf("PASS: IDs are sequential: %v", ids)
		}
	})

	t.Run("失败上传后ID连续性", func(t *testing.T) {
		req := &model.UploadFirmwareRequest{
			ModelID: modelID,
			Version:  "3.0.0",
			Md5:     correctMD5,
		}
		svc.UploadFirmware(ctx, req, testData, "fw.bin")

		reqFail := &model.UploadFirmwareRequest{
			ModelID: modelID,
			Version: "3.0.0",
			Md5:     correctMD5,
		}
		svc.UploadFirmware(ctx, reqFail, testData, "fw.bin")

		reqNext := &model.UploadFirmwareRequest{
			ModelID: modelID,
			Version: "3.0.3",
			Md5:     correctMD5,
		}
		fwNext, err := svc.UploadFirmware(ctx, reqNext, testData, "fw.bin")
		if err != nil {
			t.Fatalf("upload 3.0.3 failed: %v", err)
		}

		allFw, _ := memStore.GetAllFirmwares(ctx)
		var allIDs []model.ID
		for _, f := range allFw {
			allIDs = append(allIDs, f.ID)
		}
		sort.Slice(allIDs, func(i, j int) bool { return allIDs[i] < allIDs[j] })

		for i := 1; i < len(allIDs); i++ {
			if allIDs[i] != allIDs[i-1]+1 {
				t.Errorf("IDs not sequential after failed upload: %v", allIDs)
				allPassed = false
				failReasons = append(failReasons, fmt.Sprintf("失败上传后ID不连续: %v", allIDs))
				break
			}
		}

		if allPassed {
			t.Logf("PASS: IDs sequential after failed upload: %v", allIDs)
		} else {
			t.Logf("Note: next ID would be %d", fwNext.ID)
		}
	})

	if allPassed {
		t.Log("GREEN（绿灯，缺陷已修复）")
	} else {
		t.Logf("RED（红灯，缺陷未修复）: %v", failReasons)
		t.Fail()
	}
}