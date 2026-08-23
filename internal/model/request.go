package model

import "time"

type CreateModelRequest struct {
	Name        string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	HardwareVer string `json:"hardware_version"`
	Description string `json:"description"`
}

type UpdateModelRequest struct {
	Name        string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	HardwareVer string `json:"hardware_version"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type CreateDeviceRequest struct {
	DeviceID     string `json:"device_id"`
	ModelID      ID     `json:"model_id"`
	Name         string `json:"name"`
	IPAddress    string `json:"ip_address"`
	SerialNumber string `json:"serial_number"`
}

type UpdateDeviceRequest struct {
	Name         string `json:"name"`
	IPAddress    string `json:"ip_address"`
	SerialNumber string `json:"serial_number"`
}

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

type UploadFirmwareRequest struct {
	ModelID      ID        `json:"model_id"`
	Version      string    `json:"version"`
	Md5          string    `json:"md5"`
	Changelog    string    `json:"changelog"`
	ReleaseDate  time.Time `json:"release_date"`
}

type UpdateFirmwareRequest struct {
	Changelog   string `json:"changelog"`
	IsActive    *bool  `json:"is_active"`
}

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

type UpdateTaskRequest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	GrayscaleRatio float64  `json:"grayscale_ratio"`
	TargetDevices  []string `json:"target_devices"`
	ScheduledAt    time.Time `json:"scheduled_at"`
}

type PollUpgradeRequest struct {
	DeviceID     string `json:"device_id"`
	ModelID      ID     `json:"model_id"`
	CurrentVer   string `json:"current_version"`
	DeviceStatus string `json:"device_status"`
}

type PollUpgradeResponse struct {
	ShouldUpgrade  bool   `json:"should_upgrade"`
	TaskID         ID     `json:"task_id,omitempty"`
	FirmwareID     ID     `json:"firmware_id,omitempty"`
	FirmwareVer    string `json:"firmware_version,omitempty"`
	FirmwareURL    string `json:"firmware_url,omitempty"`
	GrayscaleRatio float64 `json:"grayscale_ratio,omitempty"`
}

type ReportProgressRequest struct {
	DeviceID    string `json:"device_id"`
	TaskID      ID     `json:"task_id"`
	Progress    int    `json:"progress"`
	Status      string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type BatchCreateDeviceRequest struct {
	Devices []CreateDeviceRequest `json:"devices"`
}

type BatchResult struct {
	SuccessCount int         `json:"success_count"`
	FailCount    int         `json:"fail_count"`
	Results      []BatchItem `json:"results"`
}

type BatchItem struct {
	Index   int    `json:"index"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      ID     `json:"id,omitempty"`
}
