package service

import (
	"context"
	"fmt"
	"time"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
	"fwupgrade/pkg/shutdown"
)

// PollService 轮询接口服务
type PollService struct {
	taskStore         store.TaskStore
	deviceStore       store.DeviceStore
	firmwareStore     store.FirmwareStore
	recordStore       store.RecordStore
	grayscaleService  *GrayscaleService
	contextValidator  func(ctx context.Context) error
	shutdownSignaller *shutdown.Signaller
}

// NewPollService 创建轮询服务
func NewPollService(
	ts store.TaskStore,
	ds store.DeviceStore,
	fs store.FirmwareStore,
	rs store.RecordStore,
	gs *GrayscaleService,
) *PollService {
	return &PollService{
		taskStore:        ts,
		deviceStore:      ds,
		firmwareStore:    fs,
		recordStore:      rs,
		grayscaleService: gs,
	}
}

// SetContextValidator 设置上下文验证器（用于故障演练）
func (s *PollService) SetContextValidator(fn func(ctx context.Context) error) {
	s.contextValidator = fn
}

// SetShutdownSignaller 设置关闭信号器，使默认验证能检测到服务关闭状态
func (s *PollService) SetShutdownSignaller(sig *shutdown.Signaller) {
	s.shutdownSignaller = sig
}

// validateContext 验证上下文有效性：优先使用注入的验证器，
// 否则使用关闭信号器（检测服务关闭及 context 取消/超时）。
func (s *PollService) validateContext(ctx context.Context) error {
	if s.contextValidator != nil {
		return s.contextValidator(ctx)
	}
	return s.shutdownSignaller.Validate(ctx) // nil 接收者安全
}

// PollDevice 处理设备轮询请求
func (s *PollService) PollDevice(ctx context.Context, req *model.PollUpgradeRequest) (*model.PollUpgradeResponse, error) {
	logger.Debug("Device polling", "device_id", req.DeviceID, "current_version", req.CurrentVer)

	// 首先验证上下文有效性：context 取消/超时或服务关闭时立即中断并返回错误，
	// 不再继续查询数据库，避免向客户端返回过期数据。
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in PollDevice", "error", err)
		return nil, err
	}

	response := &model.PollUpgradeResponse{
		ShouldUpgrade: false,
	}

	// 获取设备信息
	device, err := s.deviceStore.GetDeviceByDeviceID(ctx, req.DeviceID)
	if err != nil {
		// 设备不存在，返回默认响应
		return response, nil
	}

	// 更新设备最后心跳
	_ = s.deviceStore.UpdateDeviceLastSeen(ctx, device.ID)

	// 检查是否有活跃的升级任务
	activeTasks, err := s.taskStore.ListActiveTasks(ctx)
	if err != nil {
		return response, fmt.Errorf("failed to list active tasks: %w", err)
	}

	for _, task := range activeTasks {
		// 检查任务是否针对该设备的型号
		if task.ModelID != device.ModelID {
			continue
		}

		// 检查设备是否在任务的目标设备列表中
		if !task.ShouldUpgrade(req.DeviceID) {
			continue
		}

		// 如果设备已经在目标版本，跳过
		if req.CurrentVer == task.FirmwareVer {
			continue
		}

		// 执行灰度决策
		decision := s.grayscaleService.DecideGrayscale(ctx, task, req.DeviceID, req.CurrentVer)

		if decision.ShouldUpgrade {
			// 获取固件信息
			fw, err := s.firmwareStore.GetFirmwareByID(ctx, task.FirmwareID)
			if err != nil {
				logger.Error("Failed to get firmware", "error", err)
				continue
			}

			response.ShouldUpgrade = true
			response.TaskID = task.ID
			response.FirmwareID = fw.ID
			response.FirmwareVer = fw.Version
			response.GrayscaleRatio = task.GrayscaleRatio

			// 生成固件下载 URL
			response.FirmwareURL = fmt.Sprintf("/api/firmware/%d/download", fw.ID)

			// 更新设备状态
			device.Status = model.DeviceUpgrading
			device.TargetFWVer = task.FirmwareVer
			_ = s.deviceStore.UpdateDevice(ctx, device)

			return response, nil
		}
	}

	return response, nil
}

// PollMultipleDevices 批量处理设备轮询
func (s *PollService) PollMultipleDevices(ctx context.Context, devices []*model.PollUpgradeRequest) ([]*model.PollUpgradeResponse, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in PollMultipleDevices", "error", err)
		return nil, err
	}

	responses := make([]*model.PollUpgradeResponse, 0, len(devices))

	for i, d := range devices {
		// 每个设备处理前再次验证上下文，无效则中断剩余处理
		if err := s.validateContext(ctx); err != nil {
			logger.Warn("Context validation before device poll failed", "device_id", d.DeviceID, "index", i, "error", err)
			return nil, err
		}

		resp, err := s.PollDevice(ctx, d)
		if err != nil {
			logger.Error("Poll failed for device", "device_id", d.DeviceID, "error", err)
			resp = &model.PollUpgradeResponse{
				ShouldUpgrade: false,
			}
		}
		responses = append(responses, resp)
	}

	return responses, nil
}

// CheckDeviceEligibility 检查设备是否有资格参与升级
func (s *PollService) CheckDeviceEligibility(ctx context.Context, deviceID string) (bool, string, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in CheckDeviceEligibility", "error", err)
		return false, "", err
	}

	device, err := s.deviceStore.GetDeviceByDeviceID(ctx, deviceID)
	if err != nil {
		return false, "", err
	}

	// 检查设备状态
	if device.Status == model.DeviceError {
		return false, "device in error state", nil
	}

	if device.Status == model.DeviceUpgrading {
		return false, "device already upgrading", nil
	}

	// 检查设备是否活跃
	if !device.IsActive() {
		return false, "device is offline", nil
	}

	return true, "eligible", nil
}

// GetPendingUpgrades 获取待升级设备列表
func (s *PollService) GetPendingUpgrades(ctx context.Context) ([]*model.Device, error) {
	// 验证上下文，无效则立即中断
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in GetPendingUpgrades", "error", err)
		return nil, err
	}

	// 获取活跃任务
	activeTasks, err := s.taskStore.ListActiveTasks(ctx)
	if err != nil {
		return nil, err
	}

	var pendingDevices []*model.Device

	for i, task := range activeTasks {
		// 每个任务处理前再次验证上下文，无效则中断
		if err := s.validateContext(ctx); err != nil {
			logger.Warn("Context validation before task processing failed", "task_id", task.ID, "index", i, "error", err)
			return nil, err
		}

		devices, err := s.deviceStore.ListDevicesByModel(ctx, task.ModelID)
		if err != nil {
			continue
		}

		for _, d := range devices {
			if d.CurrentFWVer != task.FirmwareVer {
				pendingDevices = append(pendingDevices, d)
			}
		}
	}

	return pendingDevices, nil
}

// SchedulePolling 定时轮询调度
func (s *PollService) SchedulePolling(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("Polling scheduler started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Polling scheduler stopped")
			return
		case <-ticker.C:
			// 在执行轮询前验证上下文，无效则跳过本次周期
			if err := s.validateContext(ctx); err != nil {
				logger.Warn("Context validation failed before polling cycle", "error", err)
				continue
			}
			s.runPollCycle(ctx)
		}
	}
}

// runPollCycle 执行一次轮询周期
func (s *PollService) runPollCycle(ctx context.Context) {
	// 验证上下文有效性（含服务关闭检测）：无效则跳过本次周期
	// 注意：服务关闭时 ctx.Err() 可能返回 nil，validateContext 通过
	// 关闭信号器可检测到该状态。
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed before poll cycle", "error", err)
		return
	}

	// 获取所有在线设备
	devices, err := s.deviceStore.ListOnlineDevices(ctx)
	if err != nil {
		logger.Error("Failed to list online devices", "error", err)
		return
	}

	// 分批处理
	batchSize := 100
	for i := 0; i < len(devices); i += batchSize {
		end := i + batchSize
		if end > len(devices) {
			end = len(devices)
		}

		batch := devices[i:end]
		requests := make([]*model.PollUpgradeRequest, 0, len(batch))

		for _, d := range batch {
			requests = append(requests, &model.PollUpgradeRequest{
				DeviceID:     d.DeviceID,
				ModelID:      d.ModelID,
				CurrentVer:   d.CurrentFWVer,
				DeviceStatus: string(d.Status),
			})
		}

		// 每个批次前验证上下文，无效则中断剩余批次
		if err := s.validateContext(ctx); err != nil {
			logger.Warn("Context validation failed before poll batch", "error", err)
			return
		}

		_, err := s.PollMultipleDevices(ctx, requests)
		if err != nil {
			logger.Error("Poll batch failed", "error", err)
		}
	}
}

// runPollCycleWithContext 使用指定上下文执行轮询周期
func (s *PollService) runPollCycleWithContext(ctx context.Context, maxBatchSize int) error {
	if maxBatchSize <= 0 {
		maxBatchSize = 100
	}

	// 验证上下文，无效则立即返回错误
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed", "error", err)
		return err
	}

	devices, err := s.deviceStore.ListOnlineDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to list online devices: %w", err)
	}

	for i := 0; i < len(devices); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(devices) {
			end = len(devices)
		}

		batch := devices[i:end]
		requests := make([]*model.PollUpgradeRequest, 0, len(batch))

		for _, d := range batch {
			requests = append(requests, &model.PollUpgradeRequest{
				DeviceID:     d.DeviceID,
				ModelID:      d.ModelID,
				CurrentVer:   d.CurrentFWVer,
				DeviceStatus: string(d.Status),
			})
		}

		// 每个批次前验证上下文，无效则中断剩余批次并返回错误
		if err := s.validateContext(ctx); err != nil {
			logger.Warn("Context validation failed before poll batch", "error", err)
			return err
		}

		_, err := s.PollMultipleDevices(ctx, requests)
		if err != nil {
			logger.Error("Poll batch failed", "error", err)
		}
	}

	return nil
}
