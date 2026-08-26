package md5util

import (
	"crypto/md5"
	"encoding/hex"
)

// ComputeMD5String 计算字符串的MD5
func ComputeMD5String(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// ComputeMD5Hex 计算字节数据的MD5并返回hex
func ComputeMD5Hex(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// VerifyMD5Hex 验证MD5 hex字符串匹配
func VerifyMD5Hex(data []byte, expectedHex string) bool {
	actual := ComputeMD5Hex(data)
	return actual == expectedHex
}
