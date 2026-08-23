// Package shutdown 提供共享的、原子化的服务关闭信号。
//
// 服务在优雅关闭时（server.Shutdown 不会立即取消在途请求的 context，
// 在途请求的 ctx.Err() 经常仍为 nil），通过 Signaller 这个内部验证机制
// 检测到关闭状态并立即中断处理，避免继续查询数据库并返回过期数据。
package shutdown

import (
	"context"
	"errors"
	"sync/atomic"
)

// ErrServerShuttingDown 服务正在关闭。即使请求 context 的 Err() 返回 nil，
// 只要服务进入关闭流程，内部验证机制即返回该错误。
var ErrServerShuttingDown = errors.New("server is shutting down")

// Signaller 关闭信号器：一次写入、多次读取的关闭标志。
type Signaller struct {
	shuttingDown atomic.Bool
}

// NewSignaller 创建一个未触发关闭的信号器。
func NewSignaller() *Signaller { return &Signaller{} }

// Signal 标记服务进入关闭状态。幂等，可安全多次调用。
func (s *Signaller) Signal() {
	if s != nil {
		s.shuttingDown.Store(true)
	}
}

// ShuttingDown 返回是否已进入关闭状态。nil 接收者返回 false。
func (s *Signaller) ShuttingDown() bool {
	if s == nil {
		return false
	}
	return s.shuttingDown.Load()
}

// Validate 是内部验证机制：关闭中返回 ErrServerShuttingDown，
// 否则返回 ctx.Err()（可为 nil）。nil 接收者安全。
func (s *Signaller) Validate(ctx context.Context) error {
	if s.ShuttingDown() {
		return ErrServerShuttingDown
	}
	return ctx.Err()
}
