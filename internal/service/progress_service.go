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

	if req.Progress < 0 || req.Progress > 100 {
		return fmt.Errorf("progress must be between 0 and 100")
	}

	device, err := s.deviceStore.GetDeviceByDeviceID(ctx, req.DeviceID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	// 进度的单调推进由 store 层 UpdateDeviceProgress 的 max 语义保证，
	// 这里不再做独立预检，避免「读→写」之间的 check-then-act 竞态导致
	// 对合法的单调并发上报误报 "progress regression detected"。

	if req.TaskID > 0 {
		task, taskErr := s.taskStore.GetTaskByID(ctx, req.TaskID)
		if taskErr == nil {
			if task.Status != model.TaskRunning && task.Status != model.TaskPending {
				return fmt.Errorf("task %d is not active (status=%s), cannot report progress", req.TaskID, task.Status)
			}
		}
	}

	if err := s.deviceStore.UpdateDeviceProgress(ctx, device.ID, req.Progress); err != nil {
		return fmt.Errorf("failed to update device progress: %w", err)
	}

	records, err := s.recordStore.ListRecordsByTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("failed to list records: %w", err)
	}

	var matchedRecord *model.UpgradeRecord
	for _, r := range records {
		if r.DeviceID == req.DeviceID {
			matchedRecord = r
			break
		}
	}

	if matchedRecord != nil {
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
	}

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

	s.updateTaskProgress(ctx, req.TaskID)

	return nil
}

// GetDeviceProgress 获取设备升级进度
func (s *ProgressService) GetDeviceProgress(ctx context.Context, deviceID string) (*model.Device, error) {
	// store 层在读锁内返回不可变快照，单次读取即保证一致性，
	// 无需在锁外二次读取或比较取较大值（那本身也是 check-then-act 竞态）。
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
		if device.UpgradeProgress > 100 {
			return fmt.Errorf("device already at final progress: %d", device.UpgradeProgress)
		}
		if err := s.deviceStore.UpdateDeviceProgress(ctx, device.ID, 100); err != nil {
			return err
		}
		if err := s.deviceStore.UpdateDeviceStatus(ctx, device.ID, model.DeviceOnline); err != nil {
			return err
		}
	} else {
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
