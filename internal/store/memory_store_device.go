package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"fwupgrade/internal/model"
)

// ================ DeviceStore 实现 ================

// CreateDevice 创建设备
func (s *MemoryStore) CreateDevice(_ context.Context, d *model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查设备ID是否重复
	if _, exists := s.deviceIDIndex[d.DeviceID]; exists {
		return fmt.Errorf("device with id '%s' already exists", d.DeviceID)
	}

	id := s.nextID()
	d.ID = id
	s.devices[id] = d
	s.deviceIDIndex[d.DeviceID] = id

	// 更新型号设备计数
	if m, ok := s.models[d.ModelID]; ok {
		m.DeviceCount++
	}

	return nil
}

// GetDeviceByID 根据ID获取设备
func (s *MemoryStore) GetDeviceByID(_ context.Context, id model.ID) (*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.devices[id]
	if !ok {
		return nil, fmt.Errorf("device not found: id=%d", id)
	}
	return d, nil
}

// GetDeviceByDeviceID 根据设备ID获取设备
func (s *MemoryStore) GetDeviceByDeviceID(_ context.Context, deviceID string) (*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.deviceIDIndex[deviceID]
	if !ok {
		return nil, fmt.Errorf("device not found: device_id=%s", deviceID)
	}
	return s.devices[id], nil
}

// ListDevices 列出设备
func (s *MemoryStore) ListDevices(_ context.Context, page, pageSize int, modelID model.ID, status model.DeviceStatus) ([]*model.Device, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]*model.Device, 0)
	for _, d := range s.devices {
		if modelID > 0 && d.ModelID != modelID {
			continue
		}
		if status != "" && d.Status != status {
			continue
		}
		devices = append(devices, d)
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	total := int64(len(devices))
	totalCount := int(total)
	page, pageSize, start, end := paginate(page, pageSize, totalCount)

	return devices[start:end], total, nil
}

// ListDevicesByModel 根据型号列出设备
func (s *MemoryStore) ListDevicesByModel(_ context.Context, modelID model.ID) ([]*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Device
	for _, d := range s.devices {
		if d.ModelID == modelID {
			result = append(result, d)
		}
	}
	return result, nil
}

// ListDevicesByStatus 根据状态列出设备
func (s *MemoryStore) ListDevicesByStatus(_ context.Context, status model.DeviceStatus) ([]*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Device
	for _, d := range s.devices {
		if d.Status == status {
			result = append(result, d)
		}
	}
	return result, nil
}

// ListOnlineDevices 列出在线设备
func (s *MemoryStore) ListOnlineDevices(_ context.Context) ([]*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Device
	for _, d := range s.devices {
		if d.IsOnline() {
			result = append(result, d)
		}
	}
	return result, nil
}

// UpdateDevice 更新设备
func (s *MemoryStore) UpdateDevice(_ context.Context, d *model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.devices[d.ID]; !ok {
		return fmt.Errorf("device not found: id=%d", d.ID)
	}

	s.devices[d.ID] = d
	return nil
}

// UpdateDeviceStatus 更新设备状态
func (s *MemoryStore) UpdateDeviceStatus(_ context.Context, id model.ID, status model.DeviceStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.devices[id]
	if !ok {
		return fmt.Errorf("device not found: id=%d", id)
	}

	d.Status = status
	d.LastSeenAt = time.Now()
	return nil
}

// UpdateDeviceLastSeen 更新设备最后心跳时间
func (s *MemoryStore) UpdateDeviceLastSeen(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.devices[id]
	if !ok {
		return fmt.Errorf("device not found: id=%d", id)
	}

	d.LastSeenAt = time.Now()
	return nil
}

// UpdateDeviceProgress 更新升级进度
func (s *MemoryStore) UpdateDeviceProgress(_ context.Context, id model.ID, progress int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.devices[id]
	if !ok {
		return fmt.Errorf("device not found: id=%d", id)
	}

	d.UpgradeProgress = progress
	if progress >= 100 {
		d.Status = model.DeviceOnline
		d.TargetFWVer = d.CurrentFWVer
	}
	return nil
}

// DeleteDevice 删除设备
func (s *MemoryStore) DeleteDevice(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.devices[id]
	if !ok {
		return fmt.Errorf("device not found: id=%d", id)
	}

	delete(s.devices, id)
	delete(s.deviceIDIndex, d.DeviceID)

	// 更新型号设备计数
	if m, ok := s.models[d.ModelID]; ok {
		if m.DeviceCount > 0 {
			m.DeviceCount--
		}
	}

	return nil
}

// BatchCreateDevices 批量创建设备
func (s *MemoryStore) BatchCreateDevices(ctx context.Context, devices []*model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range devices {
		if _, exists := s.deviceIDIndex[d.DeviceID]; exists {
			continue // 跳过已存在的设备
		}
		id := s.nextID()
		d.ID = id
		s.devices[id] = d
		s.deviceIDIndex[d.DeviceID] = id
		if m, ok := s.models[d.ModelID]; ok {
			m.DeviceCount++
		}
	}
	return nil
}

// CountDevicesByStatus 按状态统计设备
func (s *MemoryStore) CountDevicesByStatus(_ context.Context) (map[model.DeviceStatus]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[model.DeviceStatus]int)
	for _, d := range s.devices {
		result[d.Status]++
	}
	return result, nil
}

// SearchDevices 搜索设备
func (s *MemoryStore) SearchDevices(_ context.Context, keyword string, page, pageSize int) ([]*model.Device, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keywordLower := toLower(keyword)
	var devices []*model.Device
	for _, d := range s.devices {
		if containsStr(toLower(d.DeviceID), keywordLower) ||
			containsStr(toLower(d.Name), keywordLower) ||
			containsStr(toLower(d.SerialNumber), keywordLower) {
			devices = append(devices, d)
		}
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	total := int64(len(devices))
	totalCount := int(total)
	page, pageSize, start, end := paginate(page, pageSize, totalCount)

	return devices[start:end], total, nil
}

// GetAllDevices 分页获取所有设备
func (s *MemoryStore) GetAllDevices(_ context.Context, page, pageSize int) ([]*model.Device, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]*model.Device, 0, len(s.devices))
	for _, d := range s.devices {
		devices = append(devices, d)
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	total := int64(len(devices))
	totalCount := int(total)
	page, pageSize, start, end := paginate(page, pageSize, totalCount)

	return devices[start:end], total, nil
}
