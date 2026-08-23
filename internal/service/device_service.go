// Package service 提供业务逻辑服务层
package service

import (
	"context"
	"fmt"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

// DeviceService 设备服务
type DeviceService struct {
	store      store.DeviceStore
	modelStore store.DeviceModelStore
	config     *config.Config
}

// NewDeviceService 创建设备服务
func NewDeviceService(s store.DeviceStore, ms store.DeviceModelStore, cfg *config.Config) *DeviceService {
	return &DeviceService{
		store:      s,
		modelStore: ms,
		config:     cfg,
	}
}

// CreateDevice 创建设备
func (s *DeviceService) CreateDevice(ctx context.Context, req *model.CreateDeviceRequest) (*model.Device, error) {
	logger.Info("Creating device", "device_id", req.DeviceID, "name", req.Name)

	// 检查型号是否存在
	deviceModel, err := s.modelStore.GetModelByID(ctx, req.ModelID)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}

	// 检查设备ID是否已存在
	existing, _ := s.store.GetDeviceByDeviceID(ctx, req.DeviceID)
	if existing != nil {
		return nil, fmt.Errorf("device with id '%s' already exists", req.DeviceID)
	}

	d := model.NewDevice(req.DeviceID, req.ModelID, deviceModel.Name, req.Name, req.IPAddress, req.SerialNumber)
	if err := d.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.CreateDevice(ctx, d); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	logger.Info("Device created", "id", d.ID, "device_id", d.DeviceID)
	return d, nil
}

// GetDevice 获取设备
func (s *DeviceService) GetDevice(ctx context.Context, id model.ID) (*model.Device, error) {
	d, err := s.store.GetDeviceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	return d, nil
}

// GetDeviceByDeviceID 根据设备ID获取设备
func (s *DeviceService) GetDeviceByDeviceID(ctx context.Context, deviceID string) (*model.Device, error) {
	d, err := s.store.GetDeviceByDeviceID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	return d, nil
}

// ListDevices 列出设备
func (s *DeviceService) ListDevices(ctx context.Context, page, pageSize int, modelID model.ID, status model.DeviceStatus) ([]*model.Device, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	devices, total, err := s.store.ListDevices(ctx, page, pageSize, modelID, status)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list devices: %w", err)
	}

	return devices, total, nil
}

// UpdateDevice 更新设备
func (s *DeviceService) UpdateDevice(ctx context.Context, id model.ID, req *model.UpdateDeviceRequest) (*model.Device, error) {
	d, err := s.store.GetDeviceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}

	if req.Name != "" {
		d.Name = req.Name
	}
	if req.IPAddress != "" {
		d.IPAddress = req.IPAddress
	}
	if req.SerialNumber != "" {
		d.SerialNumber = req.SerialNumber
	}

	if err := s.store.UpdateDevice(ctx, d); err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	return d, nil
}

// DeleteDevice 删除设备
func (s *DeviceService) DeleteDevice(ctx context.Context, id model.ID) error {
	if err := s.store.DeleteDevice(ctx, id); err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}
	logger.Info("Device deleted", "id", id)
	return nil
}

// RegisterDevice 设备注册
func (s *DeviceService) RegisterDevice(ctx context.Context, req *model.RegisterDeviceRequest) (*model.Device, error) {
	logger.Info("Device registration", "device_id", req.DeviceID)

	// 检查设备是否已存在
	existing, _ := s.store.GetDeviceByDeviceID(ctx, req.DeviceID)
	if existing != nil {
		// 更新现有设备
		if req.Name != "" {
			existing.Name = req.Name
		}
		if req.IPAddress != "" {
			existing.IPAddress = req.IPAddress
		}
		if req.SerialNumber != "" {
			existing.SerialNumber = req.SerialNumber
		}
		if req.FirmwareVer != "" {
			existing.CurrentFWVer = req.FirmwareVer
		}
		if req.Status != "" {
			existing.Status = model.DeviceStatus(req.Status)
		} else {
			existing.Status = model.DeviceOnline
		}

		if err := s.store.UpdateDeviceLastSeen(ctx, existing.ID); err != nil {
			return nil, err
		}
		if err := s.store.UpdateDevice(ctx, existing); err != nil {
			return nil, err
		}

		return existing, nil
	}

	// 新设备注册
	var modelName string
	if req.ModelID > 0 {
		m, err := s.modelStore.GetModelByID(ctx, req.ModelID)
		if err == nil {
			modelName = m.Name
		}
	} else if req.ModelName != "" {
		m, err := s.modelStore.GetModelByName(ctx, req.ModelName)
		if err != nil {
			return nil, fmt.Errorf("model not found: %w", err)
		}
		req.ModelID = m.ID
		modelName = m.Name
	} else {
		return nil, fmt.Errorf("model_id or model_name is required")
	}

	d := model.NewDevice(req.DeviceID, req.ModelID, modelName, req.Name, req.IPAddress, req.SerialNumber)
	d.CurrentFWVer = req.FirmwareVer

	if req.Status != "" {
		d.Status = model.DeviceStatus(req.Status)
	} else {
		d.Status = model.DeviceOnline
	}

	if err := s.store.CreateDevice(ctx, d); err != nil {
		return nil, fmt.Errorf("failed to register device: %w", err)
	}

	return d, nil
}

// SearchDevices 搜索设备
func (s *DeviceService) SearchDevices(ctx context.Context, keyword string, page, pageSize int) ([]*model.Device, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	return s.store.SearchDevices(ctx, keyword, page, pageSize)
}

// BatchCreateDevices 批量创建设备
func (s *DeviceService) BatchCreateDevices(ctx context.Context, req *model.BatchCreateDeviceRequest) (*model.BatchResult, error) {
	result := &model.BatchResult{}
	for i, devReq := range req.Devices {
		device, err := s.CreateDevice(ctx, &devReq)
		if err != nil {
			result.Results = append(result.Results, model.BatchItem{
				Index:   i,
				Success: false,
				Message: err.Error(),
			})
			result.FailCount++
		} else {
			result.Results = append(result.Results, model.BatchItem{
				Index:   i,
				Success: true,
				Message: "created",
				ID:      device.ID,
			})
			result.SuccessCount++
		}
	}
	return result, nil
}

// GetDeviceStatusStats 获取设备状态统计
func (s *DeviceService) GetDeviceStatusStats(ctx context.Context) (map[model.DeviceStatus]int, error) {
	return s.store.CountDevicesByStatus(ctx)
}
