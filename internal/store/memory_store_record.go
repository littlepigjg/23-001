package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"fwupgrade/internal/model"
)

// ================ RecordStore 实现 ================

// CreateRecord 创建记录
func (s *MemoryStore) CreateRecord(_ context.Context, r *model.UpgradeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID()
	r.ID = id
	s.records[id] = r

	return nil
}

// GetRecordByID 根据ID获取记录
func (s *MemoryStore) GetRecordByID(_ context.Context, id model.ID) (*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.records[id]
	if !ok {
		return nil, fmt.Errorf("record not found: id=%d", id)
	}
	return r, nil
}

// ListRecords 列出记录
func (s *MemoryStore) ListRecords(_ context.Context, page, pageSize int, status model.UpgradeStatus) ([]*model.UpgradeRecord, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []*model.UpgradeRecord
	for _, r := range s.records {
		if status != "" && r.Status != status {
			continue
		}
		records = append(records, r)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt.After(records[j].StartedAt)
	})

	total := int64(len(records))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.UpgradeRecord{}, total, nil
	}

	return records[start:end], total, nil
}

// ListRecordsByDevice 根据设备列出记录
func (s *MemoryStore) ListRecordsByDevice(_ context.Context, deviceID string) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeRecord
	for _, r := range s.records {
		if r.DeviceID == deviceID {
			result = append(result, r)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.After(result[j].StartedAt)
	})

	return result, nil
}

// ListRecordsByTask 根据任务列出记录
func (s *MemoryStore) ListRecordsByTask(_ context.Context, taskID model.ID) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeRecord
	for _, r := range s.records {
		if r.TaskID == taskID {
			result = append(result, r)
		}
	}
	return result, nil
}

// UpdateRecord 更新记录
func (s *MemoryStore) UpdateRecord(_ context.Context, r *model.UpgradeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.records[r.ID]; !ok {
		return fmt.Errorf("record not found: id=%d", r.ID)
	}

	s.records[r.ID] = r
	return nil
}

// UpdateRecordStatus 更新记录状态
func (s *MemoryStore) UpdateRecordStatus(_ context.Context, id model.ID, status model.UpgradeStatus, progress int, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.records[id]
	if !ok {
		return fmt.Errorf("record not found: id=%d", id)
	}

	r.Status = status
	r.Progress = progress
	if errorMsg != "" {
		r.ErrorMessage = errorMsg
	}
	if status == model.UpgradeSuccess || status == model.UpgradeFailed {
		now := time.Now()
		r.CompletedAt = &now
		r.Duration = now.Sub(r.StartedAt).Milliseconds()
	}

	return nil
}

// DeleteRecord 删除记录
func (s *MemoryStore) DeleteRecord(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.records[id]; !ok {
		return fmt.Errorf("record not found: id=%d", id)
	}

	delete(s.records, id)
	return nil
}

// GetAllRecords 获取所有记录
func (s *MemoryStore) GetAllRecords(ctx context.Context) ([]*model.UpgradeRecord, error) {
	if err := ValidateContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeRecord
	for _, r := range s.records {
		result = append(result, r)
	}
	return result, nil
}

// GetRecentRecords 获取最近记录
func (s *MemoryStore) GetRecentRecords(_ context.Context, limit int) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []*model.UpgradeRecord
	for _, r := range s.records {
		records = append(records, r)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt.After(records[j].StartedAt)
	})

	if limit > len(records) {
		limit = len(records)
	}
	return records[:limit], nil
}

// CountRecordsByStatus 按状态统计记录
func (s *MemoryStore) CountRecordsByStatus(_ context.Context) (map[model.UpgradeStatus]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[model.UpgradeStatus]int)
	for _, r := range s.records {
		result[r.Status]++
	}
	return result, nil
}

// CountTodayRecords 统计今日记录
func (s *MemoryStore) CountTodayRecords(ctx context.Context) (int, error) {
	if err := ValidateContext(ctx); err != nil {
		return 0, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	today := time.Now()
	count := 0
	for _, r := range s.records {
		if r.StartedAt.Year() == today.Year() && r.StartedAt.Month() == today.Month() && r.StartedAt.Day() == today.Day() {
			count++
		}
	}
	return count, nil
}
