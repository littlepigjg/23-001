package store

import (
	"context"

	"fwupgrade/internal/model"
)

// ================ FirmwareStore 实现 ================

// CreateFirmware 创建固件
func (s *FileStore) CreateFirmware(ctx context.Context, f *model.Firmware) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.CreateFirmware(ctx, f); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// GetFirmwareByID 根据ID获取固件
func (s *FileStore) GetFirmwareByID(ctx context.Context, id model.ID) (*model.Firmware, error) {
	return s.memStore.GetFirmwareByID(ctx, id)
}

// GetFirmwareByVersion 根据型号和版本获取固件
func (s *FileStore) GetFirmwareByVersion(ctx context.Context, modelID model.ID, version string) (*model.Firmware, error) {
	return s.memStore.GetFirmwareByVersion(ctx, modelID, version)
}

// GetLatestFirmware 获取型号的最新固件
func (s *FileStore) GetLatestFirmware(ctx context.Context, modelID model.ID) (*model.Firmware, error) {
	return s.memStore.GetLatestFirmware(ctx, modelID)
}

// ListFirmwares 列出固件
func (s *FileStore) ListFirmwares(ctx context.Context, page, pageSize int, modelID model.ID) ([]*model.Firmware, int64, error) {
	return s.memStore.ListFirmwares(ctx, page, pageSize, modelID)
}

// ListFirmwaresByModel 根据型号列出固件
func (s *FileStore) ListFirmwaresByModel(ctx context.Context, modelID model.ID) ([]*model.Firmware, error) {
	return s.memStore.ListFirmwaresByModel(ctx, modelID)
}

// UpdateFirmware 更新固件
func (s *FileStore) UpdateFirmware(ctx context.Context, f *model.Firmware) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateFirmware(ctx, f); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// SetFirmwareActive 设置固件活跃状态
func (s *FileStore) SetFirmwareActive(ctx context.Context, id model.ID, active bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.SetFirmwareActive(ctx, id, active); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// IncrementFirmwareDownload 增加下载计数
func (s *FileStore) IncrementFirmwareDownload(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.IncrementFirmwareDownload(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// DeleteFirmware 删除固件
func (s *FileStore) DeleteFirmware(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.DeleteFirmware(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// GetAllFirmwares 获取所有固件
func (s *FileStore) GetAllFirmwares(ctx context.Context) ([]*model.Firmware, error) {
	return s.memStore.GetAllFirmwares(ctx)
}

// CountFirmwaresByModel 按型号统计固件数量
func (s *FileStore) CountFirmwaresByModel(ctx context.Context) (map[model.ID]int, error) {
	return s.memStore.CountFirmwaresByModel(ctx)
}
