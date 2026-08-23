package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func TestNilPointerDefect(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemoryStore()
	cfg := config.DefaultConfig()

	// Step 1: Create a device model
	m := model.NewDeviceModel("TestModel", "TestMfg", "v1.0", "Test device model")
	if err := memStore.CreateModel(ctx, m); err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Step 2: Create firmware for this model
	fw := model.NewFirmware(m.ID, m.Name, "1.0.0", "abcdef123456", 1024,
		"/tmp/test.bin", time.Now(), "Initial firmware")
	if err := memStore.CreateFirmware(ctx, fw); err != nil {
		t.Fatalf("Failed to create firmware: %v", err)
	}

	// Step 3: Create task service
	tskSvc := service.NewTaskService(memStore, memStore, memStore, memStore, memStore, cfg)

	// Step 4: Create a task (new task with CompletedAt == nil)
	req := &model.CreateTaskRequest{
		Name:           "Test Upgrade Task",
		Description:    "Testing progress calculation for new task",
		ModelID:        m.ID,
		FirmwareID:     fw.ID,
		TaskType:       model.TaskTypeFull,
		GrayscaleRatio: 100.0,
		TargetDevices:  nil,
		CreatedBy:      "test_user",
	}
	task, err := tskSvc.CreateTask(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Step 5: Verify task was created with CompletedAt == nil
	if task.CompletedAt != nil {
		t.Fatalf("New task should have nil CompletedAt, but got: %v", task.CompletedAt)
	}
	if task.Status != model.TaskPending {
		t.Fatalf("New task should have status pending, but got: %s", task.Status)
	}

	// Step 6: Test CalculateProgress on a new task
	t.Run("CalculateProgress nil pointer test", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("RED (defect exists): CalculateProgress panicked with: %v", r)
				t.Error("Defect verified: nil pointer dereference exists")
			}
		}()

		progress := task.CalculateProgress()
		t.Logf("CalculateProgress returned: %d (no panic)", progress)
	})

	// Step 7: Test GetTaskProgress
	t.Run("GetTaskProgress returns incorrect data", func(t *testing.T) {
		resultTask, resultErr := tskSvc.GetTaskProgress(ctx, task.ID)

		t.Logf("GetTaskProgress result: progress=%d, status=%s, err=%v",
			resultTask.Progress, resultTask.Status, resultErr)

		if resultTask.Progress == 100 && resultTask.Status == model.TaskRunning {
			t.Log("RED (defect exists): Progress returns 100%, status returns running (should be 0% and pending)")
			t.Error("Defect verified: Incorrect data returned after panic recovery")
		} else {
			t.Logf("Result: progress=%d, status=%s", resultTask.Progress, resultTask.Status)
		}
	})

	fmt.Println("Defect verification summary for bug_id: fwupgrade-nil-006")
}
