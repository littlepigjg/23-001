package store

import (
	"context"
	"strings"
	"testing"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
)

func newTestURLStore() *URLStore {
	s, err := NewURLStore(config.DefaultConfig())
	if err != nil {
		panic(err)
	}
	return s
}

// TestValidateRecord_RejectsNegativeMaxVisits 校验 store 层兜底：负数 MaxVisits 必须被拒绝。
// 回归 Bug 3 —— validateRecord 旧实现用 continue 吞掉错误、永远返回 nil。
func TestValidateRecord_RejectsNegativeMaxVisits(t *testing.T) {
	s := newTestURLStore()
	u := model.NewShortURL("abcd1234", "https://example.com")
	u.MaxVisits = -1

	if err := s.Save(u, false); err == nil || !strings.Contains(err.Error(), "negative max visits") {
		t.Fatalf("expected negative max visits error, got: %v", err)
	}
	if _, err := s.Get(u.Code); err == nil {
		t.Fatal("negative max visits record should not have been persisted")
	}
}

// TestSaveWithGuard_RejectsBlockedURL 校验 panic guard 拦截时既返回错误、也不落库。
func TestSaveWithGuard_RejectsBlockedURL(t *testing.T) {
	s := newTestURLStore()
	s.SetPanicGuard(func(code, rawURL string) bool {
		return strings.Contains(rawURL, "blocked.example")
	})

	u := model.NewShortURL("block001", "https://blocked.example/evil")
	u.MaxVisits = 0

	if err := s.SaveWithGuard(u, false); err == nil || !strings.Contains(err.Error(), "panic guard triggered") {
		t.Fatalf("expected panic guard triggered error, got: %v", err)
	}
	if _, err := s.Get(u.Code); err == nil {
		t.Fatal("blocked url must not be persisted")
	}
}

// TestSave_OverwriteFalseRejectsExistingCode 确认 overwrite=false 不会静默覆盖已存在记录。
func TestSave_OverwriteFalseRejectsExistingCode(t *testing.T) {
	s := newTestURLStore()
	u1 := model.NewShortURL("samecode", "https://a.example")
	if err := s.Save(u1, false); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	u2 := model.NewShortURL("samecode", "https://b.example")
	if err := s.Save(u2, false); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected already exists error, got: %v", err)
	}

	got, err := s.Get("samecode")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got.RawURL != "https://a.example" {
		t.Fatalf("existing record was overwritten: got %q", got.RawURL)
	}
}

// TestCheckGuard_NoGuardAllowsByDefault 确认 guard 未配置时 CheckGuard 放行（沿用 SaveWithGuard 语义）。
func TestCheckGuard_NoGuardAllowsByDefault(t *testing.T) {
	s := newTestURLStore() // 未调用 SetPanicGuard
	if err := s.CheckGuard("abcd1234", "https://example.com"); err != nil {
		t.Fatalf("expected nil when no guard configured, got: %v", err)
	}
}

// TestAccessLogStore_WriteFailsWhenNotOpened 校验日志存储未初始化时 Write 返回错误。
// 回归 Bug 2 —— HandleRedirect 旧实现吞掉该错误导致重定向静默成功。
func TestAccessLogStore_WriteFailsWhenNotOpened(t *testing.T) {
	ls := NewAccessLogStore() // 未 Open
	if err := ls.Write(*model.NewShortURL("abcd1234", "https://example.com")); err == nil || !strings.Contains(err.Error(), "not opened") {
		t.Fatalf("expected not opened error, got: %v", err)
	}
	if ls.RecordCount() != 0 {
		t.Fatalf("expected 0 records, got %d", ls.RecordCount())
	}
}

// TestAccessLogStore_OpenedAllowsWrite 确认 Open 之后 Write 成功。
func TestAccessLogStore_OpenedAllowsWrite(t *testing.T) {
	ls := NewAccessLogStore()
	if err := ls.Open(context.Background()); err != nil {
		t.Fatalf("open failed: %v", err)
	}
	if err := ls.Write(*model.NewShortURL("abcd1234", "https://example.com")); err != nil {
		t.Fatalf("write after open failed: %v", err)
	}
	if ls.RecordCount() != 1 {
		t.Fatalf("expected 1 record, got %d", ls.RecordCount())
	}
}
