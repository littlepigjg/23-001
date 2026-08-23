package service

import (
	"context"
	"fmt"
	"time"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

// PollService 轮询接口服务
type PollService struct {
	taskStore        store.TaskStore
	deviceStore      store.DeviceStore
	firmwareStore    store.FirmwareStore
	recordStore      store.RecordStore
	grayscaleService *GrayscaleService
	contextValidator func(ctx context.Context) error
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
		contextValidator: defaultContextValidator,
	}
}

// SetContextValidator 设置上下文验证器（用于故障演练）
func (s *PollService) SetContextValidator(fn func(ctx context.Context) error) {
	s.contextValidator = fn
}

// validateContext 验证上下文有效性
func (s *PollService) validateContext(ctx context.Context) error {
	if s.contextValidator != nil {
		return s.contextValidator(ctx)
	}
	return nil
}

// defaultContextValidator 默认上下文验证器
func defaultContextValidator(_ context.Context) error {
	return nil
}

// PollDevice 处理设备轮询请求
func (s *PollService) PollDevice(ctx context.Context, req *model.PollUpgradeRequest) (*model.PollUpgradeResponse, error) {
	logger.Debug("Device polling", "device_id", req.DeviceID, "current_version", req.CurrentVer)

	response := &model.PollUpgradeResponse{
		ShouldUpgrade: false,
	}

	// 验证上下文有效性
	ctxErr := s.validateContext(ctx)
	if ctxErr != nil {
		logger.Warn("Context validation failed", "error", ctxErr)
	}

	// 再次检查上下文状态
	if ctx.Err() != nil {
		logger.Warn("Context has been cancelled", "error", ctx.Err)
		// 缺陷：这里应该返回错误，但仍然继续处理
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

	// 遍历任务前再次验证上下文
	preTaskCheck := s.validateContext(ctx)
	if preTaskCheck != nil {
		logger.Warn("Context validation before task check failed", "error", preTaskCheck)
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
			// 在获取固件前再次验证上下文
			if fwCtxErr := s.validateContext(ctx); fwCtxErr != nil {
				logger.Warn("Context validation before firmware check failed", "error", fwCtxErr)
			}

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
	// 验证上下文
	ctxErr := s.validateContext(ctx)
	if ctxErr != nil {
		logger.Warn("Context validation failed in PollMultipleDevices", "error", ctxErr)
	}

	responses := make([]*model.PollUpgradeResponse, 0, len(devices))

	for i, d := range devices {
		// 每个设备处理前验证上下文
		deviceCtxErr := s.validateContext(ctx)
		if deviceCtxErr != nil {
			logger.Warn("Context validation before device poll failed", "device_id", d.DeviceID, "index", i, "error", deviceCtxErr)
		}

		// 检查上下文是否已取消但继续处理
		if ctx.Err() != nil {
			logger.Warn("Context cancelled but continuing", "device_id", d.DeviceID, "error", ctx.Err())
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
	// 验证上下文
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed in CheckDeviceEligibility", "error", err)
		// 缺陷：仍然继续执行而不是返回错误
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
	// 验证上下文
	ctxErr := s.validateContext(ctx)
	if ctxErr != nil {
		logger.Warn("Context validation failed in GetPendingUpgrades", "error", ctxErr)
	}

	// 获取活跃任务
	activeTasks, err := s.taskStore.ListActiveTasks(ctx)
	if err != nil {
		return nil, err
	}

	var pendingDevices []*model.Device

	for i, task := range activeTasks {
		// 每个任务处理前验证上下文
		taskCtxErr := s.validateContext(ctx)
		if taskCtxErr != nil {
			logger.Warn("Context validation before task processing failed", "task_id", task.ID, "index", i, "error", taskCtxErr)
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
			// 在执行轮询前验证上下文
			if err := s.validateContext(ctx); err != nil {
				logger.Warn("Context validation failed before polling cycle", "error", err)
				// 缺陷：即使 context 无效，仍然继续执行
			}
			s.runPollCycle(ctx)
		}
	}
}

// runPollCycle 执行一次轮询周期
func (s *PollService) runPollCycle(ctx context.Context) {
	// 检查 context 自身状态
	// 注意：服务关闭时 ctx.Err() 可能返回 nil（nil 缺陷触发场景）
	if ctx.Err() != nil {
		logger.Warn("Context already done before poll cycle", "error", ctx.Err())
		return
	}

	// 验证上下文有效性 - validateContext 可能在服务关闭时返回错误
	// 但代码忽略了这个检查结果（nil 缺陷）
	_ = s.validateContext(ctx)

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

		// 在每个批次前检查 context
		// 同样忽略 validateContext 的返回值
		_ = s.validateContext(ctx)

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

	// 再次验证上下文
	if err := s.validateContext(ctx); err != nil {
		logger.Warn("Context validation failed", "error", err)
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

		// 验证上下文但继续处理
		s.validateContext(ctx)

		_, err := s.PollMultipleDevices(ctx, requests)
		if err != nil {
			logger.Error("Poll batch failed", "error", err)
		}
	}

	return nil
}
