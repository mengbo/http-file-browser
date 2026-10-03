# http-file-browser

零配置的 HTTP 文件浏览器：指定一个目录启动本地服务，在浏览器中像 macOS Finder 一样浏览文件系统。

本项目同时是 OpenSpec SDD（Spec-Driven Development）工作流的学习实验项目：所有行为变化都经由 Explore → Propose → Apply → Verify → Archive 演进，`openspec/` 目录记录系统当前行为与完整演进史。

## 当前状态

✅ Change 02 `directory-browsing` 已归档：浏览器中可按 Finder 风格浏览命令行指定的根目录——进入子目录、返回上级、看到当前位置，浏览位置镜像到地址栏（刷新停留在原目录、前进/后退可用、当前目录可作深链接）。列表顺序为「目录在前 + 名称不区分大小写 + 原名 tiebreak」。服务不提供根目录之外的内容。行为规范见 [openspec/specs/directory-browsing/](openspec/specs/directory-browsing/) 与 [openspec/specs/service-startup/](openspec/specs/service-startup/)。

✅ Change 03 已归档：列表的每个条目除名称与类型外还给出大小与最后修改时间，三列对齐显示，大小为人类可读形式、时间为本地时区形式。符号链接的大小是链接自身的长度而非目标大小（与服务只做字面越界判定的策略一致）；目录不给出大小；单个条目元信息取不到时该行保留名称、其余留空。列表不提供文件内容。

尚未实现：文件内容预览、编辑、远程访问。整体进度见 [docs/roadmap.md](docs/roadmap.md)。

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

在浏览器打开该地址即可看到该目录的条目列表。

几点说明：

- 目录参数是必需的。缺失、路径不存在或路径不是目录时，程序向标准错误输出中文原因并以非零状态退出，不会启动服务。
- 服务默认只监听 `127.0.0.1:8080`，本机之外的主机无法连接。端口被占用时直接报错退出，不会自动顺延。
- 前端页面与静态资源内嵌在二进制中，产物可以单独拷走运行，不需要随附资源目录。
- 当前只能浏览目录。文件条目会列出但不可点击，内容读取与预览属于后续 Change。
- 浏览位置用相对根目录的路径表示（形如 `/?path=docs/api`），因此深链接可以跨机器复用：同一个 `?path=docs` 在以另一个根目录启动的服务上照样能打开。


## 文档

| 位置 | 内容 |
|---|---|
| [docs/roadmap.md](docs/roadmap.md) | 长期规划与 Change 地图（活文档） |
| [docs/journal.md](docs/journal.md) | SDD 学习实验观察记录 |
| [docs/adr/](docs/adr/) | 架构决策记录（ADR） |
| [docs/OpenSpec_HTTP_File_Browser.md](docs/OpenSpec_HTTP_File_Browser.md) | OpenSpec 实战教材（静态参考） |
| [openspec/specs/](openspec/specs/) | 系统当前行为规范 |
| [openspec/changes/archive/](openspec/changes/archive/) | 系统演进史（Change 归档） |
