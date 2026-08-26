package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/pkg/logger"
)

// FileStore 基于文件的存储实现
type FileStore struct {
	mu       sync.RWMutex
	cfg      *config.Config
	memStore *MemoryStore
	dirty    bool
}

// NewFileStore 创建文件存储实例
func NewFileStore(cfg *config.Config) *FileStore {
	return &FileStore{
		cfg:      cfg,
		memStore: NewMemoryStore(),
	}
}

// Init 初始化文件存储
func (s *FileStore) Init(ctx context.Context) error {
	// 确保数据目录存在
	dataDir := s.cfg.Storage.DataDir
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	// 尝试从文件加载数据
	dataFile := filepath.Join(dataDir, "data.json")
	if _, err := os.Stat(dataFile); err == nil {
		if err := s.loadFromFile(dataFile); err != nil {
			logger.Warn("Failed to load data from file, starting with empty store", "error", err)
		} else {
			logger.Info("Data loaded from file", "path", dataFile)
		}
	}

	// 启动自动保存
	go s.autoSave(ctx)

	return nil
}

// Close 关闭存储
func (s *FileStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dirty {
		return s.saveToFile()
	}
	return nil
}

// autoSave 自动保存协程
func (s *FileStore) autoSave(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			if s.dirty {
				_ = s.saveToFile()
			}
			s.mu.Unlock()
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.dirty {
				if err := s.saveToFile(); err != nil {
					logger.Error("Auto save failed", "error", err)
				}
			}
			s.mu.Unlock()
		}
	}
}

// saveToFile 保存数据到文件
func (s *FileStore) saveToFile() error {
	dataDir := s.cfg.Storage.DataDir
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	dataFile := filepath.Join(dataDir, "data.json")
	data := struct {
		Models    []*model.DeviceModel    `json:"models"`
		Devices   []*model.Device         `json:"devices"`
		Firmwares []*model.Firmware       `json:"firmwares"`
		Tasks     []*model.UpgradeTask    `json:"tasks"`
		Records   []*model.UpgradeRecord  `json:"records"`
		IDCounter model.ID                `json:"id_counter"`
		SavedAt   time.Time               `json:"saved_at"`
	}{
		SavedAt: time.Now(),
	}

	// 收集所有数据
	s.memStore.mu.RLock()
	for _, m := range s.memStore.models {
		data.Models = append(data.Models, m)
	}
	for _, d := range s.memStore.devices {
		data.Devices = append(data.Devices, d)
	}
	for _, f := range s.memStore.firmwares {
		data.Firmwares = append(data.Firmwares, f)
	}
	for _, t := range s.memStore.tasks {
		data.Tasks = append(data.Tasks, t)
	}
	for _, r := range s.memStore.records {
		data.Records = append(data.Records, r)
	}
	data.IDCounter = s.memStore.idCounter
	s.memStore.mu.RUnlock()

	// 写入临时文件然后重命名（原子操作）
	tmpFile := dataFile + ".tmp"
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(tmpFile, jsonData, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, dataFile); err != nil {
		return err
	}

	s.dirty = false
	logger.Debug("Data saved to file", "path", dataFile)
	return nil
}

// loadFromFile 从文件加载数据
func (s *FileStore) loadFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var loaded struct {
		Models    []*model.DeviceModel    `json:"models"`
		Devices   []*model.Device         `json:"devices"`
		Firmwares []*model.Firmware       `json:"firmwares"`
		Tasks     []*model.UpgradeTask    `json:"tasks"`
		Records   []*model.UpgradeRecord  `json:"records"`
		IDCounter model.ID                `json:"id_counter"`
	}

	if err := json.Unmarshal(data, &loaded); err != nil {
		return err
	}

	// 恢复数据
	s.memStore.mu.Lock()
	defer s.memStore.mu.Unlock()

	for _, m := range loaded.Models {
		s.memStore.models[m.ID] = m
		s.memStore.modelNameIndex[m.Name] = m.ID
	}
	for _, d := range loaded.Devices {
		s.memStore.devices[d.ID] = d
		s.memStore.deviceIDIndex[d.DeviceID] = d.ID
	}
	for _, f := range loaded.Firmwares {
		s.memStore.firmwares[f.ID] = f
		key := fmt.Sprintf("%d:%s", f.ModelID, f.Version)
		s.memStore.firmwareVersionIndex[key] = f.ID
	}
	for _, t := range loaded.Tasks {
		s.memStore.tasks[t.ID] = t
	}
	for _, r := range loaded.Records {
		s.memStore.records[r.ID] = r
	}
	s.memStore.idCounter = loaded.IDCounter

	return nil
}

// markDirty 标记数据已修改
func (s *FileStore) markDirty() {
	s.dirty = true
}

// ================ DeviceModelStore 实现 ================

func (s *FileStore) CreateModel(ctx context.Context, m *model.DeviceModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.CreateModel(ctx, m); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) GetModelByID(ctx context.Context, id model.ID) (*model.DeviceModel, error) {
	return s.memStore.GetModelByID(ctx, id)
}

func (s *FileStore) GetModelByName(ctx context.Context, name string) (*model.DeviceModel, error) {
	return s.memStore.GetModelByName(ctx, name)
}

func (s *FileStore) ListModels(ctx context.Context, page, pageSize int) ([]*model.DeviceModel, int64, error) {
	return s.memStore.ListModels(ctx, page, pageSize)
}

func (s *FileStore) ListModelsByManufacturer(ctx context.Context, manufacturer string) ([]*model.DeviceModel, error) {
	return s.memStore.ListModelsByManufacturer(ctx, manufacturer)
}

func (s *FileStore) UpdateModel(ctx context.Context, m *model.DeviceModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateModel(ctx, m); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) DeleteModel(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.DeleteModel(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) SetModelActive(ctx context.Context, id model.ID, active bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.SetModelActive(ctx, id, active); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) CountModelDevices(ctx context.Context, modelID model.ID) (int, error) {
	return s.memStore.CountModelDevices(ctx, modelID)
}

// ================ DeviceStore 实现 ================

func (s *FileStore) CreateDevice(ctx context.Context, d *model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.CreateDevice(ctx, d); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) GetDeviceByID(ctx context.Context, id model.ID) (*model.Device, error) {
	return s.memStore.GetDeviceByID(ctx, id)
}

func (s *FileStore) GetDeviceByDeviceID(ctx context.Context, deviceID string) (*model.Device, error) {
	return s.memStore.GetDeviceByDeviceID(ctx, deviceID)
}

func (s *FileStore) ListDevices(ctx context.Context, page, pageSize int, modelID model.ID, status model.DeviceStatus) ([]*model.Device, int64, error) {
	return s.memStore.ListDevices(ctx, page, pageSize, modelID, status)
}

func (s *FileStore) ListDevicesByModel(ctx context.Context, modelID model.ID) ([]*model.Device, error) {
	return s.memStore.ListDevicesByModel(ctx, modelID)
}

func (s *FileStore) ListDevicesByStatus(ctx context.Context, status model.DeviceStatus) ([]*model.Device, error) {
	return s.memStore.ListDevicesByStatus(ctx, status)
}

func (s *FileStore) ListOnlineDevices(ctx context.Context) ([]*model.Device, error) {
	return s.memStore.ListOnlineDevices(ctx)
}

func (s *FileStore) UpdateDevice(ctx context.Context, d *model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateDevice(ctx, d); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) UpdateDeviceStatus(ctx context.Context, id model.ID, status model.DeviceStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateDeviceStatus(ctx, id, status); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) UpdateDeviceLastSeen(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateDeviceLastSeen(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) UpdateDeviceProgress(ctx context.Context, id model.ID, progress int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateDeviceProgress(ctx, id, progress); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) DeleteDevice(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.DeleteDevice(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) BatchCreateDevices(ctx context.Context, devices []*model.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.BatchCreateDevices(ctx, devices); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) CountDevicesByStatus(ctx context.Context) (map[model.DeviceStatus]int, error) {
	return s.memStore.CountDevicesByStatus(ctx)
}

func (s *FileStore) SearchDevices(ctx context.Context, keyword string, page, pageSize int) ([]*model.Device, int64, error) {
	return s.memStore.SearchDevices(ctx, keyword, page, pageSize)
}

func (s *FileStore) GetAllDevices(ctx context.Context, page, pageSize int) ([]*model.Device, int64, error) {
	return s.memStore.GetAllDevices(ctx, page, pageSize)
}

// ================ FirmwareStore 实现 ================

func (s *FileStore) CreateFirmware(ctx context.Context, f *model.Firmware) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.CreateFirmware(ctx, f); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) GetFirmwareByID(ctx context.Context, id model.ID) (*model.Firmware, error) {
	return s.memStore.GetFirmwareByID(ctx, id)
}

func (s *FileStore) GetFirmwareByVersion(ctx context.Context, modelID model.ID, version string) (*model.Firmware, error) {
	return s.memStore.GetFirmwareByVersion(ctx, modelID, version)
}

func (s *FileStore) GetLatestFirmware(ctx context.Context, modelID model.ID) (*model.Firmware, error) {
	return s.memStore.GetLatestFirmware(ctx, modelID)
}

func (s *FileStore) ListFirmwares(ctx context.Context, page, pageSize int, modelID model.ID) ([]*model.Firmware, int64, error) {
	return s.memStore.ListFirmwares(ctx, page, pageSize, modelID)
}

func (s *FileStore) ListFirmwaresByModel(ctx context.Context, modelID model.ID) ([]*model.Firmware, error) {
	return s.memStore.ListFirmwaresByModel(ctx, modelID)
}

func (s *FileStore) UpdateFirmware(ctx context.Context, f *model.Firmware) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateFirmware(ctx, f); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) SetFirmwareActive(ctx context.Context, id model.ID, active bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.SetFirmwareActive(ctx, id, active); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) IncrementFirmwareDownload(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.IncrementFirmwareDownload(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) DeleteFirmware(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.DeleteFirmware(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) GetAllFirmwares(ctx context.Context) ([]*model.Firmware, error) {
	return s.memStore.GetAllFirmwares(ctx)
}

func (s *FileStore) CountFirmwaresByModel(ctx context.Context) (map[model.ID]int, error) {
	return s.memStore.CountFirmwaresByModel(ctx)
}

// ================ TaskStore 实现 ================

func (s *FileStore) CreateTask(ctx context.Context, t *model.UpgradeTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.CreateTask(ctx, t); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) GetTaskByID(ctx context.Context, id model.ID) (*model.UpgradeTask, error) {
	return s.memStore.GetTaskByID(ctx, id)
}

func (s *FileStore) ListTasks(ctx context.Context, page, pageSize int, status model.TaskStatus) ([]*model.UpgradeTask, int64, error) {
	return s.memStore.ListTasks(ctx, page, pageSize, status)
}

func (s *FileStore) ListActiveTasks(ctx context.Context) ([]*model.UpgradeTask, error) {
	return s.memStore.ListActiveTasks(ctx)
}

func (s *FileStore) ListTasksByModel(ctx context.Context, modelID model.ID) ([]*model.UpgradeTask, error) {
	return s.memStore.ListTasksByModel(ctx, modelID)
}

func (s *FileStore) UpdateTask(ctx context.Context, t *model.UpgradeTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateTask(ctx, t); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) UpdateTaskStatus(ctx context.Context, id model.ID, status model.TaskStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateTaskStatus(ctx, id, status); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) UpdateTaskProgress(ctx context.Context, id model.ID, successCount, failCount, pendingCount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateTaskProgress(ctx, id, successCount, failCount, pendingCount); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) DeleteTask(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.DeleteTask(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) GetAllTasks(ctx context.Context) ([]*model.UpgradeTask, error) {
	return s.memStore.GetAllTasks(ctx)
}

func (s *FileStore) SearchTasks(ctx context.Context, keyword string, page, pageSize int) ([]*model.UpgradeTask, int64, error) {
	return s.memStore.SearchTasks(ctx, keyword, page, pageSize)
}

func (s *FileStore) GetRecentTasks(ctx context.Context, limit int) ([]*model.UpgradeTask, error) {
	return s.memStore.GetRecentTasks(ctx, limit)
}

// ================ RecordStore 实现 ================

func (s *FileStore) CreateRecord(ctx context.Context, r *model.UpgradeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.CreateRecord(ctx, r); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) GetRecordByID(ctx context.Context, id model.ID) (*model.UpgradeRecord, error) {
	return s.memStore.GetRecordByID(ctx, id)
}

func (s *FileStore) ListRecords(ctx context.Context, page, pageSize int, status model.UpgradeStatus) ([]*model.UpgradeRecord, int64, error) {
	return s.memStore.ListRecords(ctx, page, pageSize, status)
}

func (s *FileStore) ListRecordsByDevice(ctx context.Context, deviceID string) ([]*model.UpgradeRecord, error) {
	return s.memStore.ListRecordsByDevice(ctx, deviceID)
}

func (s *FileStore) ListRecordsByTask(ctx context.Context, taskID model.ID) ([]*model.UpgradeRecord, error) {
	return s.memStore.ListRecordsByTask(ctx, taskID)
}

func (s *FileStore) UpdateRecord(ctx context.Context, r *model.UpgradeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateRecord(ctx, r); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) UpdateRecordStatus(ctx context.Context, id model.ID, status model.UpgradeStatus, progress int, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.UpdateRecordStatus(ctx, id, status, progress, errorMsg); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) DeleteRecord(ctx context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.memStore.DeleteRecord(ctx, id); err != nil {
		return err
	}
	s.markDirty()
	return nil
}

func (s *FileStore) GetAllRecords(ctx context.Context) ([]*model.UpgradeRecord, error) {
	return s.memStore.GetAllRecords(ctx)
}

func (s *FileStore) GetRecentRecords(ctx context.Context, limit int) ([]*model.UpgradeRecord, error) {
	return s.memStore.GetRecentRecords(ctx, limit)
}

func (s *FileStore) CountRecordsByStatus(ctx context.Context) (map[model.UpgradeStatus]int, error) {
	return s.memStore.CountRecordsByStatus(ctx)
}

func (s *FileStore) CountTodayRecords(ctx context.Context) (int, error) {
	return s.memStore.CountTodayRecords(ctx)
}
