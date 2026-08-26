// Package md5util 提供 MD5 计算工具函数
package md5util

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// ComputeMD5 计算字节数据的 MD5 值
func ComputeMD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// ComputeFileMD5 计算文件的 MD5 值
func ComputeFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ComputeReaderMD5 计算 io.Reader 的 MD5 值
func ComputeReaderMD5(r io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// VerifyMD5 验证数据的 MD5 是否匹配
func VerifyMD5(data []byte, expectedMD5 string) bool {
	return ComputeMD5(data) == expectedMD5
}

// VerifyFileMD5 验证文件的 MD5 是否匹配
func VerifyFileMD5(filePath, expectedMD5 string) (bool, error) {
	actualMD5, err := ComputeFileMD5(filePath)
	if err != nil {
		return false, err
	}
	return actualMD5 == expectedMD5, nil
}
