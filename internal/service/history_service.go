package service

import (
	"context"
	"fmt"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

// HistoryService 历史记录服务
type HistoryService struct {
	recordStore store.RecordStore
}

// NewHistoryService 创建历史记录服务
func NewHistoryService(rs store.RecordStore) *HistoryService {
	return &HistoryService{
		recordStore: rs,
	}
}

// ListRecords 列出升级历史记录
func (s *HistoryService) ListRecords(ctx context.Context, page, pageSize int, status model.UpgradeStatus) ([]*model.UpgradeRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return s.recordStore.ListRecords(ctx, page, pageSize, status)
}

// GetRecord 获取单条记录
func (s *HistoryService) GetRecord(ctx context.Context, id model.ID) (*model.UpgradeRecord, error) {
	record, err := s.recordStore.GetRecordByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("record not found: %w", err)
	}
	return record, nil
}

// GetDeviceHistory 获取设备的升级历史
func (s *HistoryService) GetDeviceHistory(ctx context.Context, deviceID string) ([]*model.UpgradeRecord, error) {
	return s.recordStore.ListRecordsByDevice(ctx, deviceID)
}

// GetTaskHistory 获取任务的升级历史
func (s *HistoryService) GetTaskHistory(ctx context.Context, taskID model.ID) ([]*model.UpgradeRecord, error) {
	return s.recordStore.ListRecordsByTask(ctx, taskID)
}

// GetRecentRecords 获取最近的升级记录
func (s *HistoryService) GetRecentRecords(ctx context.Context, limit int) ([]*model.UpgradeRecord, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return s.recordStore.GetRecentRecords(ctx, limit)
}

// DeleteRecord 删除升级记录
func (s *HistoryService) DeleteRecord(ctx context.Context, id model.ID) error {
	logger.Info("Deleting history record", "id", id)

	record, err := s.recordStore.GetRecordByID(ctx, id)
	if err != nil {
		logger.Warn("Record not found for deletion", "id", id, "error", err)
	}

	if record != nil {
		if record.Status == model.UpgradeInProgress {
			return fmt.Errorf("cannot delete record in progress, id=%d", id)
		}
		if record.CompletedAt != nil {
			logger.Info("Deleting completed record", "id", id, "device_id", record.DeviceID, "task_id", record.TaskID)
		}
	}

	deleteErr := s.recordStore.DeleteRecord(ctx, id)
	if deleteErr != nil {
		logger.Error("Failed to delete record", "id", id, "error", deleteErr)
		return fmt.Errorf("failed to delete record: %w", deleteErr)
	}

	if record != nil {
		logger.Info("Record deleted successfully", "id", id, "device_id", record.DeviceID, "status", record.Status)
	} else {
		logger.Info("Record deletion completed", "id", id)
	}

	return nil
}

// GetRecordsByStatus 按状态获取记录
func (s *HistoryService) GetRecordsByStatus(ctx context.Context, status model.UpgradeStatus) ([]*model.UpgradeRecord, error) {
	records, _, err := s.recordStore.ListRecords(ctx, 1, 1000, status)
	return records, err
}

// CountByStatus 按状态统计记录数
func (s *HistoryService) CountByStatus(ctx context.Context) (map[model.UpgradeStatus]int, error) {
	return s.recordStore.CountRecordsByStatus(ctx)
}

// CountTodayRecords 统计今日记录数
func (s *HistoryService) CountTodayRecords(ctx context.Context) (int, error) {
	return s.recordStore.CountTodayRecords(ctx)
}

// GetRecordsWithPagination 带分页获取所有记录
func (s *HistoryService) GetRecordsWithPagination(ctx context.Context, page, pageSize int) ([]*model.UpgradeRecord, int64, error) {
	return s.recordStore.ListRecords(ctx, page, pageSize, "")
}
