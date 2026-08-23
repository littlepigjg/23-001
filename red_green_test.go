package fwupgrade_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	s := store.NewMemoryStore()
	svc := service.NewStatsService(s, s, s, s, s)

	runtime.GC()
	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	iterations := 30
	for i := 0; i < iterations; i++ {
		_, _ = svc.GetDashboard(ctx)
	}

	time.Sleep(300 * time.Millisecond)
	runtime.GC()

	current := runtime.NumGoroutine()
	growth := current - baseline

	if growth > iterations/3 {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Errorf("goroutine leak detected: baseline=%d, current=%d, growth=%d", baseline, current, growth)
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}
