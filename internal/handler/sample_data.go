package handler

import (
	"fmt"
	"net/http"

	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/pkg/logger"
)

// createSampleData 创建示例数据
func createSampleData(r *http.Request,
	modelSvc *service.DeviceModelService,
	deviceSvc *service.DeviceService,
	firmwareSvc *service.FirmwareService,
	taskSvc *service.TaskService,
	grayscaleSvc *service.GrayscaleService,
) {
	ctx := r.Context()

	// 创建设备型号
	model1, err := modelSvc.CreateModel(ctx, &model.CreateModelRequest{
		Name:         "SmartSensor-X1",
		Manufacturer: "TechCorp",
		HardwareVer:  "1.0",
		Description:  "智能传感器 X1 型号",
	})
	if err != nil {
		logger.Error("Failed to create model", "error", err)
		return
	}

	model2, err := modelSvc.CreateModel(ctx, &model.CreateModelRequest{
		Name:         "Gateway-G2",
		Manufacturer: "TechCorp",
		HardwareVer:  "2.0",
		Description:  "网关 G2 型号",
	})
	if err != nil {
		logger.Error("Failed to create model", "error", err)
		return
	}

	logger.Info("Created sample models", "count", 2)
	_ = model1
	_ = model2

	// 创建设备
	for i := 1; i <= 10; i++ {
		deviceSvc.CreateDevice(ctx, &model.CreateDeviceRequest{
			DeviceID:     fmt.Sprintf("sensor-%03d", i),
			ModelID:      model1.ID,
			Name:         fmt.Sprintf("传感器 #%03d", i),
			IPAddress:    fmt.Sprintf("192.168.1.%d", 100+i),
			SerialNumber: fmt.Sprintf("SN%010d", i),
		})
	}

	for i := 1; i <= 5; i++ {
		deviceSvc.CreateDevice(ctx, &model.CreateDeviceRequest{
			DeviceID:     fmt.Sprintf("gateway-%03d", i),
			ModelID:      model2.ID,
			Name:         fmt.Sprintf("网关 #%03d", i),
			IPAddress:    fmt.Sprintf("192.168.2.%d", 100+i),
			SerialNumber: fmt.Sprintf("GW%010d", i),
		})
	}

	logger.Info("Created sample devices", "count", 15)

	// 创建固件
	firmwareSvc.UploadFirmware(ctx, &model.UploadFirmwareRequest{
		ModelID: model1.ID,
		Version: "1.0.0",
		Md5:     "",
	}, []byte("sample firmware v1.0.0 for X1"), "firmware_v1.0.0.bin")

	firmwareSvc.UploadFirmware(ctx, &model.UploadFirmwareRequest{
		ModelID: model1.ID,
		Version: "1.1.0",
		Md5:     "",
	}, []byte("sample firmware v1.1.0 for X1 - new features"), "firmware_v1.1.0.bin")

	firmwareSvc.UploadFirmware(ctx, &model.UploadFirmwareRequest{
		ModelID: model2.ID,
		Version: "2.0.0",
		Md5:     "",
	}, []byte("sample firmware v2.0.0 for G2"), "firmware_v2.0.0.bin")

	logger.Info("Created sample firmwares", "count", 3)
	_ = grayscaleSvc
	_ = taskSvc
}
