package store

import (
	"context"

	"fwupgrade/internal/model"
)

// ================ TaskStore 实现 ================

// CreateTask 创建任务
func (s *FileStore) CreateTask(ctx context.Context, t *model.UpgradeTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.CreateTask(ctx, t); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// GetTaskByID 根据ID获取任务
func (s *FileStore) GetTaskByID(ctx context.Context, id model.ID) (*model.UpgradeTask, error) {
	return s.memStore.GetTaskByID(ctx, id)
}

// ListTasks 列出任务
func (s *FileStore) ListTasks(ctx context.Context, page, pageSize int, status model.TaskStatus) ([]*model.UpgradeTask, int64, error) {
	return s.memStore.ListTasks(ctx, page, pageSize, status)
}

// ListActiveTasks 列出活跃任务
func (s *FileStore) ListActiveTasks(ctx context.Context) ([]*model.UpgradeTask, error) {
	return s.memStore.ListActiveTasks(ctx)
}

// ListTasksByModel 根据型号列出任务
func (s *FileStore) ListTasksByModel(ctx context.Context, modelID model.ID) ([]*model.UpgradeTask, error) {
	return s.memStore.ListTasksByModel(ctx, modelID)
}

// UpdateTask 更新任务
func (s *FileStore) UpdateTask(ctx context.Context, t *model.UpgradeTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateTask(ctx, t); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// UpdateTaskStatus 更新任务状态
func (s *FileStore) UpdateTaskStatus(ctx context.Context, id model.ID, status model.TaskStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateTaskStatus(ctx, id, status); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// UpdateTaskProgress 更新任务进度
func (s *FileStore) UpdateTaskProgress(ctx context.Context, id model.ID, successCount, failCount, pendingCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateTaskProgress(ctx, id, successCount, failCount, pendingCount); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// DeleteTask 删除任务
func (s *FileStore) DeleteTask(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.DeleteTask(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// GetAllTasks 获取所有任务
func (s *FileStore) GetAllTasks(ctx context.Context) ([]*model.UpgradeTask, error) {
	return s.memStore.GetAllTasks(ctx)
}

// SearchTasks 搜索任务
func (s *FileStore) SearchTasks(ctx context.Context, keyword string, page, pageSize int) ([]*model.UpgradeTask, int64, error) {
	return s.memStore.SearchTasks(ctx, keyword, page, pageSize)
}

// GetRecentTasks 获取最近任务
func (s *FileStore) GetRecentTasks(ctx context.Context, limit int) ([]*model.UpgradeTask, error) {
	return s.memStore.GetRecentTasks(ctx, limit)
}
