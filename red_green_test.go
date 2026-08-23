package fwupgrade

import (
	"strings"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	cfg := config.Default()

	// ============ Test 1: PanicGuard error swallowed (URLService.Create) ============
	t.Run("PanicGuard error swallowed by err shadowing", func(t *testing.T) {
		urlStore, err := store.NewURLStore(cfg)
		if err != nil {
			t.Fatalf("Cannot create URLStore: %v", err)
		}

		urlStore.SetPanicGuard(func(code, rawURL string) bool {
			return strings.Contains(rawURL, "blocked")
		})

		urlService, err := service.NewURLService(cfg, urlStore)
		if err != nil {
			t.Fatalf("Cannot create URLService: %v", err)
		}

		req := &model.CreateReq{
			RawURL:    "https://blocked.example.com/malware",
			MaxVisits: 100,
		}

		_, createErr := urlService.Create(nil, req)

		if createErr == nil {
			t.Errorf("RED: Create should have rejected blocked URL but returned success - defect is PRESENT (err variable shadowing)")
		} else {
			t.Log("GREEN: Create properly rejected blocked URL - defect is FIXED")
		}
	})

	// ============ Test 2: Valid URL creation still works ============
	t.Run("Valid URL creation works", func(t *testing.T) {
		urlStore, err := store.NewURLStore(cfg)
		if err != nil {
			t.Fatalf("Cannot create URLStore: %v", err)
		}

		urlService, err := service.NewURLService(cfg, urlStore)
		if err != nil {
			t.Fatalf("Cannot create URLService: %v", err)
		}

		validReq := &model.CreateReq{
			RawURL:    "https://example.com/valid-page",
			MaxVisits: 50,
		}

		_, validErr := urlService.Create(nil, validReq)
		if validErr != nil {
			t.Errorf("FAIL: Valid URL creation failed: %v", validErr)
		}
	})

	// ============ Test 3: Access log failure swallowed (RedirectService) ============
	t.Run("Access log failure swallowed by err shadowing", func(t *testing.T) {
		urlStore, err := store.NewURLStore(cfg)
		if err != nil {
			t.Fatalf("Cannot create URLStore: %v", err)
		}

		urlService, err := service.NewURLService(cfg, urlStore)
		if err != nil {
			t.Fatalf("Cannot create URLService: %v", err)
		}

		// First create a valid URL
		validReq := &model.CreateReq{
			RawURL:    "https://example.com/valid-page",
			MaxVisits: 50,
		}
		validURL, validErr := urlService.Create(nil, validReq)
		if validErr != nil {
			t.Fatalf("Cannot create valid URL: %v", validErr)
		}

		// Create a log store but DON'T open it
		logStore := store.NewAccessLogStore()
		redirectService, err := service.NewRedirectService(urlStore, logStore)
		if err != nil {
			t.Fatalf("Cannot create RedirectService: %v", err)
		}

		redirectReq := &model.RedirectRequest{
			Code: validURL.Code,
		}

		_, redirectErr := redirectService.HandleRedirect(nil, redirectReq)

		if redirectErr == nil {
			t.Errorf("RED: Redirect should have failed but returned success - defect is PRESENT (err variable shadowing)")
		} else {
			t.Log("GREEN: Redirect properly failed when access log was not opened - defect is FIXED")
		}
	})

	// ============ Test 4: Redirect works when log store is opened ============
	t.Run("Redirect with properly opened log store", func(t *testing.T) {
		urlStore, err := store.NewURLStore(cfg)
		if err != nil {
			t.Fatalf("Cannot create URLStore: %v", err)
		}

		urlService, err := service.NewURLService(cfg, urlStore)
		if err != nil {
			t.Fatalf("Cannot create URLService: %v", err)
		}

		// Create a valid URL
		validReq := &model.CreateReq{
			RawURL:    "https://example.com/redirect-test",
			MaxVisits: 50,
		}
		validURL, validErr := urlService.Create(nil, validReq)
		if validErr != nil {
			t.Fatalf("Cannot create valid URL: %v", validErr)
		}

		// Open log store first
		logStore := store.NewAccessLogStore()
		logStore.Open(nil)

		redirectService, err := service.NewRedirectService(urlStore, logStore)
		if err != nil {
			t.Fatalf("Cannot create RedirectService: %v", err)
		}

		redirectReq := &model.RedirectRequest{
			Code: validURL.Code,
		}

		_, redirectErr := redirectService.HandleRedirect(nil, redirectReq)
		if redirectErr != nil {
			t.Errorf("FAIL: Redirect with opened log store failed: %v", redirectErr)
		}

		logCount := logStore.RecordCount()
		if logCount == 0 {
			t.Errorf("FAIL: Access log record was not written")
		}
	})

	// ============ Test 5: validateRecord error swallowed (URLStore.validateRecord) ============
	t.Run("validateRecord error swallowed by err shadowing", func(t *testing.T) {
		urlStore, err := store.NewURLStore(cfg)
		if err != nil {
			t.Fatalf("Cannot create URLStore: %v", err)
		}

		// Create a ShortURL with invalid data (negative MaxVisits)
		// ShortURL.Validate() doesn't check MaxVisits, but validateRecord does
		u := model.NewShortURL("test1", "https://example.com/test")
		u.MaxVisits = -5 // Invalid! validateRecord should reject this

		// Call Save directly - it calls validateRecord internally
		saveErr := urlStore.Save(u, false)

		if saveErr == nil {
			t.Errorf("RED: Save should have rejected invalid record but returned success - defect is PRESENT (err variable shadowing in validateRecord)")
		} else {
			t.Log("GREEN: Save properly rejected invalid record - defect is FIXED")
		}
	})
}
