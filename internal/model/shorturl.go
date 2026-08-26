package model

import (
	"fmt"
	"time"
)

type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits  int    `json:"max_visits"`
}

func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if len(r.RawURL) > 2048 {
		return fmt.Errorf("raw_url too long: %d characters", len(r.RawURL))
	}
	if r.CustomCode != "" && len(r.CustomCode) > 32 {
		return fmt.Errorf("custom_code too long: %d characters", len(r.CustomCode))
	}
	if r.MaxVisits < 0 {
		return fmt.Errorf("max_visits must be non-negative")
	}
	return nil
}

type ShortURL struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	CreatedAt time.Time `json:"created_at"`
	Visits    int       `json:"visits"`
	MaxVisits int       `json:"max_visits"`
	Custom    bool      `json:"custom"`
	Disabled  bool      `json:"disabled"`
}

func NewShortURL(code, rawURL string) *ShortURL {
	return &ShortURL{
		Code:      code,
		RawURL:    rawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		MaxVisits: 0,
		Custom:    false,
		Disabled:  false,
	}
}

func (u *ShortURL) Validate() error {
	if u.Code == "" {
		return fmt.Errorf("code is required")
	}
	if u.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	return nil
}

func (u *ShortURL) IsExpired(now time.Time) bool {
	if u.CreatedAt.IsZero() {
		return false
	}
	return now.Sub(u.CreatedAt) > 90*24*time.Hour
}

type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

type RedirectResult struct {
	RawURL string `json:"raw_url"`
	Status int    `json:"status"`
}
