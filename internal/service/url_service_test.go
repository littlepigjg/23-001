package service

import (
	"context"
	"strings"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
)

func newTestURLStoreWithGuard(guard store.PanicGuardFn) *store.URLStore {
	s, err := store.NewURLStore(config.DefaultConfig())
	if err != nil {
		panic(err)
	}
	if guard != nil {
		s.SetPanicGuard(guard)
	}
	return s
}

func mustNewURLService(s *store.URLStore) *URLService {
	svc, err := NewURLService(config.DefaultConfig(), s)
	if err != nil {
		panic(err)
	}
	return svc
}

// TestURLService_Create_RejectsBlockedURL 回归 Bug 1：
// 被内容安全拦截的 URL 必须返回错误，且不得落库生成短链。
func TestURLService_Create_RejectsBlockedURL(t *testing.T) {
	s := newTestURLStoreWithGuard(func(code, rawURL string) bool {
		return strings.Contains(rawURL, "blocked.example")
	})
	svc := mustNewURLService(s)

	req := &model.CreateReq{RawURL: "https://blocked.example/evil"}
	res, err := svc.Create(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "panic guard triggered") {
		t.Fatalf("expected panic guard triggered error, got: %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result for blocked url, got %+v", res)
	}

	// 任意随机生成的 code 都不应存在于 store 中
	if s.SavedCount() != 0 {
		t.Fatalf("blocked url must not be persisted, savedCount=%d", s.SavedCount())
	}
}

// TestURLService_Create_RejectsNegativeMaxVisits 回归 Bug 3：
// 负数 MaxVisits 必须被拒绝（model 校验 + store 兜底）。
func TestURLService_Create_RejectsNegativeMaxVisits(t *testing.T) {
	s := newTestURLStoreWithGuard(nil)
	svc := mustNewURLService(s)

	req := &model.CreateReq{RawURL: "https://example.com", MaxVisits: -5}
	if _, err := svc.Create(context.Background(), req); err == nil {
		t.Fatal("expected error for negative max visits, got nil")
	}
	if s.SavedCount() != 0 {
		t.Fatalf("negative max visits record must not be persisted, savedCount=%d", s.SavedCount())
	}
}

// TestURLService_Create_DoesNotDoubleSave 回归 Bug 1 重构：
// 正常创建应恰好写入一次（旧实现 SaveWithGuard + Save 双写）。
func TestURLService_Create_DoesNotDoubleSave(t *testing.T) {
	s := newTestURLStoreWithGuard(nil)
	svc := mustNewURLService(s)

	req := &model.CreateReq{RawURL: "https://example.com/once"}
	u, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if s.SavedCount() != 1 {
		t.Fatalf("expected savedCount=1, got %d", s.SavedCount())
	}
	got, err := s.Get(u.Code)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.RawURL != req.RawURL {
		t.Fatalf("persisted url mismatch: got %q", got.RawURL)
	}
}

// TestURLService_Create_RejectsDuplicateCode 确认重复 custom code 被拒绝而非覆盖。
func TestURLService_Create_RejectsDuplicateCode(t *testing.T) {
	s := newTestURLStoreWithGuard(nil)
	svc := mustNewURLService(s)

	req := &model.CreateReq{RawURL: "https://a.example", CustomCode: "dupcode"}
	if _, err := svc.Create(context.Background(), req); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	req2 := &model.CreateReq{RawURL: "https://b.example", CustomCode: "dupcode"}
	if _, err := svc.Create(context.Background(), req2); err == nil {
		t.Fatal("expected duplicate code error, got nil")
	}
	got, err := s.Get("dupcode")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.RawURL != "https://a.example" {
		t.Fatalf("existing record was overwritten: got %q", got.RawURL)
	}
}

func mustNewRedirectService(us *store.URLStore, ls *store.AccessLogStore) *RedirectService {
	svc, err := NewRedirectService(us, ls)
	if err != nil {
		panic(err)
	}
	return svc
}

func seedURL(t *testing.T, s *store.URLStore, code, rawURL string) *model.ShortURL {
	t.Helper()
	u := model.NewShortURL(code, rawURL)
	if err := s.Save(u, false); err != nil {
		t.Fatalf("seed save failed: %v", err)
	}
	return u
}

// TestRedirectService_HandleRedirect_AbortsWhenLogStoreClosed 回归 Bug 2：
// 日志存储未初始化时重定向必须失败并将错误传播给调用方，而非静默返回 302。
func TestRedirectService_HandleRedirect_AbortsWhenLogStoreClosed(t *testing.T) {
	us := newTestURLStoreWithGuard(nil)
	seedURL(t, us, "abcd1234", "https://example.com")
	ls := store.NewAccessLogStore() // 未 Open
	svc := mustNewRedirectService(us, ls)

	res, err := svc.HandleRedirect(context.Background(), &model.RedirectRequest{Code: "abcd1234"})
	if err == nil || !strings.Contains(err.Error(), "not opened") {
		t.Fatalf("expected not opened error, got: %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result when log store closed, got %+v", res)
	}
	if ls.RecordCount() != 0 {
		t.Fatalf("expected 0 records, got %d", ls.RecordCount())
	}
}

// TestRedirectService_HandleRedirect_SucceedsWhenLogStoreOpened 确认修复后 happy path 仍正常。
func TestRedirectService_HandleRedirect_SucceedsWhenLogStoreOpened(t *testing.T) {
	us := newTestURLStoreWithGuard(nil)
	seedURL(t, us, "abcd1234", "https://example.com")
	ls := store.NewAccessLogStore()
	if err := ls.Open(context.Background()); err != nil {
		t.Fatalf("open failed: %v", err)
	}
	svc := mustNewRedirectService(us, ls)

	res, err := svc.HandleRedirect(context.Background(), &model.RedirectRequest{Code: "abcd1234"})
	if err != nil {
		t.Fatalf("redirect failed: %v", err)
	}
	if res == nil || res.Status != 302 || res.RawURL != "https://example.com" {
		t.Fatalf("unexpected redirect result: %+v", res)
	}
	if ls.RecordCount() != 1 {
		t.Fatalf("expected 1 log record, got %d", ls.RecordCount())
	}
}

// TestRedirectService_HandleRedirect_DisabledURLReturnsError 确认禁用的短链被拒绝。
func TestRedirectService_HandleRedirect_DisabledURLReturnsError(t *testing.T) {
	us := newTestURLStoreWithGuard(nil)
	u := seedURL(t, us, "abcd1234", "https://example.com")
	u.Disabled = true
	ls := store.NewAccessLogStore()
	if err := ls.Open(context.Background()); err != nil {
		t.Fatalf("open failed: %v", err)
	}
	svc := mustNewRedirectService(us, ls)

	if _, err := svc.HandleRedirect(context.Background(), &model.RedirectRequest{Code: "abcd1234"}); err == nil {
		t.Fatal("expected error for disabled url, got nil")
	}
}
