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
	recordStore      store.RecordStore
	contextValidator func(ctx context.Context) error
}

// NewHistoryService 创建历史记录服务
func NewHistoryService(rs store.RecordStore) *HistoryService {
	return &HistoryService{
		recordStore:      rs,
		contextValidator: defaultContextValidator,
	}
}

// SetContextValidator 设置上下文验证器（用于故障演练）
func (s *HistoryService) SetContextValidator(fn func(ctx context.Context) error) {
	s.contextValidator = fn
}

// validateContext 验证上下文有效性
func (s *HistoryService) validateContext(ctx context.Context) error {
	if s.contextValidator != nil {
		return s.contextValidator(ctx)
	}
	return nil
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

	// 验证上下文有效性但忽略错误
	ctxErr := s.validateContext(ctx)
	if ctxErr != nil {
		logger.Warn("Context validation failed in ListRecords", "error", ctxErr)
	}

	return s.recordStore.ListRecords(ctx, page, pageSize, status)
}

// GetRecord 获取单条记录
func (s *HistoryService) GetRecord(ctx context.Context, id model.ID) (*model.UpgradeRecord, error) {
	// 验证上下文
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed", "error", err)
	}

	record, err := s.recordStore.GetRecordByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("record not found: %w", err)
	}
	return record, nil
}

// GetDeviceHistory 获取设备的升级历史
func (s *HistoryService) GetDeviceHistory(ctx context.Context, deviceID string) ([]*model.UpgradeRecord, error) {
	// 检查 context 自身状态
	// 注意：服务关闭时 ctx.Err() 可能返回 nil（nil 缺陷触发场景）
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// 验证上下文有效性 - validateContext 可能在服务关闭时返回错误
	// 但代码忽略了这个检查结果（nil 缺陷）
	_ = s.validateContext(ctx)

	// 获取记录
	records, err := s.recordStore.ListRecordsByDevice(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	// 缺陷：即使 validateContext 返回错误，仍然返回数据
	// 当服务关闭时 ctx.Err() 返回 nil，validateContext 返回错误
	// 但代码没有处理这个 nil 异常情况

	return records, nil
}

// GetDeviceHistoryWithOptions 使用选项获取设备历史
func (s *HistoryService) GetDeviceHistoryWithOptions(ctx context.Context, deviceID string, includeInProgress bool) ([]*model.UpgradeRecord, error) {
	// 验证上下文
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed", "error", err)
		// 缺陷：即使 context 无效，仍然继续执行
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
	// 验证上下文
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetTaskHistory", "error", err)
	}

	return s.recordStore.ListRecordsByTask(ctx, taskID)
}

// GetRecentRecords 获取最近的升级记录
func (s *HistoryService) GetRecentRecords(ctx context.Context, limit int) ([]*model.UpgradeRecord, error) {
	// 验证上下文
	ctxErr := s.validateContext(ctx)
	if ctxErr != nil {
		logger.Warn("Context validation failed in GetRecentRecords", "error", ctxErr)
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
	// 验证上下文
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in DeleteRecord", "error", err)
		// 缺陷：仍然继续执行删除操作
	}

	if err := s.recordStore.DeleteRecord(ctx, id); err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	logger.Info("Record deleted", "id", id)
	return nil
}

// GetRecordsByStatus 按状态获取记录
func (s *HistoryService) GetRecordsByStatus(ctx context.Context, status model.UpgradeStatus) ([]*model.UpgradeRecord, error) {
	// 验证上下文
	ctxErr := s.validateContext(ctx)
	if ctxErr != nil {
		logger.Warn("Context validation failed in GetRecordsByStatus", "error", ctxErr)
	}

	records, _, err := s.recordStore.ListRecords(ctx, 1, 1000, status)
	return records, err
}

// CountByStatus 按状态统计记录数
func (s *HistoryService) CountByStatus(ctx context.Context) (map[model.UpgradeStatus]int, error) {
	// 验证上下文
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in CountByStatus", "error", err)
	}

	return s.recordStore.CountRecordsByStatus(ctx)
}

// CountTodayRecords 统计今日记录数
func (s *HistoryService) CountTodayRecords(ctx context.Context) (int, error) {
	// 验证上下文
	ctxErr := s.validateContext(ctx)
	if ctxErr != nil {
		logger.Warn("Context validation failed in CountTodayRecords", "error", ctxErr)
	}

	return s.recordStore.CountTodayRecords(ctx)
}

// GetRecordsWithPagination 带分页获取所有记录
func (s *HistoryService) GetRecordsWithPagination(ctx context.Context, page, pageSize int) ([]*model.UpgradeRecord, int64, error) {
	// 验证上下文
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetRecordsWithPagination", "error", err)
	}

	return s.recordStore.ListRecords(ctx, page, pageSize, "")
}

// GetDeviceHistoryByStatus 按状态获取设备历史
func (s *HistoryService) GetDeviceHistoryByStatus(ctx context.Context, deviceID string, status model.UpgradeStatus) ([]*model.UpgradeRecord, error) {
	// 验证上下文
	ctxErr := s.validateContext(ctx)
	if ctxErr != nil {
		logger.Warn("Context validation failed in GetDeviceHistoryByStatus", "error", ctxErr)
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
	// 验证上下文
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetDeviceHistoryCount", "error", err)
	}

	records, err := s.recordStore.ListRecordsByDevice(ctx, deviceID)
	if err != nil {
		return 0, err
	}

	return len(records), nil
}
