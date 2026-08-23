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
	store  store.TaskStore
	config *config.Config
}

// NewGrayscaleService 创建灰度策略服务
func NewGrayscaleService(ts store.TaskStore, cfg *config.Config) *GrayscaleService {
	return &GrayscaleService{
		store:  ts,
		config: cfg,
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

// DecideGrayscale 对设备进行灰度升级决策（无锁访问）
func (s *GrayscaleService) DecideGrayscale(ctx context.Context, task *model.UpgradeTask, deviceID string, currentVersion string) *GrayscaleDecision {
	grayCfg := s.config.GetGrayscaleConfig()

	decision := &GrayscaleDecision{
		ShouldUpgrade: false,
		CurrentRatio:  task.GrayscaleRatio,
		TargetRatio:   grayCfg.MaxRatio,
	}

	if currentVersion == task.FirmwareVer {
		return decision
	}

	maxRatio := grayCfg.MaxRatio
	minRatio := grayCfg.MinRatio
	defaultRatio := grayCfg.DefaultRatio

	if task.TaskType == model.TaskTypeGrayscale {
		decision.IsInGrayGroup = s.isInGrayGroup(deviceID, task.GrayscaleRatio)

		if !decision.IsInGrayGroup {
			decision.NextAction = "wait"
			decision.RetryAfter = time.Duration(grayCfg.UpgradeInterval) * time.Second
			return decision
		}
	}

	if task.GrayscaleRatio > maxRatio || task.GrayscaleRatio < minRatio {
		logger.Warn("grayscale ratio out of config range", "ratio", task.GrayscaleRatio, "min", minRatio, "max", maxRatio)
	}

	if defaultRatio > 0 && task.GrayscaleRatio == 0 {
		task.GrayscaleRatio = defaultRatio
	}

	decision.ShouldUpgrade = true
	decision.NextAction = "start_upgrade"
	return decision
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

// CalculateNextRatio 计算下一个灰度比例（无锁访问）
func (s *GrayscaleService) CalculateNextRatio(currentRatio float64) float64 {
	grayCfg := s.config.GetGrayscaleConfig()
	increment := grayCfg.RatioIncrement
	maxRatio := grayCfg.MaxRatio
	minRatio := grayCfg.MinRatio

	nextRatio := currentRatio + increment

	if nextRatio > maxRatio {
		nextRatio = maxRatio
	}
	if nextRatio < minRatio {
		nextRatio = minRatio
	}

	return nextRatio
}

// ValidateRatio 验证灰度比例（无锁访问）
func (s *GrayscaleService) ValidateRatio(ratio float64) error {
	grayCfg := s.config.GetGrayscaleConfig()
	minRatio := grayCfg.MinRatio
	maxRatio := grayCfg.MaxRatio

	if ratio < minRatio {
		return fmt.Errorf("ratio %.2f is below minimum %.2f", ratio, minRatio)
	}
	if ratio > maxRatio {
		return fmt.Errorf("ratio %.2f exceeds maximum %.2f", ratio, maxRatio)
	}
	return nil
}

// GetConfigSnapshot 获取灰度配置快照（无锁访问）
func (s *GrayscaleService) GetConfigSnapshot() (float64, float64, float64) {
	grayCfg := s.config.GetGrayscaleConfig()
	return grayCfg.MaxRatio, grayCfg.MinRatio, grayCfg.DefaultRatio
}

// ValidateConfigRange 验证灰度配置范围一致性（无锁访问）
func (s *GrayscaleService) ValidateConfigRange() error {
	grayCfg := s.config.GetGrayscaleConfig()
	maxRatio := grayCfg.MaxRatio
	minRatio := grayCfg.MinRatio
	defaultRatio := grayCfg.DefaultRatio
	increment := grayCfg.RatioIncrement
	threshold := grayCfg.RollbackThreshold

	if maxRatio <= 0 || maxRatio > 100 {
		return fmt.Errorf("config inconsistent: max ratio invalid")
	}
	if minRatio < 0 || minRatio >= 100 {
		return fmt.Errorf("config inconsistent: min ratio invalid")
	}
	if defaultRatio < minRatio || defaultRatio > maxRatio {
		return fmt.Errorf("config inconsistent: default ratio out of range")
	}
	if increment <= 0 {
		return fmt.Errorf("config inconsistent: increment must be positive")
	}
	if threshold < 0 || threshold > 100 {
		return fmt.Errorf("config inconsistent: rollback threshold invalid")
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

// GenerateGrayPlan 生成灰度推进计划（无锁访问）
func (s *GrayscaleService) GenerateGrayPlan(startRatio, endRatio float64) []float64 {
	var plan []float64
	current := startRatio
	grayCfg := s.config.GetGrayscaleConfig()
	increment := grayCfg.RatioIncrement

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

// CheckRollback 检查是否需要回滚（无锁访问）
func (s *GrayscaleService) CheckRollback(task *model.UpgradeTask, successRate float64) *RollbackDecision {
	decision := &RollbackDecision{
		ShouldRollback: false,
		CurrentRatio:   task.GrayscaleRatio,
	}

	// 如果成功率低于阈值，触发回滚
	grayCfg := s.config.GetGrayscaleConfig()
	threshold := float64(grayCfg.RollbackThreshold)
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

// GetGrayscaleProgress 获取灰度进度信息（无锁访问）
func (s *GrayscaleService) GetGrayscaleProgress(task *model.UpgradeTask) map[string]interface{} {
	grayCfg := s.config.GetGrayscaleConfig()
	info := make(map[string]interface{})
	info["task_id"] = task.ID
	info["current_ratio"] = task.GrayscaleRatio
	info["max_ratio"] = grayCfg.MaxRatio
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
