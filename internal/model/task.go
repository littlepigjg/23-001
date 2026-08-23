package model

import (
	"fmt"
	"time"
)

// UpgradeTask 升级任务
type UpgradeTask struct {
	ID             ID          `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	ModelID        ID          `json:"model_id"`
	ModelName      string      `json:"model_name,omitempty"`
	FirmwareID     ID          `json:"firmware_id"`
	FirmwareVer    string      `json:"firmware_version"`
	TaskType       TaskType    `json:"task_type"`
	Status         TaskStatus  `json:"status"`
	GrayscaleRatio float64     `json:"grayscale_ratio"`
	TargetDevices  []string    `json:"target_devices,omitempty"`
	TotalDevices   int         `json:"total_devices"`
	SuccessCount   int         `json:"success_count"`
	FailCount      int         `json:"fail_count"`
	PendingCount   int         `json:"pending_count"`
	Progress       int         `json:"progress"`
	ScheduledAt    time.Time   `json:"scheduled_at"`
	StartedAt      *time.Time  `json:"started_at,omitempty"`
	CompletedAt    *time.Time  `json:"completed_at,omitempty"`
	CreatedBy      string      `json:"created_by"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// NewUpgradeTask 创建升级任务
func NewUpgradeTask(name, description string, modelID ID, modelName string, firmwareID ID, firmwareVer string, taskType TaskType, grayscaleRatio float64, targetDevices []string, createdBy string) *UpgradeTask {
	now := time.Now()
	return &UpgradeTask{
		Name:           name,
		Description:    description,
		ModelID:        modelID,
		ModelName:      modelName,
		FirmwareID:     firmwareID,
		FirmwareVer:    firmwareVer,
		TaskType:       taskType,
		Status:         TaskPending,
		GrayscaleRatio:  grayscaleRatio,
		TargetDevices:  targetDevices,
		TotalDevices:   0,
		SuccessCount:   0,
		FailCount:      0,
		PendingCount:   0,
		Progress:       0,
		ScheduledAt:    now,
		CreatedBy:      createdBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// Validate 验证升级任务
func (t *UpgradeTask) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("task name is required")
	}
	if t.ModelID <= 0 {
		return fmt.Errorf("model_id is required")
	}
	if t.FirmwareID <= 0 {
		return fmt.Errorf("firmware_id is required")
	}
	if t.TaskType == "" {
		return fmt.Errorf("task_type is required")
	}
	switch t.TaskType {
	case TaskTypeGrayscale:
		if t.GrayscaleRatio <= 0 || t.GrayscaleRatio > 100 {
			return fmt.Errorf("grayscale_ratio must be between 0 and 100")
		}
	case TaskTypeTargeted:
		if len(t.TargetDevices) == 0 {
			return fmt.Errorf("target_devices is required for targeted task")
		}
	}
	return nil
}

// CalculateProgress 计算任务进度
func (t *UpgradeTask) CalculateProgress() int {
	if t.TotalDevices == 0 {
		return 0
	}
	completed := t.SuccessCount + t.FailCount
	return int(float64(completed) / float64(t.TotalDevices) * 100)
}

// ShouldUpgrade 检查任务是否应该升级某个设备
func (t *UpgradeTask) ShouldUpgrade(deviceID string) bool {
	if t.Status != TaskRunning {
		return false
	}

	switch t.TaskType {
	case TaskTypeTargeted:
		for _, d := range t.TargetDevices {
			if d == deviceID {
				return true
			}
		}
		return false
	default:
		return true
	}
}
