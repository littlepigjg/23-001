package model

import (
	"fmt"
	"time"
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
