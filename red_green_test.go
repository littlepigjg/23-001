package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	s := store.NewMemoryStore()
	ps := service.NewProgressService(s, s, s)

	device := model.NewDevice("device-001", 1, "Model-A", "Device-1", "10.0.0.1", "SN001")
	device.Status = model.DeviceUpgrading
	device.UpgradeProgress = 0
	if err := s.CreateDevice(context.Background(), device); err != nil {
		t.Fatal(err)
	}

	const numGoroutines = 100
	progressValues := make([]int, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		progressValues[i] = i + 1
	}

	var wg sync.WaitGroup
	var errCount int32
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(progress int) {
			defer wg.Done()
			req := &model.ReportProgressRequest{
				DeviceID: "device-001",
				TaskID:   1,
				Progress: progress,
				Status:   string(model.UpgradeInProgress),
			}
			if err := ps.ReportProgress(context.Background(), req); err != nil {
				atomic.AddInt32(&errCount, 1)
			}
		}(progressValues[i])
	}
	wg.Wait()

	finalDevice, err := s.GetDeviceByDeviceID(context.Background(), "device-001")
	if err != nil {
		t.Fatal(err)
	}

	maxProgress := numGoroutines
	t.Logf("Final progress: %d, errCount: %d", finalDevice.UpgradeProgress, atomic.LoadInt32(&errCount))

	if finalDevice.UpgradeProgress < maxProgress {
		fmt.Printf("RED (红灯，缺陷未修复): progress regressed from %d to %d, data race detected\n", maxProgress, finalDevice.UpgradeProgress)
		t.FailNow()
	}

	fmt.Printf("GREEN (绿灯，缺陷已修复): progress is %d, all writes preserved\n", finalDevice.UpgradeProgress)
}

func TestProgressMonotonicity(t *testing.T) {
	s := store.NewMemoryStore()
	ps := service.NewProgressService(s, s, s)

	device := model.NewDevice("device-002", 1, "Model-A", "Device-2", "10.0.0.2", "SN002")
	device.Status = model.DeviceUpgrading
	device.UpgradeProgress = 0
	if err := s.CreateDevice(context.Background(), device); err != nil {
		t.Fatal(err)
	}

	const numGoroutines = 30
	var wg sync.WaitGroup
	reported := make([]int32, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			val := int32((idx + 1) * 10)
			req := &model.ReportProgressRequest{
				DeviceID: "device-002",
				TaskID:   2,
				Progress: int(val),
				Status:   string(model.UpgradeInProgress),
			}
			_ = ps.ReportProgress(context.Background(), req)
			reported[idx] = val
		}(i)
	}
	wg.Wait()

	finalDevice, err := s.GetDeviceByDeviceID(context.Background(), "device-002")
	if err != nil {
		t.Fatal(err)
	}

	maxVal := int32(0)
	for _, p := range reported {
		if p > maxVal {
			maxVal = p
		}
	}

	t.Logf("TestProgressMonotonicity: final=%d, maxReported=%d", finalDevice.UpgradeProgress, maxVal)

	if finalDevice.UpgradeProgress < int(maxVal) {
		fmt.Printf("RED (红灯，缺陷未修复): progress dropped from %d to %d due to non-atomic read-write race\n", maxVal, finalDevice.UpgradeProgress)
		t.FailNow()
	}

	fmt.Printf("GREEN (绿灯，缺陷已修复): progress is %d, monotonicity preserved\n", finalDevice.UpgradeProgress)
}

func TestConcurrentProgressQuery(t *testing.T) {
	s := store.NewMemoryStore()
	ps := service.NewProgressService(s, s, s)

	device := model.NewDevice("device-003", 1, "Model-A", "Device-3", "10.0.0.3", "SN003")
	device.Status = model.DeviceUpgrading
	device.UpgradeProgress = 0
	if err := s.CreateDevice(context.Background(), device); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	writeCount := 20
	readCount := 20

	for i := 0; i < writeCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := &model.ReportProgressRequest{
				DeviceID: "device-003",
				TaskID:   3,
				Progress: (idx + 1) * 5,
				Status:   string(model.UpgradeInProgress),
			}
			_ = ps.ReportProgress(context.Background(), req)
		}(i)
	}

	readResults := make([]int32, readCount)
	for i := 0; i < readCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			d, err := ps.GetDeviceProgress(context.Background(), "device-003")
			if err == nil && d != nil {
				readResults[idx] = int32(d.UpgradeProgress)
			}
		}(i)
	}

	wg.Wait()

	finalDevice, err := s.GetDeviceByDeviceID(context.Background(), "device-003")
	if err != nil {
		t.Fatal(err)
	}

	maxExpected := writeCount * 5
	t.Logf("TestConcurrentProgressQuery: final=%d, maxExpected=%d", finalDevice.UpgradeProgress, maxExpected)

	if finalDevice.UpgradeProgress < maxExpected {
		fmt.Printf("RED (红灯，缺陷未修复): concurrent read-write caused progress loss (%d -> %d)\n", maxExpected, finalDevice.UpgradeProgress)
		t.FailNow()
	}

	allValid := true
	for i, r := range readResults {
		if r < 0 || r > 100 {
			allValid = false
			t.Logf("  invalid read result[%d]=%d", i, r)
		}
	}

	if !allValid {
		fmt.Printf("RED (红灯，缺陷未修复): inconsistent progress values observed during concurrent read-write\n")
		t.FailNow()
	}

	fmt.Printf("GREEN (绿灯，缺陷已修复): progress is %d, all reads consistent\n", finalDevice.UpgradeProgress)
}