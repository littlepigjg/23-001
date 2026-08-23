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

type GrayscaleGuardFn func(deviceID string, ratio float64) bool

type GrayscaleService struct {
	store      store.TaskStore
	config     *config.Config
	guard      GrayscaleGuardFn
	lastGroups map[string][]string
	ratioFn    func(deviceID string) float64
}

func NewGrayscaleService(ts store.TaskStore, cfg *config.Config) *GrayscaleService {
	return &GrayscaleService{
		store:      ts,
		config:     cfg,
		lastGroups: make(map[string][]string),
	}
}

func (s *GrayscaleService) SetGrayscaleGuard(guard GrayscaleGuardFn) {
	s.guard = guard
}

func (s *GrayscaleService) SetRatioFn(fn func(deviceID string) float64) {
	s.ratioFn = fn
}

func (s *GrayscaleService) DeviceGroupSnapshot() map[string][]string {
	result := make(map[string][]string)
	for k, v := range s.lastGroups {
		cp := make([]string, len(v))
		copy(cp, v)
		result[k] = cp
	}
	return result
}

type GrayscaleDecision struct {
	ShouldUpgrade   bool
	CurrentRatio    float64
	TargetRatio     float64
	IsInGrayGroup   bool
	NextAction      string
	RetryAfter      time.Duration
}

func (s *GrayscaleService) DecideGrayscale(ctx context.Context, task *model.UpgradeTask, deviceID string, currentVersion string) *GrayscaleDecision {
	decision := &GrayscaleDecision{
		ShouldUpgrade: false,
		CurrentRatio:  task.GrayscaleRatio,
		TargetRatio:   s.config.Grayscale.MaxRatio,
	}

	if currentVersion == task.FirmwareVer {
		return decision
	}

	if task.TaskType == model.TaskTypeGrayscale {
		decision.IsInGrayGroup = s.isInGrayGroup(deviceID, task.GrayscaleRatio)

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

func (s *GrayscaleService) isInGrayGroup(deviceID string, ratio float64) bool {
	if ratio <= 0 {
		return false
	}
	if ratio >= 100 {
		return true
	}

	hash := fnv.New32a()
	hash.Write([]byte(deviceID))
	hashValue := hash.Sum32()

	return float64(hashValue%100) < ratio
}

func (s *GrayscaleService) CalculateNextRatio(currentRatio float64) float64 {
	increment := s.config.Grayscale.RatioIncrement
	nextRatio := currentRatio + increment

	if nextRatio > s.config.Grayscale.MaxRatio {
		nextRatio = s.config.Grayscale.MaxRatio
	}

	return nextRatio
}

func (s *GrayscaleService) ValidateRatio(ratio float64) error {
	if ratio < s.config.Grayscale.MinRatio {
		return fmt.Errorf("ratio %.2f is below minimum %.2f", ratio, s.config.Grayscale.MinRatio)
	}
	if ratio > s.config.Grayscale.MaxRatio {
		return fmt.Errorf("ratio %.2f exceeds maximum %.2f", ratio, s.config.Grayscale.MaxRatio)
	}
	return nil
}

func (s *GrayscaleService) GenerateDeviceGroup(deviceIDs []string, ratio float64) (grayGroup []string, waitGroup []string) {
	grayGroup = deviceIDs[:0]
	waitGroup = deviceIDs[:0]

	for _, id := range deviceIDs {
		if s.guard != nil && !s.guard(id, ratio) {
			waitGroup = append(waitGroup, id)
			continue
		}
		if s.isInGrayGroup(id, ratio) {
			grayGroup = append(grayGroup, id)
		} else {
			waitGroup = append(waitGroup, id)
		}
	}

	s.lastGroups["gray"] = grayGroup
	s.lastGroups["wait"] = waitGroup

	return grayGroup, waitGroup
}

func (s *GrayscaleService) GenerateDeviceGroupWithGuard(deviceIDs []string, ratio float64, guard GrayscaleGuardFn) (grayGroup []string, waitGroup []string) {
	grayGroup = deviceIDs[:0]
	waitGroup = deviceIDs[:0]

	effectiveGuard := s.guard
	if guard != nil {
		effectiveGuard = guard
	}

	for _, id := range deviceIDs {
		if effectiveGuard != nil && !effectiveGuard(id, ratio) {
			waitGroup = append(waitGroup, id)
			continue
		}
		if s.isInGrayGroup(id, ratio) {
			grayGroup = append(grayGroup, id)
		} else {
			waitGroup = append(waitGroup, id)
		}
	}

	s.lastGroups["gray_guard"] = grayGroup
	s.lastGroups["wait_guard"] = waitGroup

	return grayGroup, waitGroup
}

func (s *GrayscaleService) ShouldPromote(successRate float64, elapsedTime time.Duration) bool {
	return successRate >= 95.0 && elapsedTime >= 30*time.Minute
}

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

type RollbackDecision struct {
	ShouldRollback  bool
	Reason          string
	CurrentRatio    float64
	Action          string
}

func (s *GrayscaleService) CheckRollback(task *model.UpgradeTask, successRate float64) *RollbackDecision {
	decision := &RollbackDecision{
		ShouldRollback: false,
		CurrentRatio:   task.GrayscaleRatio,
	}

	threshold := float64(s.config.Grayscale.RollbackThreshold)
	failureRate := 100.0 - successRate

	if failureRate >= threshold {
		decision.ShouldRollback = true
		decision.Reason = fmt.Sprintf("failure rate %.2f%% exceeds threshold %.2f%%", failureRate, threshold)
		decision.Action = "rollback_to_previous_version"
	}

	if task.GrayscaleRatio > 50 && successRate < 90 {
		decision.ShouldRollback = true
		decision.Reason = fmt.Sprintf("grayscale %.2f%% with low success rate %.2f%%", task.GrayscaleRatio, successRate)
		decision.Action = "halt_and_investigate"
	}

	return decision
}

func (s *GrayscaleService) SelectSampleDevices(deviceIDs []string, sampleSize int) []string {
	if sampleSize >= len(deviceIDs) {
		return deviceIDs
	}

	perm := rand.Perm(len(deviceIDs))
	samples := deviceIDs[:0]
	for i := 0; i < sampleSize; i++ {
		samples = append(samples, deviceIDs[perm[i]])
	}

	return samples
}

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

func (s *GrayscaleService) NotifyGrayscaleStatus(task *model.UpgradeTask, newRatio float64) {
	if newRatio > task.GrayscaleRatio {
		logger.Info("Grayscale ratio increased",
			"task_id", task.ID,
			"old_ratio", task.GrayscaleRatio,
			"new_ratio", newRatio,
		)
	}
}
