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

// ValidateVersionName 校验版本号作为文件名的安全性
// 检查版本号是否包含可能导致路径遍历的危险字符
func ValidateVersionName(version string) error {
	if version == "" {
		return fmt.Errorf("version name is required")
	}

	// 检查是否包含路径分隔符
	if strings.Contains(version, "/") || strings.Contains(version, "\\") {
		return fmt.Errorf("version name contains path separator")
	}

	// 检查是否以点号开头（隐藏文件或相对路径）
	if strings.HasPrefix(version, ".") {
		return fmt.Errorf("version name starts with dot")
	}

	// 检查是否包含 ".."
	if strings.Contains(version, "..") {
		return fmt.Errorf("version name contains path traversal sequence")
	}

	// 检查是否包含空格
	if strings.Contains(version, " ") {
		return fmt.Errorf("version name contains spaces")
	}

	// 检查版本号长度
	if len(version) > 100 {
		return fmt.Errorf("version name too long: %d characters", len(version))
	}

	return nil
}

// SafeFileName 生成安全的文件名
// 将版本号转换为可安全用作文件名的格式
func SafeFileName(version string) string {
	result := version

	// 移除或替换危险字符
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, "\\", "_")
	result = strings.ReplaceAll(result, "..", "_")
	result = strings.ReplaceAll(result, " ", "_")

	// 移除开头的点号
	result = strings.TrimPrefix(result, ".")

	if result == "" {
		result = "unnamed"
	}

	return result
}

// BuildFirmwarePath 构建固件文件的安全路径
func BuildFirmwarePath(modelID int64, version, uploadDir, ext string) (string, error) {
	// 先清理版本号
	safeVersion := SafeFileName(version)

	// 校验清理后的版本号
	if err := ValidateVersionName(safeVersion); err != nil {
		return "", fmt.Errorf("invalid version for filename: %w", err)
	}

	modelDir := filepath.Join(uploadDir, fmt.Sprintf("model_%d", modelID))
	filename := fmt.Sprintf("model_%d_v_%s%s", modelID, safeVersion, ext)
	fullPath := filepath.Join(modelDir, filename)

	// 验证最终路径在预期目录下
	absUploadDir, err := filepath.Abs(uploadDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path of upload dir: %w", err)
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path of firmware file: %w", err)
	}

	if !strings.HasPrefix(absFullPath, absUploadDir) {
		return "", fmt.Errorf("firmware path escapes upload directory")
	}

	return fullPath, nil
}
