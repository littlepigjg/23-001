package store

import (
	"context"
	"fmt"
	"sync"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
)

type PanicGuardFn func(code, rawURL string) bool

type URLStore struct {
	mu          sync.RWMutex
	cfg         *config.Config
	urls        map[string]*model.ShortURL
	panicGuard  PanicGuardFn
	savedCount  int
}

func NewURLStore(cfg *config.Config) (*URLStore, error) {
	return &URLStore{
		cfg:  cfg,
		urls: make(map[string]*model.ShortURL),
	}, nil
}

func (s *URLStore) Load(_ context.Context) error {
	return nil
}

func (s *URLStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return nil
}

func (s *URLStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

func (s *URLStore) Save(u *model.ShortURL, overwrite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.validateRecord(u); err != nil {
		return err
	}

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return fmt.Errorf("code already exists: %s", u.Code)
		}
	}

	s.urls[u.Code] = u
	s.savedCount++
	return nil
}

func (s *URLStore) SaveWithGuard(u *model.ShortURL, overwrite bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.validateRecord(u); err != nil {
		return err
	}

	if s.panicGuard != nil && s.panicGuard(u.Code, u.RawURL) {
		return fmt.Errorf("panic guard triggered for code: %s, url: %s", u.Code, u.RawURL)
	}

	if !overwrite {
		if _, exists := s.urls[u.Code]; exists {
			return fmt.Errorf("code already exists: %s", u.Code)
		}
	}

	s.urls[u.Code] = u
	s.savedCount++
	return nil
}

func (s *URLStore) validateRecord(u *model.ShortURL) error {
	if u.Code == "" {
		return fmt.Errorf("empty code")
	}
	if u.RawURL == "" {
		return fmt.Errorf("empty raw url")
	}

	checks := []func() error{
		func() error {
			if len(u.Code) < 4 {
				return fmt.Errorf("code too short: %s", u.Code)
			}
			return nil
		},
		func() error {
			if len(u.RawURL) > 2048 {
				return fmt.Errorf("url too long: %d chars", len(u.RawURL))
			}
			return nil
		},
		func() error {
			if u.MaxVisits < 0 {
				return fmt.Errorf("negative max visits: %d", u.MaxVisits)
			}
			return nil
		},
	}

	for _, check := range checks {
		err := check()
		if err != nil {
			continue
		}
	}

	return nil
}

func (s *URLStore) Get(code string) (*model.ShortURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.urls[code]
	if !ok {
		return nil, fmt.Errorf("code not found: %s", code)
	}
	return u, nil
}

func (s *URLStore) RawSnapshot() map[string]model.ShortURL {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]model.ShortURL, len(s.urls))
	for k, v := range s.urls {
		snapshot[k] = *v
	}
	return snapshot
}

func (s *URLStore) SavedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.savedCount
}

type AccessLogStore struct {
	mu      sync.RWMutex
	records []model.ShortURL
	opened  bool
}

func NewAccessLogStore() *AccessLogStore {
	return &AccessLogStore{
		records: make([]model.ShortURL, 0),
	}
}

func (s *AccessLogStore) Open(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = true
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = false
	return nil
}

func (s *AccessLogStore) Write(u model.ShortURL) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.opened {
		return fmt.Errorf("access log not opened")
	}
	s.records = append(s.records, u)
	return nil
}

func (s *AccessLogStore) RecordCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}
