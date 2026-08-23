package store

import (
	"context"

	"fwupgrade/internal/model"
)

// ================ DeviceStore 实现 ================

// CreateDevice 创建设备
func (s *FileStore) CreateDevice(ctx context.Context, d *model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.CreateDevice(ctx, d); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// GetDeviceByID 根据ID获取设备
func (s *FileStore) GetDeviceByID(ctx context.Context, id model.ID) (*model.Device, error) {
	return s.memStore.GetDeviceByID(ctx, id)
}

// GetDeviceByDeviceID 根据设备ID获取设备
func (s *FileStore) GetDeviceByDeviceID(ctx context.Context, deviceID string) (*model.Device, error) {
	return s.memStore.GetDeviceByDeviceID(ctx, deviceID)
}

// ListDevices 列出设备
func (s *FileStore) ListDevices(ctx context.Context, page, pageSize int, modelID model.ID, status model.DeviceStatus) ([]*model.Device, int64, error) {
	return s.memStore.ListDevices(ctx, page, pageSize, modelID, status)
}

// ListDevicesByModel 根据型号列出设备
func (s *FileStore) ListDevicesByModel(ctx context.Context, modelID model.ID) ([]*model.Device, error) {
	return s.memStore.ListDevicesByModel(ctx, modelID)
}

// ListDevicesByStatus 根据状态列出设备
func (s *FileStore) ListDevicesByStatus(ctx context.Context, status model.DeviceStatus) ([]*model.Device, error) {
	return s.memStore.ListDevicesByStatus(ctx, status)
}

// ListOnlineDevices 列出在线设备
func (s *FileStore) ListOnlineDevices(ctx context.Context) ([]*model.Device, error) {
	return s.memStore.ListOnlineDevices(ctx)
}

// UpdateDevice 更新设备
func (s *FileStore) UpdateDevice(ctx context.Context, d *model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateDevice(ctx, d); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// UpdateDeviceStatus 更新设备状态
func (s *FileStore) UpdateDeviceStatus(ctx context.Context, id model.ID, status model.DeviceStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateDeviceStatus(ctx, id, status); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// UpdateDeviceLastSeen 更新设备最后心跳时间
func (s *FileStore) UpdateDeviceLastSeen(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateDeviceLastSeen(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// UpdateDeviceProgress 更新升级进度
func (s *FileStore) UpdateDeviceProgress(ctx context.Context, id model.ID, progress int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateDeviceProgress(ctx, id, progress); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// DeleteDevice 删除设备
func (s *FileStore) DeleteDevice(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.DeleteDevice(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// BatchCreateDevices 批量创建设备
func (s *FileStore) BatchCreateDevices(ctx context.Context, devices []*model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.BatchCreateDevices(ctx, devices); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// CountDevicesByStatus 按状态统计设备
func (s *FileStore) CountDevicesByStatus(ctx context.Context) (map[model.DeviceStatus]int, error) {
	return s.memStore.CountDevicesByStatus(ctx)
}

// SearchDevices 搜索设备
func (s *FileStore) SearchDevices(ctx context.Context, keyword string, page, pageSize int) ([]*model.Device, int64, error) {
	return s.memStore.SearchDevices(ctx, keyword, page, pageSize)
}

// GetAllDevices 分页获取所有设备
func (s *FileStore) GetAllDevices(ctx context.Context, page, pageSize int) ([]*model.Device, int64, error) {
	return s.memStore.GetAllDevices(ctx, page, pageSize)
}
