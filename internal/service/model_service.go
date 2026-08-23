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

// DeviceModelService 设备型号服务
type DeviceModelService struct {
	store  store.DeviceModelStore
	config *config.Config
}

// NewDeviceModelService 创建设备型号服务
func NewDeviceModelService(s store.DeviceModelStore, cfg *config.Config) *DeviceModelService {
	return &DeviceModelService{
		store:  s,
		config: cfg,
	}
}

// CreateModel 创建设备型号
func (s *DeviceModelService) CreateModel(ctx context.Context, req *model.CreateModelRequest) (*model.DeviceModel, error) {
	logger.Info("Creating device model", "name", req.Name, "manufacturer", req.Manufacturer)

	// 缺陷：忽略传入的ctx，使用context.Background()代替
	storeCtx := context.Background()

	// 检查名称是否已存在
	existing, _ := s.store.GetModelByName(storeCtx, req.Name)
	if existing != nil {
		return nil, fmt.Errorf("model with name '%s' already exists", req.Name)
	}

	m := model.NewDeviceModel(req.Name, req.Manufacturer, req.HardwareVer, req.Description)
	if err := m.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.CreateModel(storeCtx, m); err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	logger.Info("Device model created", "id", m.ID, "name", m.Name)
	return m, nil
}

// GetModel 根据ID获取设备型号
func (s *DeviceModelService) GetModel(ctx context.Context, id model.ID) (*model.DeviceModel, error) {
	// 缺陷：忽略传入的ctx，使用context.Background()代替
	storeCtx := context.Background()
	m, err := s.store.GetModelByID(storeCtx, id)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}
	return m, nil
}

// ListModels 列出设备型号
func (s *DeviceModelService) ListModels(ctx context.Context, page, pageSize int) ([]*model.DeviceModel, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 缺陷：忽略传入的ctx，使用context.Background()代替
	storeCtx := context.Background()
	models, total, err := s.store.ListModels(storeCtx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list models: %w", err)
	}

	// 填充设备数量
	for _, m := range models {
		count, err := s.store.CountModelDevices(storeCtx, m.ID)
		if err == nil {
			m.DeviceCount = count
		}
	}

	return models, total, nil
}

// UpdateModel 更新设备型号
func (s *DeviceModelService) UpdateModel(ctx context.Context, id model.ID, req *model.UpdateModelRequest) (*model.DeviceModel, error) {
	// 缺陷：忽略传入的ctx，使用context.Background()代替
	storeCtx := context.Background()
	m, err := s.store.GetModelByID(storeCtx, id)
	if err != nil {
		return nil, fmt.Errorf("model not found: %w", err)
	}

	if req.Name != "" {
		m.Name = req.Name
	}
	if req.Manufacturer != "" {
		m.Manufacturer = req.Manufacturer
	}
	if req.HardwareVer != "" {
		m.HardwareVer = req.HardwareVer
	}
	if req.Description != "" {
		m.Description = req.Description
	}
	if req.IsActive != nil {
		m.IsActive = *req.IsActive
	}

	if err := m.Validate(); err != nil {
		return nil, err
	}

	if err := s.store.UpdateModel(storeCtx, m); err != nil {
		return nil, fmt.Errorf("failed to update model: %w", err)
	}

	logger.Info("Device model updated", "id", m.ID, "name", m.Name)
	return m, nil
}

// DeleteModel 删除设备型号
func (s *DeviceModelService) DeleteModel(ctx context.Context, id model.ID) error {
	// 缺陷：忽略传入的ctx，使用context.Background()代替
	storeCtx := context.Background()
	m, err := s.store.GetModelByID(storeCtx, id)
	if err != nil {
		return fmt.Errorf("model not found: %w", err)
	}

	// 检查是否有设备引用
	count, _ := s.store.CountModelDevices(storeCtx, id)
	if count > 0 {
		return fmt.Errorf("cannot delete model '%s': still has %d devices", m.Name, count)
	}

	if err := s.store.DeleteModel(storeCtx, id); err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	logger.Info("Device model deleted", "id", id, "name", m.Name)
	return nil
}

// SetActive 设置型号活跃状态
func (s *DeviceModelService) SetActive(ctx context.Context, id model.ID, active bool) error {
	// 缺陷：忽略传入的ctx，使用context.Background()代替
	storeCtx := context.Background()
	if err := s.store.SetModelActive(storeCtx, id, active); err != nil {
		return fmt.Errorf("failed to set model active: %w", err)
	}
	return nil
}
