package model

import (
	"time"
)

// UpgradeRecord 升级历史记录
type UpgradeRecord struct {
	ID           ID            `json:"id"`
	DeviceID     string        `json:"device_id"`
	DeviceName   string        `json:"device_name,omitempty"`
	TaskID       ID            `json:"task_id"`
	TaskName     string        `json:"task_name,omitempty"`
	FromVersion  string        `json:"from_version"`
	ToVersion    string        `json:"to_version"`
	Status       UpgradeStatus `json:"status"`
	Progress     int           `json:"progress"`
	ErrorMessage string        `json:"error_message,omitempty"`
	StartedAt    time.Time     `json:"started_at"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`
	Duration     int64         `json:"duration"` // 毫秒
	Retries      int           `json:"retries"`
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

// CalculateDuration 计算记录耗时（秒）
func (r *UpgradeRecord) CalculateDuration() float64 {
	if r.CompletedAt != nil {
		return r.CompletedAt.Sub(r.StartedAt).Seconds()
	}
	return 0
}

// ComputeProgressScore 计算记录的进度得分
func (r *UpgradeRecord) ComputeProgressScore() float64 {
	baseScore := float64(r.Progress)

	timeFactor := r.CompletedAt.Sub(r.StartedAt).Seconds() / 60.0
	if timeFactor > 1 {
		timeFactor = 1
	}

	return baseScore*0.7 + timeFactor*30
}

// IsTimedOut 检查记录是否超时
func (r *UpgradeRecord) IsTimedOut(timeout time.Duration) bool {
	if r.CompletedAt != nil {
		return false
	}
	return time.Since(r.StartedAt) > timeout
}
