package store

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"fwupgrade/internal/model"
)

// PanicGuardFn 故障守卫函数类型，用于故障演练和诊断
type PanicGuardFn func(code, rawURL string) bool

// MemoryStore 基于内存的数据存储实现
type MemoryStore struct {
	mu sync.RWMutex

	// 自增ID计数器
	idCounter model.ID

	// 型号数据
	models map[model.ID]*model.DeviceModel
	// 型号名称索引
	modelNameIndex map[string]model.ID

	// 设备数据
	devices map[model.ID]*model.Device
	// 设备ID索引
	deviceIDIndex map[string]model.ID

	// 固件数据
	firmwares map[model.ID]*model.Firmware
	// 固件版本索引 (modelID + version)
	firmwareVersionIndex map[string]model.ID

	// 任务数据
	tasks map[model.ID]*model.UpgradeTask

	// 升级记录
	records map[model.ID]*model.UpgradeRecord

	// 故障守卫钩子
	panicGuard PanicGuardFn
}

// NewMemoryStore 创建内存存储实例
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		idCounter:           0,
		models:              make(map[model.ID]*model.DeviceModel),
		modelNameIndex:      make(map[string]model.ID),
		devices:             make(map[model.ID]*model.Device),
		deviceIDIndex:       make(map[string]model.ID),
		firmwares:           make(map[model.ID]*model.Firmware),
		firmwareVersionIndex: make(map[string]model.ID),
		tasks:               make(map[model.ID]*model.UpgradeTask),
		records:             make(map[model.ID]*model.UpgradeRecord),
	}
}

// nextID 获取下一个自增ID
func (s *MemoryStore) nextID() model.ID {
	s.idCounter++
	return s.idCounter
}

// Init 初始化存储
func (s *MemoryStore) Init(_ context.Context) error {
	return nil
}

// Close 关闭存储
func (s *MemoryStore) Close() error {
	return nil
}

// ================ DeviceModelStore 实现 ================

// CreateModel 创建设备型号
func (s *MemoryStore) CreateModel(_ context.Context, m *model.DeviceModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查名称是否重复
	if _, exists := s.modelNameIndex[m.Name]; exists {
		return fmt.Errorf("model with name '%s' already exists", m.Name)
	}

	id := s.nextID()
	m.ID = id
	s.models[id] = m
	s.modelNameIndex[m.Name] = id

	return nil
}

// GetModelByID 根据ID获取设备型号
func (s *MemoryStore) GetModelByID(_ context.Context, id model.ID) (*model.DeviceModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.models[id]
	if !ok {
		return nil, fmt.Errorf("model not found: id=%d", id)
	}
	return m, nil
}

// GetModelByName 根据名称获取设备型号
func (s *MemoryStore) GetModelByName(_ context.Context, name string) (*model.DeviceModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.modelNameIndex[name]
	if !ok {
		return nil, fmt.Errorf("model not found: name=%s", name)
	}
	return s.models[id], nil
}

// ListModels 列出设备型号
func (s *MemoryStore) ListModels(_ context.Context, page, pageSize int) ([]*model.DeviceModel, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := int64(len(s.models))
	models := make([]*model.DeviceModel, 0, len(s.models))

	for _, m := range s.models {
		models = append(models, m)
	}

	// 按ID排序
	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})

	// 分页
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.DeviceModel{}, total, nil
	}

	return models[start:end], total, nil
}

// ListModelsByManufacturer 根据厂商列出设备型号
func (s *MemoryStore) ListModelsByManufacturer(_ context.Context, manufacturer string) ([]*model.DeviceModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.DeviceModel
	for _, m := range s.models {
		if m.Manufacturer == manufacturer {
			result = append(result, m)
		}
	}
	return result, nil
}

// UpdateModel 更新设备型号
func (s *MemoryStore) UpdateModel(_ context.Context, m *model.DeviceModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.models[m.ID]
	if !ok {
		return fmt.Errorf("model not found: id=%d", m.ID)
	}

	// 如果名称改变，更新索引
	if existing.Name != m.Name {
		if _, exists := s.modelNameIndex[m.Name]; exists {
			return fmt.Errorf("model with name '%s' already exists", m.Name)
		}
		delete(s.modelNameIndex, existing.Name)
		s.modelNameIndex[m.Name] = m.ID
	}

	m.UpdatedAt = time.Now()
	s.models[m.ID] = m

	// 同步更新相关设备的型号名
	for _, d := range s.devices {
		if d.ModelID == m.ID {
			d.ModelName = m.Name
		}
	}

	return nil
}

// DeleteModel 删除设备型号
func (s *MemoryStore) DeleteModel(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.models[id]
	if !ok {
		return fmt.Errorf("model not found: id=%d", id)
	}

	// 检查是否有设备使用此型号
	for _, d := range s.devices {
		if d.ModelID == id {
			return fmt.Errorf("cannot delete model: model has devices")
		}
	}

	delete(s.models, id)
	delete(s.modelNameIndex, m.Name)

	return nil
}

// SetModelActive 设置型号活跃状态
func (s *MemoryStore) SetModelActive(ctx context.Context, id model.ID, active bool) error {
	m, err := s.GetModelByID(ctx, id)
	if err != nil {
		return err
	}
	m.IsActive = active
	return s.UpdateModel(ctx, m)
}

// CountModelDevices 统计型号下的设备数量
func (s *MemoryStore) CountModelDevices(_ context.Context, modelID model.ID) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, d := range s.devices {
		if d.ModelID == modelID {
			count++
		}
	}
	return count, nil
}

// ================ DeviceStore 实现 ================

// CreateDevice 创建设备
func (s *MemoryStore) CreateDevice(_ context.Context, d *model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查设备ID是否重复
	if _, exists := s.deviceIDIndex[d.DeviceID]; exists {
		return fmt.Errorf("device with id '%s' already exists", d.DeviceID)
	}

	id := s.nextID()
	d.ID = id
	s.devices[id] = d
	s.deviceIDIndex[d.DeviceID] = id

	// 更新型号设备计数
	if m, ok := s.models[d.ModelID]; ok {
		m.DeviceCount++
	}

	return nil
}

// GetDeviceByID 根据ID获取设备
func (s *MemoryStore) GetDeviceByID(_ context.Context, id model.ID) (*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	d, ok := s.devices[id]
	if !ok {
		return nil, fmt.Errorf("device not found: id=%d", id)
	}
	return d, nil
}

// GetDeviceByDeviceID 根据设备ID获取设备
func (s *MemoryStore) GetDeviceByDeviceID(_ context.Context, deviceID string) (*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.deviceIDIndex[deviceID]
	if !ok {
		return nil, fmt.Errorf("device not found: device_id=%s", deviceID)
	}
	return s.devices[id], nil
}

// ListDevices 列出设备
func (s *MemoryStore) ListDevices(_ context.Context, page, pageSize int, modelID model.ID, status model.DeviceStatus) ([]*model.Device, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]*model.Device, 0)
	for _, d := range s.devices {
		if modelID > 0 && d.ModelID != modelID {
			continue
		}
		if status != "" && d.Status != status {
			continue
		}
		devices = append(devices, d)
	}

	// 排序
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	total := int64(len(devices))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.Device{}, total, nil
	}

	return devices[start:end], total, nil
}

// ListDevicesByModel 根据型号列出设备
func (s *MemoryStore) ListDevicesByModel(_ context.Context, modelID model.ID) ([]*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Device
	for _, d := range s.devices {
		if d.ModelID == modelID {
			result = append(result, d)
		}
	}
	return result, nil
}

// ListDevicesByStatus 根据状态列出设备
func (s *MemoryStore) ListDevicesByStatus(_ context.Context, status model.DeviceStatus) ([]*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Device
	for _, d := range s.devices {
		if d.Status == status {
			result = append(result, d)
		}
	}
	return result, nil
}

// ListOnlineDevices 列出在线设备
func (s *MemoryStore) ListOnlineDevices(_ context.Context) ([]*model.Device, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Device
	for _, d := range s.devices {
		if d.IsOnline() {
			result = append(result, d)
		}
	}
	return result, nil
}

// UpdateDevice 更新设备
func (s *MemoryStore) UpdateDevice(_ context.Context, d *model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.devices[d.ID]; !ok {
		return fmt.Errorf("device not found: id=%d", d.ID)
	}

	s.devices[d.ID] = d
	return nil
}

// UpdateDeviceStatus 更新设备状态
func (s *MemoryStore) UpdateDeviceStatus(_ context.Context, id model.ID, status model.DeviceStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.devices[id]
	if !ok {
		return fmt.Errorf("device not found: id=%d", id)
	}

	d.Status = status
	d.LastSeenAt = time.Now()
	return nil
}

// UpdateDeviceLastSeen 更新设备最后心跳时间
func (s *MemoryStore) UpdateDeviceLastSeen(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.devices[id]
	if !ok {
		return fmt.Errorf("device not found: id=%d", id)
	}

	d.LastSeenAt = time.Now()
	return nil
}

// UpdateDeviceProgress 更新升级进度
func (s *MemoryStore) UpdateDeviceProgress(_ context.Context, id model.ID, progress int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.devices[id]
	if !ok {
		return fmt.Errorf("device not found: id=%d", id)
	}

	d.UpgradeProgress = progress
	if progress >= 100 {
		d.Status = model.DeviceOnline
		d.TargetFWVer = d.CurrentFWVer
	}
	return nil
}

// DeleteDevice 删除设备
func (s *MemoryStore) DeleteDevice(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	d, ok := s.devices[id]
	if !ok {
		return fmt.Errorf("device not found: id=%d", id)
	}

	delete(s.devices, id)
	delete(s.deviceIDIndex, d.DeviceID)

	// 更新型号设备计数
	if m, ok := s.models[d.ModelID]; ok {
		if m.DeviceCount > 0 {
			m.DeviceCount--
		}
	}

	return nil
}

// BatchCreateDevices 批量创建设备
func (s *MemoryStore) BatchCreateDevices(ctx context.Context, devices []*model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, d := range devices {
		if _, exists := s.deviceIDIndex[d.DeviceID]; exists {
			continue // 跳过已存在的设备
		}
		id := s.nextID()
		d.ID = id
		s.devices[id] = d
		s.deviceIDIndex[d.DeviceID] = id
		if m, ok := s.models[d.ModelID]; ok {
			m.DeviceCount++
		}
	}
	return nil
}

// CountDevicesByStatus 按状态统计设备
func (s *MemoryStore) CountDevicesByStatus(_ context.Context) (map[model.DeviceStatus]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[model.DeviceStatus]int)
	for _, d := range s.devices {
		result[d.Status]++
	}
	return result, nil
}

// SearchDevices 搜索设备
func (s *MemoryStore) SearchDevices(_ context.Context, keyword string, page, pageSize int) ([]*model.Device, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keywordLower := toLower(keyword)
	var devices []*model.Device
	for _, d := range s.devices {
		if containsStr(toLower(d.DeviceID), keywordLower) ||
			containsStr(toLower(d.Name), keywordLower) ||
			containsStr(toLower(d.SerialNumber), keywordLower) {
			devices = append(devices, d)
		}
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	total := int64(len(devices))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.Device{}, total, nil
	}

	return devices[start:end], total, nil
}

// GetAllDevices 分页获取所有设备
func (s *MemoryStore) GetAllDevices(_ context.Context, page, pageSize int) ([]*model.Device, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]*model.Device, 0, len(s.devices))
	for _, d := range s.devices {
		devices = append(devices, d)
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].ID < devices[j].ID
	})

	total := int64(len(devices))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.Device{}, total, nil
	}

	return devices[start:end], total, nil
}

// toLower 转换字符串为小写
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

// containsStr 检查字符串是否包含子串（小写比较）
func containsStr(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// SetPanicGuard 设置故障守卫钩子，用于故障演练和诊断
func (s *MemoryStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

// checkPanicGuard 检查故障守卫
func (s *MemoryStore) checkPanicGuard(code, rawURL string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.panicGuard != nil {
		return s.panicGuard(code, rawURL)
	}
	return false
}

// ================ FirmwareStore 实现 ================

// CreateFirmware 创建固件
func (s *MemoryStore) CreateFirmware(_ context.Context, f *model.Firmware) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID()
	f.ID = id
	s.firmwares[id] = f

	// 添加版本索引
	key := fmt.Sprintf("%d:%s", f.ModelID, f.Version)
	s.firmwareVersionIndex[key] = id

	return nil
}

// GetFirmwareByID 根据ID获取固件
func (s *MemoryStore) GetFirmwareByID(_ context.Context, id model.ID) (*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, ok := s.firmwares[id]
	if !ok {
		return nil, fmt.Errorf("firmware not found: id=%d", id)
	}
	return f, nil
}

// GetFirmwareByVersion 根据型号和版本获取固件
func (s *MemoryStore) GetFirmwareByVersion(_ context.Context, modelID model.ID, version string) (*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("%d:%s", modelID, version)
	id, ok := s.firmwareVersionIndex[key]
	if !ok {
		return nil, fmt.Errorf("firmware not found: model_id=%d, version=%s", modelID, version)
	}
	return s.firmwares[id], nil
}

// GetLatestFirmware 获取型号的最新固件
func (s *MemoryStore) GetLatestFirmware(_ context.Context, modelID model.ID) (*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest *model.Firmware
	for _, f := range s.firmwares {
		if f.ModelID == modelID {
			if latest == nil || f.ReleaseDate.After(latest.ReleaseDate) {
				latest = f
			}
		}
	}

	if latest == nil {
		return nil, fmt.Errorf("no firmware found for model: %d", modelID)
	}
	return latest, nil
}

// ListFirmwares 列出固件
func (s *MemoryStore) ListFirmwares(_ context.Context, page, pageSize int, modelID model.ID) ([]*model.Firmware, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var firmwares []*model.Firmware
	for _, f := range s.firmwares {
		if modelID > 0 && f.ModelID != modelID {
			continue
		}
		firmwares = append(firmwares, f)
	}

	sort.Slice(firmwares, func(i, j int) bool {
		return firmwares[i].CreatedAt.After(firmwares[j].CreatedAt)
	})

	total := int64(len(firmwares))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.Firmware{}, total, nil
	}

	return firmwares[start:end], total, nil
}

// ListFirmwaresByModel 根据型号列出固件
func (s *MemoryStore) ListFirmwaresByModel(_ context.Context, modelID model.ID) ([]*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Firmware
	for _, f := range s.firmwares {
		if f.ModelID == modelID {
			result = append(result, f)
		}
	}
	return result, nil
}

// UpdateFirmware 更新固件
func (s *MemoryStore) UpdateFirmware(_ context.Context, f *model.Firmware) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.firmwares[f.ID]; !ok {
		return fmt.Errorf("firmware not found: id=%d", f.ID)
	}

	f.UpdatedAt = time.Now()
	s.firmwares[f.ID] = f
	return nil
}

// SetFirmwareActive 设置固件活跃状态
func (s *MemoryStore) SetFirmwareActive(ctx context.Context, id model.ID, active bool) error {
	f, err := s.GetFirmwareByID(ctx, id)
	if err != nil {
		return err
	}
	f.IsActive = active
	return s.UpdateFirmware(ctx, f)
}

// IncrementFirmwareDownload 增加下载计数
func (s *MemoryStore) IncrementFirmwareDownload(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, ok := s.firmwares[id]
	if !ok {
		return fmt.Errorf("firmware not found: id=%d", id)
	}

	f.DownloadCount++
	return nil
}

// DeleteFirmware 删除固件
func (s *MemoryStore) DeleteFirmware(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, ok := s.firmwares[id]
	if !ok {
		return fmt.Errorf("firmware not found: id=%d", id)
	}

	delete(s.firmwares, id)
	key := fmt.Sprintf("%d:%s", f.ModelID, f.Version)
	delete(s.firmwareVersionIndex, key)

	return nil
}

// GetAllFirmwares 获取所有固件
func (s *MemoryStore) GetAllFirmwares(_ context.Context) ([]*model.Firmware, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.Firmware
	for _, f := range s.firmwares {
		result = append(result, f)
	}
	return result, nil
}

// CountFirmwaresByModel 按型号统计固件数量
func (s *MemoryStore) CountFirmwaresByModel(_ context.Context) (map[model.ID]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[model.ID]int)
	for _, f := range s.firmwares {
		result[f.ModelID]++
	}
	return result, nil
}

// ================ TaskStore 实现 ================

// CreateTask 创建任务
func (s *MemoryStore) CreateTask(_ context.Context, t *model.UpgradeTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID()
	t.ID = id
	s.tasks[id] = t

	return nil
}

// GetTaskByID 根据ID获取任务
func (s *MemoryStore) GetTaskByID(_ context.Context, id model.ID) (*model.UpgradeTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: id=%d", id)
	}
	return t, nil
}

// ListTasks 列出任务
func (s *MemoryStore) ListTasks(_ context.Context, page, pageSize int, status model.TaskStatus) ([]*model.UpgradeTask, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*model.UpgradeTask
	for _, t := range s.tasks {
		if status != "" && t.Status != status {
			continue
		}
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})

	total := int64(len(tasks))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.UpgradeTask{}, total, nil
	}

	return tasks[start:end], total, nil
}

// ListActiveTasks 列出活跃任务
func (s *MemoryStore) ListActiveTasks(_ context.Context) ([]*model.UpgradeTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeTask
	for _, t := range s.tasks {
		if t.Status == model.TaskPending || t.Status == model.TaskRunning {
			result = append(result, t)
		}
	}
	return result, nil
}

// ListTasksByModel 根据型号列出任务
func (s *MemoryStore) ListTasksByModel(_ context.Context, modelID model.ID) ([]*model.UpgradeTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeTask
	for _, t := range s.tasks {
		if t.ModelID == modelID {
			result = append(result, t)
		}
	}
	return result, nil
}

// UpdateTask 更新任务
func (s *MemoryStore) UpdateTask(_ context.Context, t *model.UpgradeTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[t.ID]; !ok {
		return fmt.Errorf("task not found: id=%d", t.ID)
	}

	t.UpdatedAt = time.Now()
	s.tasks[t.ID] = t
	return nil
}

// UpdateTaskStatus 更新任务状态
func (s *MemoryStore) UpdateTaskStatus(_ context.Context, id model.ID, status model.TaskStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: id=%d", id)
	}

	t.Status = status
	t.UpdatedAt = time.Now()

	if status == model.TaskRunning && t.StartedAt == nil {
		now := time.Now()
		t.StartedAt = &now
	}
	if status == model.TaskCompleted || status == model.TaskFailed || status == model.TaskCancelled {
		now := time.Now()
		t.CompletedAt = &now
	}

	return nil
}

// UpdateTaskProgress 更新任务进度
func (s *MemoryStore) UpdateTaskProgress(_ context.Context, id model.ID, successCount, failCount, pendingCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: id=%d", id)
	}

	t.SuccessCount = successCount
	t.FailCount = failCount
	t.PendingCount = pendingCount
	t.Progress = t.CalculateProgress()
	t.UpdatedAt = time.Now()

	return nil
}

// DeleteTask 删除任务
func (s *MemoryStore) DeleteTask(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return fmt.Errorf("task not found: id=%d", id)
	}

	delete(s.tasks, id)
	return nil
}

// GetAllTasks 获取所有任务
func (s *MemoryStore) GetAllTasks(_ context.Context) ([]*model.UpgradeTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeTask
	for _, t := range s.tasks {
		result = append(result, t)
	}
	return result, nil
}

// SearchTasks 搜索任务
func (s *MemoryStore) SearchTasks(_ context.Context, keyword string, page, pageSize int) ([]*model.UpgradeTask, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keywordLower := toLower(keyword)
	var tasks []*model.UpgradeTask
	for _, t := range s.tasks {
		if containsStr(toLower(t.Name), keywordLower) ||
			containsStr(toLower(t.Description), keywordLower) {
			tasks = append(tasks, t)
		}
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})

	total := int64(len(tasks))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.UpgradeTask{}, total, nil
	}

	return tasks[start:end], total, nil
}

// GetRecentTasks 获取最近任务
func (s *MemoryStore) GetRecentTasks(_ context.Context, limit int) ([]*model.UpgradeTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*model.UpgradeTask
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})

	if limit > len(tasks) {
		limit = len(tasks)
	}
	return tasks[:limit], nil
}

// ================ RecordStore 实现 ================

// CreateRecord 创建记录
func (s *MemoryStore) CreateRecord(_ context.Context, r *model.UpgradeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := s.nextID()
	r.ID = id
	s.records[id] = r

	return nil
}

// GetRecordByID 根据ID获取记录
func (s *MemoryStore) GetRecordByID(_ context.Context, id model.ID) (*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.records[id]
	if !ok {
		return nil, fmt.Errorf("record not found: id=%d", id)
	}
	return r, nil
}

// ListRecords 列出记录
func (s *MemoryStore) ListRecords(_ context.Context, page, pageSize int, status model.UpgradeStatus) ([]*model.UpgradeRecord, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []*model.UpgradeRecord
	for _, r := range s.records {
		if status != "" && r.Status != status {
			continue
		}
		records = append(records, r)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt.After(records[j].StartedAt)
	})

	total := int64(len(records))
	start := (page - 1) * pageSize
	if start > int(total) {
		start = int(total)
	}
	end := start + pageSize
	if end > int(total) {
		end = int(total)
	}

	if start >= int(total) {
		return []*model.UpgradeRecord{}, total, nil
	}

	return records[start:end], total, nil
}

// ListRecordsByDevice 根据设备列出记录
func (s *MemoryStore) ListRecordsByDevice(_ context.Context, deviceID string) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeRecord
	for _, r := range s.records {
		if r.DeviceID == deviceID {
			result = append(result, r)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.After(result[j].StartedAt)
	})

	return result, nil
}

// ListRecordsByTask 根据任务列出记录
func (s *MemoryStore) ListRecordsByTask(_ context.Context, taskID model.ID) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeRecord
	for _, r := range s.records {
		if r.TaskID == taskID {
			result = append(result, r)
		}
	}
	return result, nil
}

// UpdateRecord 更新记录
func (s *MemoryStore) UpdateRecord(_ context.Context, r *model.UpgradeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.records[r.ID]; !ok {
		return fmt.Errorf("record not found: id=%d", r.ID)
	}

	s.records[r.ID] = r
	return nil
}

// UpdateRecordStatus 更新记录状态
func (s *MemoryStore) UpdateRecordStatus(_ context.Context, id model.ID, status model.UpgradeStatus, progress int, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.records[id]
	if !ok {
		return fmt.Errorf("record not found: id=%d", id)
	}

	r.Status = status
	r.Progress = progress
	if errorMsg != "" {
		r.ErrorMessage = errorMsg
	}
	if status == model.UpgradeSuccess || status == model.UpgradeFailed {
		now := time.Now()
		r.CompletedAt = &now
		r.Duration = now.Sub(r.StartedAt).Milliseconds()
	}

	return nil
}

// DeleteRecord 删除记录
func (s *MemoryStore) DeleteRecord(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.records[id]; !ok {
		return fmt.Errorf("record not found: id=%d", id)
	}

	delete(s.records, id)
	return nil
}

// GetAllRecords 获取所有记录
func (s *MemoryStore) GetAllRecords(_ context.Context) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.UpgradeRecord
	for _, r := range s.records {
		result = append(result, r)
	}
	return result, nil
}

// GetRecentRecords 获取最近记录
func (s *MemoryStore) GetRecentRecords(_ context.Context, limit int) ([]*model.UpgradeRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var records []*model.UpgradeRecord
	for _, r := range s.records {
		records = append(records, r)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt.After(records[j].StartedAt)
	})

	if limit > len(records) {
		limit = len(records)
	}
	return records[:limit], nil
}

// CountRecordsByStatus 按状态统计记录
func (s *MemoryStore) CountRecordsByStatus(_ context.Context) (map[model.UpgradeStatus]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[model.UpgradeStatus]int)
	for _, r := range s.records {
		result[r.Status]++
	}
	return result, nil
}

// CountTodayRecords 统计今日记录
func (s *MemoryStore) CountTodayRecords(_ context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	today := time.Now()
	count := 0
	for _, r := range s.records {
		if r.StartedAt.Year() == today.Year() && r.StartedAt.Month() == today.Month() && r.StartedAt.Day() == today.Day() {
			count++
		}
	}
	return count, nil
}

// RawSnapshot 返回存储的原始快照，用于诊断
func (s *MemoryStore) RawSnapshot() map[string]model.UpgradeTask {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]model.UpgradeTask)
	for id, task := range s.tasks {
		result[fmt.Sprintf("%d", id)] = *task
	}
	return result
}
