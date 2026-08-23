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
	bufferAllocator BufferAllocator
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

// SetBufferAllocator 允许自定义缓冲区容量计算策略
func (s *ProgressService) SetBufferAllocator(fn BufferAllocator) {
	s.bufferAllocator = fn
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

	// 更新设备进度
	if err := s.deviceStore.UpdateDeviceProgress(ctx, device.ID, req.Progress); err != nil {
		return fmt.Errorf("failed to update device progress: %w", err)
	}

	// 查找对应的升级记录
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

// estimateProgressBufferSize 估算进度缓冲区容量
func (s *ProgressService) estimateProgressBufferSize(itemCount int) int {
	if s.bufferAllocator != nil {
		return s.bufferAllocator(itemCount)
	}
	return itemCount / 2
}

// BatchReportProgress 批量上报进度
func (s *ProgressService) BatchReportProgress(ctx context.Context, reports []*model.ReportProgressRequest) map[string]error {
	results := make(map[string]error)

	bufSize := s.estimateProgressBufferSize(len(reports))
	errDevices := make([]string, bufSize)
	errErrors := make([]error, bufSize)
	errIdx := 0

	for _, req := range reports {
		err := s.ReportProgress(ctx, req)
		if err != nil {
			results[req.DeviceID] = err
			errDevices[errIdx] = req.DeviceID
			errErrors[errIdx] = err
			errIdx++
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

	bufSize := s.estimateProgressBufferSize(len(records))
	statusCounts := make([]int, bufSize)
	countIdx := 0

	successCount := 0
	failCount := 0
	inProgressCount := 0

	for _, r := range records {
		switch r.Status {
		case model.UpgradeSuccess:
			successCount++
			statusCounts[countIdx] = 0
			countIdx++
		case model.UpgradeFailed:
			failCount++
			statusCounts[countIdx] = 1
			countIdx++
		case model.UpgradeInProgress:
			inProgressCount++
			statusCounts[countIdx] = 2
			countIdx++
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
