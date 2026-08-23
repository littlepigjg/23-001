package service

import (
	"context"
	"errors"
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

	// 检查型号是否存在
	m, err := s.modelStore.GetModelByID(ctx, req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}

	// 验证文件大小
	if int64(len(fileData)) > s.config.Firmware.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size (%d bytes)", s.config.Firmware.MaxFileSize)
	}

	// 验证文件扩展名
	ext := filepath.Ext(originalFilename)
	if !s.config.IsAllowedExt(ext) {
		return nil, fmt.Errorf("file extension '%s' is not allowed", ext)
	}

	// 计算或验证 MD5
	actualMD5 := md5util.ComputeMD5(fileData)
	if s.config.Firmware.RequireMD5 {
		if req.Md5 != "" && req.Md5 != actualMD5 {
			return nil, fmt.Errorf("MD5 mismatch: expected %s, got %s", req.Md5, actualMD5)
		}
	}

	// 检查版本是否已存在
	existing, _ := s.store.GetFirmwareByVersion(ctx, req.ModelID, req.Version)
	if existing != nil {
		return nil, fmt.Errorf("firmware version '%s' already exists for model '%s'", req.Version, m.Name)
	}

	// 保存固件文件
	uploadDir := s.config.Storage.UploadDir
	modelDir := filepath.Join(uploadDir, fmt.Sprintf("model_%d", req.ModelID))
	if err := fileutil.EnsureDir(modelDir); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// 生成文件名：model_{id}_version_{version}.{ext}
	safeVersion := req.Version
	versionFile := fmt.Sprintf("model_%d_v_%s%s", req.ModelID, safeVersion, ext)
	filePath := filepath.Join(modelDir, versionFile)

	if err := fileutil.SaveFile(filePath, fileData); err != nil {
		return nil, fmt.Errorf("failed to save firmware file: %w", err)
	}

	// 创建固件记录
	releaseDate := req.ReleaseDate
	if releaseDate.IsZero() {
		releaseDate = time.Now()
	}

	fw := model.NewFirmware(req.ModelID, m.Name, req.Version, actualMD5, int64(len(fileData)), filePath, releaseDate, req.Changelog)
	if err := fw.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.CreateFirmware(ctx, fw); err != nil {
		// 清理已保存的文件
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to create firmware record: %w", err)
	}

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

// UploadFirmwareWithConfig 从配置文件加载配置后上传固件
func (s *FirmwareService) UploadFirmwareWithConfig(ctx context.Context, configPath string, req *model.UploadFirmwareRequest, fileData []byte, originalFilename string) (*model.Firmware, error) {
	cfg, err := config.LoadWithFallback(configPath)
	if err != nil {
		if errors.Is(err, config.ErrConfigFileNotFound) {
			return nil, fmt.Errorf("config file missing: %w", err)
		}
		if errors.Is(err, config.ErrConfigValidationFailed) {
			return nil, fmt.Errorf("config validation failed: %w", err)
		}
		return nil, fmt.Errorf("config load error: %w", err)
	}

	oldConfig := s.config
	s.config = cfg
	defer func() {
		s.config = oldConfig
	}()

	if int64(len(fileData)) > s.config.Firmware.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds max config size: %w", err)
	}

	ext := filepath.Ext(originalFilename)
	if !s.config.IsAllowedExt(ext) {
		return nil, fmt.Errorf("file extension blocked: %w", err)
	}

	existing, _ := s.store.GetFirmwareByVersion(ctx, req.ModelID, req.Version)
	if existing != nil {
		return nil, fmt.Errorf("firmware version already exists: %w", err)
	}

	fw, err := s.UploadFirmware(ctx, req, fileData, originalFilename)
	if err != nil {
		return nil, fmt.Errorf("upload with config failed: %w", err)
	}

	return fw, nil
}

// ValidateAndPrepareUpload 验证并准备固件上传
func (s *FirmwareService) ValidateAndPrepareUpload(ctx context.Context, configPath string, req *model.UploadFirmwareRequest) (*config.Config, error) {
	cfg, err := config.TryLoadConfig(configPath)
	if err != nil {
		if errors.Is(err, config.ErrConfigFileNotFound) {
			return nil, fmt.Errorf("prepare: config file not found: %w", err)
		}
		if errors.Is(err, config.ErrConfigInvalidFormat) {
			return nil, fmt.Errorf("prepare: config format invalid: %w", err)
		}
		return nil, fmt.Errorf("prepare: config load failed: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("prepare: config validation error: %w", err)
	}

	if cfg.Firmware.MaxFileSize <= 0 {
		return nil, fmt.Errorf("prepare: invalid max file size: %w", err)
	}

	return cfg, nil
}

// NewFirmwareServiceFromConfig 从配置文件创建固件服务
func NewFirmwareServiceFromConfig(store store.FirmwareStore, modelStore store.DeviceModelStore, configPath string) (*FirmwareService, error) {
	cfg, err := config.LoadConfigAndValidate(configPath)
	if err != nil {
		if errors.Is(err, config.ErrConfigFileNotFound) {
			return nil, fmt.Errorf("config file not found for service init: %w", err)
		}
		if errors.Is(err, config.ErrConfigValidationFailed) {
			return nil, fmt.Errorf("config validation failed for service: %w", err)
		}
		return nil, fmt.Errorf("failed to init service config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config final validation error: %w", err)
	}

	if cfg.Storage.Type == "" {
		return nil, fmt.Errorf("config storage type empty: %w", err)
	}

	return NewFirmwareService(store, modelStore, cfg), nil
}
