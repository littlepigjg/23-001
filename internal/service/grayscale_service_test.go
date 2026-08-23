package service

import (
	"fmt"
	"sort"
	"testing"

	"fwupgrade/internal/config"
)

// newTestGrayscaleService 构造一个用于测试的灰度服务。
// GenerateDeviceGroup / SelectSampleDevices 不依赖 store，传 nil 即可。
func newTestGrayscaleService() *GrayscaleService {
	return NewGrayscaleService(nil, config.DefaultConfig())
}

// stableIDs 构造 n 个形如 dev-0001 的设备 ID，保证顺序稳定、可读。
func stableIDs(n int) []string {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("dev-%04d", i+1)
	}
	return ids
}

// TestGenerateDeviceGroup_DoesNotMutateInput 回归测试：
// 修复前 grayGroup/waitGroup 复用 deviceIDs 底层数组，append 会覆盖
// deviceIDs 尚未读取的元素，导致设备 ID 丢失/重复。修复后入参必须保持不变。
//
// 使用 ratio=50 让 gray/wait 交错（dev-0001..dev-0010 在 fnv 下恰为
// gray/wait/wait/gray/wait/gray/gray/wait/wait/gray），这样 bug 版本里
// 两个组共享底层数组互相覆盖时，会把尚未读取的元素改写成别的值，
// 从而被"入参未破坏"断言可靠捕获。ratio=100 时所有元素同向、
// 覆盖值恰好等于原值，无法暴露污染。
func TestGenerateDeviceGroup_DoesNotMutateInput(t *testing.T) {
	s := newTestGrayscaleService()

	ids := stableIDs(10)
	idsCopy := append([]string(nil), ids...)

	gray, wait := s.GenerateDeviceGroup(ids, 50)

	if len(gray)+len(wait) != 10 {
		t.Fatalf("expected 10 devices split across groups, got gray=%d wait=%d", len(gray), len(wait))
	}
	if len(gray) == 0 || len(wait) == 0 {
		t.Fatalf("test requires a mixed split to expose aliasing; got gray=%d wait=%d", len(gray), len(wait))
	}

	// 关键断言：入参 slice 内容不被破坏。
	for i, want := range idsCopy {
		if ids[i] != want {
			t.Fatalf("input deviceIDs mutated at index %d: got %q want %q (this is the slice-aliasing bug)", i, ids[i], want)
		}
	}
}

// TestGenerateDeviceGroup_NoDuplicatesAcrossGroups 验证灰度组与等待组互斥、
// 合并后等于全集且无重复。修复前两个组共享底层数组会互相覆盖产生重复。
func TestGenerateDeviceGroup_NoDuplicatesAcrossGroups(t *testing.T) {
	s := newTestGrayscaleService()
	ids := stableIDs(50)

	gray, wait := s.GenerateDeviceGroup(ids, 50)

	seen := make(map[string]bool, len(ids))
	for _, id := range gray {
		if seen[id] {
			t.Fatalf("duplicate device %q appears more than once in gray group", id)
		}
		seen[id] = true
	}
	for _, id := range wait {
		if seen[id] {
			t.Fatalf("device %q appears in both gray and wait groups", id)
		}
		seen[id] = true
	}

	if len(seen) != len(ids) {
		t.Fatalf("expected %d unique devices across groups, got %d (devices were lost)", len(ids), len(seen))
	}
}

// TestGenerateDeviceGroup_IdempotentAcrossRuns 回归"多跑几次丢失越来越严重"：
// 修复前 lastGroups 缓存的是被污染的别名 slice，反复调用会持续覆盖底层数组。
// 修复后连续多次调用结果应稳定且每次入参都完整。
func TestGenerateDeviceGroup_IdempotentAcrossRuns(t *testing.T) {
	s := newTestGrayscaleService()
	idsCopy := stableIDs(10)

	var firstGray []string
	for run := 0; run < 5; run++ {
		// 每次都用一份全新的副本调用，避免上一轮残留影响。
		input := append([]string(nil), idsCopy...)
		gray, _ := s.GenerateDeviceGroup(input, 100)

		if len(gray) != 10 {
			t.Fatalf("run %d: expected 10 gray devices, got %d (%v)", run, len(gray), gray)
		}
		// 每次入参都应保持原样。
		for i, want := range idsCopy {
			if input[i] != want {
				t.Fatalf("run %d: input mutated at %d: got %q want %q", run, i, input[i], want)
			}
		}
		if run == 0 {
			firstGray = append([]string(nil), gray...)
		} else if !sameSet(firstGray, gray) {
			t.Fatalf("run %d: gray set drifted from first run (non-deterministic / cumulative corruption)", run)
		}
	}
}

// TestGenerateDeviceGroupWithGuard_DoesNotMutateInput 同样验证 WithGuard 变体。
func TestGenerateDeviceGroupWithGuard_DoesNotMutateInput(t *testing.T) {
	s := newTestGrayscaleService()
	ids := stableIDs(10)
	idsCopy := append([]string(nil), ids...)

	// guard 让偶数下标进入灰度组。
	guard := func(deviceID string, ratio float64) bool {
		idx := sort.Search(len(ids), func(i int) bool { return ids[i] >= deviceID })
		if idx < len(ids) && ids[idx] == deviceID {
			return idx%2 == 0
		}
		return false
	}
	gray, wait := s.GenerateDeviceGroupWithGuard(ids, 100, guard)

	// 入参未破坏。
	for i, want := range idsCopy {
		if ids[i] != want {
			t.Fatalf("input mutated at %d: got %q want %q", i, ids[i], want)
		}
	}
	// 两组互斥且并集为全集。
	seen := map[string]bool{}
	for _, id := range gray {
		if seen[id] {
			t.Fatalf("dup in gray: %q", id)
		}
		seen[id] = true
	}
	for _, id := range wait {
		if seen[id] {
			t.Fatalf("device %q in both groups", id)
		}
		seen[id] = true
	}
	if len(seen) != len(ids) {
		t.Fatalf("expected %d unique, got %d", len(ids), len(seen))
	}
}

// TestSelectSampleDevices_DoesNotMutateInput 回归 SelectSampleDevices 的同类别名缺陷。
func TestSelectSampleDevices_DoesNotMutateInput(t *testing.T) {
	s := newTestGrayscaleService()
	ids := stableIDs(10)
	idsCopy := append([]string(nil), ids...)

	samples := s.SelectSampleDevices(ids, 5)

	if len(samples) != 5 {
		t.Fatalf("expected 5 samples, got %d", len(samples))
	}
	// 样本必须都来自原始集合。
	valid := map[string]bool{}
	for _, id := range idsCopy {
		valid[id] = true
	}
	for _, id := range samples {
		if !valid[id] {
			t.Fatalf("sample %q is not in the original set (input was corrupted)", id)
		}
	}
	// 入参未破坏。
	for i, want := range idsCopy {
		if ids[i] != want {
			t.Fatalf("input mutated at %d: got %q want %q", i, ids[i], want)
		}
	}
	// 样本自身无重复。
	seen := map[string]bool{}
	for _, id := range samples {
		if seen[id] {
			t.Fatalf("duplicate sample %q", id)
		}
		seen[id] = true
	}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sa := append([]string(nil), a...)
	sb := append([]string(nil), b...)
	sort.Strings(sa)
	sort.Strings(sb)
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}
