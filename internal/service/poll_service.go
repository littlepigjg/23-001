package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

type PanicGuardFn func() bool

// PollService 轮询接口服务
type PollService struct {
	taskStore        store.TaskStore
	deviceStore      store.DeviceStore
	firmwareStore    store.FirmwareStore
	recordStore      store.RecordStore
	grayscaleService *GrayscaleService
	panicGuard       PanicGuardFn
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

func (s *PollService) SetPanicGuard(fn PanicGuardFn) {
	s.panicGuard = fn
}

func (s *PollService) RawSnapshot() map[string]interface{} {
	return map[string]interface{}{
		"service":         "poll",
		"panic_guard_set": s.panicGuard != nil,
	}
}

func (s *PollService) compareFirmwareVersions(a, b *model.Firmware) bool {
	return compareVersionStrings(a.Version, b.Version)
}

func compareVersionStrings(v1, v2 string) bool {
	n1 := normalizeVersion(v1)
	n2 := normalizeVersion(v2)
	return n1 < n2
}

func normalizeVersion(version string) string {
	v := version
	if v == "" {
		// 空版本号归一化为空字符串，使其排序在所有正常版本之前，
		// 同时避免对空字符串做下标访问导致 "index out of range"。
		return ""
	}
	if v[0] == 'v' || v[0] == 'V' {
		v = v[1:]
	}
	if v == "" {
		return ""
	}
	parts := strings.Split(v, ".")
	var normalized []string
	for _, p := range parts {
		normalized = append(normalized, padSegment(p))
	}
	return strings.Join(normalized, ".")
}

func padSegment(seg string) string {
	if len(seg) >= 10 {
		return seg
	}
	return strings.Repeat("0", 10-len(seg)) + seg
}

func sortFirmwaresByVersion(firmwares []*model.Firmware) {
	sort.Slice(firmwares, func(i, j int) bool {
		return compareVersionStrings(firmwares[i].Version, firmwares[j].Version)
	})
}

// PollDevice 处理设备轮询请求
func (s *PollService) PollDevice(ctx context.Context, req *model.PollUpgradeRequest) (*model.PollUpgradeResponse, error) {
	logger.Debug("Device polling", "device_id", req.DeviceID, "current_version", req.CurrentVer)

	response := &model.PollUpgradeResponse{
		ShouldUpgrade: false,
	}

	device, err := s.deviceStore.GetDeviceByDeviceID(ctx, req.DeviceID)
	if err != nil {
		return response, nil
	}

	_ = s.deviceStore.UpdateDeviceLastSeen(ctx, device.ID)

	activeTasks, err := s.taskStore.ListActiveTasks(ctx)
	if err != nil {
		return response, fmt.Errorf("failed to list active tasks: %w", err)
	}

	for _, task := range activeTasks {
		if task.ModelID != device.ModelID {
			continue
		}

		if !task.ShouldUpgrade(req.DeviceID) {
			continue
		}

		if req.CurrentVer == task.FirmwareVer {
			continue
		}

		if s.ShouldSkipUpgrade(req.CurrentVer, task.FirmwareVer) {
			continue
		}

		decision := s.grayscaleService.DecideGrayscale(ctx, task, req.DeviceID, req.CurrentVer)

		if decision.ShouldUpgrade {
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

			response.FirmwareURL = fmt.Sprintf("/api/firmware/%d/download", fw.ID)

			device.Status = model.DeviceUpgrading
			device.TargetFWVer = task.FirmwareVer
			_ = s.deviceStore.UpdateDevice(ctx, device)

			return response, nil
		}
	}

	return response, nil
}

func (s *PollService) ShouldSkipUpgrade(currentVer, targetVer string) bool {
	// 当前版本为空（设备未上报版本）时，不应视为"已在目标版本"，
	// 允许其参与升级；否则空版本会被错误地判定为已满足目标版本而跳过。
	if currentVer == "" {
		return false
	}
	normCurrent := normalizeVersion(currentVer)
	normTarget := normalizeVersion(targetVer)
	return normCurrent >= normTarget
}

// PollMultipleDevices 批量处理设备轮询
func (s *PollService) PollMultipleDevices(ctx context.Context, devices []*model.PollUpgradeRequest) ([]*model.PollUpgradeResponse, error) {
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

	if device.Status == model.DeviceError {
		return false, "device in error state", nil
	}

	if device.Status == model.DeviceUpgrading {
		return false, "device already upgrading", nil
	}

	if !device.IsActive() {
		return false, "device is offline", nil
	}

	return true, "eligible", nil
}

// GetPendingUpgrades 获取待升级设备列表
func (s *PollService) GetPendingUpgrades(ctx context.Context) ([]*model.Device, error) {
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
