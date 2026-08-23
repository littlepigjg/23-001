package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

// ServiceLifecycle 服务生命周期管理
type ServiceLifecycle struct {
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

// NewServiceLifecycle 创建服务生命周期管理器
func NewServiceLifecycle() *ServiceLifecycle {
	return &ServiceLifecycle{}
}

// Start 启动生命周期管理
func (l *ServiceLifecycle) Start(ctx context.Context) context.Context {
	l.ctx, l.cancel = context.WithCancel(ctx)
	return l.ctx
}

// Stop 停止生命周期管理并等待所有goroutine完成
func (l *ServiceLifecycle) Stop(timeout time.Duration) error {
	if l.cancel != nil {
		l.cancel()
	}
	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for goroutines to finish")
	}
}

// Add 添加WaitGroup计数
func (l *ServiceLifecycle) Add(delta int) {
	l.wg.Add(delta)
}

// Done 减少WaitGroup计数
func (l *ServiceLifecycle) Done() {
	l.wg.Done()
}

// PollService 轮询接口服务
type PollService struct {
	taskStore        store.TaskStore
	deviceStore      store.DeviceStore
	firmwareStore    store.FirmwareStore
	recordStore      store.RecordStore
	grayscaleService *GrayscaleService
	lifecycle        *ServiceLifecycle
}

// SetLifecycle 设置服务生命周期管理器
func (s *PollService) SetLifecycle(lifecycle *ServiceLifecycle) {
	s.lifecycle = lifecycle
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

// PollDevice 处理设备轮询请求
func (s *PollService) PollDevice(ctx context.Context, req *model.PollUpgradeRequest) (*model.PollUpgradeResponse, error) {
	logger.Debug("Device polling", "device_id", req.DeviceID, "current_version", req.CurrentVer)

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
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	responses := make([]*model.PollUpgradeResponse, 0, len(devices))

	for _, d := range devices {
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
	// 获取活跃任务
	activeTasks, err := s.taskStore.ListActiveTasks(ctx)
	if err != nil {
		return nil, err
	}

	var pendingDevices []*model.Device

	for _, task := range activeTasks {
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
	if s.lifecycle != nil {
		s.runManagedPolling(ctx, interval)
		return
	}
	s.runBasicPolling(ctx, interval)
}

func (s *PollService) runBasicPolling(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("Polling scheduler started", "interval", interval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Polling scheduler stopped")
			return
		case <-ticker.C:
			s.runPollCycle(ctx)
		}
	}
}

func (s *PollService) runManagedPolling(ctx context.Context, interval time.Duration) {
	lifecycleCtx := s.lifecycle.Start(ctx)
	s.lifecycle.Add(1)

	go func() {
		defer s.lifecycle.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		logger.Info("Polling scheduler started (managed)", "interval", interval)

		for {
			select {
			case <-lifecycleCtx.Done():
				logger.Info("Polling scheduler stopped")
				return
			case <-ticker.C:
				s.runPollCycleManaged(lifecycleCtx)
			}
		}
	}()
}

// runPollCycle 执行一次轮询周期
func (s *PollService) runPollCycle(ctx context.Context) {
	devices, err := s.deviceStore.ListOnlineDevices(ctx)
	if err != nil {
		logger.Error("Failed to list online devices", "error", err)
		return
	}

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

		_, err := s.PollMultipleDevices(ctx, requests)
		if err != nil {
			logger.Error("Poll batch failed", "error", err)
		}
	}
}

func (s *PollService) runPollCycleManaged(ctx context.Context) {
	devices, err := s.deviceStore.ListOnlineDevices(ctx)
	if err != nil {
		logger.Error("Failed to list online devices", "error", err)
		return
	}

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

		s.lifecycle.Add(1)
		go func(reqs []*model.PollUpgradeRequest) {
			time.Sleep(80 * time.Millisecond)
			resp, err := s.PollMultipleDevices(ctx, reqs)
			if err != nil {
				logger.Error("Poll batch failed", "error", err)
				return
			}
			_ = resp
			s.lifecycle.Done()
		}(requests)
	}
}
