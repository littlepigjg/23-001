package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/md5util"
)

// newTestFirmwareService 构建一个指向临时上传目录的 FirmwareService，并预置一个设备型号。
func newTestFirmwareService(t *testing.T) (*FirmwareService, *store.MemoryStore, *config.Config, *model.DeviceModel) {
	t.Helper()

	uploadDir := filepath.Join(t.TempDir(), "uploads")
	cfg := &config.Config{
		Storage:   config.StorageConfig{Type: "memory", UploadDir: uploadDir},
		Firmware:  config.FirmwareConfig{MaxFileSize: 10 << 20, AllowedExts: ".bin,.fw", RequireMD5: true},
	}

	memStore := store.NewMemoryStore()
	m := model.NewDeviceModel("TestModel", "Acme", "hw1", "desc")
	if err := memStore.CreateModel(context.Background(), m); err != nil {
		t.Fatalf("create model: %v", err)
	}

	svc := NewFirmwareService(memStore, memStore, cfg)
	return svc, memStore, cfg, m
}

// TestUploadFirmware_MD5MismatchCleansUpFile 验证 MD5 校验失败时，
// 已写入磁盘的固件文件应被清理，不应残留。
func TestUploadFirmware_MD5MismatchCleansUpFile(t *testing.T) {
	svc, memStore, cfg, m := newTestFirmwareService(t)
	ctx := context.Background()

	data := []byte("firmware content")
	req := &model.UploadFirmwareRequest{
		ModelID: m.ID,
		Version: "1.0.0",
		Md5:     "deadbeefdeadbeefdeadbeefdeadbeef", // 故意错误的 MD5
	}

	_, err := svc.UploadFirmware(ctx, req, data, "fw.bin")
	if err == nil {
		t.Fatal("expected MD5 mismatch error, got nil")
	}

	// 计算文件落地路径，校验文件已被删除
	modelDir := filepath.Join(cfg.Storage.UploadDir, fmt.Sprintf("model_%d", m.ID))
	filePath := filepath.Join(modelDir, fmt.Sprintf("model_%d_v_1.0.0.bin", m.ID))
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("firmware file should be removed after MD5 mismatch, stat err=%v", err)
	}

	// 确保固件记录也未创建
	if fws, err := memStore.GetAllFirmwares(ctx); err == nil && len(fws) != 0 {
		t.Fatalf("no firmware record should be created, got %d", len(fws))
	}
	// 上传目录下不应残留任何文件
	if entries, _ := os.ReadDir(cfg.Storage.UploadDir); len(entries) != 0 {
		t.Fatalf("upload dir should be empty, found %d entries", len(entries))
	}
}

// TestUploadFirmware_SuccessKeepsFile 验证正常上传时文件保留且 MD5 写入正确。
func TestUploadFirmware_SuccessKeepsFile(t *testing.T) {
	svc, memStore, _, m := newTestFirmwareService(t)
	ctx := context.Background()

	data := []byte("good firmware")
	req := &model.UploadFirmwareRequest{
		ModelID: m.ID,
		Version: "1.0.0",
		Md5:     md5util.ComputeMD5(data),
	}

	fw, err := svc.UploadFirmware(ctx, req, data, "fw.bin")
	if err != nil {
		t.Fatalf("upload: %v", err)
	}

	if _, err := os.Stat(fw.FilePath); err != nil {
		t.Fatalf("firmware file should exist after successful upload: %v", err)
	}
	if fw.Md5 != md5util.ComputeMD5(data) {
		t.Fatalf("stored md5 = %s, want %s", fw.Md5, md5util.ComputeMD5(data))
	}
	if fws, _ := memStore.GetAllFirmwares(ctx); len(fws) != 1 {
		t.Fatalf("expected 1 firmware record, got %d", len(fws))
	}
}
