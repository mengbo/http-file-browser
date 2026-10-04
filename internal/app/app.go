package app

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mengbo/http-file-browser/internal/server"
	"github.com/mengbo/http-file-browser/web"
)

// defaultListenAddress 是未指定 --listen 时的监听地址：仅回环，默认行为与引入
// 远程访问前完全一致（design D3；service-startup: Loopback-only binding by default）。
const defaultListenAddress = "127.0.0.1:8080"

type listenFunc func(network, address string) (net.Listener, error)

// Run 启动文件浏览服务：校验根目录参数、按 --listen 监听地址、报告访问地址并开始服务。
// 返回非 nil 错误时服务未进入运行状态，调用方应以非零状态退出。
func Run(args []string, stdout, stderr io.Writer) error {
	return run(args, stdout, stderr, web.FS, net.Listen)
}

func run(args []string, stdout, stderr io.Writer, assets fs.FS, listen listenFunc) error {
	cfg, err := parseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "错误：%v\n", err)
		return err
	}

	root, err := rootDir(cfg.rootArg)
	if err != nil {
		fmt.Fprintf(stderr, "错误：%v\n", err)
		return err
	}

	// 仅当监听非回环地址时启用认证：生成一次性凭证并交给根 handler（design D4、D5）。
	// 回环启动完全不生成凭证，认证层因此保持关闭，现有行为零回归。
	var token string
	if !cfg.loopback {
		token, err = server.GenerateToken()
		if err != nil {
			err = fmt.Errorf("生成访问凭证失败：%w", err)
			fmt.Fprintf(stderr, "错误：%v\n", err)
			return err
		}
	}

	handler, err := server.NewHandler(root, assets, server.Auth{Token: token})
	if err != nil {
		fmt.Fprintf(stderr, "错误：%v\n", err)
		return err
	}

	listener, err := listen("tcp", cfg.listen)
	if err != nil {
		err = fmt.Errorf("监听地址 %s 不可用：%w", cfg.listen, err)
		fmt.Fprintf(stderr, "错误：%v\n", err)
		return err
	}
	defer listener.Close()

	fmt.Fprintf(stdout, "服务已就绪，访问 http://%s 浏览 %s\n", listener.Addr(), root)
	if !cfg.loopback {
		reportRemoteAccess(stdout, listener.Addr().String(), token)
	}

	return http.Serve(listener, handler)
}

// startupConfig 是启动层从命令行解析出的配置。
type startupConfig struct {
	rootArg  string
	listen   string
	loopback bool
}

// parseArgs 把命令行参数解析为启动配置：--listen 可前可后，<目录> 仍是唯一位置参数
// （design D3）。显式监听地址的非法值按启动错误返回，由调用方报错退出。
func parseArgs(args []string) (startupConfig, error) {
	cfg := startupConfig{listen: defaultListenAddress}
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--listen":
			if i+1 >= len(args) {
				return startupConfig{}, errors.New("--listen 缺少取值，用法：--listen <host:port>")
			}
			i++
			cfg.listen = args[i]
		case strings.HasPrefix(arg, "--listen="):
			cfg.listen = strings.TrimPrefix(arg, "--listen=")
		default:
			positional = append(positional, arg)
		}
	}

	if len(positional) == 0 {
		return startupConfig{}, errors.New("缺少根目录参数，用法：http-file-browser [--listen <host:port>] <目录>")
	}
	if len(positional) > 1 {
		return startupConfig{}, fmt.Errorf("只接受一个根目录参数，收到 %d 个", len(positional))
	}
	cfg.rootArg = positional[0]

	loopback, err := loopbackListen(cfg.listen)
	if err != nil {
		return startupConfig{}, err
	}
	cfg.loopback = loopback
	return cfg, nil
}

// loopbackListen 判定监听地址是否仅回环，并顺带校验地址本身的合法性（design D3）。
//
// 规则：空主机与 0.0.0.0/:: 表示监听所有网卡（非回环）；localhost 与回环 IP 为回环；
// 解析不出 IP 且非 localhost 的一律按非回环处理——安全默认：不认识就当远程。
// 缺端口、端口非 0-65535 数字等非法值返回错误。
func loopbackListen(address string) (bool, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return false, fmt.Errorf("监听地址 %q 非法：%w", address, err)
	}
	if port == "" {
		return false, fmt.Errorf("监听地址 %q 非法：缺少端口", address)
	}
	if n, err := strconv.Atoi(port); err != nil || n < 0 || n > 65535 {
		return false, fmt.Errorf("监听地址 %q 非法：端口必须是 0-65535 的数字", address)
	}

	switch {
	case host == "":
		return false, nil
	case host == "localhost":
		return true, nil
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false, nil
	}
	return ip.IsLoopback(), nil
}

// reportRemoteAccess 在非回环启动时追加远程访问提示（design D8；
// service-startup: Remote access is enabled）：说明已启用远程访问、给出本次运行
// 唯一的凭证，并提供一条同机可直接打开的链接。监听主机是 0.0.0.0/:: 时地址行
// 自身不可浏览，故用 localhost 另行给出。
func reportRemoteAccess(stdout io.Writer, addr, token string) {
	fmt.Fprintln(stdout, "已启用远程访问：同网段的其他设备可访问本服务，访问需提供下方凭证。")
	fmt.Fprintf(stdout, "凭证：%s\n", token)
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	fmt.Fprintf(stdout, "同机访问：http://localhost:%s/?token=%s\n", port, token)
}

func rootDir(arg string) (string, error) {
	root, err := filepath.Abs(arg)
	if err != nil {
		return "", fmt.Errorf("根目录 %q 无法解析：%w", arg, err)
	}

	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("根目录 %q 不存在", arg)
		}
		return "", fmt.Errorf("根目录 %q 不可用：%w", arg, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("根目录 %q 不是目录", arg)
	}

	return root, nil
}
