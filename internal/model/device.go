// Package model 定义系统的数据模型和业务实体
package model

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
