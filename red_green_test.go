package fwupgrade_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	binPath := os.Args[0]

	cmd := exec.Command(binPath, "-test.run", "^TestConcurrent$", "-test.v")
	cmd.Env = append(os.Environ(), "GOFLAGS=-race")

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 {
				t.Logf("RED (红灯，缺陷未修复) - 并发竞态导致测试进程崩溃, 退出码: %d", exitErr.ExitCode())
				t.Logf("输出摘要: %s", truncateStr(outputStr, 500))
				t.FailNow()
				return
			}
		} else {
			t.Logf("RED (红灯，缺陷未修复) - 执行错误: %v", err)
			t.Logf("输出: %s", outputStr)
			t.FailNow()
			return
		}
	}

	if strings.Contains(outputStr, "DATA RACE") {
		t.Logf("RED (红灯，缺陷未修复) - 检测到数据竞争 (DATA RACE)")
		t.Logf("输出摘要: %s", truncateStr(outputStr, 500))
		t.FailNow()
		return
	}

	if strings.Contains(outputStr, "concurrent map") || strings.Contains(outputStr, "fatal error") {
		t.Logf("RED (红灯，缺陷未修复) - 检测到并发map访问异常")
		t.Logf("输出摘要: %s", truncateStr(outputStr, 500))
		t.FailNow()
		return
	}

	t.Log("GREEN (绿灯，缺陷已修复) - 并发测试通过，无数据竞争")
}

func TestConcurrent(t *testing.T) {
	s := store.NewMemoryStore()
	ctx := context.Background()

	m := model.NewDeviceModel("ConcurrencyModel", "TestMfr", "v1.0", "concurrency test model")
	if err := s.CreateModel(ctx, m); err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	fw := model.NewFirmware(m.ID, m.Name, "1.0.0", "abc123", 2048, "/tmp/test.bin", time.Now(), "initial")
	if err := s.CreateFirmware(ctx, fw); err != nil {
		t.Fatalf("failed to create firmware: %v", err)
	}

	s.SetPanicGuard(func(tag string) bool {
		return false
	})

	snap := s.RawSnapshot()
	if snap == nil {
		t.Fatal("raw snapshot is nil")
	}

	iterations := 20
	done := make(chan bool, iterations*3)

	for i := 0; i < iterations; i++ {
		go func(idx int) {
			defer func() { done <- true }()
			f := model.NewFirmware(
				m.ID, m.Name,
				fmt.Sprintf("2.%d.%d", idx, idx),
				fmt.Sprintf("md5_%d", idx),
				1024,
				fmt.Sprintf("/tmp/fw_%d.bin", idx),
				time.Now(),
				fmt.Sprintf("firmware update %d", idx),
			)
			_ = s.CreateFirmware(ctx, f)
		}(i)

		go func(idx int) {
			defer func() { done <- true }()
			d := model.NewDevice(
				fmt.Sprintf("device-%03d", idx),
				m.ID, m.Name,
				fmt.Sprintf("Device %d", idx),
				fmt.Sprintf("192.168.1.%d", idx%254+1),
				fmt.Sprintf("SN-%03d", idx),
			)
			_ = s.CreateDevice(ctx, d)
		}(i)

		go func(idx int) {
			defer func() { done <- true }()
			task := model.NewUpgradeTask(
				fmt.Sprintf("Task-%d", idx),
				fmt.Sprintf("Concurrency test task %d", idx),
				m.ID, m.Name,
				fw.ID, fw.Version,
				model.TaskTypeFull,
				100,
				nil,
				"test-runner",
			)
			_ = s.CreateTask(ctx, task)
		}(i)
	}

	timeout := time.After(10 * time.Second)
	for i := 0; i < iterations*3; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("concurrent operations timed out")
		}
	}
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
