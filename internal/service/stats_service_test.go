package service

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"fwupgrade/internal/store"
)

// TestGetDashboardNoGoroutineLeak verifies that concurrent (and prematurely
// cancelled) GetDashboard calls do not leak goroutines blocked on channel send.
//
// 回归测试：修复前 GetDashboard 用无缓冲 channel 发送两次结果，但主调用方
// 只读取一次即返回，第二次 ch <- partial2 永远无接收方 → 内部 goroutine
// 永久阻塞在 chan send 上而泄漏。并发压测下 goroutine 数持续增长。
// 修复后 channel 带缓冲，内部 goroutine 必然能写入并退出，无泄漏。
func TestGetDashboardNoGoroutineLeak(t *testing.T) {
	svc := NewStatsService(
		store.NewMemoryStore(),
		store.NewMemoryStore(),
		store.NewMemoryStore(),
		store.NewMemoryStore(),
		store.NewMemoryStore(),
	)

	// 预热：让一次正常调用跑完。
	if _, err := svc.GetDashboard(context.Background()); err != nil {
		t.Fatalf("initial GetDashboard failed: %v", err)
	}

	// 给上轮 goroutine 退出时间，记录基线。
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	baseline := runtime.NumGoroutine()

	const concurrency = 200
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			// 一半请求立刻取消 context（模拟请求超时/客户端断开），
			// 另一半正常完成。修复前这两种路径都会泄漏 goroutine。
			if i := i; i%2 == 0 {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				_, _ = svc.GetDashboard(ctx)
				return
			}
			_, _ = svc.GetDashboard(context.Background())
		}()
	}
	wg.Wait()

	// 等待所有派生 goroutine 退出。泄漏的 goroutine 永远不会退出，
	// 因此 NumGoroutine 会远超 baseline。
	time.Sleep(300 * time.Millisecond)
	runtime.GC()
	after := runtime.NumGoroutine()

	// 允许少量波动，但绝不应接近 concurrency 级别的增长。
	if after-baseline > concurrency/4 {
		t.Fatalf("goroutine leak detected: baseline=%d, after=%d (delta=%d)",
			baseline, after, after-baseline)
	}
	t.Logf("goroutine baseline=%d after=%d delta=%d", baseline, after, after-baseline)
}
