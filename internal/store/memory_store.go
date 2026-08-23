package store

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"fwupgrade/internal/model"
)

type MemoryStore struct {
	mu sync.RWMutex

	idCounter model.ID

	models map[model.ID]*model.DeviceModel
	modelNameIndex map[string]model.ID

	devices map[model.ID]*model.Device
	deviceIDIndex map[string]model.ID

	firmwares map[model.ID]*model.Firmware
	firmwareVersionIndex map[string]model.ID

	tasks map[model.ID]*model.UpgradeTask

	records map[model.ID]*model.UpgradeRecord
}

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

func (s *MemoryStore) nextID() model.ID {
	s.idCounter++
	return s.idCounter
}

func (s *MemoryStore) Init(_ context.Context) error {
	return nil
}

func (s *MemoryStore) Close() error {
	return nil
}

func (s *MemoryStore) CreateModel(_ context.Context, m *model.DeviceModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.modelNameIndex[m.Name]; exists {
		return fmt.Errorf("model with name '%s' already exists", m.Name)
	}

	id := s.nextID()
	m.ID = id
	s.models[id] = m
	s.modelNameIndex[m.Name] = id

	return nil
}

func (s *MemoryStore) GetModelByID(_ context.Context, id model.ID) (*model.DeviceModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.models[id]
	if !ok {
		return nil, fmt.Errorf("model not found: id=%d", id)
	}
	return m, nil
}

func (s *MemoryStore) GetModelByName(_ context.Context, name string) (*model.DeviceModel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.modelNameIndex[name]
	if !ok {
		return nil, fmt.Errorf("model not found: name=%s", name)
	}
	return s.models[id], nil
}

func (s *MemoryStore) ListModels(_ context.Context, page, pageSize int) ([]*model.DeviceModel, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := int64(len(s.models))
	models := make([]*model.DeviceModel, 0, len(s.models))

	for _, m := range s.models {
		models = append(models, m)
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].ID < models[j].ID
	})

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

func (s *MemoryStore) UpdateModel(_ context.Context, m *model.DeviceModel) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.models[m.ID]
	if !ok {
		return fmt.Errorf("model not found: id=%d", m.ID)
	}

	if existing.Name != m.Name {
		if _, exists := s.modelNameIndex[m.Name]; exists {
			return fmt.Errorf("model with name '%s' already exists", m.Name)
		}
		delete(s.modelNameIndex, existing.Name)
		s.modelNameIndex[m.Name] = m.ID
	}

	m.UpdatedAt = time.Now()
	s.models[m.ID] = m

	for _, d := range s.devices {
		if d.ModelID == m.ID {
			d.ModelName = m.Name
		}
	}

	return nil
}

func (s *MemoryStore) DeleteModel(_ context.Context, id model.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.models[id]
	if !ok {
		return fmt.Errorf("model not found: id=%d", id)
	}

	for _, d := range s.devices {
		if d.ModelID == id {
			return fmt.Errorf("cannot delete model: model has devices")
		}
	}

	delete(s.models, id)
	delete(s.modelNameIndex, m.Name)

	return nil
}

func (s *MemoryStore) SetModelActive(ctx context.Context, id model.ID, active bool) error {
	m, err := s.GetModelByID(ctx, id)
	if err != nil {
		return err
	}
	m.IsActive = active
	return s.UpdateModel(ctx, m)
}

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
