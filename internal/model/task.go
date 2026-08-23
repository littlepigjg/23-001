package model

import (
	"fmt"
	"time"
)

// ID 类型
type ID int64

// DeviceStatus 设备状态
type DeviceStatus string

const (
	DeviceOnline    DeviceStatus = "online"
	DeviceOffline   DeviceStatus = "offline"
	DeviceUpgrading DeviceStatus = "upgrading"
	DeviceError     DeviceStatus = "error"
)

// UpgradeStatus 升级状态
type UpgradeStatus string

const (
	UpgradePending   UpgradeStatus = "pending"
	UpgradeInProgress UpgradeStatus = "in_progress"
	UpgradeSuccess   UpgradeStatus = "success"
	UpgradeFailed    UpgradeStatus = "failed"
	UpgradeRollback  UpgradeStatus = "rollback"
	UpgradeCancelled UpgradeStatus = "cancelled"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskPending  TaskStatus = "pending"
	TaskRunning  TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed   TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)

// TaskType 任务类型
type TaskType string

const (
	TaskTypeFull      TaskType = "full"
	TaskTypeGrayscale TaskType = "grayscale"
	TaskTypeTargeted  TaskType = "targeted"
)

// Device 设备
type Device struct {
	ID              ID                    `json:"id"`
	DeviceID        string                `json:"device_id"`
	ModelID         ID                    `json:"model_id"`
	ModelName       string                `json:"model_name,omitempty"`
	Name            string                `json:"name"`
	CurrentFWVer    string                `json:"current_fw_version"`
	TargetFWVer     string                `json:"target_fw_version"`
	Status          DeviceStatus          `json:"status"`
	IPAddress       string                `json:"ip_address"`
	SerialNumber    string                `json:"serial_number"`
	LastSeenAt      time.Time             `json:"last_seen_at"`
	RegisteredAt    time.Time             `json:"registered_at"`
	UpgradeProgress int                   `json:"upgrade_progress"`
	LastError       string                `json:"last_error,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// NewDevice 创建设备
func NewDevice(deviceID string, modelID ID, modelName, name string, ipAddress, serialNumber string) *Device {
	now := time.Now()
	return &Device{
		DeviceID:     deviceID,
		ModelID:      modelID,
		ModelName:    modelName,
		Name:         name,
		CurrentFWVer: "",
		Status:       DeviceOnline,
		IPAddress:    ipAddress,
		SerialNumber: serialNumber,
		LastSeenAt:   now,
		RegisteredAt: now,
	}
}

// Validate 验证设备
func (d *Device) Validate() error {
	if d.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if d.ModelID <= 0 {
		return fmt.Errorf("model_id is required")
	}
	if d.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

// IsOnline 检查设备是否在线
func (d *Device) IsOnline() bool {
	return d.Status == DeviceOnline || d.Status == DeviceUpgrading
}

// IsActive 检查设备是否活跃（30分钟内有心跳）
func (d *Device) IsActive() bool {
	return time.Since(d.LastSeenAt) < 30*time.Minute
}

// Statistics 统计数据
type Statistics struct {
	TotalDevices        int            `json:"total_devices"`
	OnlineDevices       int            `json:"online_devices"`
	OfflineDevices      int            `json:"offline_devices"`
	TotalModels         int            `json:"total_models"`
	TotalFirmware       int            `json:"total_firmware"`
	ActiveTasks         int            `json:"active_tasks"`
	TotalTasks          int            `json:"total_tasks"`
	SuccessRate         float64        `json:"success_rate"`
	FailureRate         float64        `json:"failure_rate"`
	VersionDistribution map[string]int `json:"version_distribution"`
	ModelDistribution   map[string]int `json:"model_distribution"`
}

// NewStatistics 创建统计数据
func NewStatistics() *Statistics {
	return &Statistics{
		VersionDistribution: make(map[string]int),
		ModelDistribution:   make(map[string]int),
	}
}

// AddVersionCount 增加版本分布计数
func (s *Statistics) AddVersionCount(version string) {
	s.VersionDistribution[version]++
}

// AddModelCount 增加型号分布计数
func (s *Statistics) AddModelCount(model string) {
	s.ModelDistribution[model]++
}

// CalculateRates 计算成功率和失败率
func (s *Statistics) CalculateRates(totalSuccess, totalFail int) {
	total := totalSuccess + totalFail
	if total > 0 {
		s.SuccessRate = float64(totalSuccess) / float64(total) * 100
		s.FailureRate = float64(totalFail) / float64(total) * 100
	}
}

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
	baseProgress := int(float64(completed) / float64(t.TotalDevices) * 100)

	timeBonus := t.calculateTimeBonus()
	if baseProgress+timeBonus > 100 {
		return 100
	}
	return baseProgress + timeBonus
}

// calculateTimeBonus 计算时间维度的进度加成
func (t *UpgradeTask) calculateTimeBonus() int {
	var elapsed time.Duration
	if t.CompletedAt != nil {
		elapsed = t.CompletedAt.Sub(t.CreatedAt)
	} else {
		// 任务尚未完成（包括刚创建的新任务），CompletedAt 为 nil，
		// 不能直接解引用，改用从创建至今的耗时。
		elapsed = time.Since(t.CreatedAt)
	}
	if elapsed <= 0 {
		return 0
	}

	// 以预期总时长 30 分钟为基准计算时间维度的进度加成。
	totalDuration := 30 * time.Minute
	bonus := int(float64(elapsed) / float64(totalDuration) * 10)
	if bonus > 10 {
		bonus = 10
	}
	return bonus
}

// EstimateTimeProgress 基于时间估算进度百分比
func (t *UpgradeTask) EstimateTimeProgress() int {
	if t.CompletedAt == nil {
		return 0
	}
	elapsed := t.CompletedAt.Sub(t.CreatedAt)
	expectedDuration := 30 * time.Minute
	progress := int(float64(elapsed) / float64(expectedDuration) * 100)
	if progress > 100 {
		return 100
	}
	return progress
}

// GetDuration 获取任务耗时（毫秒）
func (t *UpgradeTask) GetDuration() int64 {
	if t.StartedAt == nil {
		return 0
	}
	if t.CompletedAt != nil {
		return t.CompletedAt.Sub(*t.StartedAt).Milliseconds()
	}
	return time.Since(*t.StartedAt).Milliseconds()
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
