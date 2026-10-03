//go:build unix

package server

import (
	"syscall"
	"testing"
)

// makeFifo 在给定路径创建命名管道。
//
// 非普通文件只有命名管道、Socket、设备这几类可稳定构造，其中命名管道是唯一在
// 普通文件系统上一条命令就能造出来的。标准库没有跨平台的 API，因此把 syscall.Mkfifo
// 用 build tag 隔离在 unix 上——测试代码同样受 design 约束：这里若直接调 syscall，
// 整个测试包在 Windows 上就编译不过，而 ADR-0001 看重的正是交叉编译。
func makeFifo(t *testing.T, path string) {
	t.Helper()
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("无法创建命名管道，跳过：%v", err)
	}
}
