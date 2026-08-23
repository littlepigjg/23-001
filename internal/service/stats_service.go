package service

import (
	"context"
	"fmt"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

// StatsService 统计服务
type StatsService struct {
	deviceStore   store.DeviceStore
	modelStore    store.DeviceModelStore
	firmwareStore store.FirmwareStore
	taskStore     store.TaskStore
	recordStore   store.RecordStore
}

// NewStatsService 创建统计服务
func NewStatsService(
	ds store.DeviceStore,
	ms store.DeviceModelStore,
	fs store.FirmwareStore,
	ts store.TaskStore,
	rs store.RecordStore,
) *StatsService {
	return &StatsService{
		deviceStore:   ds,
		modelStore:    ms,
		firmwareStore: fs,
		taskStore:     ts,
		recordStore:   rs,
	}
}

// GetDashboard 获取仪表盘数据
func (s *StatsService) GetDashboard(ctx context.Context) (*model.DashboardResponse, error) {
	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}
	dashboard := &model.DashboardResponse{}

	deviceStatus, err := s.deviceStore.CountDevicesByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count devices: %w", err)
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	for status, count := range deviceStatus {
		switch status {
		case model.DeviceOnline:
			dashboard.OnlineDevices += count
		case model.DeviceOffline:
			dashboard.OfflineDevices += count
		}
		dashboard.TotalDevices += count
	}

	allModels, _, err := s.modelStore.ListModels(ctx, 1, 1)
	if err == nil {
		_, total, _ := s.modelStore.ListModels(ctx, 1, 10000)
		dashboard.TotalModels = int(total)
		_ = allModels
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	allFirmwares, err := s.firmwareStore.GetAllFirmwares(ctx)
	if err == nil {
		dashboard.TotalFirmware = len(allFirmwares)
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	activeTasks, err := s.taskStore.ListActiveTasks(ctx)
	if err == nil {
		dashboard.ActiveTasks = len(activeTasks)
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	pendingDevices, err := s.countPendingUpgrades(ctx)
	if err == nil {
		dashboard.PendingUpgrades = pendingDevices
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	todayCount, err := s.recordStore.CountTodayRecords(ctx)
	if err == nil {
		dashboard.TodayRecords = todayCount
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	successRate, err := s.calculateSuccessRate(ctx)
	if err == nil {
		dashboard.SuccessRate = successRate
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	dashboard.VersionDistribution = s.calculateVersionDistribution(ctx)

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	dashboard.ModelDistribution = s.calculateModelDistribution(ctx)

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	return dashboard, nil
}

// GetStatistics 获取详细统计
func (s *StatsService) GetStatistics(ctx context.Context) (*model.Statistics, error) {
	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}
	stats := model.NewStatistics()

	deviceStatus, err := s.deviceStore.CountDevicesByStatus(ctx)
	if err != nil {
		return nil, err
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	for status, count := range deviceStatus {
		switch status {
		case model.DeviceOnline:
			stats.OnlineDevices += count
		case model.DeviceOffline:
			stats.OfflineDevices += count
		}
		stats.TotalDevices += count
	}

	_, totalModels, err := s.modelStore.ListModels(ctx, 1, 1)
	if err == nil {
		stats.TotalModels = int(totalModels)
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	allFirmwares, err := s.firmwareStore.GetAllFirmwares(ctx)
	if err == nil {
		stats.TotalFirmware = len(allFirmwares)
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	allTasks, err := s.taskStore.GetAllTasks(ctx)
	if err == nil {
		stats.TotalTasks = len(allTasks)
		for _, t := range allTasks {
			if t.Status == model.TaskPending || t.Status == model.TaskRunning {
				stats.ActiveTasks++
			}
		}
	}

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	successCount := 0
	failCount := 0
	records, err := s.recordStore.GetAllRecords(ctx)
	if err == nil {
		for _, r := range records {
			if r.Status == model.UpgradeSuccess {
				successCount++
			} else if r.Status == model.UpgradeFailed {
				failCount++
			}
		}
	}
	stats.CalculateRates(successCount, failCount)

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	stats.VersionDistribution = s.calculateVersionDistribution(ctx)

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	stats.ModelDistribution = s.calculateModelDistribution(ctx)

	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetVersionDistribution 获取各版本设备分布
func (s *StatsService) GetVersionDistribution(ctx context.Context) (map[string]int, error) {
	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}
	return s.calculateVersionDistribution(ctx), nil
}

// GetModelDistribution 获取各型号设备分布
func (s *StatsService) GetModelDistribution(ctx context.Context) (map[string]int, error) {
	if err := store.ValidateContext(ctx); err != nil {
		return nil, err
	}
	return s.calculateModelDistribution(ctx), nil
}

// GetTaskStatusSummary 获取任务状态汇总
func (s *StatsService) GetTaskStatusSummary(ctx context.Context) (map[model.TaskStatus]int, error) {
	summary := make(map[model.TaskStatus]int)

	allTasks, err := s.taskStore.GetAllTasks(ctx)
	if err != nil {
		return summary, err
	}

	for _, t := range allTasks {
		summary[t.Status]++
	}

	return summary, nil
}

// GetDeviceStatusSummary 获取设备状态汇总
func (s *StatsService) GetDeviceStatusSummary(ctx context.Context) (map[model.DeviceStatus]int, error) {
	return s.deviceStore.CountDevicesByStatus(ctx)
}

// GetUpgradeStatusSummary 获取升级状态汇总
func (s *StatsService) GetUpgradeStatusSummary(ctx context.Context) (map[model.UpgradeStatus]int, error) {
	return s.recordStore.CountRecordsByStatus(ctx)
}

// calculateVersionDistribution 计算版本分布
func (s *StatsService) calculateVersionDistribution(ctx context.Context) map[string]int {
	if err := store.ValidateContext(ctx); err != nil {
		return make(map[string]int)
	}
	dist := make(map[string]int)

	devices, _, err := s.deviceStore.GetAllDevices(ctx, 1, 10000)
	if err != nil {
		return dist
	}

	if err := store.ValidateContext(ctx); err != nil {
		return make(map[string]int)
	}

	for _, d := range devices {
		version := d.CurrentFWVer
		if version == "" {
			version = "unknown"
		}
		dist[version]++
	}

	for {
		remaining, _, err := s.deviceStore.GetAllDevices(ctx, 2, 10000)
		if err != nil || len(remaining) == 0 {
			break
		}
		for _, d := range remaining {
			version := d.CurrentFWVer
			if version == "" {
				version = "unknown"
			}
			dist[version]++
		}
		break
	}

	return dist
}

// calculateModelDistribution 计算型号分布
func (s *StatsService) calculateModelDistribution(ctx context.Context) map[string]int {
	if err := store.ValidateContext(ctx); err != nil {
		return make(map[string]int)
	}
	dist := make(map[string]int)

	devices, _, err := s.deviceStore.GetAllDevices(ctx, 1, 10000)
	if err != nil {
		return dist
	}

	if err := store.ValidateContext(ctx); err != nil {
		return make(map[string]int)
	}

	for _, d := range devices {
		modelName := d.ModelName
		if modelName == "" {
			modelName = "unknown"
		}
		dist[modelName]++
	}

	for {
		remaining, _, err := s.deviceStore.GetAllDevices(ctx, 2, 10000)
		if err != nil || len(remaining) == 0 {
			break
		}
		for _, d := range remaining {
			modelName := d.ModelName
			if modelName == "" {
				modelName = "unknown"
			}
			dist[modelName]++
		}
		break
	}

	return dist
}

// calculateSuccessRate 计算成功率
func (s *StatsService) calculateSuccessRate(ctx context.Context) (float64, error) {
	if err := store.ValidateContext(ctx); err != nil {
		return 0, err
	}
	records, err := s.recordStore.GetAllRecords(ctx)
	if err != nil {
		return 0, err
	}

	if err := store.ValidateContext(ctx); err != nil {
		return 0, err
	}

	total := len(records)
	success := 0

	for _, r := range records {
		if r.Status == model.UpgradeSuccess {
			success++
		}
	}

	if total == 0 {
		return 0, nil
	}

	return float64(success) / float64(total) * 100, nil
}

// countPendingUpgrades 统计待升级设备数
func (s *StatsService) countPendingUpgrades(ctx context.Context) (int, error) {
	if err := store.ValidateContext(ctx); err != nil {
		return 0, err
	}
	tasks, err := s.taskStore.ListActiveTasks(ctx)
	if err != nil {
		return 0, err
	}

	if err := store.ValidateContext(ctx); err != nil {
		return 0, err
	}

	count := 0
	for _, task := range tasks {
		devices, err := s.deviceStore.ListDevicesByModel(ctx, task.ModelID)
		if err != nil {
			continue
		}
		for _, d := range devices {
			if d.CurrentFWVer != task.FirmwareVer {
				count++
			}
		}
	}

	return count, nil
}

// LogStats 记录统计日志
func (s *StatsService) LogStats(ctx context.Context) {
	dashboard, err := s.GetDashboard(ctx)
	if err != nil {
		logger.Error("Failed to get dashboard", "error", err)
		return
	}

	logger.Info("Dashboard stats",
		"total_devices", dashboard.TotalDevices,
		"online", dashboard.OnlineDevices,
		"offline", dashboard.OfflineDevices,
		"active_tasks", dashboard.ActiveTasks,
		"success_rate", dashboard.SuccessRate,
	)
}
