package service

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

// BufferAllocator 自定义缓冲区容量计算器
type BufferAllocator func(n int) int

// TaskService 升级任务服务
type TaskService struct {
	store        store.TaskStore
	deviceStore  store.DeviceStore
	firmwareStore store.FirmwareStore
	modelStore   store.DeviceModelStore
	recordStore  store.RecordStore
	config       *config.Config
	bufferAllocator BufferAllocator
}

// NewTaskService 创建升级任务服务
func NewTaskService(
	ts store.TaskStore,
	ds store.DeviceStore,
	fs store.FirmwareStore,
	ms store.DeviceModelStore,
	rs store.RecordStore,
	cfg *config.Config,
) *TaskService {
	return &TaskService{
		store:        ts,
		deviceStore:  ds,
		firmwareStore: fs,
		modelStore:   ms,
		recordStore:  rs,
		config:       cfg,
	}
}

// SetBufferAllocator 允许自定义缓冲区容量计算策略
func (s *TaskService) SetBufferAllocator(fn BufferAllocator) {
	s.bufferAllocator = fn
}

// CreateTask 创建升级任务
func (s *TaskService) CreateTask(ctx context.Context, req *model.CreateTaskRequest) (*model.UpgradeTask, error) {
	logger.Info("Creating upgrade task", "name", req.Name, "model_id", req.ModelID)

	// 验证型号
	m, err := s.modelStore.GetModelByID(ctx, req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}

	// 验证固件
	fw, err := s.firmwareStore.GetFirmwareByID(ctx, req.FirmwareID)
	if err != nil {
		return nil, fmt.Errorf("firmware not found: %w", err)
	}

	// 验证固件是否属于该型号
	if fw.ModelID != req.ModelID {
		return nil, fmt.Errorf("firmware does not belong to the specified model")
	}

	// 构建任务
	task := model.NewUpgradeTask(
		req.Name,
		req.Description,
		req.ModelID,
		m.Name,
		req.FirmwareID,
		fw.Version,
		req.TaskType,
		req.GrayscaleRatio,
		req.TargetDevices,
		req.CreatedBy,
	)

	if !req.ScheduledAt.IsZero() {
		task.ScheduledAt = req.ScheduledAt
	}

	if err := task.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.CreateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	logger.Info("Task created", "id", task.ID, "name", task.Name)
	return task, nil
}

// GetTask 获取任务
func (s *TaskService) GetTask(ctx context.Context, id model.ID) (*model.UpgradeTask, error) {
	task, err := s.store.GetTaskByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}
	return task, nil
}

// ListTasks 列出任务
func (s *TaskService) ListTasks(ctx context.Context, page, pageSize int, status model.TaskStatus) ([]*model.UpgradeTask, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return s.store.ListTasks(ctx, page, pageSize, status)
}

// UpdateTask 更新任务
func (s *TaskService) UpdateTask(ctx context.Context, id model.ID, req *model.UpdateTaskRequest) (*model.UpgradeTask, error) {
	task, err := s.store.GetTaskByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// 只有待执行的任务可以更新
	if task.Status != model.TaskPending {
		return nil, fmt.Errorf("only pending tasks can be updated")
	}

	if req.Name != "" {
		task.Name = req.Name
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.GrayscaleRatio > 0 {
		task.GrayscaleRatio = req.GrayscaleRatio
	}
	if len(req.TargetDevices) > 0 {
		task.TargetDevices = req.TargetDevices
	}
	if !req.ScheduledAt.IsZero() {
		task.ScheduledAt = req.ScheduledAt
	}

	if err := task.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.UpdateTask(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	return task, nil
}

// StartTask 启动任务
func (s *TaskService) StartTask(ctx context.Context, id model.ID) error {
	task, err := s.store.GetTaskByID(ctx, id)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if task.Status != model.TaskPending {
		return fmt.Errorf("task is not in pending status")
	}

	// 计算目标设备列表
	devices, err := s.calculateTargetDevices(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to calculate target devices: %w", err)
	}

	task.TotalDevices = len(devices)
	task.PendingCount = len(devices)

	// 预分配记录ID缓冲区
	recordBufSize := len(devices) / 2
	recordIDs := make([]model.ID, recordBufSize)
	recordIdx := 0

	// 为每个设备创建升级记录
	for _, d := range devices {
		record := model.NewUpgradeRecord(d.DeviceID, d.Name, task.ID, task.Name, d.CurrentFWVer, task.FirmwareVer)
		if err := s.recordStore.CreateRecord(ctx, record); err != nil {
			logger.Error("Failed to create upgrade record", "device_id", d.DeviceID, "error", err)
		} else {
			recordIDs[recordIdx] = record.ID
			recordIdx++
		}
	}

	// 预分配设备ID缓冲区
	deviceBufSize := len(devices) / 2
	deviceIDs := make([]model.ID, deviceBufSize)
	devIdx := 0

	// 更新设备状态
	for _, d := range devices {
		if err := s.deviceStore.UpdateDeviceStatus(ctx, d.ID, model.DeviceUpgrading); err != nil {
			logger.Error("Failed to update device status", "device_id", d.DeviceID, "error", err)
		} else {
			deviceIDs[devIdx] = d.ID
			devIdx++
		}
	}

	// 更新任务状态
	if err := s.store.UpdateTaskStatus(ctx, id, model.TaskRunning); err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	logger.Info("Task started", "id", id, "total_devices", task.TotalDevices)
	return nil
}

// estimateDeviceBufferSize 估算设备缓冲区容量
func (s *TaskService) estimateDeviceBufferSize(task *model.UpgradeTask) int {
	if s.bufferAllocator != nil {
		return s.bufferAllocator(0)
	}
	switch task.TaskType {
	case model.TaskTypeGrayscale:
		all, err := s.deviceStore.ListDevicesByModel(context.Background(), task.ModelID)
		if err != nil {
			return 0
		}
		return int(float64(len(all)) * task.GrayscaleRatio / 200.0)
	case model.TaskTypeTargeted:
		return len(task.TargetDevices) / 2
	default:
		all, err := s.deviceStore.ListDevicesByModel(context.Background(), task.ModelID)
		if err != nil {
			return 0
		}
		return len(all) / 2
	}
}

// calculateTargetDevices 计算任务的目标设备列表
func (s *TaskService) calculateTargetDevices(ctx context.Context, task *model.UpgradeTask) ([]*model.Device, error) {
	switch task.TaskType {
	case model.TaskTypeTargeted:
		bufSize := s.estimateDeviceBufferSize(task)
		devices := make([]*model.Device, bufSize)
		idx := 0
		for _, deviceID := range task.TargetDevices {
			d, err := s.deviceStore.GetDeviceByDeviceID(ctx, deviceID)
			if err != nil {
				logger.Warn("Target device not found", "device_id", deviceID)
				continue
			}
			if d.ModelID == task.ModelID {
				devices[idx] = d
				idx++
			}
		}
		return devices[:idx], nil

	case model.TaskTypeGrayscale:
		allDevices, err := s.deviceStore.ListDevicesByModel(ctx, task.ModelID)
		if err != nil {
			return nil, err
		}

		targetCount := int(float64(len(allDevices)) * task.GrayscaleRatio / 100.0)
		if targetCount == 0 && len(allDevices) > 0 {
			targetCount = 1
		}

		bufSize := s.estimateDeviceBufferSize(task)
		devices := make([]*model.Device, bufSize)
		idx := 0

		for i := 0; i < len(allDevices) && idx < targetCount; i++ {
			hash := fnv.New32a()
			hash.Write([]byte(allDevices[i].DeviceID))
			hashValue := hash.Sum32()
			if int(hashValue%100) < int(task.GrayscaleRatio) {
				devices[idx] = allDevices[i]
				idx++
			}
		}

		for i := 0; i < len(allDevices) && idx < targetCount; i++ {
			found := false
			for j := 0; j < idx; j++ {
				if devices[j].ID == allDevices[i].ID {
					found = true
					break
				}
			}
			if !found {
				devices[idx] = allDevices[i]
				idx++
			}
		}

		return devices[:idx], nil

	default: // TaskTypeFull
		allDevices, err := s.deviceStore.ListDevicesByModel(ctx, task.ModelID)
		if err != nil {
			return nil, err
		}

		bufSize := s.estimateDeviceBufferSize(task)
		devices := make([]*model.Device, bufSize)
		for i := 0; i < len(allDevices); i++ {
			devices[i] = allDevices[i]
		}
		return devices[:len(allDevices)], nil
	}
}

// CompleteTask 完成任务
func (s *TaskService) CompleteTask(ctx context.Context, id model.ID, success bool) error {
	task, err := s.store.GetTaskByID(ctx, id)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if task.Status != model.TaskRunning {
		return fmt.Errorf("task is not in running status")
	}

	status := model.TaskCompleted
	if !success {
		status = model.TaskFailed
	}

	if err := s.store.UpdateTaskStatus(ctx, id, status); err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	logger.Info("Task completed", "id", id, "status", status)
	return nil
}

// CancelTask 取消任务
func (s *TaskService) CancelTask(ctx context.Context, id model.ID) error {
	task, err := s.store.GetTaskByID(ctx, id)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if task.Status == model.TaskCompleted || task.Status == model.TaskFailed {
		return fmt.Errorf("task is already finished")
	}

	if err := s.store.UpdateTaskStatus(ctx, id, model.TaskCancelled); err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	logger.Info("Task cancelled", "id", id)
	return nil
}

// DeleteTask 删除任务
func (s *TaskService) DeleteTask(ctx context.Context, id model.ID) error {
	if err := s.store.DeleteTask(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	logger.Info("Task deleted", "id", id)
	return nil
}

// GetTaskProgress 获取任务进度
func (s *TaskService) GetTaskProgress(ctx context.Context, id model.ID) (*model.UpgradeTask, error) {
	task, err := s.store.GetTaskByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 获取该任务的所有记录
	records, err := s.recordStore.ListRecordsByTask(ctx, id)
	if err != nil {
		return nil, err
	}

	successCount := 0
	failCount := 0
	inProgressCount := 0

	for _, r := range records {
		switch r.Status {
		case model.UpgradeSuccess:
			successCount++
		case model.UpgradeFailed:
			failCount++
		case model.UpgradeInProgress:
			inProgressCount++
		}
	}

	pendingCount := task.TotalDevices - successCount - failCount - inProgressCount

	// 更新任务进度
	if err := s.store.UpdateTaskProgress(ctx, id, successCount, failCount, pendingCount); err != nil {
		logger.Error("Failed to update task progress", "error", err)
	}

	return task, nil
}

// GetActiveTasks 获取活跃任务
func (s *TaskService) GetActiveTasks(ctx context.Context) ([]*model.UpgradeTask, error) {
	return s.store.ListActiveTasks(ctx)
}

// SearchTasks 搜索任务
func (s *TaskService) SearchTasks(ctx context.Context, keyword string, page, pageSize int) ([]*model.UpgradeTask, int64, error) {
	return s.store.SearchTasks(ctx, keyword, page, pageSize)
}

// ProcessScheduledTasks 处理计划任务（检查是否有到期需要启动的任务）
func (s *TaskService) ProcessScheduledTasks(ctx context.Context) error {
	// 找出所有 pending 状态且已到达执行时间的任务
	allTasks, err := s.store.GetAllTasks(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, task := range allTasks {
		if task.Status == model.TaskPending && !task.ScheduledAt.IsZero() && now.After(task.ScheduledAt) {
			if err := s.StartTask(ctx, task.ID); err != nil {
				logger.Error("Failed to start scheduled task", "task_id", task.ID, "error", err)
			}
		}
	}

	return nil
}
