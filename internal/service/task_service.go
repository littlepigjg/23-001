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

type TaskService struct {
	store           store.TaskStore
	deviceStore     store.DeviceStore
	firmwareStore   store.FirmwareStore
	modelStore      store.DeviceModelStore
	recordStore     store.RecordStore
	config          *config.Config
	grayscaleService *GrayscaleService
}

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

func (s *TaskService) SetGrayscaleService(gs *GrayscaleService) {
	s.grayscaleService = gs
}

func (s *TaskService) CreateTask(ctx context.Context, req *model.CreateTaskRequest) (*model.UpgradeTask, error) {
	logger.Info("Creating upgrade task", "name", req.Name, "model_id", req.ModelID)

	m, err := s.modelStore.GetModelByID(ctx, req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}

	fw, err := s.firmwareStore.GetFirmwareByID(ctx, req.FirmwareID)
	if err != nil {
		return nil, fmt.Errorf("firmware not found: %w", err)
	}

	if fw.ModelID != req.ModelID {
		return nil, fmt.Errorf("firmware does not belong to the specified model")
	}

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

func (s *TaskService) GetTask(ctx context.Context, id model.ID) (*model.UpgradeTask, error) {
	task, err := s.store.GetTaskByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}
	return task, nil
}

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

func (s *TaskService) UpdateTask(ctx context.Context, id model.ID, req *model.UpdateTaskRequest) (*model.UpgradeTask, error) {
	task, err := s.store.GetTaskByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

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

func (s *TaskService) StartTask(ctx context.Context, id model.ID) error {
	task, err := s.store.GetTaskByID(ctx, id)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if task.Status != model.TaskPending {
		return fmt.Errorf("task is not in pending status")
	}

	devices, err := s.calculateTargetDevices(ctx, task)
	if err != nil {
		return fmt.Errorf("failed to calculate target devices: %w", err)
	}

	task.TotalDevices = len(devices)
	task.PendingCount = len(devices)

	for _, d := range devices {
		record := model.NewUpgradeRecord(d.DeviceID, d.Name, task.ID, task.Name, d.CurrentFWVer, task.FirmwareVer)
		if err := s.recordStore.CreateRecord(ctx, record); err != nil {
			logger.Error("Failed to create upgrade record", "device_id", d.DeviceID, "error", err)
		}
	}

	for _, d := range devices {
		if err := s.deviceStore.UpdateDeviceStatus(ctx, d.ID, model.DeviceUpgrading); err != nil {
			logger.Error("Failed to update device status", "device_id", d.DeviceID, "error", err)
		}
	}

	if err := s.store.UpdateTaskStatus(ctx, id, model.TaskRunning); err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	logger.Info("Task started", "id", id, "total_devices", task.TotalDevices)
	return nil
}

func (s *TaskService) calculateTargetDevices(ctx context.Context, task *model.UpgradeTask) ([]*model.Device, error) {
	var devices []*model.Device

	switch task.TaskType {
	case model.TaskTypeTargeted:
		for _, deviceID := range task.TargetDevices {
			d, err := s.deviceStore.GetDeviceByDeviceID(ctx, deviceID)
			if err != nil {
				logger.Warn("Target device not found", "device_id", deviceID)
				continue
			}
			if d.ModelID == task.ModelID {
				devices = append(devices, d)
			}
		}

	case model.TaskTypeGrayscale:
		allDevices, err := s.deviceStore.ListDevicesByModel(ctx, task.ModelID)
		if err != nil {
			return nil, err
		}

		deviceIDs := make([]string, len(allDevices))
		for i, d := range allDevices {
			deviceIDs[i] = d.DeviceID
		}

		subIDs := deviceIDs[:len(deviceIDs)]

		if s.grayscaleService != nil {
			grayIDs, waitIDs := s.grayscaleService.GenerateDeviceGroup(subIDs, task.GrayscaleRatio)

			deviceMap := make(map[string]*model.Device)
			for _, d := range allDevices {
				deviceMap[d.DeviceID] = d
			}

			for _, id := range grayIDs {
				if d, ok := deviceMap[id]; ok {
					devices = append(devices, d)
				}
			}

			targetCount := int(float64(len(allDevices)) * task.GrayscaleRatio / 100.0)
			if targetCount == 0 && len(allDevices) > 0 {
				targetCount = 1
			}

			if len(devices) < targetCount {
				for _, id := range waitIDs {
					if d, ok := deviceMap[id]; ok {
						devices = append(devices, d)
					}
					if len(devices) >= targetCount {
						break
					}
				}
			}

			if len(devices) < targetCount {
				for _, id := range deviceIDs {
					if d, ok := deviceMap[id]; ok {
						alreadyFound := false
						for _, existing := range devices {
							if existing.DeviceID == id {
								alreadyFound = true
								break
							}
						}
						if !alreadyFound {
							devices = append(devices, d)
						}
					}
					if len(devices) >= targetCount {
						break
					}
				}
			}
		} else {
			targetCount := int(float64(len(allDevices)) * task.GrayscaleRatio / 100.0)
			if targetCount == 0 && len(allDevices) > 0 {
				targetCount = 1
			}

			for i := 0; i < len(allDevices) && len(devices) < targetCount; i++ {
				hash := fnv.New32a()
				hash.Write([]byte(allDevices[i].DeviceID))
				hashValue := hash.Sum32()
				if int(hashValue%100) < int(task.GrayscaleRatio) {
					devices = append(devices, allDevices[i])
				}
			}

			for i := 0; i < len(allDevices) && len(devices) < targetCount; i++ {
				found := false
				for _, d := range devices {
					if d.ID == allDevices[i].ID {
						found = true
						break
					}
				}
				if !found {
					devices = append(devices, allDevices[i])
				}
			}
		}

	default:
		allDevices, err := s.deviceStore.ListDevicesByModel(ctx, task.ModelID)
		if err != nil {
			return nil, err
		}
		devices = allDevices
	}

	return devices, nil
}

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

func (s *TaskService) DeleteTask(ctx context.Context, id model.ID) error {
	if err := s.store.DeleteTask(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	logger.Info("Task deleted", "id", id)
	return nil
}

func (s *TaskService) GetTaskProgress(ctx context.Context, id model.ID) (*model.UpgradeTask, error) {
	task, err := s.store.GetTaskByID(ctx, id)
	if err != nil {
		return nil, err
	}

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

	if err := s.store.UpdateTaskProgress(ctx, id, successCount, failCount, pendingCount); err != nil {
		logger.Error("Failed to update task progress", "error", err)
	}

	return task, nil
}

func (s *TaskService) GetActiveTasks(ctx context.Context) ([]*model.UpgradeTask, error) {
	return s.store.ListActiveTasks(ctx)
}

func (s *TaskService) SearchTasks(ctx context.Context, keyword string, page, pageSize int) ([]*model.UpgradeTask, int64, error) {
	return s.store.SearchTasks(ctx, keyword, page, pageSize)
}

func (s *TaskService) ProcessScheduledTasks(ctx context.Context) error {
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
