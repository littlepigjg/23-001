package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"fwupgrade/internal/model"
)

// ================ FirmwareStore 实现 ================

// CreateFirmware 创建固件
func (s *MemoryStore) CreateFirmware(_ context.Context, f *model.Firmware) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("%d:%s", f.ModelID, f.Version)
	if _, exists := s.firmwareVersionIndex[key]; exists {
		return fmt.Errorf("firmware version '%s' already exists for model %d", f.Version, f.ModelID)
	}

	id := s.nextID()
	f.ID = id
	s.firmwares[id] = f
	s.firmwareVersionIndex[key] = id

	return nil
}

// GetFirmwareByID 根据ID获取固件
func (s *MemoryStore) GetFirmwareByID(_ context.Context, id model.ID) (*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, ok := s.firmwares[id]
	if !ok {
		return nil, fmt.Errorf("firmware not found: id=%d", id)
	}
	return f, nil
}

// GetFirmwareByVersion 根据型号和版本获取固件
func (s *MemoryStore) GetFirmwareByVersion(_ context.Context, modelID model.ID, version string) (*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%d:%s", modelID, version)
	id, ok := s.firmwareVersionIndex[key]
	if !ok {
		return nil, fmt.Errorf("firmware not found: model=%d, version=%s", modelID, version)
	}
	return s.firmwares[id], nil
}

// GetLatestFirmware 获取型号的最新固件
func (s *MemoryStore) GetLatestFirmware(_ context.Context, modelID model.ID) (*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest *model.Firmware
	for _, f := range s.firmwares {
		if f.ModelID == modelID && f.IsActive {
			if latest == nil || f.ReleaseDate.After(latest.ReleaseDate) {
				latest = f
			}
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no firmware found for model %d", modelID)
	}
	return latest, nil
}

// ListFirmwares 列出固件
func (s *MemoryStore) ListFirmwares(_ context.Context, page, pageSize int, modelID model.ID) ([]*model.Firmware, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var firmwares []*model.Firmware
	for _, f := range s.firmwares {
		if modelID > 0 && f.ModelID != modelID {
			continue
		}
		firmwares = append(firmwares, f)
	}

	sort.Slice(firmwares, func(i, j int) bool {
		return firmwares[i].ID < firmwares[j].ID
	})

	total := int64(len(firmwares))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.Firmware{}, total, nil
	}

	return firmwares[start:end], total, nil
}

// ListFirmwaresByModel 根据型号列出固件
func (s *MemoryStore) ListFirmwaresByModel(_ context.Context, modelID model.ID) ([]*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Firmware
	for _, f := range s.firmwares {
		if f.ModelID == modelID {
			result = append(result, f)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ReleaseDate.After(result[j].ReleaseDate)
	})

	return result, nil
}

// UpdateFirmware 更新固件
func (s *MemoryStore) UpdateFirmware(_ context.Context, f *model.Firmware) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.firmwares[f.ID]; !ok {
		return fmt.Errorf("firmware not found: id=%d", f.ID)
	}

	f.UpdatedAt = time.Now()
	s.firmwares[f.ID] = f
	return nil
}

// SetFirmwareActive 设置固件活跃状态
func (s *MemoryStore) SetFirmwareActive(_ context.Context, id model.ID, active bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, ok := s.firmwares[id]
	if !ok {
		return fmt.Errorf("firmware not found: id=%d", id)
	}

	f.IsActive = active
	return nil
}

// IncrementFirmwareDownload 增加下载计数
func (s *MemoryStore) IncrementFirmwareDownload(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, ok := s.firmwares[id]
	if !ok {
		return fmt.Errorf("firmware not found: id=%d", id)
	}

	f.DownloadCount++
	return nil
}

// DeleteFirmware 删除固件
func (s *MemoryStore) DeleteFirmware(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, ok := s.firmwares[id]
	if !ok {
		return fmt.Errorf("firmware not found: id=%d", id)
	}

	delete(s.firmwares, id)
	key := fmt.Sprintf("%d:%s", f.ModelID, f.Version)
	delete(s.firmwareVersionIndex, key)

	return nil
}

// GetAllFirmwares 获取所有固件
func (s *MemoryStore) GetAllFirmwares(_ context.Context) ([]*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Firmware
	for _, f := range s.firmwares {
		result = append(result, f)
	}
	return result, nil
}

// CountFirmwaresByModel 按型号统计固件数量
func (s *MemoryStore) CountFirmwaresByModel(_ context.Context) (map[model.ID]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[model.ID]int)
	for _, f := range s.firmwares {
		result[f.ModelID]++
	}
	return result, nil
}
