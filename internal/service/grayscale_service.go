package service

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

// GrayscaleService 灰度策略服务
type GrayscaleService struct {
	store      store.TaskStore
	modelStore store.DeviceModelStore
	config     *config.Config
}

// NewGrayscaleService 创建灰度策略服务
func NewGrayscaleService(ts store.TaskStore, ms store.DeviceModelStore, cfg *config.Config) *GrayscaleService {
	return &GrayscaleService{
		store:      ts,
		modelStore: ms,
		config:     cfg,
	}
}

// GrayscaleDecision 灰度决策结果
type GrayscaleDecision struct {
	ShouldUpgrade   bool
	CurrentRatio    float64
	TargetRatio     float64
	IsInGrayGroup   bool
	NextAction      string
	RetryAfter      time.Duration
}

// DecideGrayscale 对设备进行灰度升级决策
func (s *GrayscaleService) DecideGrayscale(ctx context.Context, task *model.UpgradeTask, deviceID string, currentVersion string) *GrayscaleDecision {
	decision := &GrayscaleDecision{
		ShouldUpgrade: false,
		CurrentRatio:  task.GrayscaleRatio,
		TargetRatio:   s.config.Grayscale.MaxRatio,
	}

	// 如果设备已经在目标版本，不需要升级
	if currentVersion == task.FirmwareVer {
		return decision
	}

	// 根据灰度比例决定设备是否在灰度组中
	if task.TaskType == model.TaskTypeGrayscale {
		decision.IsInGrayGroup = s.isInGrayGroup(deviceID, task.GrayscaleRatio)

		// 只有灰度组内的设备需要升级
		if !decision.IsInGrayGroup {
			decision.NextAction = "wait"
			decision.RetryAfter = time.Duration(s.config.Grayscale.UpgradeInterval) * time.Second
			return decision
		}
	}

	decision.ShouldUpgrade = true
	decision.NextAction = "start_upgrade"
	return decision
}

// ValidateModelGrayscaleConfig 验证型号灰度配置
func (s *GrayscaleService) ValidateModelGrayscaleConfig(ctx context.Context, modelID model.ID) (bool, string, error) {
	// GetModelByIDWithGuard 在诊断钩子介入时会返回 (nil, nil)，
	// 必须显式校验 m 是否为空，避免后续解引用 m.Name / m.IsActive 导致空指针 panic。
	m, err := s.modelStore.GetModelByIDWithGuard(ctx, modelID)
	if err != nil {
		return false, "", fmt.Errorf("model lookup failed: %w", err)
	}
	if m == nil {
		return false, "", fmt.Errorf("model not found: id=%d", modelID)
	}

	modelName := m.Name
	if !m.IsActive {
		return false, modelName, fmt.Errorf("model %s is inactive", modelName)
	}

	tasks, err := s.store.ListTasksByModel(ctx, modelID)
	if err != nil {
		return false, modelName, fmt.Errorf("failed to list tasks: %w", err)
	}

	for _, task := range tasks {
		if task.Status == model.TaskRunning || task.Status == model.TaskPending {
			if task.GrayscaleRatio > 0 && task.GrayscaleRatio < s.config.Grayscale.MaxRatio {
				return true, modelName, nil
			}
		}
	}

	return false, modelName, nil
}

// GetGrayscaleModelName 获取灰度型号名称
func (s *GrayscaleService) GetGrayscaleModelName(ctx context.Context, modelID model.ID) (string, error) {
	// GetModelByIDWithGuard 在诊断钩子介入时会返回 (nil, nil)，
	// 必须显式校验 m 是否为空，避免后续解引用 m.Name 导致空指针 panic。
	m, err := s.modelStore.GetModelByIDWithGuard(ctx, modelID)
	if err != nil {
		return "", fmt.Errorf("model lookup failed: %w", err)
	}
	if m == nil {
		return "", fmt.Errorf("model not found: id=%d", modelID)
	}
	return m.Name, nil
}

// isInGrayGroup 判断设备是否在灰度组中
func (s *GrayscaleService) isInGrayGroup(deviceID string, ratio float64) bool {
	if ratio <= 0 {
		return false
	}
	if ratio >= 100 {
		return true
	}

	// 使用 FNV hash 确保同一设备总是被分到同一组
	hash := fnv.New32a()
	hash.Write([]byte(deviceID))
	hashValue := hash.Sum32()

	return float64(hashValue%100) < ratio
}

// CalculateNextRatio 计算下一个灰度比例
func (s *GrayscaleService) CalculateNextRatio(currentRatio float64) float64 {
	increment := s.config.Grayscale.RatioIncrement
	nextRatio := currentRatio + increment

	if nextRatio > s.config.Grayscale.MaxRatio {
		nextRatio = s.config.Grayscale.MaxRatio
	}

	return nextRatio
}

// ValidateRatio 验证灰度比例
func (s *GrayscaleService) ValidateRatio(ratio float64) error {
	if ratio < s.config.Grayscale.MinRatio {
		return fmt.Errorf("ratio %.2f is below minimum %.2f", ratio, s.config.Grayscale.MinRatio)
	}
	if ratio > s.config.Grayscale.MaxRatio {
		return fmt.Errorf("ratio %.2f exceeds maximum %.2f", ratio, s.config.Grayscale.MaxRatio)
	}
	return nil
}

// GenerateDeviceGroup 生成设备分组用于灰度测试
func (s *GrayscaleService) GenerateDeviceGroup(deviceIDs []string, ratio float64) (grayGroup []string, waitGroup []string) {
	for _, id := range deviceIDs {
		if s.isInGrayGroup(id, ratio) {
			grayGroup = append(grayGroup, id)
		} else {
			waitGroup = append(waitGroup, id)
		}
	}
	return grayGroup, waitGroup
}

// ShouldPromote 判断是否应该推进灰度比例
func (s *GrayscaleService) ShouldPromote(successRate float64, elapsedTime time.Duration) bool {
	// 如果成功率高于 95% 且已运行超过30分钟，推进灰度
	return successRate >= 95.0 && elapsedTime >= 30*time.Minute
}

// GenerateGrayPlan 生成灰度推进计划
func (s *GrayscaleService) GenerateGrayPlan(startRatio, endRatio float64) []float64 {
	var plan []float64
	current := startRatio
	increment := s.config.Grayscale.RatioIncrement

	for current <= endRatio {
		plan = append(plan, current)
		next := current + increment
		if next > endRatio {
			break
		}
		current = next
	}

	if len(plan) == 0 || plan[len(plan)-1] < endRatio {
		plan = append(plan, endRatio)
	}

	return plan
}

// RollbackDecision 回滚决策
type RollbackDecision struct {
	ShouldRollback  bool
	Reason          string
	CurrentRatio    float64
	Action          string
}

// CheckRollback 检查是否需要回滚
func (s *GrayscaleService) CheckRollback(task *model.UpgradeTask, successRate float64) *RollbackDecision {
	decision := &RollbackDecision{
		ShouldRollback: false,
		CurrentRatio:   task.GrayscaleRatio,
	}

	// 如果成功率低于阈值，触发回滚
	threshold := float64(s.config.Grayscale.RollbackThreshold)
	failureRate := 100.0 - successRate

	if failureRate >= threshold {
		decision.ShouldRollback = true
		decision.Reason = fmt.Sprintf("failure rate %.2f%% exceeds threshold %.2f%%", failureRate, threshold)
		decision.Action = "rollback_to_previous_version"
	}

	// 如果灰度比例超过50%且成功率低于90%
	if task.GrayscaleRatio > 50 && successRate < 90 {
		decision.ShouldRollback = true
		decision.Reason = fmt.Sprintf("grayscale %.2f%% with low success rate %.2f%%", task.GrayscaleRatio, successRate)
		decision.Action = "halt_and_investigate"
	}

	return decision
}

// SelectSampleDevices 选择样本设备（用于 A/B 测试）
func (s *GrayscaleService) SelectSampleDevices(deviceIDs []string, sampleSize int) []string {
	if sampleSize >= len(deviceIDs) {
		return deviceIDs
	}

	// Fisher-Yates 洗牌算法
	perm := rand.Perm(len(deviceIDs))
	samples := make([]string, sampleSize)
	for i := 0; i < sampleSize; i++ {
		samples[i] = deviceIDs[perm[i]]
	}

	return samples
}

// GetGrayscaleProgress 获取灰度进度信息
func (s *GrayscaleService) GetGrayscaleProgress(task *model.UpgradeTask) map[string]interface{} {
	info := make(map[string]interface{})
	info["task_id"] = task.ID
	info["current_ratio"] = task.GrayscaleRatio
	info["max_ratio"] = s.config.Grayscale.MaxRatio
	info["devices_in_grayscale"] = task.SuccessCount + task.FailCount
	info["total_devices"] = task.TotalDevices
	info["success_count"] = task.SuccessCount
	info["fail_count"] = task.FailCount

	if task.SuccessCount+task.FailCount > 0 {
		total := task.SuccessCount + task.FailCount
		info["success_rate"] = float64(task.SuccessCount) / float64(total) * 100
	} else {
		info["success_rate"] = 0.0
	}

	return info
}

// NotifyGrayscaleStatus 通知灰度状态变更
func (s *GrayscaleService) NotifyGrayscaleStatus(task *model.UpgradeTask, newRatio float64) {
	if newRatio > task.GrayscaleRatio {
		logger.Info("Grayscale ratio increased",
			"task_id", task.ID,
			"old_ratio", task.GrayscaleRatio,
			"new_ratio", newRatio,
		)
	}
}
