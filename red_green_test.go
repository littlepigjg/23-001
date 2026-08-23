package fwupgrade

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

func TestRedGreen(t *testing.T) {
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

	// Step 6: Query progress of the new task
	resultTask, resultErr := tskSvc.GetTaskProgress(ctx, task.ID)

	// Step 7: Evaluate the result - RED means defect is present
	if resultErr != nil {
		fmt.Println("RED (红灯，缺陷未修复) - GetTaskProgress returned error when it shouldn't")
		t.Fatalf("GetTaskProgress returned error: %v", resultErr)
	}

	if resultTask == nil {
		fmt.Println("RED (红灯，缺陷未修复) - GetTaskProgress returned nil task")
		t.Fatal("GetTaskProgress returned nil task")
	}

	var hasDefect bool
	var defectMsg string

	// For a brand new pending task with no records:
	// - Progress should be 0 (not 100)
	// - Status should remain TaskPending (not TaskRunning)

	if resultTask.Progress != 0 {
		hasDefect = true
		defectMsg = fmt.Sprintf("New task progress should be 0, but got %d (panic recovery returned incorrect data)", resultTask.Progress)
	}

	if resultTask.Status != model.TaskPending {
		hasDefect = true
		defectMsg += fmt.Sprintf(" | New task status should be pending, but got %s (panic recovery changed status)", resultTask.Status)
	}

	if hasDefect {
		fmt.Printf("RED (红灯，缺陷未修复) - %s\n", defectMsg)
		t.Fatalf("Defect detected: %s", defectMsg)
	}

	// Additional verification: CalculateProgress should not panic for nil CompletedAt
	calcResult := task.CalculateProgress()
	if calcResult != 0 {
		fmt.Printf("RED (红灯，缺陷未修复) - CalculateProgress returned %d for new task, expected 0\n", calcResult)
		t.Fatalf("CalculateProgress returned incorrect value: %d", calcResult)
	}

	fmt.Println("GREEN (绿灯，缺陷已修复) - New task progress query works correctly")
}
