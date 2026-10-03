package app

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func unusedListen(t *testing.T) listenFunc {
	t.Helper()
	return func(string, string) (net.Listener, error) {
		t.Error("在参数校验失败时不应开始监听")
		return nil, errors.New("不应被调用")
	}
}

func emptyAssets() fs.FS {
	return fstest.MapFS{}
}

func TestRootDirArgumentIsProvidedAndUsable(t *testing.T) {
	root := t.TempDir()

	// listener 在交给 run 之前已关闭，使 run 的服务循环立即返回而不真正对外服务。
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("准备回环监听失败：%v", err)
	}
	listener.Close()

	var stdout, stderr bytes.Buffer
	listen := func(network, address string) (net.Listener, error) {
		if address != listenAddress {
			t.Errorf("监听地址 = %q，期望 %q", address, listenAddress)
		}
		return listener, nil
	}

	if err := run([]string{root}, &stdout, &stderr, emptyAssets(), listen); err == nil {
		t.Error("listener 已关闭时 run 应返回错误")
	}

	if stderr.Len() != 0 {
		t.Errorf("stderr = %q，期望为空", stderr.String())
	}
	if !strings.Contains(stdout.String(), "服务已就绪") {
		t.Errorf("stdout = %q，期望报告服务已就绪", stdout.String())
	}
	if !strings.Contains(stdout.String(), root) {
		t.Errorf("stdout = %q，期望包含根目录 %q", stdout.String(), root)
	}
}

func TestRootDirArgumentIsMissing(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if err := run(nil, &stdout, &stderr, emptyAssets(), unusedListen(t)); err == nil {
		t.Error("缺少根目录参数时应返回错误")
	}

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q，期望为空", stdout.String())
	}
	if !strings.Contains(stderr.String(), "缺少根目录参数") {
		t.Errorf("stderr = %q，期望说明缺少根目录参数", stderr.String())
	}
}

func TestRootDirArgumentDoesNotExist(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "不存在")

	var stdout, stderr bytes.Buffer

	if err := run([]string{missing}, &stdout, &stderr, emptyAssets(), unusedListen(t)); err == nil {
		t.Error("根目录不存在时应返回错误")
	}

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q，期望为空", stdout.String())
	}
	if !strings.Contains(stderr.String(), "不存在") {
		t.Errorf("stderr = %q，期望说明路径不存在", stderr.String())
	}
}

func TestRootDirArgumentIsNotADirectory(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("准备普通文件失败：%v", err)
	}

	var stdout, stderr bytes.Buffer

	if err := run([]string{file}, &stdout, &stderr, emptyAssets(), unusedListen(t)); err == nil {
		t.Error("根目录不是目录时应返回错误")
	}

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q，期望为空", stdout.String())
	}
	if !strings.Contains(stderr.String(), "不是目录") {
		t.Errorf("stderr = %q，期望说明该路径不是目录", stderr.String())
	}
}

func TestStartupReportContainsSchemeHostAndPort(t *testing.T) {
	root := t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("准备回环监听失败：%v", err)
	}
	listener.Close()

	var stdout, stderr bytes.Buffer
	listen := func(string, string) (net.Listener, error) { return listener, nil }

	if err := run([]string{root}, &stdout, &stderr, emptyAssets(), listen); err == nil {
		t.Error("listener 已关闭时 run 应返回错误")
	}

	report := stdout.String()
	if !strings.Contains(report, "http://") {
		t.Errorf("stdout = %q，期望包含协议", report)
	}
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("解析 listener 地址失败：%v", err)
	}
	if !strings.Contains(report, host) {
		t.Errorf("stdout = %q，期望包含主机 %q", report, host)
	}
	if !strings.Contains(report, port) {
		t.Errorf("stdout = %q，期望包含端口 %q", report, port)
	}
}

// TestDefaultListenerIsLoopbackOnly 断言 run 请求的监听地址是回环地址。
// 这里刻意不真的绑定端口：绑定会在开发者本地跑着程序时直接失败，
// 与 design.md D7「测试不绑定真实端口」冲突。真实 socket 的回环属性
// 由端到端观察（lsof 显示 127.0.0.1:8080 LISTEN）承担。
func TestDefaultListenerIsLoopbackOnly(t *testing.T) {
	root := t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("准备回环监听失败：%v", err)
	}
	listener.Close()

	var requested string
	listen := func(network, address string) (net.Listener, error) {
		requested = address
		return listener, nil
	}

	run([]string{root}, io.Discard, io.Discard, emptyAssets(), listen)

	host, port, err := net.SplitHostPort(requested)
	if err != nil {
		t.Fatalf("run 请求的监听地址 %q 无法解析：%v", requested, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		t.Fatalf("run 请求的监听主机 %q 不是 IP 地址", host)
	}
	if !ip.IsLoopback() {
		t.Errorf("run 请求的监听地址 %s 不是回环地址", requested)
	}
	if port != "8080" {
		t.Errorf("run 请求的端口 = %q，期望默认端口 8080", port)
	}
}

func TestListenAddressCannotBeBound(t *testing.T) {
	root := t.TempDir()

	var stdout, stderr bytes.Buffer
	listen := func(string, string) (net.Listener, error) {
		return nil, errors.New("address already in use")
	}

	if err := run([]string{root}, &stdout, &stderr, emptyAssets(), listen); err == nil {
		t.Error("监听地址不可用时应返回错误")
	}

	if stdout.Len() != 0 {
		t.Errorf("stdout = %q，期望为空（未进入服务状态）", stdout.String())
	}
	if !strings.Contains(stderr.String(), listenAddress) {
		t.Errorf("stderr = %q，期望说明监听地址 %s 不可用", stderr.String(), listenAddress)
	}
}
