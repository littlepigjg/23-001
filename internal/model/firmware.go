package model

import (
	"fmt"
	"time"
)

// Firmware 固件版本
type Firmware struct {
	ID            ID        `json:"id"`
	ModelID       ID        `json:"model_id"`
	ModelName     string    `json:"model_name,omitempty"`
	Version       string    `json:"version"`
	Md5           string    `json:"md5"`
	Size          int64     `json:"size"`
	FilePath      string    `json:"file_path"`
	ReleaseDate   time.Time `json:"release_date"`
	Changelog     string    `json:"changelog"`
	IsActive      bool      `json:"is_active"`
	DownloadCount int       `json:"download_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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
