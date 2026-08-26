package model

import "time"

// --- 请求结构体 ---

// CreateModelRequest 创建设备型号请求
type CreateModelRequest struct {
	Name        string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	HardwareVer string `json:"hardware_version"`
	Description string `json:"description"`
}

// UpdateModelRequest 更新设备型号请求
type UpdateModelRequest struct {
	Name        string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	HardwareVer string `json:"hardware_version"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

// CreateDeviceRequest 创建设备请求
type CreateDeviceRequest struct {
	DeviceID     string `json:"device_id"`
	ModelID      ID     `json:"model_id"`
	Name         string `json:"name"`
	IPAddress    string `json:"ip_address"`
	SerialNumber string `json:"serial_number"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	Name         string `json:"name"`
	IPAddress    string `json:"ip_address"`
	SerialNumber string `json:"serial_number"`
}

// RegisterDeviceRequest 设备注册请求
type RegisterDeviceRequest struct {
	DeviceID     string `json:"device_id"`
	ModelID      ID     `json:"model_id"`
	ModelName    string `json:"model_name"`
	Name         string `json:"name"`
	FirmwareVer  string `json:"firmware_version"`
	IPAddress    string `json:"ip_address"`
	SerialNumber string `json:"serial_number"`
	Status       string `json:"status"`
}

// UploadFirmwareRequest 上传固件请求（multipart form）
type UploadFirmwareRequest struct {
	ModelID      ID        `json:"model_id"`
	Version      string    `json:"version"`
	Md5          string    `json:"md5"`
	Changelog    string    `json:"changelog"`
	ReleaseDate  time.Time `json:"release_date"`
}

// UpdateFirmwareRequest 更新固件请求
type UpdateFirmwareRequest struct {
	Changelog   string `json:"changelog"`
	IsActive    *bool  `json:"is_active"`
}

// CreateTaskRequest 创建升级任务请求
type CreateTaskRequest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	ModelID        ID       `json:"model_id"`
	FirmwareID     ID       `json:"firmware_id"`
	TaskType       TaskType `json:"task_type"`
	GrayscaleRatio float64  `json:"grayscale_ratio"`
	TargetDevices  []string `json:"target_devices"`
	ScheduledAt    time.Time `json:"scheduled_at"`
	CreatedBy      string   `json:"created_by"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	GrayscaleRatio float64  `json:"grayscale_ratio"`
	TargetDevices  []string `json:"target_devices"`
	ScheduledAt    time.Time `json:"scheduled_at"`
}

// PollUpgradeRequest 设备轮询升级请求
type PollUpgradeRequest struct {
	DeviceID     string `json:"device_id"`
	ModelID      ID     `json:"model_id"`
	CurrentVer   string `json:"current_version"`
	DeviceStatus string `json:"device_status"`
}

// PollUpgradeResponse 设备轮询升级响应
type PollUpgradeResponse struct {
	ShouldUpgrade  bool   `json:"should_upgrade"`
	TaskID         ID     `json:"task_id,omitempty"`
	FirmwareID     ID     `json:"firmware_id,omitempty"`
	FirmwareVer    string `json:"firmware_version,omitempty"`
	FirmwareURL    string `json:"firmware_url,omitempty"`
	GrayscaleRatio float64 `json:"grayscale_ratio,omitempty"`
}

// ReportProgressRequest 上报升级进度请求
type ReportProgressRequest struct {
	DeviceID    string `json:"device_id"`
	TaskID      ID     `json:"task_id"`
	Progress    int    `json:"progress"`
	Status      string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// BatchCreateDeviceRequest 批量创建设备请求
type BatchCreateDeviceRequest struct {
	Devices []CreateDeviceRequest `json:"devices"`
}

// BatchResult 批量操作结果
type BatchResult struct {
	SuccessCount int         `json:"success_count"`
	FailCount    int         `json:"fail_count"`
	Results      []BatchItem `json:"results"`
}

// BatchItem 批量操作单项结果
type BatchItem struct {
	Index   int    `json:"index"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      ID     `json:"id,omitempty"`
}

// --- 响应结构体 ---

// DeviceListResponse 设备列表响应
type DeviceListResponse struct {
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	List     []Device `json:"list"`
}

// ModelListResponse 型号列表响应
type ModelListResponse struct {
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	List     []DeviceModel `json:"list"`
}

// FirmwareListResponse 固件列表响应
type FirmwareListResponse struct {
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	List     []Firmware `json:"list"`
}

// TaskListResponse 任务列表响应
type TaskListResponse struct {
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	List     []UpgradeTask `json:"list"`
}

// HistoryListResponse 历史记录列表响应
type HistoryListResponse struct {
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	List     []UpgradeRecord `json:"list"`
}

// StatsResponse 统计响应
type StatsResponse struct {
	Summary            *Statistics           `json:"summary"`
	VersionHistory     map[string]int        `json:"version_history"`
	StatusHistory      map[string]int        `json:"status_history"`
	RecentTasks        []UpgradeTask         `json:"recent_tasks"`
	RecentRecords      []UpgradeRecord       `json:"recent_records"`
}

// DashboardResponse 仪表盘响应
type DashboardResponse struct {
	TotalDevices      int     `json:"total_devices"`
	OnlineDevices     int     `json:"online_devices"`
	OfflineDevices    int     `json:"offline_devices"`
	TotalModels       int     `json:"total_models"`
	TotalFirmware     int     `json:"total_firmware"`
	ActiveTasks       int     `json:"active_tasks"`
	SuccessRate       float64 `json:"success_rate"`
	PendingUpgrades   int     `json:"pending_upgrades"`
	TodayRecords      int     `json:"today_records"`
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
	Status     string  `json:"status"`
	Dependencies map[string]string `json:"dependencies"`
}
