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
	fmt.Println("=== Test 1: CalculateProgress with nil CompletedAt ===")
	t.Run("CalculateProgress nil pointer test", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("RED (defect exists): CalculateProgress panicked with: %v\n", r)
				t.Error("Defect verified: nil pointer dereference exists in calculateTimeBonus")
			}
		}()

		progress := task.CalculateProgress()
		fmt.Printf("CalculateProgress returned: %d (no panic)\n", progress)
	})

	// Step 7: Test GetTaskProgress which catches the panic but returns wrong data
	fmt.Println("\n=== Test 2: GetTaskProgress returns incorrect data ===")
	t.Run("GetTaskProgress returns incorrect data", func(t *testing.T) {
		resultTask, resultErr := tskSvc.GetTaskProgress(ctx, task.ID)

		fmt.Printf("GetTaskProgress result: progress=%d, status=%s, err=%v\n",
			resultTask.Progress, resultTask.Status, resultErr)

		if resultTask.Progress == 100 && resultTask.Status == model.TaskRunning {
			fmt.Println("RED (defect exists): Progress returns 100%, status returns running (should be 0% and pending)")
			t.Error("Defect verified: Incorrect data returned after panic recovery")
		} else if resultTask.Progress == 0 && resultTask.Status == model.TaskPending {
			fmt.Println("GREEN (defect fixed): Progress returns 0%, status returns pending")
		} else {
			fmt.Printf("Unexpected result: progress=%d, status=%s\n", resultTask.Progress, resultTask.Status)
		}
	})

	fmt.Println("\n========== Defect Verification Summary ==========")
	fmt.Println("bug_id: fwupgrade-nil-006")
	fmt.Println("bug_category: nil pointer dereference")
	fmt.Println("location: internal/model/task.go.calculateTimeBonus")
	fmt.Println("==================================================")
}
