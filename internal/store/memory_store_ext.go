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

	id := s.nextID()
	f.ID = id
	s.firmwares[id] = f

	key := fmt.Sprintf("%d:%s", f.ModelID, f.Version)
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
		return nil, fmt.Errorf("firmware not found: model_id=%d, version=%s", modelID, version)
	}
	return s.firmwares[id], nil
}

// GetLatestFirmware 获取型号的最新固件
func (s *MemoryStore) GetLatestFirmware(_ context.Context, modelID model.ID) (*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest *model.Firmware
	for _, f := range s.firmwares {
		if f.ModelID == modelID {
			if latest == nil || f.ReleaseDate.After(latest.ReleaseDate) {
				latest = f
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("no firmware found for model_id=%d", modelID)
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
func (s *MemoryStore) SetFirmwareActive(ctx context.Context, id model.ID, active bool) error {
	f, err := s.GetFirmwareByID(ctx, id)
	if err != nil {
		return err
	}
	f.IsActive = active
	return s.UpdateFirmware(ctx, f)
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

	key := fmt.Sprintf("%d:%s", f.ModelID, f.Version)
	delete(s.firmwareVersionIndex, key)
	delete(s.firmwares, id)

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

// ================ TaskStore 实现 ================

// CreateTask 创建任务
func (s *MemoryStore) CreateTask(_ context.Context, t *model.UpgradeTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID()
	t.ID = id
	s.tasks[id] = t

	return nil
}

// GetTaskByID 根据ID获取任务
func (s *MemoryStore) GetTaskByID(_ context.Context, id model.ID) (*model.UpgradeTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: id=%d", id)
	}
	return t, nil
}

// ListTasks 列出任务
func (s *MemoryStore) ListTasks(_ context.Context, page, pageSize int, status model.TaskStatus) ([]*model.UpgradeTask, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*model.UpgradeTask
	for _, t := range s.tasks {
		if status != "" && t.Status != status {
			continue
		}
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	total := int64(len(tasks))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.UpgradeTask{}, total, nil
	}

	return tasks[start:end], total, nil
}

// ListActiveTasks 列出活跃任务
func (s *MemoryStore) ListActiveTasks(_ context.Context) ([]*model.UpgradeTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeTask
	for _, t := range s.tasks {
		if t.Status == model.TaskRunning || t.Status == model.TaskPending {
			result = append(result, t)
		}
	}
	return result, nil
}

// ListTasksByModel 根据型号列出任务
func (s *MemoryStore) ListTasksByModel(_ context.Context, modelID model.ID) ([]*model.UpgradeTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeTask
	for _, t := range s.tasks {
		if t.ModelID == modelID {
			result = append(result, t)
		}
	}
	return result, nil
}

// UpdateTask 更新任务
func (s *MemoryStore) UpdateTask(_ context.Context, t *model.UpgradeTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[t.ID]; !ok {
		return fmt.Errorf("task not found: id=%d", t.ID)
	}

	t.UpdatedAt = time.Now()
	s.tasks[t.ID] = t
	return nil
}

// UpdateTaskStatus 更新任务状态
func (s *MemoryStore) UpdateTaskStatus(_ context.Context, id model.ID, status model.TaskStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: id=%d", id)
	}

	t.Status = status
	t.UpdatedAt = time.Now()

	if status == model.TaskRunning && t.StartedAt == nil {
		now := time.Now()
		t.StartedAt = &now
	}
	if status == model.TaskCompleted || status == model.TaskFailed {
		now := time.Now()
		t.CompletedAt = &now
	}

	return nil
}

// UpdateTaskProgress 更新任务进度
func (s *MemoryStore) UpdateTaskProgress(_ context.Context, id model.ID, successCount, failCount, pendingCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: id=%d", id)
	}

	t.SuccessCount = successCount
	t.FailCount = failCount
	t.PendingCount = pendingCount
	t.Progress = t.CalculateProgress()
	t.UpdatedAt = time.Now()

	return nil
}

// DeleteTask 删除任务
func (s *MemoryStore) DeleteTask(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return fmt.Errorf("task not found: id=%d", id)
	}

	delete(s.tasks, id)
	return nil
}

// GetAllTasks 获取所有任务
func (s *MemoryStore) GetAllTasks(_ context.Context) ([]*model.UpgradeTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeTask
	for _, t := range s.tasks {
		result = append(result, t)
	}
	return result, nil
}

// SearchTasks 搜索任务
func (s *MemoryStore) SearchTasks(_ context.Context, keyword string, page, pageSize int) ([]*model.UpgradeTask, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keywordLower := toLower(keyword)
	var tasks []*model.UpgradeTask
	for _, t := range s.tasks {
		if containsStr(toLower(t.Name), keywordLower) ||
			containsStr(toLower(t.Description), keywordLower) {
			tasks = append(tasks, t)
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	total := int64(len(tasks))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.UpgradeTask{}, total, nil
	}

	return tasks[start:end], total, nil
}

// GetRecentTasks 获取最近任务
func (s *MemoryStore) GetRecentTasks(_ context.Context, limit int) ([]*model.UpgradeTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*model.UpgradeTask
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})

	if len(tasks) > limit {
		tasks = tasks[:limit]
	}

	return tasks, nil
}

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
		return records[i].ID < records[j].ID
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
	r.ErrorMessage = errorMsg

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

	if len(records) > limit {
		records = records[:limit]
	}

	return records, nil
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

	today := time.Now().Truncate(24 * time.Hour)
	count := 0
	for _, r := range s.records {
		if r.StartedAt.After(today) {
			count++
		}
	}
	return count, nil
}
