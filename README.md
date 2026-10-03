# http-file-browser

零配置的 HTTP 文件浏览器：指定一个目录启动本地服务，在浏览器中像 macOS Finder 一样浏览文件系统。

本项目同时是 OpenSpec SDD（Spec-Driven Development）工作流的学习实验项目：所有行为变化都经由 Explore → Propose → Apply → Verify → Archive 演进，`openspec/` 目录记录系统当前行为与完整演进史。

## 当前状态

✅ Change 01 `bootstrap-http-server` 已归档：命令行启动服务、返回内嵌前端页面、`/api/health` 前后端往返均可用，行为规范见 [openspec/specs/service-startup/](openspec/specs/service-startup/)。目录浏览尚未实现。整体进度见 [docs/roadmap.md](docs/roadmap.md)。

## 快速开始

需要 Go 1.27 或更高版本，除此之外无任何依赖（无 npm、无打包器）。

```bash
go build -o http-file-browser .
./http-file-browser <要浏览的目录>
```

启动成功时标准输出打印访问地址：

```
服务已就绪，访问 http://127.0.0.1:8080 浏览 /path/to/dir
```

在浏览器打开该地址即可看到前端页面显示「服务就绪」。

几点说明：

- 目录参数是必需的。缺失、路径不存在或路径不是目录时，程序向标准错误输出中文原因并以非零状态退出，不会启动服务。
- 服务默认只监听 `127.0.0.1:8080`，本机之外的主机无法连接。端口被占用时直接报错退出，不会自动顺延。
- 前端页面与静态资源内嵌在二进制中，产物可以单独拷走运行，不需要随附资源目录。

## 文档

| 位置 | 内容 |
|---|---|
| [docs/roadmap.md](docs/roadmap.md) | 长期规划与 Change 地图（活文档） |
| [docs/journal.md](docs/journal.md) | SDD 学习实验观察记录 |
| [docs/adr/](docs/adr/) | 架构决策记录（ADR） |
| [docs/OpenSpec_HTTP_File_Browser.md](docs/OpenSpec_HTTP_File_Browser.md) | OpenSpec 实战教材（静态参考） |
| [openspec/specs/](openspec/specs/) | 系统当前行为规范 |
| [openspec/changes/archive/](openspec/changes/archive/) | 系统演进史（Change 归档） |
