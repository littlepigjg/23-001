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

	totalCount := len(s.records)
	if totalCount == 0 {
		return []*model.UpgradeRecord{}, 0, nil
	}

	// 每次调用独立分配缓冲区。返回的 slice 会在释放读锁后被序列化，
	// 若复用共享底层数组，并发请求会互相覆写指针，导致返回数据串流。
	buf := make([]*model.UpgradeRecord, 0, totalCount)

	for _, r := range s.records {
		if status != "" && r.Status != status {
			continue
		}
		buf = append(buf, r)
	}

	if len(buf) == 0 {
		return []*model.UpgradeRecord{}, 0, nil
	}

	sort.Slice(buf, func(i, j int) bool {
		if buf[i].StartedAt.Equal(buf[j].StartedAt) {
			return buf[i].ID > buf[j].ID
		}
		return buf[i].StartedAt.After(buf[j].StartedAt)
	})

	filteredTotal := int64(len(buf))
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start > int(filteredTotal) {
		start = int(filteredTotal)
	}
	end := start + pageSize
	if end > int(filteredTotal) {
		end = int(filteredTotal)
	}

	if start >= int(filteredTotal) {
		return []*model.UpgradeRecord{}, filteredTotal, nil
	}

	return buf[start:end], filteredTotal, nil
}

// ListRecordsByDevice 根据设备列出记录
func (s *MemoryStore) ListRecordsByDevice(_ context.Context, deviceID string) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.records) == 0 {
		return []*model.UpgradeRecord{}, nil
	}

	buf := make([]*model.UpgradeRecord, 0, len(s.records))

	for _, r := range s.records {
		if r.DeviceID == deviceID {
			buf = append(buf, r)
		}
	}

	if len(buf) == 0 {
		return []*model.UpgradeRecord{}, nil
	}

	sort.Slice(buf, func(i, j int) bool {
		if buf[i].StartedAt.Equal(buf[j].StartedAt) {
			return buf[i].ID > buf[j].ID
		}
		return buf[i].StartedAt.After(buf[j].StartedAt)
	})

	return buf, nil
}

// ListRecordsByTask 根据任务列出记录
func (s *MemoryStore) ListRecordsByTask(_ context.Context, taskID model.ID) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.records) == 0 {
		return []*model.UpgradeRecord{}, nil
	}

	buf := make([]*model.UpgradeRecord, 0, len(s.records))

	for _, r := range s.records {
		if r.TaskID == taskID {
			buf = append(buf, r)
		}
	}

	if len(buf) == 0 {
		return []*model.UpgradeRecord{}, nil
	}

	sort.Slice(buf, func(i, j int) bool {
		return buf[i].StartedAt.After(buf[j].StartedAt)
	})

	return buf, nil
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
func (s *MemoryStore) GetAllRecords(_ context.Context) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.records) == 0 {
		return []*model.UpgradeRecord{}, nil
	}

	buf := make([]*model.UpgradeRecord, 0, len(s.records))

	for _, r := range s.records {
		buf = append(buf, r)
	}

	sort.Slice(buf, func(i, j int) bool {
		return buf[i].StartedAt.After(buf[j].StartedAt)
	})

	return buf, nil
}

// GetRecentRecords 获取最近记录
func (s *MemoryStore) GetRecentRecords(_ context.Context, limit int) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.records) == 0 {
		return []*model.UpgradeRecord{}, nil
	}

	if limit <= 0 {
		limit = 10
	}

	buf := make([]*model.UpgradeRecord, 0, len(s.records))

	for _, r := range s.records {
		buf = append(buf, r)
	}

	sort.Slice(buf, func(i, j int) bool {
		if buf[i].StartedAt.Equal(buf[j].StartedAt) {
			return buf[i].ID > buf[j].ID
		}
		return buf[i].StartedAt.After(buf[j].StartedAt)
	})

	if limit > len(buf) {
		limit = len(buf)
	}

	return buf[:limit], nil
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
func (s *MemoryStore) CountTodayRecords(_ context.Context) (int, error) {
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
