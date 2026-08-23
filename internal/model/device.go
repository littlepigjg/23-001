// Package model 定义系统的数据模型和业务实体
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
	DeviceOnline   DeviceStatus = "online"
	DeviceOffline  DeviceStatus = "offline"
	DeviceUpgrading DeviceStatus = "upgrading"
	DeviceError    DeviceStatus = "error"
)

// UpgradeStatus 升级状态
type UpgradeStatus string

const (
	UpgradePending    UpgradeStatus = "pending"
	UpgradeInProgress UpgradeStatus = "in_progress"
	UpgradeSuccess    UpgradeStatus = "success"
	UpgradeFailed     UpgradeStatus = "failed"
	UpgradeRollback   UpgradeStatus = "rollback"
	UpgradeCancelled  UpgradeStatus = "cancelled"
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
	TaskTypeFull     TaskType = "full"
	TaskTypeGrayscale TaskType = "grayscale"
	TaskTypeTargeted TaskType = "targeted"
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
