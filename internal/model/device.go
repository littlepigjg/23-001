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
	// DeviceOnline 设备在线
	DeviceOnline DeviceStatus = "online"
	// DeviceOffline 设备离线
	DeviceOffline DeviceStatus = "offline"
	// DeviceUpgrading 设备升级中
	DeviceUpgrading DeviceStatus = "upgrading"
	// DeviceError 设备错误状态
	DeviceError DeviceStatus = "error"
)

// UpgradeStatus 升级状态
type UpgradeStatus string

const (
	// UpgradePending 等待升级
	UpgradePending UpgradeStatus = "pending"
	// UpgradeInProgress 升级进行中
	UpgradeInProgress UpgradeStatus = "in_progress"
	// UpgradeSuccess 升级成功
	UpgradeSuccess UpgradeStatus = "success"
	// UpgradeFailed 升级失败
	UpgradeFailed UpgradeStatus = "failed"
	// UpgradeRollback 回滚中
	UpgradeRollback UpgradeStatus = "rollback"
	// UpgradeCancelled 升级取消
	UpgradeCancelled UpgradeStatus = "cancelled"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	// TaskPending 待执行
	TaskPending TaskStatus = "pending"
	// TaskRunning 执行中
	TaskRunning TaskStatus = "running"
	// TaskCompleted 已完成
	TaskCompleted TaskStatus = "completed"
	// TaskFailed 已失败
	TaskFailed TaskStatus = "failed"
	// TaskCancelled 已取消
	TaskCancelled TaskStatus = "cancelled"
)

// TaskType 任务类型
type TaskType string

const (
	// TaskTypeFull 全量升级
	TaskTypeFull TaskType = "full"
	// TaskTypeGrayscale 灰度升级
	TaskTypeGrayscale TaskType = "grayscale"
	// TaskTypeTargeted 指定设备升级
	TaskTypeTargeted TaskType = "targeted"
)

// DeviceModel 设备型号
type DeviceModel struct {
	ID          ID        `json:"id"`
	Name        string    `json:"name"`
	Manufacturer string   `json:"manufacturer"`
	HardwareVer string    `json:"hardware_version"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	DeviceCount int       `json:"device_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewDeviceModel 创建设备型号
func NewDeviceModel(name, manufacturer, hardwareVer, description string) *DeviceModel {
	now := time.Now()
	return &DeviceModel{
		Name:        name,
		Manufacturer: manufacturer,
		HardwareVer: hardwareVer,
		Description: description,
		IsActive:    true,
		DeviceCount: 0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Validate 验证设备型号
func (m *DeviceModel) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("model name is required")
	}
	if len(m.Name) > 100 {
		return fmt.Errorf("model name too long: %d characters", len(m.Name))
	}
	if m.Manufacturer == "" {
		return fmt.Errorf("manufacturer is required")
	}
	if len(m.Manufacturer) > 50 {
		return fmt.Errorf("manufacturer too long: %d characters", len(m.Manufacturer))
	}
	return nil
}

// Device 设备
type Device struct {
	ID            ID           `json:"id"`
	DeviceID      string       `json:"device_id"`
	ModelID       ID           `json:"model_id"`
	ModelName     string       `json:"model_name,omitempty"`
	Name          string       `json:"name"`
	CurrentFWVer  string       `json:"current_fw_version"`
	TargetFWVer   string       `json:"target_fw_version"`
	Status        DeviceStatus `json:"status"`
	IPAddress     string       `json:"ip_address"`
	SerialNumber  string       `json:"serial_number"`
	LastSeenAt    time.Time    `json:"last_seen_at"`
	RegisteredAt  time.Time    `json:"registered_at"`
	UpgradeProgress int        `json:"upgrade_progress"`
	LastError     string       `json:"last_error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
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

// Firmware 固件版本
type Firmware struct {
	ID          ID        `json:"id"`
	ModelID     ID        `json:"model_id"`
	ModelName   string    `json:"model_name,omitempty"`
	Version     string    `json:"version"`
	Md5         string    `json:"md5"`
	Size        int64     `json:"size"`
	FilePath    string    `json:"file_path"`
	ReleaseDate time.Time `json:"release_date"`
	Changelog   string    `json:"changelog"`
	IsActive    bool      `json:"is_active"`
	DownloadCount int     `json:"download_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewFirmware 创建固件版本
func NewFirmware(modelID ID, modelName, version, md5 string, size int64, filePath string, releaseDate time.Time, changelog string) *Firmware {
	now := time.Now()
	return &Firmware{
		ModelID:      modelID,
		ModelName:    modelName,
		Version:      version,
		Md5:          md5,
		Size:         size,
		FilePath:     filePath,
		ReleaseDate:  releaseDate,
		Changelog:    changelog,
		IsActive:     true,
		DownloadCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Validate 验证固件
func (f *Firmware) Validate() error {
	if f.ModelID <= 0 {
		return fmt.Errorf("model_id is required")
	}
	if f.Version == "" {
		return fmt.Errorf("version is required")
	}
	if len(f.Version) > 50 {
		return fmt.Errorf("version too long: %d characters", len(f.Version))
	}
	if f.Md5 == "" {
		return fmt.Errorf("md5 is required")
	}
	if f.Size <= 0 {
		return fmt.Errorf("file size must be positive")
	}
	if f.FilePath == "" {
		return fmt.Errorf("file_path is required")
	}
	return nil
}

// IsReleased 检查固件是否已发布
func (f *Firmware) IsReleased() bool {
	return !f.ReleaseDate.IsZero() && f.ReleaseDate.Before(time.Now())
}

// UpgradeTask 升级任务
type UpgradeTask struct {
	ID            ID          `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	ModelID       ID          `json:"model_id"`
	ModelName     string      `json:"model_name,omitempty"`
	FirmwareID    ID          `json:"firmware_id"`
	FirmwareVer   string      `json:"firmware_version"`
	TaskType      TaskType    `json:"task_type"`
	Status        TaskStatus  `json:"status"`
	GrayscaleRatio float64    `json:"grayscale_ratio"`
	TargetDevices []string    `json:"target_devices,omitempty"`
	TotalDevices  int         `json:"total_devices"`
	SuccessCount  int         `json:"success_count"`
	FailCount     int         `json:"fail_count"`
	PendingCount  int         `json:"pending_count"`
	Progress      int         `json:"progress"`
	ScheduledAt   time.Time   `json:"scheduled_at"`
	StartedAt     *time.Time  `json:"started_at,omitempty"`
	CompletedAt   *time.Time  `json:"completed_at,omitempty"`
	CreatedBy     string      `json:"created_by"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
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

// UpgradeRecord 升级历史记录
type UpgradeRecord struct {
	ID            ID            `json:"id"`
	DeviceID      string        `json:"device_id"`
	DeviceName    string        `json:"device_name,omitempty"`
	TaskID        ID            `json:"task_id"`
	TaskName      string        `json:"task_name,omitempty"`
	FromVersion   string        `json:"from_version"`
	ToVersion     string        `json:"to_version"`
	Status        UpgradeStatus `json:"status"`
	Progress      int           `json:"progress"`
	ErrorMessage  string        `json:"error_message,omitempty"`
	StartedAt     time.Time     `json:"started_at"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
	Duration      int64         `json:"duration"` // 毫秒
	Retries       int           `json:"retries"`
}

// NewUpgradeRecord 创建升级记录
func NewUpgradeRecord(deviceID, deviceName string, taskID ID, taskName, fromVersion, toVersion string) *UpgradeRecord {
	return &UpgradeRecord{
		DeviceID:    deviceID,
		DeviceName:  deviceName,
		TaskID:      taskID,
		TaskName:    taskName,
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Status:      UpgradeInProgress,
		Progress:    0,
		StartedAt:   time.Now(),
		Retries:     0,
	}
}

// Complete 完成升级记录
func (r *UpgradeRecord) Complete(success bool, errorMsg string) {
	now := time.Now()
	r.CompletedAt = &now
	if success {
		r.Status = UpgradeSuccess
		r.Progress = 100
	} else {
		r.Status = UpgradeFailed
		r.ErrorMessage = errorMsg
	}
	r.Duration = now.Sub(r.StartedAt).Milliseconds()
}

// IsSuccess 检查升级是否成功
func (r *UpgradeRecord) IsSuccess() bool {
	return r.Status == UpgradeSuccess
}

// Statistics 统计数据
type Statistics struct {
	TotalDevices      int     `json:"total_devices"`
	OnlineDevices     int     `json:"online_devices"`
	OfflineDevices    int     `json:"offline_devices"`
	TotalModels       int     `json:"total_models"`
	TotalFirmware     int     `json:"total_firmware"`
	ActiveTasks       int     `json:"active_tasks"`
	TotalTasks        int     `json:"total_tasks"`
	SuccessRate       float64 `json:"success_rate"`
	FailureRate       float64 `json:"failure_rate"`
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

// DashboardResponse 仪表盘响应
type DashboardResponse struct {
	TotalDevices      int            `json:"total_devices"`
	OnlineDevices     int            `json:"online_devices"`
	OfflineDevices    int            `json:"offline_devices"`
	TotalModels       int            `json:"total_models"`
	TotalFirmware     int            `json:"total_firmware"`
	ActiveTasks       int            `json:"active_tasks"`
	PendingUpgrades   int            `json:"pending_upgrades"`
	TodayRecords      int            `json:"today_records"`
	SuccessRate       float64        `json:"success_rate"`
	VersionDistribution map[string]int `json:"version_distribution"`
	ModelDistribution   map[string]int `json:"model_distribution"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	Timestamp string `json:"timestamp"`
}

// ReadyResponse 就绪检查响应
type ReadyResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies"`
}
