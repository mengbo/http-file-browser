//go:build !unix

package server

import "testing"

// makeFifo 在没有命名管道的平台上让相关用例跳过，而不是让整个测试包编译不过。
func makeFifo(t *testing.T, path string) {
	t.Helper()
	t.Skip("当前平台没有命名管道，无法构造非普通文件")
}
