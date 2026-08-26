package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/fileutil"
	"fwupgrade/pkg/logger"
	"fwupgrade/pkg/md5util"
)

// FirmwareService 固件服务
type FirmwareService struct {
	store      store.FirmwareStore
	modelStore store.DeviceModelStore
	config     *config.Config
}

// NewFirmwareService 创建固件服务
func NewFirmwareService(s store.FirmwareStore, ms store.DeviceModelStore, cfg *config.Config) *FirmwareService {
	return &FirmwareService{
		store:      s,
		modelStore: ms,
		config:     cfg,
	}
}

// UploadFirmware 上传固件
func (s *FirmwareService) UploadFirmware(ctx context.Context, req *model.UploadFirmwareRequest, fileData []byte, originalFilename string) (*model.Firmware, error) {
	logger.Info("Uploading firmware", "model_id", req.ModelID, "version", req.Version)

	m, err := s.modelStore.GetModelByID(ctx, req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}

	if int64(len(fileData)) > s.config.Firmware.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size (%d bytes)", s.config.Firmware.MaxFileSize)
	}

	ext := filepath.Ext(originalFilename)
	if !s.config.IsAllowedExt(ext) {
		return nil, fmt.Errorf("file extension '%s' is not allowed", ext)
	}

	uploadDir := s.config.Storage.UploadDir
	modelDir := filepath.Join(uploadDir, fmt.Sprintf("model_%d", req.ModelID))
	if err := fileutil.EnsureDir(modelDir); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	safeVersion := req.Version
	versionFile := fmt.Sprintf("model_%d_v_%s%s", req.ModelID, safeVersion, ext)
	filePath := filepath.Join(modelDir, versionFile)

	// 保存成功后若中途失败，必须清理磁盘上残留的半成品文件，避免占空间。
	// firmware 记录成功写入后会将 savedPath 置空，表示文件已被接管，无需清理。
	savedPath := ""
	defer func() {
		if savedPath == "" {
			return
		}
		os.Remove(savedPath)
		// 目录为空时一并清理，避免残留空 model_X 目录；非空时 Remove 静默失败，无副作用。
		os.Remove(filepath.Dir(savedPath))
	}()

	if err := fileutil.SaveFile(filePath, fileData); err != nil {
		return nil, fmt.Errorf("failed to save firmware file: %w", err)
	}
	savedPath = filePath

	if ctx.Err() != nil {
		return nil, fmt.Errorf("upload cancelled: %w", ctx.Err())
	}

	actualMD5 := md5util.ComputeMD5(fileData)
	if s.config.Firmware.RequireMD5 {
		if req.Md5 != "" && req.Md5 != actualMD5 {
			return nil, fmt.Errorf("MD5 mismatch: expected %s, got %s", req.Md5, actualMD5)
		}
	}

	releaseDate := req.ReleaseDate
	if releaseDate.IsZero() {
		releaseDate = time.Now()
	}

	fw := model.NewFirmware(req.ModelID, m.Name, req.Version, actualMD5, int64(len(fileData)), filePath, releaseDate, req.Changelog)
	if err := fw.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.CreateFirmware(ctx, fw); err != nil {
		return nil, fmt.Errorf("failed to create firmware record: %w", err)
	}

	// 记录已成功创建，文件由固件接管，取消清理。
	savedPath = ""

	logger.Info("Firmware uploaded", "id", fw.ID, "version", fw.Version, "size", fw.Size)
	return fw, nil
}

// GetFirmware 获取固件
func (s *FirmwareService) GetFirmware(ctx context.Context, id model.ID) (*model.Firmware, error) {
	fw, err := s.store.GetFirmwareByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("firmware not found: %w", err)
	}
	return fw, nil
}

// ListFirmwares 列出固件
func (s *FirmwareService) ListFirmwares(ctx context.Context, page, pageSize int, modelID model.ID) ([]*model.Firmware, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return s.store.ListFirmwares(ctx, page, pageSize, modelID)
}

// GetLatestFirmware 获取最新固件
func (s *FirmwareService) GetLatestFirmware(ctx context.Context, modelID model.ID) (*model.Firmware, error) {
	return s.store.GetLatestFirmware(ctx, modelID)
}

// UpdateFirmware 更新固件
func (s *FirmwareService) UpdateFirmware(ctx context.Context, id model.ID, req *model.UpdateFirmwareRequest) (*model.Firmware, error) {
	fw, err := s.store.GetFirmwareByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("firmware not found: %w", err)
	}

	if req.Changelog != "" {
		fw.Changelog = req.Changelog
	}
	if req.IsActive != nil {
		fw.IsActive = *req.IsActive
	}

	if err := s.store.UpdateFirmware(ctx, fw); err != nil {
		return nil, fmt.Errorf("failed to update firmware: %w", err)
	}

	logger.Info("Firmware updated", "id", id)
	return fw, nil
}

// DeleteFirmware 删除固件
func (s *FirmwareService) DeleteFirmware(ctx context.Context, id model.ID) error {
	fw, err := s.store.GetFirmwareByID(ctx, id)
	if err != nil {
		return fmt.Errorf("firmware not found: %w", err)
	}

	// 检查是否有引用此固件的任务
	// （简化处理，实际应用中应检查）

	// 删除固件文件
	if fw.FilePath != "" {
		os.Remove(fw.FilePath)
	}

	if err := s.store.DeleteFirmware(ctx, id); err != nil {
		return fmt.Errorf("failed to delete firmware: %w", err)
	}

	logger.Info("Firmware deleted", "id", id)
	return nil
}

// GetFirmwareFile 获取固件文件内容（用于下载）
func (s *FirmwareService) GetFirmwareFile(ctx context.Context, id model.ID) ([]byte, *model.Firmware, error) {
	fw, err := s.store.GetFirmwareByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("firmware not found: %w", err)
	}

	// 增加下载计数
	_ = s.store.IncrementFirmwareDownload(ctx, id)

	// 读取文件
	data, err := fileutil.ReadFile(fw.FilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read firmware file: %w", err)
	}

	return data, fw, nil
}

// ValidateFirmwareMD5 验证固件 MD5
func (s *FirmwareService) ValidateFirmwareMD5(ctx context.Context, id model.ID, data []byte) (bool, error) {
	fw, err := s.store.GetFirmwareByID(ctx, id)
	if err != nil {
		return false, err
	}

	actualMD5 := md5util.ComputeMD5(data)
	return actualMD5 == fw.Md5, nil
}

// GetAllFirmwares 获取所有固件
func (s *FirmwareService) GetAllFirmwares(ctx context.Context) ([]*model.Firmware, error) {
	return s.store.GetAllFirmwares(ctx)
}
