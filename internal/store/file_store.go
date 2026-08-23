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
