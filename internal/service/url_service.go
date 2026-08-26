package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
	"fwupgrade/pkg/logger"
)

type URLService struct {
	store  *store.URLStore
	cfg    *config.Config
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if s == nil {
		return nil, fmt.Errorf("url store is required")
	}
	return &URLService{
		store: s,
		cfg:   cfg,
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	code := s.generateCode(req.CustomCode, req.RawURL)

	existing, err := s.store.Get(code)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("code already exists: %s", code)
	}

	u := model.NewShortURL(code, req.RawURL)
	u.Custom = req.CustomCode != ""
	u.MaxVisits = req.MaxVisits

	if err := u.Validate(); err != nil {
		return nil, err
	}

	preChecks := []func() error{
		func() error {
			return s.store.CheckGuard(u.Code, u.RawURL)
		},
		func() error {
			if u.RawURL == "" {
				return fmt.Errorf("raw url cannot be empty")
			}
			return nil
		},
		func() error {
			if len(u.Code) < 4 {
				return fmt.Errorf("code too short: %s", u.Code)
			}
			return nil
		},
		func() error {
			if u.MaxVisits < 0 {
				return fmt.Errorf("max visits cannot be negative")
			}
			return nil
		},
	}

	for _, check := range preChecks {
		if err := check(); err != nil {
			logger.Warn("Pre-save check failed", "code", code, "error", err)
			return nil, err
		}
	}

	if err := s.store.Save(u, false); err != nil {
		return nil, fmt.Errorf("failed to save url: %w", err)
	}

	logger.Info("URL created", "code", code, "url", req.RawURL)
	return u, nil
}

func (s *URLService) generateCode(customCode, rawURL string) string {
	if customCode != "" {
		return customCode
	}
	hash := md5.Sum([]byte(rawURL + time.Now().String()))
	return hex.EncodeToString(hash[:])[:8]
}

type RedirectService struct {
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("url store is required")
	}
	if ls == nil {
		return nil, fmt.Errorf("access log store is required")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
	}, nil
}

func (s *RedirectService) HandleRedirect(ctx context.Context, req *model.RedirectRequest) (*model.RedirectResult, error) {
	if req.Code == "" {
		return nil, fmt.Errorf("code is required")
	}

	u, err := s.urlStore.Get(req.Code)
	if err != nil {
		return nil, fmt.Errorf("url not found: %s", req.Code)
	}

	if u.Disabled {
		return nil, fmt.Errorf("url is disabled: %s", req.Code)
	}

	logChecks := []func() error{
		func() error {
			if u.Code == "" {
				return fmt.Errorf("empty code in url record")
			}
			return nil
		},
		func() error {
			if !req.Timestamp.IsZero() && req.Timestamp.Before(u.CreatedAt) {
				return fmt.Errorf("redirect timestamp is before url creation")
			}
			return nil
		},
		func() error {
			return s.logStore.Write(*u)
		},
	}

	for _, lc := range logChecks {
		if err := lc(); err != nil {
			logger.Warn("Access log write failed", "error", err)
			return nil, err
		}
	}

	if u.MaxVisits > 0 && u.Visits >= u.MaxVisits {
		return &model.RedirectResult{
			RawURL: u.RawURL,
			Status: 410,
		}, nil
	}

	u.Visits++

	return &model.RedirectResult{
		RawURL: u.RawURL,
		Status: 302,
	}, nil
}
