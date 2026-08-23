package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"fwupgrade/internal/model"
)

// ================ TaskStore 实现 ================

// CreateTask 创建任务
func (s *MemoryStore) CreateTask(ctx context.Context, t *model.UpgradeTask) error {
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
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
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
		if t.Status == model.TaskPending || t.Status == model.TaskRunning {
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
func (s *MemoryStore) UpdateTaskStatus(ctx context.Context, id model.ID, status model.TaskStatus) error {
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
	if status == model.TaskCompleted || status == model.TaskFailed || status == model.TaskCancelled {
		now := time.Now()
		t.CompletedAt = &now
	}

	return nil
}

// UpdateTaskProgress 更新任务进度
func (s *MemoryStore) UpdateTaskProgress(ctx context.Context, id model.ID, successCount, failCount, pendingCount int) error {
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
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
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

	if limit > len(tasks) {
		limit = len(tasks)
	}
	return tasks[:limit], nil
}
