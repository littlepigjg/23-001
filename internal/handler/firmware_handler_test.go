package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

// newTestFirmwareHandler 构建一个基于内存存储的 FirmwareHandler，并预置一个有效固件，
// 返回 handler、有效固件 ID 与底层 store 便于构造请求。
func newTestFirmwareHandler(t *testing.T) (*FirmwareHandler, model.ID) {
	t.Helper()
	cfg := config.DefaultConfig()
	ms := store.NewMemoryStore()
	fwSvc := service.NewFirmwareService(ms, ms, cfg)
	h := NewFirmwareHandler(fwSvc, cfg)

	m := model.NewDeviceModel("TestModel", "TestCorp", "1.0", "desc")
	if err := ms.CreateModel(context.Background(), m); err != nil {
		t.Fatalf("seed model: %v", err)
	}

	fw := model.NewFirmware(m.ID, m.Name, "1.0.0", "d41d8cd98f00b204e980", 4, "/tmp/firmware_1.0.0.bin", time.Now(), "init")
	if err := ms.CreateFirmware(context.Background(), fw); err != nil {
		t.Fatalf("seed firmware: %v", err)
	}
	return h, fw.ID
}

// newRequest 构造一个带路径的请求。
func newRequest(t *testing.T, method, path string, body []byte) *http.Request {
	t.Helper()
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	r.URL.Path = path
	return r
}

func assertStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rr.Code != want {
		t.Fatalf("status: got %d, want %d (body=%s)", rr.Code, want, rr.Body.String())
	}
}

// TestGetFirmware_MissingReturns404 缺失固件 ID 的查询应返回 404，而非 200+空对象。
func TestGetFirmware_MissingReturns404(t *testing.T) {
	h, _ := newTestFirmwareHandler(t)
	rr := httptest.NewRecorder()
	h.Get(rr, newRequest(t, http.MethodGet, "/api/firmware/999999", nil))
	assertStatus(t, rr, http.StatusNotFound)

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["data"] != nil {
		t.Errorf("expected no data field for not-found, got %v", body["data"])
	}
}

// TestGetFirmware_ExistsReturns200 有效固件 ID 仍应返回 200（无回归）。
func TestGetFirmware_ExistsReturns200(t *testing.T) {
	h, id := newTestFirmwareHandler(t)
	rr := httptest.NewRecorder()
	h.Get(rr, newRequest(t, http.MethodGet, "/api/firmware/"+itoa(id), nil))
	assertStatus(t, rr, http.StatusOK)
}

// TestUpdateFirmware_MissingReturns404 更新缺失固件应返回 404（此前为 500）。
func TestUpdateFirmware_MissingReturns404(t *testing.T) {
	h, _ := newTestFirmwareHandler(t)
	body, _ := json.Marshal(model.UpdateFirmwareRequest{Changelog: "x"})
	rr := httptest.NewRecorder()
	h.Update(rr, newRequest(t, http.MethodPut, "/api/firmware/999999", body))
	assertStatus(t, rr, http.StatusNotFound)
}

// TestDeleteFirmware_MissingReturns404 删除缺失固件应返回 404（此前为 500）。
func TestDeleteFirmware_MissingReturns404(t *testing.T) {
	h, _ := newTestFirmwareHandler(t)
	rr := httptest.NewRecorder()
	h.Delete(rr, newRequest(t, http.MethodDelete, "/api/firmware/999999", nil))
	assertStatus(t, rr, http.StatusNotFound)
}

// TestGetDevice_MissingReturns404 镜像设备场景：查询缺失设备应返回 404。
func TestGetDevice_MissingReturns404(t *testing.T) {
	cfg := config.DefaultConfig()
	ms := store.NewMemoryStore()
	devSvc := service.NewDeviceService(ms, ms, cfg)
	h := NewDeviceHandler(devSvc, cfg)

	rr := httptest.NewRecorder()
	h.Get(rr, newRequest(t, http.MethodGet, "/api/devices/999999", nil))
	assertStatus(t, rr, http.StatusNotFound)
}

// TestUpdateDevice_MissingReturns404 更新缺失设备应返回 404（此前为 200+空或 500）。
func TestUpdateDevice_MissingReturns404(t *testing.T) {
	cfg := config.DefaultConfig()
	ms := store.NewMemoryStore()
	devSvc := service.NewDeviceService(ms, ms, cfg)
	h := NewDeviceHandler(devSvc, cfg)

	body, _ := json.Marshal(model.UpdateDeviceRequest{Name: "x"})
	rr := httptest.NewRecorder()
	h.Update(rr, newRequest(t, http.MethodPut, "/api/devices/999999", body))
	assertStatus(t, rr, http.StatusNotFound)
}

// TestDeleteDevice_MissingReturns404 删除缺失设备应返回 404（此前为 500）。
func TestDeleteDevice_MissingReturns404(t *testing.T) {
	cfg := config.DefaultConfig()
	ms := store.NewMemoryStore()
	devSvc := service.NewDeviceService(ms, ms, cfg)
	h := NewDeviceHandler(devSvc, cfg)

	rr := httptest.NewRecorder()
	h.Delete(rr, newRequest(t, http.MethodDelete, "/api/devices/999999", nil))
	assertStatus(t, rr, http.StatusNotFound)
}

// itoa 将 model.ID 转为字符串用于路径构造。
func itoa(id model.ID) string {
	return strconv.FormatInt(int64(id), 10)
}
