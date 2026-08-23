package store

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"fwupgrade/internal/model"
)

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

// SnapshotData 数据快照，用于持久化。
// 快照中的元素是结构体值的拷贝，调用方在释放锁后可安全地进行序列化，
// 不会与并发的写操作产生数据竞争。
type SnapshotData struct {
	Models    []*model.DeviceModel
	Devices   []*model.Device
	Firmwares []*model.Firmware
	Tasks     []*model.UpgradeTask
	Records   []*model.UpgradeRecord
	IDCounter model.ID
}

// Snapshot 在读锁保护下获取所有数据的一致性快照拷贝。
// 快照按 ID 排序，保证持久化结果确定且便于核对。
func (s *MemoryStore) Snapshot() *SnapshotData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := &SnapshotData{
		IDCounter: s.idCounter,
		Models:    make([]*model.DeviceModel, 0, len(s.models)),
		Devices:   make([]*model.Device, 0, len(s.devices)),
		Firmwares: make([]*model.Firmware, 0, len(s.firmwares)),
		Tasks:     make([]*model.UpgradeTask, 0, len(s.tasks)),
		Records:   make([]*model.UpgradeRecord, 0, len(s.records)),
	}

	// 拷贝结构体值而非共享指针，避免序列化期间并发写修改同一对象导致竞态。
	// 注意：Device.Metadata 与 UpgradeTask.TargetDevices 目前在代码中仅在创建时赋值、
	// 不会被并发原地修改，因此共享底层数据是安全的。
	for _, m := range s.models {
		cp := *m
		snap.Models = append(snap.Models, &cp)
	}
	for _, d := range s.devices {
		cp := *d
		snap.Devices = append(snap.Devices, &cp)
	}
	for _, f := range s.firmwares {
		cp := *f
		snap.Firmwares = append(snap.Firmwares, &cp)
	}
	for _, t := range s.tasks {
		cp := *t
		snap.Tasks = append(snap.Tasks, &cp)
	}
	for _, r := range s.records {
		cp := *r
		snap.Records = append(snap.Records, &cp)
	}

	// 按 ID 排序，保证输出确定
	sort.Slice(snap.Models, func(i, j int) bool { return snap.Models[i].ID < snap.Models[j].ID })
	sort.Slice(snap.Devices, func(i, j int) bool { return snap.Devices[i].ID < snap.Devices[j].ID })
	sort.Slice(snap.Firmwares, func(i, j int) bool { return snap.Firmwares[i].ID < snap.Firmwares[j].ID })
	sort.Slice(snap.Tasks, func(i, j int) bool { return snap.Tasks[i].ID < snap.Tasks[j].ID })
	sort.Slice(snap.Records, func(i, j int) bool { return snap.Records[i].ID < snap.Records[j].ID })

	return snap
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
