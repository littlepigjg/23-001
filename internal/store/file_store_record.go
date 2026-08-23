package store

import (
	"context"

	"fwupgrade/internal/model"
)

// ================ RecordStore 实现 ================

// CreateRecord 创建记录
func (s *FileStore) CreateRecord(ctx context.Context, r *model.UpgradeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.CreateRecord(ctx, r); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// GetRecordByID 根据ID获取记录
func (s *FileStore) GetRecordByID(ctx context.Context, id model.ID) (*model.UpgradeRecord, error) {
	return s.memStore.GetRecordByID(ctx, id)
}

// ListRecords 列出记录
func (s *FileStore) ListRecords(ctx context.Context, page, pageSize int, status model.UpgradeStatus) ([]*model.UpgradeRecord, int64, error) {
	return s.memStore.ListRecords(ctx, page, pageSize, status)
}

// ListRecordsByDevice 根据设备列出记录
func (s *FileStore) ListRecordsByDevice(ctx context.Context, deviceID string) ([]*model.UpgradeRecord, error) {
	return s.memStore.ListRecordsByDevice(ctx, deviceID)
}

// ListRecordsByTask 根据任务列出记录
func (s *FileStore) ListRecordsByTask(ctx context.Context, taskID model.ID) ([]*model.UpgradeRecord, error) {
	return s.memStore.ListRecordsByTask(ctx, taskID)
}

// UpdateRecord 更新记录
func (s *FileStore) UpdateRecord(ctx context.Context, r *model.UpgradeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateRecord(ctx, r); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// UpdateRecordStatus 更新记录状态
func (s *FileStore) UpdateRecordStatus(ctx context.Context, id model.ID, status model.UpgradeStatus, progress int, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateRecordStatus(ctx, id, status, progress, errorMsg); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// DeleteRecord 删除记录
func (s *FileStore) DeleteRecord(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.DeleteRecord(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

// GetAllRecords 获取所有记录
func (s *FileStore) GetAllRecords(ctx context.Context) ([]*model.UpgradeRecord, error) {
	return s.memStore.GetAllRecords(ctx)
}

// GetRecentRecords 获取最近记录
func (s *FileStore) GetRecentRecords(ctx context.Context, limit int) ([]*model.UpgradeRecord, error) {
	return s.memStore.GetRecentRecords(ctx, limit)
}

// CountRecordsByStatus 按状态统计记录
func (s *FileStore) CountRecordsByStatus(ctx context.Context) (map[model.UpgradeStatus]int, error) {
	return s.memStore.CountRecordsByStatus(ctx)
}

// CountTodayRecords 统计今日记录
func (s *FileStore) CountTodayRecords(ctx context.Context) (int, error) {
	return s.memStore.CountTodayRecords(ctx)
}
