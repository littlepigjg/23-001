package service

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

type dashboardPartial struct {
	onlineDevices   int
	offlineDevices  int
	totalDevices    int
	totalModels     int
	totalFirmware   int
	activeTasks     int
	pendingUpgrades int
	todayRecords    int
	successRate     float64
	versionDist     map[string]int
	modelDist       map[string]int
	err             error
}

type metricsCollector struct {
	eventCh   chan struct{}
	resultCh  chan *dashboardPartial
	guardFn   func(op string) bool
	mu        sync.Mutex
	active    bool
	snapshot  map[string]int
	closeOnce sync.Once
}

// StatsService 统计服务
type StatsService struct {
	deviceStore   store.DeviceStore
	modelStore    store.DeviceModelStore
	firmwareStore store.FirmwareStore
	taskStore     store.TaskStore
	recordStore   store.RecordStore
	collector     *metricsCollector
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
		collector:     newMetricsCollector(),
	}
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		eventCh:  make(chan struct{}, 1),
		resultCh: make(chan *dashboardPartial, 2),
		snapshot: make(map[string]int),
	}
}

func (c *metricsCollector) SetDiagnosticGuard(fn func(op string) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.guardFn = fn
}

func (c *metricsCollector) RawSnapshot() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make(map[string]int, len(c.snapshot))
	for k, v := range c.snapshot {
		result[k] = v
	}
	return result
}

func (c *metricsCollector) recordMetric(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.snapshot[name]++
}

func (c *metricsCollector) shouldSkip(op string) bool {
	c.mu.Lock()
	fn := c.guardFn
	c.mu.Unlock()
	if fn != nil {
		return fn(op)
	}
	return false
}

// SetDiagnosticGuard 设置诊断守护函数
func (s *StatsService) SetDiagnosticGuard(fn func(op string) bool) {
	s.collector.SetDiagnosticGuard(fn)
}

// RawSnapshot 获取原始诊断快照
func (s *StatsService) RawSnapshot() map[string]int {
	return s.collector.RawSnapshot()
}

// StartCollector 启动指标收集器
func (s *StatsService) StartCollector(ctx context.Context) {
	s.collector.mu.Lock()
	if s.collector.active {
		s.collector.mu.Unlock()
		return
	}
	s.collector.active = true
	s.collector.mu.Unlock()

	go s.collectorLoop(ctx)
}

func (s *StatsService) collectorLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.collector.recordMetric("tick")
			_, _ = s.GetDashboard(ctx)
		case <-s.collector.eventCh:
			s.collector.recordMetric("event")
		}
	}
}

// GetDashboard 获取仪表盘数据
func (s *StatsService) GetDashboard(ctx context.Context) (*model.DashboardResponse, error) {
	dashboard := &model.DashboardResponse{}

	// 使用带缓冲的 channel，确保内部 goroutine 即使在主调用方提前返回
	// （例如 ctx 超时取消）后，仍能把结果写入后退出，不会因无接收方而
	// 永久阻塞在 channel send 上，从而避免 goroutine 泄漏。
	ch := make(chan *dashboardPartial, 2)

	go func() {
		partial := &dashboardPartial{}

		deviceStatus, err := s.deviceStore.CountDevicesByStatus(ctx)
		if err != nil {
			partial.err = fmt.Errorf("failed to count devices: %w", err)
			ch <- partial
			return
		}

		for status, count := range deviceStatus {
			switch status {
			case model.DeviceOnline:
				partial.onlineDevices += count
			case model.DeviceOffline:
				partial.offlineDevices += count
			}
			partial.totalDevices += count
		}

		_, total, _ := s.modelStore.ListModels(ctx, 1, 10000)
		partial.totalModels = int(total)

		allFirmwares, _ := s.firmwareStore.GetAllFirmwares(ctx)
		partial.totalFirmware = len(allFirmwares)

		activeTasks, _ := s.taskStore.ListActiveTasks(ctx)
		partial.activeTasks = len(activeTasks)

		pendingDevices, _ := s.countPendingUpgrades(ctx)
		partial.pendingUpgrades = pendingDevices

		todayCount, _ := s.recordStore.CountTodayRecords(ctx)
		partial.todayRecords = todayCount

		successRate, _ := s.calculateSuccessRate(ctx)
		partial.successRate = successRate

		partial.versionDist = s.calculateVersionDistribution(ctx)
		partial.modelDist = s.calculateModelDistribution(ctx)

		ch <- partial

		if s.collector.shouldSkip("secondary") {
			return
		}

		partial2 := &dashboardPartial{}
		partial2.totalDevices = partial.totalDevices
		partial2.onlineDevices = partial.onlineDevices
		partial2.offlineDevices = partial.offlineDevices
		partial2.totalModels = partial.totalModels
		partial2.totalFirmware = partial.totalFirmware
		partial2.activeTasks = partial.activeTasks
		partial2.pendingUpgrades = partial.pendingUpgrades
		partial2.todayRecords = partial.todayRecords
		partial2.successRate = partial.successRate
		partial2.versionDist = partial.versionDist
		partial2.modelDist = partial.modelDist

		s.collector.recordMetric("secondary_collect")
		time.Sleep(10 * time.Millisecond)

		// 带缓冲的 channel 保证此次发送不会阻塞：主调用方只读取第一个
		// 结果即返回，第二个结果落入缓冲区后被丢弃，goroutine 可正常退出。
		ch <- partial2
	}()

	select {
	case result := <-ch:
		if result.err != nil {
			return nil, result.err
		}
		dashboard.OnlineDevices = result.onlineDevices
		dashboard.OfflineDevices = result.offlineDevices
		dashboard.TotalDevices = result.totalDevices
		dashboard.TotalModels = result.totalModels
		dashboard.TotalFirmware = result.totalFirmware
		dashboard.ActiveTasks = result.activeTasks
		dashboard.PendingUpgrades = result.pendingUpgrades
		dashboard.TodayRecords = result.todayRecords
		dashboard.SuccessRate = result.successRate
		dashboard.VersionDistribution = result.versionDist
		dashboard.ModelDistribution = result.modelDist
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return dashboard, nil
}

// GetStatistics 获取详细统计
func (s *StatsService) GetStatistics(ctx context.Context) (*model.Statistics, error) {
	stats := model.NewStatistics()

	deviceStatus, err := s.deviceStore.CountDevicesByStatus(ctx)
	if err != nil {
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

	allFirmwares, err := s.firmwareStore.GetAllFirmwares(ctx)
	if err == nil {
		stats.TotalFirmware = len(allFirmwares)
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

	stats.VersionDistribution = s.calculateVersionDistribution(ctx)
	stats.ModelDistribution = s.calculateModelDistribution(ctx)

	return stats, nil
}

// GetVersionDistribution 获取各版本设备分布
func (s *StatsService) GetVersionDistribution(ctx context.Context) (map[string]int, error) {
	return s.calculateVersionDistribution(ctx), nil
}

// GetModelDistribution 获取各型号设备分布
func (s *StatsService) GetModelDistribution(ctx context.Context) (map[string]int, error) {
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
	dist := make(map[string]int)

	devices, _, err := s.deviceStore.GetAllDevices(ctx, 1, 10000)
	if err != nil {
		return dist
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
	dist := make(map[string]int)

	devices, _, err := s.deviceStore.GetAllDevices(ctx, 1, 10000)
	if err != nil {
		return dist
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
	records, err := s.recordStore.GetAllRecords(ctx)
	if err != nil {
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
	tasks, err := s.taskStore.ListActiveTasks(ctx)
	if err != nil {
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

// GoroutineCount 获取当前goroutine数量
func (s *StatsService) GoroutineCount() int {
	return runtime.NumGoroutine()
}