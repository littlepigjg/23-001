package model

type DeviceListResponse struct {
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
	List     []Device `json:"list"`
}

type ModelListResponse struct {
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	List     []DeviceModel `json:"list"`
}

type FirmwareListResponse struct {
	Total    int64     `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	List     []Firmware `json:"list"`
}

type TaskListResponse struct {
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	List     []UpgradeTask `json:"list"`
}

type HistoryListResponse struct {
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	List     []UpgradeRecord `json:"list"`
}

type StatsResponse struct {
	Summary        *Statistics     `json:"summary"`
	VersionHistory map[string]int  `json:"version_history"`
	StatusHistory  map[string]int  `json:"status_history"`
	RecentTasks    []UpgradeTask   `json:"recent_tasks"`
	RecentRecords  []UpgradeRecord `json:"recent_records"`
}

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

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	Timestamp string `json:"timestamp"`
}

type ReadyResponse struct {
	Status       string            `json:"status"`
	Dependencies map[string]string `json:"dependencies"`
}
