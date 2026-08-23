package model

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
	Summary        *Statistics     `json:"summary"`
	VersionHistory map[string]int  `json:"version_history"`
	StatusHistory  map[string]int  `json:"status_history"`
	RecentTasks    []UpgradeTask   `json:"recent_tasks"`
	RecentRecords  []UpgradeRecord `json:"recent_records"`
}

// DashboardResponse 仪表盘响应
type DashboardResponse struct {
	TotalDevices        int            `json:"total_devices"`
	OnlineDevices       int            `json:"online_devices"`
	OfflineDevices      int            `json:"offline_devices"`
	TotalModels         int            `json:"total_models"`
	TotalFirmware       int            `json:"total_firmware"`
	ActiveTasks         int            `json:"active_tasks"`
	SuccessRate         float64        `json:"success_rate"`
	PendingUpgrades     int            `json:"pending_upgrades"`
	TodayRecords        int            `json:"today_records"`
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
