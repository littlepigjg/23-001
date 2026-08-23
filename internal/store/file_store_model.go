package store

import (
	"context"

	"fwupgrade/internal/model"
)

// ================ DeviceModelStore 实现 ================

// CreateModel 创建设备型号
func (s *FileStore) CreateModel(ctx context.Context, m *model.DeviceModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.CreateModel(ctx, m); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// GetModelByID 根据ID获取设备型号
func (s *FileStore) GetModelByID(ctx context.Context, id model.ID) (*model.DeviceModel, error) {
	return s.memStore.GetModelByID(ctx, id)
}

// GetModelByName 根据名称获取设备型号
func (s *FileStore) GetModelByName(ctx context.Context, name string) (*model.DeviceModel, error) {
	return s.memStore.GetModelByName(ctx, name)
}

// ListModels 列出设备型号
func (s *FileStore) ListModels(ctx context.Context, page, pageSize int) ([]*model.DeviceModel, int64, error) {
	return s.memStore.ListModels(ctx, page, pageSize)
}

// ListModelsByManufacturer 根据厂商列出设备型号
func (s *FileStore) ListModelsByManufacturer(ctx context.Context, manufacturer string) ([]*model.DeviceModel, error) {
	return s.memStore.ListModelsByManufacturer(ctx, manufacturer)
}

// UpdateModel 更新设备型号
func (s *FileStore) UpdateModel(ctx context.Context, m *model.DeviceModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateModel(ctx, m); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// DeleteModel 删除设备型号
func (s *FileStore) DeleteModel(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.DeleteModel(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// SetModelActive 设置型号活跃状态
func (s *FileStore) SetModelActive(ctx context.Context, id model.ID, active bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.SetModelActive(ctx, id, active); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// CountModelDevices 统计型号下的设备数量
func (s *FileStore) CountModelDevices(ctx context.Context, modelID model.ID) (int, error) {
	return s.memStore.CountModelDevices(ctx, modelID)
}
