package service

import (
	"testing"

	"fwupgrade/internal/model"
)

func TestNormalizeVersionHandlesEmpty(t *testing.T) {
	// 空版本号历史上会触发 "index out of range [0] with length 0"，
	// 现在必须安全地归一化为空字符串。
	if got := normalizeVersion(""); got != "" {
		t.Fatalf("normalizeVersion(\"\") = %q, want \"\"", got)
	}
	// 仅前缀 "v"/"V" 后为空也算空版本。
	for _, v := range []string{"v", "V"} {
		if got := normalizeVersion(v); got != "" {
			t.Fatalf("normalizeVersion(%q) = %q, want \"\"", v, got)
		}
	}
}

func TestNormalizeVersionStable(t *testing.T) {
	cases := map[string]string{
		"1.2.3":   "0000000001.0000000002.0000000003",
		"v1.2.3":  "0000000001.0000000002.0000000003",
		"V1.10.2": "0000000001.0000000010.0000000002",
	}
	for in, want := range cases {
		if got := normalizeVersion(in); got != want {
			t.Fatalf("normalizeVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCompareVersionStringsOrdering(t *testing.T) {
	// 正常版本之间的相对顺序不应被空版本处理影响。
	if !compareVersionStrings("1.2.3", "1.2.4") {
		t.Fatalf("1.2.3 should sort before 1.2.4")
	}
	if compareVersionStrings("1.10.0", "1.2.0") {
		t.Fatalf("1.10.0 should sort after 1.2.0 (numeric, not lexical)")
	}
	// 空版本排在所有正常版本之前。
	if !compareVersionStrings("", "1.0.0") {
		t.Fatalf("empty version should sort before 1.0.0")
	}
}

func TestShouldSkipUpgradeEmptyCurrent(t *testing.T) {
	s := &PollService{}
	// 空当前版本不应被判为"已在目标版本"而被跳过，否则空版本设备永远拿不到升级。
	if s.ShouldSkipUpgrade("", "1.2.3") {
		t.Fatalf("empty current version should not skip upgrade")
	}
	// 注意：current==target 且均为空的退化场景在 PollDevice 中已由
	// req.CurrentVer == task.FirmwareVer 的相等性判断提前 continue 处理，
	// 这里只校验存在有效目标时空当前版本一律放行的核心语义。
	// 正常语义保持：当前已是目标版本应跳过。
	if !s.ShouldSkipUpgrade("1.2.3", "1.2.3") {
		t.Fatalf("current==target should skip")
	}
	// 正常语义保持：当前低于目标不跳过。
	if s.ShouldSkipUpgrade("1.2.3", "1.2.4") {
		t.Fatalf("lower current should not skip")
	}
	// 正常语义保持：当前高于目标应跳过。
	if !s.ShouldSkipUpgrade("1.2.4", "1.2.3") {
		t.Fatalf("higher current should skip")
	}
}

func TestSortFirmwaresByVersionEmptySafe(t *testing.T) {
	// 含空版本固件时排序不得 panic，且空版本排在最前。
	fws := []*model.Firmware{
		{Version: "1.2.3"},
		{Version: ""},
		{Version: "1.2.10"},
		{Version: "v1.2.1"},
	}
	sortFirmwaresByVersion(fws)
	if fws[0].Version != "" {
		t.Fatalf("expected empty version first, got %q", fws[0].Version)
	}
	if fws[1].Version != "v1.2.1" {
		t.Fatalf("expected 1.2.1 second, got %q", fws[1].Version)
	}
	if fws[2].Version != "1.2.3" {
		t.Fatalf("expected 1.2.3 third, got %q", fws[2].Version)
	}
	if fws[3].Version != "1.2.10" {
		t.Fatalf("expected 1.2.10 last, got %q", fws[3].Version)
	}
}

func TestFirmwareServiceNormalizeForSortEmpty(t *testing.T) {
	s := &FirmwareService{}
	if got := s.normalizeVersionForSort(""); got != "" {
		t.Fatalf("normalizeVersionForSort(\"\") = %q, want \"\"", got)
	}
}

func TestFirmwareServiceSortFirmwaresEmptySafe(t *testing.T) {
	s := &FirmwareService{}
	fws := []*model.Firmware{
		{Version: "1.2.3"},
		{Version: ""},
		{Version: "1.2.10"},
	}
	s.sortFirmwares(fws)
	if fws[0].Version != "" {
		t.Fatalf("expected empty version first, got %q", fws[0].Version)
	}
	if fws[1].Version != "1.2.3" {
		t.Fatalf("expected 1.2.3 second, got %q", fws[1].Version)
	}
	if fws[2].Version != "1.2.10" {
		t.Fatalf("expected 1.2.10 last, got %q", fws[2].Version)
	}
}
