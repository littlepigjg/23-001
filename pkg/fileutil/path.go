package fileutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// BuildPath 构建文件路径
func BuildPath(parts ...string) string {
	return filepath.Join(parts...)
}

// NormalizePath 规范化路径
func NormalizePath(path string) string {
	return filepath.Clean(path)
}

// EnsureTrailingSlash 确保路径以斜杠结尾
func EnsureTrailingSlash(path string) string {
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		return path + string(filepath.Separator)
	}
	return path
}

// SplitExtension 拆分文件名和扩展名
func SplitExtension(filename string) (name, ext string) {
	ext = filepath.Ext(filename)
	name = strings.TrimSuffix(filename, ext)
	return name, ext
}

// IsAbsolutePath 判断是否为绝对路径
func IsAbsolutePath(path string) bool {
	return filepath.IsAbs(path)
}

// RelativePath 获取相对路径
func RelativePath(basePath, targetPath string) (string, error) {
	rel, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}
	return rel, nil
}

// MatchExtension 判断文件是否匹配扩展名
func MatchExtension(filename string, extensions []string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, e := range extensions {
		if strings.ToLower(e) == ext {
			return true
		}
	}
	return false
}
