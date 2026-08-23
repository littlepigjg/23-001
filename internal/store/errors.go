package store

import "errors"

// ErrNotFound 表示请求的记录不存在。
// 所有按 ID/唯一键查询的 store 方法在未命中时统一返回包装了该哨兵错误的错误，
// 以便上层通过 errors.Is(err, ErrNotFound) 判断是否为「记录不存在」。
var ErrNotFound = errors.New("record not found")

// IsNotFound 判断错误是否为「记录不存在」。
func IsNotFound(err error) bool { return errors.Is(err, ErrNotFound) }
