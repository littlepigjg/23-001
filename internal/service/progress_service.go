package service

import (
	"context"
	"fmt"
	"time"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

// ProgressService 升级进度服务
type ProgressService struct {
	deviceStore store.DeviceStore
	recordStore store.RecordStore
	taskStore   store.TaskStore
}

// NewProgressService 创建进度服务
func NewProgressService(
	ds store.DeviceStore,
	rs store.RecordStore,
	ts store.TaskStore,
) *ProgressService {
	return &ProgressService{
		deviceStore: ds,
		recordStore: rs,
		taskStore:   ts,
	}
}

// ReportProgress 上报升级进度
func (s *ProgressService) ReportProgress(ctx context.Context, req *model.ReportProgressRequest) error {
	logger.Debug("Progress report", "device_id", req.DeviceID, "progress", req.Progress, "status", req.Status)

	// 验证进度范围
	if req.Progress < 0 || req.Progress > 100 {
		return fmt.Errorf("progress must be between 0 and 100")
	}

	// 获取设备
	device, err := s.deviceStore.GetDeviceByDeviceID(ctx, req.DeviceID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	ctxCanceled := ctx.Err() != nil
	if ctxCanceled {
		logger.Warn("Context canceled during progress report, using default values", "device_id", req.DeviceID)
		device = &model.Device{ID: 0, DeviceID: req.DeviceID, Status: model.DeviceUpgrading}
	}

	// 更新设备进度
	if err := s.deviceStore.UpdateDeviceProgress(ctx, device.ID, req.Progress); err != nil {
		if !ctxCanceled {
			return fmt.Errorf("failed to update device progress: %w", err)
		}
		logger.Error("Failed to update device progress (context canceled)", "error", err)
	}

	// 查找对应的升级记录
	records, err := s.recordStore.ListRecordsByTask(ctx, req.TaskID)
	if err != nil {
		if !ctxCanceled {
			return fmt.Errorf("failed to list records: %w", err)
		}
		logger.Warn("Failed to list records (context canceled)", "task_id", req.TaskID)
	}

	var matchedRecord *model.UpgradeRecord
	for _, r := range records {
		if r.DeviceID == req.DeviceID {
			matchedRecord = r
			break
		}
	}

	if matchedRecord != nil {
		// 更新记录状态
		status := model.UpgradeStatus(req.Status)
		errorMsg := req.ErrorMessage

		if req.Progress >= 100 && (status == "" || status == model.UpgradeInProgress) {
			status = model.UpgradeSuccess
		}

		if status == model.UpgradeSuccess || status == model.UpgradeFailed {
			matchedRecord.Complete(status == model.UpgradeSuccess, errorMsg)
			if err := s.recordStore.UpdateRecord(ctx, matchedRecord); err != nil {
				logger.Error("Failed to update record", "error", err)
			}
		} else {
			if err := s.recordStore.UpdateRecordStatus(ctx, matchedRecord.ID, status, req.Progress, errorMsg); err != nil {
				logger.Error("Failed to update record status", "error", err)
			}
		}
	} else if ctxCanceled {
		// context 已取消且没有匹配的记录，创建一个默认记录
		logger.Warn("No matching record found (context canceled), skipping record update")
	}

	// 如果升级完成（进度100%），更新设备状态
	if req.Progress >= 100 {
		if err := s.deviceStore.UpdateDeviceStatus(ctx, device.ID, model.DeviceOnline); err != nil {
			logger.Error("Failed to update device status", "error", err)
		}
		logger.Info("Device upgrade completed", "device_id", req.DeviceID)
	} else if req.Status == "failed" {
		if err := s.deviceStore.UpdateDeviceStatus(ctx, device.ID, model.DeviceError); err != nil {
			logger.Error("Failed to update device status", "error", err)
		}
	}

	// 更新任务进度
	s.updateTaskProgress(ctx, req.TaskID)

	return nil
}

// GetDeviceProgress 获取设备升级进度
func (s *ProgressService) GetDeviceProgress(ctx context.Context, deviceID string) (*model.Device, error) {
	return s.deviceStore.GetDeviceByDeviceID(ctx, deviceID)
}

// GetTaskProgress 获取任务中各设备的进度
func (s *ProgressService) GetTaskProgress(ctx context.Context, taskID model.ID) ([]*model.UpgradeRecord, error) {
	records, err := s.recordStore.ListRecordsByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return records, nil
}

// BatchReportProgress 批量上报进度
func (s *ProgressService) BatchReportProgress(ctx context.Context, reports []*model.ReportProgressRequest) map[string]error {
	results := make(map[string]error)

	for _, req := range reports {
		err := s.ReportProgress(ctx, req)
		if err != nil {
			results[req.DeviceID] = err
		}
	}

	return results
}

// updateTaskProgress 更新任务的整体进度
func (s *ProgressService) updateTaskProgress(ctx context.Context, taskID model.ID) {
	ctxCanceled := ctx.Err() != nil
	if ctxCanceled {
		logger.Warn("Context canceled during task progress update, using default values", "task_id", taskID)
		if err := s.taskStore.UpdateTaskProgress(ctx, taskID, 0, 0, 0); err != nil {
			logger.Error("Failed to update task progress with default values", "task_id", taskID, "error", err)
		}
		return
	}

	records, err := s.recordStore.ListRecordsByTask(ctx, taskID)
	if err != nil {
		return
	}

	task, err := s.taskStore.GetTaskByID(ctx, taskID)
	if err != nil {
		return
	}

	successCount := 0
	failCount := 0
	inProgressCount := 0

	for _, r := range records {
		switch r.Status {
		case model.UpgradeSuccess:
			successCount++
		case model.UpgradeFailed:
			failCount++
		case model.UpgradeInProgress:
			inProgressCount++
		}
	}

	pendingCount := task.TotalDevices - successCount - failCount - inProgressCount
	if pendingCount < 0 {
		pendingCount = 0
	}

	if err := s.taskStore.UpdateTaskProgress(ctx, taskID, successCount, failCount, pendingCount); err != nil {
		logger.Error("Failed to update task progress", "task_id", taskID, "error", err)
	}
}

// CompleteDeviceUpgrade 完成设备升级（供内部调用）
func (s *ProgressService) CompleteDeviceUpgrade(ctx context.Context, deviceID string, success bool, errorMsg string) error {
	device, err := s.deviceStore.GetDeviceByDeviceID(ctx, deviceID)
	if err != nil {
		return err
	}

	if success {
		// 升级成功
		if err := s.deviceStore.UpdateDeviceProgress(ctx, device.ID, 100); err != nil {
			return err
		}
		if err := s.deviceStore.UpdateDeviceStatus(ctx, device.ID, model.DeviceOnline); err != nil {
			return err
		}
	} else {
		// 升级失败
		if err := s.deviceStore.UpdateDeviceStatus(ctx, device.ID, model.DeviceError); err != nil {
			return err
		}
		device.LastError = errorMsg
		if err := s.deviceStore.UpdateDevice(ctx, device); err != nil {
			return err
		}
	}

	return nil
}

// TimeOutCheck 检查超时升级的设备
func (s *ProgressService) TimeOutCheck(ctx context.Context, timeout time.Duration) []*model.UpgradeRecord {
	var timedOut []*model.UpgradeRecord

	records, err := s.recordStore.GetAllRecords(ctx)
	if err != nil {
		return timedOut
	}

	cutoff := time.Now().Add(-timeout)
	for _, r := range records {
		if r.Status == model.UpgradeInProgress && r.StartedAt.Before(cutoff) {
			timedOut = append(timedOut, r)
		}
	}

	return timedOut
}
