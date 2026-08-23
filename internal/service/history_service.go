package service

import (
	"context"
	"fmt"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
	"fwupgrade/pkg/shutdown"
)

// HistoryService 历史记录服务
type HistoryService struct {
	recordStore       store.RecordStore
	contextValidator  func(ctx context.Context) error
	shutdownSignaller *shutdown.Signaller
}

// NewHistoryService 创建历史记录服务
func NewHistoryService(rs store.RecordStore) *HistoryService {
	return &HistoryService{
		recordStore: rs,
	}
}

// SetContextValidator 设置上下文验证器（用于故障演练）
func (s *HistoryService) SetContextValidator(fn func(ctx context.Context) error) {
	s.contextValidator = fn
}

// SetShutdownSignaller 设置关闭信号器，使默认验证能检测到服务关闭状态
func (s *HistoryService) SetShutdownSignaller(sig *shutdown.Signaller) {
	s.shutdownSignaller = sig
}

// validateContext 验证上下文有效性：优先使用注入的验证器，
// 否则使用关闭信号器（检测服务关闭及 context 取消/超时）。
func (s *HistoryService) validateContext(ctx context.Context) error {
	if s.contextValidator != nil {
		return s.contextValidator(ctx)
	}
	return s.shutdownSignaller.Validate(ctx) // nil 接收者安全
}

// ListRecords 列出升级历史记录
func (s *HistoryService) ListRecords(ctx context.Context, page, pageSize int, status model.UpgradeStatus) ([]*model.UpgradeRecord, int64, error) {
	// 验证上下文，无效则立即中断，避免返回过期数据
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in ListRecords", "error", err)
		return nil, 0, err
	}

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
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetRecord", "error", err)
		return nil, err
	}

	record, err := s.recordStore.GetRecordByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("record not found: %w", err)
	}
	return record, nil
}

// GetDeviceHistory 获取设备的升级历史
func (s *HistoryService) GetDeviceHistory(ctx context.Context, deviceID string) ([]*model.UpgradeRecord, error) {
	// 验证上下文有效性（含服务关闭检测）：即使 ctx.Err() 返回 nil，
	// 关闭信号器也能检测到服务关闭状态并立即中断，避免返回过期数据。
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetDeviceHistory", "error", err)
		return nil, err
	}

	// 获取记录
	records, err := s.recordStore.ListRecordsByDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return records, nil
}

// GetDeviceHistoryWithOptions 使用选项获取设备历史
func (s *HistoryService) GetDeviceHistoryWithOptions(ctx context.Context, deviceID string, includeInProgress bool) ([]*model.UpgradeRecord, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetDeviceHistoryWithOptions", "error", err)
		return nil, err
	}

	records, err := s.recordStore.ListRecordsByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device history: %w", err)
	}

	if !includeInProgress {
		filtered := make([]*model.UpgradeRecord, 0)
		for _, r := range records {
			if r.Status != model.UpgradeInProgress {
				filtered = append(filtered, r)
			}
		}
		records = filtered
	}

	return records, nil
}

// GetTaskHistory 获取任务的升级历史
func (s *HistoryService) GetTaskHistory(ctx context.Context, taskID model.ID) ([]*model.UpgradeRecord, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetTaskHistory", "error", err)
		return nil, err
	}

	return s.recordStore.ListRecordsByTask(ctx, taskID)
}

// GetRecentRecords 获取最近的升级记录
func (s *HistoryService) GetRecentRecords(ctx context.Context, limit int) ([]*model.UpgradeRecord, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetRecentRecords", "error", err)
		return nil, err
	}

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
	// 验证上下文，无效则立即中断，避免在关闭过程中执行删除操作
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in DeleteRecord", "error", err)
		return err
	}

	if err := s.recordStore.DeleteRecord(ctx, id); err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	logger.Info("Record deleted", "id", id)
	return nil
}

// GetRecordsByStatus 按状态获取记录
func (s *HistoryService) GetRecordsByStatus(ctx context.Context, status model.UpgradeStatus) ([]*model.UpgradeRecord, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetRecordsByStatus", "error", err)
		return nil, err
	}

	records, _, err := s.recordStore.ListRecords(ctx, 1, 1000, status)
	return records, err
}

// CountByStatus 按状态统计记录数
func (s *HistoryService) CountByStatus(ctx context.Context) (map[model.UpgradeStatus]int, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in CountByStatus", "error", err)
		return nil, err
	}

	return s.recordStore.CountRecordsByStatus(ctx)
}

// CountTodayRecords 统计今日记录数
func (s *HistoryService) CountTodayRecords(ctx context.Context) (int, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in CountTodayRecords", "error", err)
		return 0, err
	}

	return s.recordStore.CountTodayRecords(ctx)
}

// GetRecordsWithPagination 带分页获取所有记录
func (s *HistoryService) GetRecordsWithPagination(ctx context.Context, page, pageSize int) ([]*model.UpgradeRecord, int64, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetRecordsWithPagination", "error", err)
		return nil, 0, err
	}

	return s.recordStore.ListRecords(ctx, page, pageSize, "")
}

// GetDeviceHistoryByStatus 按状态获取设备历史
func (s *HistoryService) GetDeviceHistoryByStatus(ctx context.Context, deviceID string, status model.UpgradeStatus) ([]*model.UpgradeRecord, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetDeviceHistoryByStatus", "error", err)
		return nil, err
	}

	records, err := s.recordStore.ListRecordsByDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	if status != "" {
		filtered := make([]*model.UpgradeRecord, 0)
		for _, r := range records {
			if r.Status == status {
				filtered = append(filtered, r)
			}
		}
		records = filtered
	}

	return records, nil
}

// GetDeviceHistoryCount 获取设备历史记录数
func (s *HistoryService) GetDeviceHistoryCount(ctx context.Context, deviceID string) (int, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetDeviceHistoryCount", "error", err)
		return 0, err
	}

	records, err := s.recordStore.ListRecordsByDevice(ctx, deviceID)
	if err != nil {
		return 0, err
	}

	return len(records), nil
}
