package app

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/mengbo/http-file-browser/web"
)

// syncBuffer 是带锁的 bytes.Buffer，供测试在服务 goroutine 写 stdout 的同时安全读取。
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// advertisedListener 是真实只在回环接受连接、但对外报告另一地址的 listener。
// 它让 run 以为自己在监听非回环地址而走启用认证的分支，测试却仍只在回环上
// 接受连接——沿用「测试不真正对外暴露」（design 风险条目）。
type advertisedListener struct {
	net.Listener
	addr net.Addr
}

func (l advertisedListener) Addr() net.Addr { return l.addr }

type stringAddr string

func (a stringAddr) Network() string { return "tcp" }
func (a stringAddr) String() string  { return string(a) }

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
		if address != defaultListenAddress {
			t.Errorf("监听地址 = %q，期望 %q", address, defaultListenAddress)
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
	if !strings.Contains(stderr.String(), defaultListenAddress) {
		t.Errorf("stderr = %q，期望说明监听地址 %s 不可用", stderr.String(), defaultListenAddress)
	}
}

// TestParseArgsListenAddress 覆盖 --listen 的四种情形：默认无参、显式回环、
// 显式非回环与非法值（task 1.1）。
func TestParseArgsListenAddress(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		listen   string
		loopback bool
		wantErr  bool
	}{
		{name: "默认无参", args: []string{"/tmp"}, listen: defaultListenAddress, loopback: true},
		{name: "显式回环 127.0.0.1", args: []string{"--listen", "127.0.0.1:9000", "/tmp"}, listen: "127.0.0.1:9000", loopback: true},
		{name: "显式回环 localhost", args: []string{"--listen", "localhost:9000", "/tmp"}, listen: "localhost:9000", loopback: true},
		{name: "显式回环 ::1", args: []string{"--listen", "[::1]:9000", "/tmp"}, listen: "[::1]:9000", loopback: true},
		{name: "显式非回环 0.0.0.0", args: []string{"--listen", "0.0.0.0:8080", "/tmp"}, listen: "0.0.0.0:8080", loopback: false},
		{name: "显式非回环 空主机", args: []string{"--listen", ":8080", "/tmp"}, listen: ":8080", loopback: false},
		{name: "显式非回环 任意主机名", args: []string{"--listen", "example.local:8080", "/tmp"}, listen: "example.local:8080", loopback: false},
		{name: "等号形式", args: []string{"/tmp", "--listen=0.0.0.0:9000"}, listen: "0.0.0.0:9000", loopback: false},
		{name: "缺端口", args: []string{"--listen", "127.0.0.1", "/tmp"}, wantErr: true},
		{name: "端口非数字", args: []string{"--listen", "127.0.0.1:abc", "/tmp"}, wantErr: true},
		{name: "端口越界", args: []string{"--listen", "127.0.0.1:70000", "/tmp"}, wantErr: true},
		{name: "listen 缺取值", args: []string{"/tmp", "--listen"}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := parseArgs(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseArgs(%q) = %+v，期望错误", tc.args, cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs(%q) 返回错误：%v", tc.args, err)
			}
			if cfg.listen != tc.listen {
				t.Errorf("listen = %q，期望 %q", cfg.listen, tc.listen)
			}
			if cfg.loopback != tc.loopback {
				t.Errorf("loopback = %v，期望 %v", cfg.loopback, tc.loopback)
			}
		})
	}
}

// TestRunUsesConfiguredListenAddress 断言 run 把解析出的 --listen 值交给监听 seam。
func TestRunUsesConfiguredListenAddress(t *testing.T) {
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

	// 用显式回环地址配合空资源：非回环分支要求登录页在场，另行在认证用例覆盖。
	run([]string{"--listen", "127.0.0.1:9000", root}, io.Discard, io.Discard, emptyAssets(), listen)

	if requested != "127.0.0.1:9000" {
		t.Errorf("请求的监听地址 = %q，期望 127.0.0.1:9000", requested)
	}
}

// TestNonLoopbackStartupReport 断言非回环启动额外报告远程访问与凭证（task 3.3）。
func TestNonLoopbackStartupReport(t *testing.T) {
	root := t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("准备回环监听失败：%v", err)
	}
	listener.Close()

	var stdout bytes.Buffer
	listen := func(string, string) (net.Listener, error) { return listener, nil }

	if err := run([]string{"--listen", "0.0.0.0:8080", root}, &stdout, io.Discard, web.FS, listen); err == nil {
		t.Error("listener 已关闭时 run 应返回错误")
	}

	report := stdout.String()
	if !strings.Contains(report, "已启用远程访问") {
		t.Errorf("非回环启动 stdout = %q，期望报告已启用远程访问", report)
	}
	if !strings.Contains(report, "凭证：") {
		t.Errorf("非回环启动 stdout = %q，期望给出凭证", report)
	}

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("解析 listener 地址失败：%v", err)
	}
	if !strings.Contains(report, "http://localhost:"+port+"/?token=") {
		t.Errorf("非回环启动 stdout = %q，期望包含同机 localhost 访问链接", report)
	}
	// 链接里的 token 必须与报告的凭证一致。
	token := credentialFromReport(report)
	if token == "" {
		t.Fatalf("未能从 stdout 解析出凭证：%q", report)
	}
	if !strings.Contains(report, "?token="+token) {
		t.Errorf("stdout = %q，期望访问链接携带凭证 %q", report, token)
	}
}

// TestLoopbackStartupReportHasNoCredential 断言回环启动不报告远程访问、不生成凭证。
func TestLoopbackStartupReportHasNoCredential(t *testing.T) {
	root := t.TempDir()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("准备回环监听失败：%v", err)
	}
	listener.Close()

	var stdout bytes.Buffer
	listen := func(string, string) (net.Listener, error) { return listener, nil }

	if err := run([]string{root}, &stdout, io.Discard, emptyAssets(), listen); err == nil {
		t.Error("listener 已关闭时 run 应返回错误")
	}

	report := stdout.String()
	if !strings.Contains(report, "服务已就绪") {
		t.Errorf("回环启动 stdout = %q，期望与现有报告一致", report)
	}
	if strings.Contains(report, "已启用远程访问") || strings.Contains(report, "凭证：") || strings.Contains(report, "?token=") {
		t.Errorf("回环启动 stdout = %q，不应出现远程访问、凭证或 token", report)
	}
}

// credentialFromReport 从启动输出中取出「凭证：<token>」行的 token。
func credentialFromReport(report string) string {
	const label = "凭证："
	idx := strings.Index(report, label)
	if idx < 0 {
		return ""
	}
	rest := report[idx+len(label):]
	if end := strings.IndexByte(rest, '\n'); end >= 0 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

// TestNonLoopbackAuthenticationIsEnabled 端到端断言：非回环启动后 handler 处于
// 认证启用状态——未认证得到 401、页面得到登录入口、携带有效凭证得到资源（task 3.4）。
// 真实连接只落在回环上（advertisedListener），不真正对外暴露。
func TestNonLoopbackAuthenticationIsEnabled(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatalf("准备文件失败：%v", err)
	}

	real, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("准备回环监听失败：%v", err)
	}
	_, port, err := net.SplitHostPort(real.Addr().String())
	if err != nil {
		t.Fatalf("解析 listener 地址失败：%v", err)
	}

	var stdout syncBuffer
	listen := func(string, string) (net.Listener, error) {
		return advertisedListener{Listener: real, addr: stringAddr("0.0.0.0:" + port)}, nil
	}
	done := make(chan error, 1)
	go func() {
		done <- run([]string{"--listen", "0.0.0.0:8080", root}, &stdout, io.Discard, web.FS, listen)
	}()
	defer func() {
		real.Close()
		<-done
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	base := "http://127.0.0.1:" + port

	get := func(path string, cookie *http.Cookie) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, base+path, nil)
		if err != nil {
			t.Fatalf("构造请求失败：%v", err)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("请求 %s 失败：%v", path, err)
		}
		return resp
	}
	waitReady := func() {
		t.Helper()
		for i := 0; i < 200; i++ {
			resp, err := client.Get(base + "/api/health")
			if err == nil {
				resp.Body.Close()
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("服务未在预期时间内就绪")
	}
	waitReady()

	// 未认证的 /api/ 请求：401 JSON 错误信封，标识 unauthorized。
	unauth := get("/api/health", nil)
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("未认证 /api/health 状态码 = %d，期望 401", unauth.StatusCode)
	}

	// 未认证的页面请求：401 + 登录入口，而非页面内容。
	page := get("/", nil)
	defer page.Body.Close()
	if page.StatusCode != http.StatusUnauthorized {
		t.Fatalf("未认证 / 状态码 = %d，期望 401", page.StatusCode)
	}

	// 携带有效凭证的请求：放行。
	token := credentialFromReport(stdout.String())
	if token == "" {
		t.Fatalf("未能从启动输出解析出凭证：%q", stdout.String())
	}
	authed := get("/api/health", &http.Cookie{Name: "access_token", Value: token})
	defer authed.Body.Close()
	if authed.StatusCode != http.StatusOK {
		t.Fatalf("携带有效凭证的 /api/health 状态码 = %d，期望 200", authed.StatusCode)
	}
}

// TestLoopbackAuthenticationIsDisabled 端到端断言：默认回环启动下无凭证亦可访问，
// 且不生成凭证、行为与现状一致（task 3.4）。
func TestLoopbackAuthenticationIsDisabled(t *testing.T) {
	root := t.TempDir()

	real, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("准备回环监听失败：%v", err)
	}
	_, port, err := net.SplitHostPort(real.Addr().String())
	if err != nil {
		t.Fatalf("解析 listener 地址失败：%v", err)
	}

	var stdout syncBuffer
	listen := func(string, string) (net.Listener, error) { return real, nil }
	done := make(chan error, 1)
	go func() {
		done <- run([]string{root}, &stdout, io.Discard, emptyAssets(), listen)
	}()
	defer func() {
		real.Close()
		<-done
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	base := "http://127.0.0.1:" + port
	var resp *http.Response
	for i := 0; i < 200; i++ {
		resp, err = client.Get(base + "/api/health")
		if err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("服务未在预期时间内就绪：%v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("回环下未携带凭证的 /api/health 状态码 = %d，期望 200", resp.StatusCode)
	}
	if strings.Contains(stdout.String(), "凭证：") || strings.Contains(stdout.String(), "?token=") {
		t.Errorf("回环启动不应生成凭证：%q", stdout.String())
	}
}
