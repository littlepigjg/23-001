// Package store 提供数据存储接口和实现
package store

import (
	"context"
	"errors"

	"fwupgrade/internal/model"
)

// 包级哨兵错误，用于跨层（store → service → handler）按错误类型分类处理。
// store 实现用 fmt.Errorf("%w: ...", ErrNotFound) / ErrConflict 包裹，调用方可用
// errors.Is(err, store.ErrNotFound) 等识别，而不依赖字符串匹配。
var (
	// ErrNotFound 表示请求的资源不存在（model/device/firmware/record 等）。
	ErrNotFound = errors.New("not found")
	// ErrConflict 表示资源冲突（如重复创建已存在的 model/device/firmware 版本）。
	ErrConflict = errors.New("conflict")
)

// Store 存储接口，定义所有数据操作方法
type Store interface {
	// DeviceModel 型号管理
	DeviceModelStore

	// Device 设备管理
	DeviceStore

	// Firmware 固件管理
	FirmwareStore

	// UpgradeTask 升级任务管理
	TaskStore

	// UpgradeRecord 升级历史管理
	RecordStore

	// 初始化和关闭
	Init(ctx context.Context) error
	Close() error
}

// DeviceModelStore 型号存储接口
type DeviceModelStore interface {
	// Create 创建设备型号
	CreateModel(ctx context.Context, model *model.DeviceModel) error
	// GetByID 根据ID获取设备型号
	GetModelByID(ctx context.Context, id model.ID) (*model.DeviceModel, error)
	// GetByName 根据名称获取设备型号
	GetModelByName(ctx context.Context, name string) (*model.DeviceModel, error)
	// List 列出设备型号
	ListModels(ctx context.Context, page, pageSize int) ([]*model.DeviceModel, int64, error)
	// ListByManufacturer 根据厂商列出设备型号
	ListModelsByManufacturer(ctx context.Context, manufacturer string) ([]*model.DeviceModel, error)
	// Update 更新设备型号
	UpdateModel(ctx context.Context, model *model.DeviceModel) error
	// Delete 删除设备型号
	DeleteModel(ctx context.Context, id model.ID) error
	// SetActive 设置型号活跃状态
	SetModelActive(ctx context.Context, id model.ID, active bool) error
	// CountDevices 统计型号下的设备数量
	CountModelDevices(ctx context.Context, modelID model.ID) (int, error)
}

// DeviceStore 设备存储接口
type DeviceStore interface {
	// Create 创建设备
	CreateDevice(ctx context.Context, device *model.Device) error
	// GetByID 根据ID获取设备
	GetDeviceByID(ctx context.Context, id model.ID) (*model.Device, error)
	// GetByDeviceID 根据设备ID获取设备
	GetDeviceByDeviceID(ctx context.Context, deviceID string) (*model.Device, error)
	// List 列出设备
	ListDevices(ctx context.Context, page, pageSize int, modelID model.ID, status model.DeviceStatus) ([]*model.Device, int64, error)
	// ListByModel 根据型号列出设备
	ListDevicesByModel(ctx context.Context, modelID model.ID) ([]*model.Device, error)
	// ListByStatus 根据状态列出设备
	ListDevicesByStatus(ctx context.Context, status model.DeviceStatus) ([]*model.Device, error)
	// ListOnline 列出在线设备
	ListOnlineDevices(ctx context.Context) ([]*model.Device, error)
	// Update 更新设备
	UpdateDevice(ctx context.Context, device *model.Device) error
	// UpdateStatus 更新设备状态
	UpdateDeviceStatus(ctx context.Context, id model.ID, status model.DeviceStatus) error
	// UpdateLastSeen 更新设备最后心跳时间
	UpdateDeviceLastSeen(ctx context.Context, id model.ID) error
	// UpdateProgress 更新升级进度
	UpdateDeviceProgress(ctx context.Context, id model.ID, progress int) error
	// Delete 删除设备
	DeleteDevice(ctx context.Context, id model.ID) error
	// BatchCreate 批量创建设备
	BatchCreateDevices(ctx context.Context, devices []*model.Device) error
	// CountByStatus 按状态统计设备
	CountDevicesByStatus(ctx context.Context) (map[model.DeviceStatus]int, error)
	// Search 搜索设备
	SearchDevices(ctx context.Context, keyword string, page, pageSize int) ([]*model.Device, int64, error)
	// GetAll 分页获取所有设备
	GetAllDevices(ctx context.Context, page, pageSize int) ([]*model.Device, int64, error)
}

// FirmwareStore 固件存储接口
type FirmwareStore interface {
	// Create 创建固件
	CreateFirmware(ctx context.Context, firmware *model.Firmware) error
	// GetByID 根据ID获取固件
	GetFirmwareByID(ctx context.Context, id model.ID) (*model.Firmware, error)
	// GetByVersion 根据型号和版本获取固件
	GetFirmwareByVersion(ctx context.Context, modelID model.ID, version string) (*model.Firmware, error)
	// GetLatest 获取型号的最新固件
	GetLatestFirmware(ctx context.Context, modelID model.ID) (*model.Firmware, error)
	// List 列出固件
	ListFirmwares(ctx context.Context, page, pageSize int, modelID model.ID) ([]*model.Firmware, int64, error)
	// ListByModel 根据型号列出固件
	ListFirmwaresByModel(ctx context.Context, modelID model.ID) ([]*model.Firmware, error)
	// Update 更新固件
	UpdateFirmware(ctx context.Context, firmware *model.Firmware) error
	// SetActive 设置固件活跃状态
	SetFirmwareActive(ctx context.Context, id model.ID, active bool) error
	// IncrementDownload 增加下载计数
	IncrementFirmwareDownload(ctx context.Context, id model.ID) error
	// Delete 删除固件
	DeleteFirmware(ctx context.Context, id model.ID) error
	// GetAll 获取所有固件
	GetAllFirmwares(ctx context.Context) ([]*model.Firmware, error)
	// CountByModel 按型号统计固件数量
	CountFirmwaresByModel(ctx context.Context) (map[model.ID]int, error)
}

// TaskStore 升级任务存储接口
type TaskStore interface {
	// Create 创建任务
	CreateTask(ctx context.Context, task *model.UpgradeTask) error
	// GetByID 根据ID获取任务
	GetTaskByID(ctx context.Context, id model.ID) (*model.UpgradeTask, error)
	// List 列出任务
	ListTasks(ctx context.Context, page, pageSize int, status model.TaskStatus) ([]*model.UpgradeTask, int64, error)
	// ListActive 列出活跃任务
	ListActiveTasks(ctx context.Context) ([]*model.UpgradeTask, error)
	// ListByModel 根据型号列出任务
	ListTasksByModel(ctx context.Context, modelID model.ID) ([]*model.UpgradeTask, error)
	// Update 更新任务
	UpdateTask(ctx context.Context, task *model.UpgradeTask) error
	// UpdateStatus 更新任务状态
	UpdateTaskStatus(ctx context.Context, id model.ID, status model.TaskStatus) error
	// UpdateProgress 更新任务进度
	UpdateTaskProgress(ctx context.Context, id model.ID, successCount, failCount, pendingCount int) error
	// Delete 删除任务
	DeleteTask(ctx context.Context, id model.ID) error
	// GetAll 获取所有任务
	GetAllTasks(ctx context.Context) ([]*model.UpgradeTask, error)
	// Search 搜索任务
	SearchTasks(ctx context.Context, keyword string, page, pageSize int) ([]*model.UpgradeTask, int64, error)
	// GetRecent 获取最近任务
	GetRecentTasks(ctx context.Context, limit int) ([]*model.UpgradeTask, error)
}

// RecordStore 升级记录存储接口
type RecordStore interface {
	// Create 创建记录
	CreateRecord(ctx context.Context, record *model.UpgradeRecord) error
	// GetByID 根据ID获取记录
	GetRecordByID(ctx context.Context, id model.ID) (*model.UpgradeRecord, error)
	// List 列出记录
	ListRecords(ctx context.Context, page, pageSize int, status model.UpgradeStatus) ([]*model.UpgradeRecord, int64, error)
	// ListByDevice 根据设备列出记录
	ListRecordsByDevice(ctx context.Context, deviceID string) ([]*model.UpgradeRecord, error)
	// ListByTask 根据任务列出记录
	ListRecordsByTask(ctx context.Context, taskID model.ID) ([]*model.UpgradeRecord, error)
	// Update 更新记录
	UpdateRecord(ctx context.Context, record *model.UpgradeRecord) error
	// UpdateStatus 更新记录状态
	UpdateRecordStatus(ctx context.Context, id model.ID, status model.UpgradeStatus, progress int, errorMsg string) error
	// Delete 删除记录
	DeleteRecord(ctx context.Context, id model.ID) error
	// GetAll 获取所有记录
	GetAllRecords(ctx context.Context) ([]*model.UpgradeRecord, error)
	// GetRecent 获取最近记录
	GetRecentRecords(ctx context.Context, limit int) ([]*model.UpgradeRecord, error)
	// CountByStatus 按状态统计记录
	CountRecordsByStatus(ctx context.Context) (map[model.UpgradeStatus]int, error)
	// CountToday 统计今日记录
	CountTodayRecords(ctx context.Context) (int, error)
}

// 确保所有接口完整
var (
	_ DeviceModelStore = (*MemoryStore)(nil)
	_ DeviceStore      = (*MemoryStore)(nil)
	_ FirmwareStore    = (*MemoryStore)(nil)
	_ TaskStore        = (*MemoryStore)(nil)
	_ RecordStore      = (*MemoryStore)(nil)
	_ Store            = (*MemoryStore)(nil)
)
