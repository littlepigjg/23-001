// Package fileutil 提供文件操作工具函数
package fileutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EnsureDir 确保目录存在，如果不存在则创建
func EnsureDir(dirPath string) error {
	return os.MkdirAll(dirPath, 0755)
}

// FileExists 检查文件是否存在
func FileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// DirExists 检查目录是否存在
func DirExists(dirPath string) bool {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// GetFileSize 获取文件大小（字节）
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// MoveFile 移动文件
func MoveFile(src, dst string) error {
	if err := EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	return os.Rename(src, dst)
}

// DeleteFile 删除文件
func DeleteFile(filePath string) error {
	if !FileExists(filePath) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}
	return os.Remove(filePath)
}

// DeleteDir 递归删除目录
func DeleteDir(dirPath string) error {
	return os.RemoveAll(dirPath)
}

// SaveFile 保存数据到文件
func SaveFile(filePath string, data []byte) error {
	if err := EnsureDir(filepath.Dir(filePath)); err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// ReadFile 读取文件内容
func ReadFile(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

// ListFiles 列出目录下的所有文件
func ListFiles(dirPath string) ([]string, error) {
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ListFilesInDir 列出指定目录下的文件（不递归）
func ListFilesInDir(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(dirPath, entry.Name()))
		}
	}
	return files, nil
}

// ListFilesByExt 列出指定扩展名的文件
func ListFilesByExt(dirPath, ext string) ([]string, error) {
	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), strings.ToLower(ext)) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// GetFileName 获取文件名（含扩展名）
func GetFileName(filePath string) string {
	return filepath.Base(filePath)
}

// GetFileNameWithoutExt 获取文件名（不含扩展名）
func GetFileNameWithoutExt(filePath string) string {
	ext := filepath.Ext(filePath)
	name := filepath.Base(filePath)
	return strings.TrimSuffix(name, ext)
}

// GetFileExt 获取文件扩展名
func GetFileExt(filePath string) string {
	return filepath.Ext(filePath)
}

// GetFileDir 获取文件所在目录
func GetFileDir(filePath string) string {
	return filepath.Dir(filePath)
}

// JoinPath 拼接路径
func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

// AbsPath 获取绝对路径
func AbsPath(path string) (string, error) {
	return filepath.Abs(path)
}

// IsImageFile 检查是否为图片文件
func IsImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg":
		return true
	default:
		return false
	}
}

// IsFirmwareFile 检查是否为固件文件
func IsFirmwareFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".bin", ".hex", ".img", ".firmware", ".fw", ".tar.gz", ".tgz":
		return true
	default:
		return false
	}
}

// FormatFileSize 格式化文件大小显示
func FormatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
