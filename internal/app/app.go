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

	"github.com/mengbo/http-file-browser/internal/server"
	"github.com/mengbo/http-file-browser/web"
)

const listenAddress = "127.0.0.1:8080"

type listenFunc func(network, address string) (net.Listener, error)

// Run 启动文件浏览服务：校验根目录参数、监听回环地址、报告访问地址并开始服务。
// 返回非 nil 错误时服务未进入运行状态，调用方应以非零状态退出。
func Run(args []string, stdout, stderr io.Writer) error {
	return run(args, stdout, stderr, web.FS, net.Listen)
}

func run(args []string, stdout, stderr io.Writer, assets fs.FS, listen listenFunc) error {
	root, err := rootDir(args)
	if err != nil {
		fmt.Fprintf(stderr, "错误：%v\n", err)
		return err
	}

	listener, err := listen("tcp", listenAddress)
	if err != nil {
		err = fmt.Errorf("监听地址 %s 不可用：%w", listenAddress, err)
		fmt.Fprintf(stderr, "错误：%v\n", err)
		return err
	}
	defer listener.Close()

	fmt.Fprintf(stdout, "服务已就绪，访问 http://%s 浏览 %s\n", listener.Addr(), root)

	return serve(listener, root, assets)
}

func serve(listener net.Listener, root string, assets fs.FS) error {
	return http.Serve(listener, server.NewHandler(root, assets))
}

func rootDir(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("缺少根目录参数，用法：http-file-browser <目录>")
	}
	if len(args) > 1 {
		return "", fmt.Errorf("只接受一个根目录参数，收到 %d 个", len(args))
	}

	root, err := filepath.Abs(args[0])
	if err != nil {
		return "", fmt.Errorf("根目录 %q 无法解析：%w", args[0], err)
	}

	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("根目录 %q 不存在", args[0])
		}
		return "", fmt.Errorf("根目录 %q 不可用：%w", args[0], err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("根目录 %q 不是目录", args[0])
	}

	return root, nil
}
