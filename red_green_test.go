package fwupgrade

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fwupgrade/internal/model"
	"fwupgrade/internal/store"
)

func TestRedGreen(t *testing.T) {
	s := store.NewMemoryStore()
	ctx := context.Background()
	baseTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	for i := 5; i >= 1; i-- {
		r := model.NewUpgradeRecord(
			fmt.Sprintf("device-%02d", i),
			fmt.Sprintf("Device %02d", i),
			model.ID(i),
			fmt.Sprintf("task-%02d", i),
			"v1.0.0",
			fmt.Sprintf("v2.0.%d", i),
		)
		r.StartedAt = baseTime.Add(-time.Duration(i) * time.Minute)
		r.Status = model.UpgradeSuccess
		if err := s.CreateRecord(ctx, r); err != nil {
			t.Fatalf("CreateRecord failed: %v", err)
		}
	}

	recent, err := s.GetRecentRecords(ctx, 3)
	if err != nil {
		t.Fatalf("GetRecentRecords failed: %v", err)
	}
	if len(recent) != 3 {
		t.Fatalf("Expected 3 recent records, got %d", len(recent))
	}

	originalID0 := recent[0].ID
	originalDev0 := recent[0].DeviceID
	originalStartedAt0 := recent[0].StartedAt

	_, err = s.ListRecordsByDevice(ctx, "device-03")
	if err != nil {
		t.Fatalf("ListRecordsByDevice failed: %v", err)
	}

	corrupted := false
	if recent[0].ID != originalID0 {
		corrupted = true
	}
	if recent[0].DeviceID != originalDev0 {
		corrupted = true
	}
	if !recent[0].StartedAt.Equal(originalStartedAt0) {
		corrupted = true
	}

	if corrupted {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Errorf("recent records data corrupted after subsequent store call: "+
			"recent[0].ID changed from %d to %d, DeviceID changed from %s to %s",
			originalID0, recent[0].ID, originalDev0, recent[0].DeviceID)
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}
