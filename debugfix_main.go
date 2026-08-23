//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	
	"fwupgrade/internal/config"
	"fwupgrade/internal/model"
	"fwupgrade/internal/service"
	"fwupgrade/internal/store"
)

func main() {
	tmpDir, _ := os.MkdirTemp("", "debug_test")
	defer os.RemoveAll(tmpDir)
	
	cfgPath1 := filepath.Join(tmpDir, "config1.conf")
	cfgPath2 := filepath.Join(tmpDir, "config2.conf")
	
	cfgContent1 := fmt.Sprintf(
		"storage.type=file\n"+
			"storage.data_dir=%s\n"+
			"storage.upload_dir=%s\n"+
			"firmware.max_file_size=104857600\n"+
			"firmware.require_md5=true\n",
		tmpDir, filepath.Join(tmpDir, "uploads1"),
	)
	cfgContent2 := fmt.Sprintf(
		"storage.type=memory\n"+
			"storage.data_dir=%s\n"+
			"storage.upload_dir=%s\n"+
			"firmware.max_file_size=52428800\n"+
			"firmware.require_md5=false\n",
		tmpDir, filepath.Join(tmpDir, "uploads2"),
	)
	
	os.WriteFile(cfgPath1, []byte(cfgContent1), 0644)
	os.WriteFile(cfgPath2, []byte(cfgContent2), 0644)
	
	cfg := config.DefaultConfig()
	memStore := store.NewMemoryStore()
	fileStore := store.NewFileStore(cfg)
	fwSvc := service.NewFirmwareService(fileStore, memStore, cfg)
	
	ctx := context.Background()
	modelID := model.ID(1)
	
	_ = memStore.CreateModel(ctx, &model.DeviceModel{
		ID: modelID, Name: "test", Manufacturer: "mfr",
		HardwareVer: "v1", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	_ = fileStore.CreateModel(ctx, &model.DeviceModel{
		ID: modelID, Name: "test-fs", Manufacturer: "mfr",
		HardwareVer: "v1", IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	_ = memStore.CreateFirmware(ctx, &model.Firmware{
		ID: model.ID(1), ModelID: modelID, ModelName: "test",
		Version: "1.0.0", Md5: "abcdef", Size: 1024,
		FilePath: filepath.Join(tmpDir, "test.bin"),
		ReleaseDate: time.Now(), IsActive: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	
	// Test ReloadConfig directly
	fmt.Println("=== Test ReloadConfig ===")
	err := cfg.ReloadConfig(cfgPath1)
	if err != nil {
		fmt.Printf("Reload error: %v\n", err)
	}
	ms, md5, ud := cfg.GetConfigSnapshot()
	fmt.Printf("After reload1: maxSize=%d, reqMD5=%v, uploadDir=%s\n", ms, md5, ud)
	
	err = cfg.ReloadConfig(cfgPath2)
	if err != nil {
		fmt.Printf("Reload error: %v\n", err)
	}
	ms, md5, ud = cfg.GetConfigSnapshot()
	fmt.Printf("After reload2: maxSize=%d, reqMD5=%v, uploadDir=%s\n", ms, md5, ud)
	
	// Test concurrent access
	fmt.Println("\n=== Test concurrent access ===")
	stop := make(chan struct{})
	var wg sync.WaitGroup
	
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				select {
				case <-stop:
					return
				default:
					if id%2 == 0 {
						cfg.ReloadConfig(cfgPath1)
					} else {
						cfg.ReloadConfig(cfgPath2)
					}
				}
			}
		}(i)
	}
	
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				select {
				case <-stop:
					return
				default:
					maxSize, reqMD5, uploadDir := fwSvc.GetConfigSnapshot()
					isCfg1 := (maxSize == 104857600 && reqMD5 == true && len(uploadDir) > 0 && uploadDir[len(uploadDir)-8:] == "uploads1")
					isCfg2 := (maxSize == 52428800 && reqMD5 == false && len(uploadDir) > 0 && uploadDir[len(uploadDir)-8:] == "uploads2")
					if !isCfg1 && !isCfg2 && uploadDir != "uploads" {
						fmt.Printf("INCONSISTENT: maxSize=%d, reqMD5=%v, uploadDir=%s\n", maxSize, reqMD5, uploadDir)
					}
				}
			}
		}()
	}
	
	time.Sleep(500 * time.Millisecond)
	close(stop)
	wg.Wait()
	
	ms, md5, ud = fwSvc.GetConfigSnapshot()
	fmt.Printf("Final: maxSize=%d, reqMD5=%v, uploadDir=%s\n", ms, md5, ud)
}
