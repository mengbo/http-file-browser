# Tasks

## 1. 项目骨架

- [x] 1.1 初始化 Go module（`go mod init`），确认仓库根出现 `go.mod` 且不引入任何第三方依赖（`go list -m all` 仅列出 module 自身）
- [x] 1.2 按 design.md D1 建立 `internal/app`、`internal/server`、`web/` 三个目录，确认 `go build ./...` 在无源码时报无错误

## 2. 启动流程与参数校验

- [x] 2.1 实现 `internal/app` 的 `run(args []string, stdout, stderr io.Writer) error`：解析根目录参数、校验路径存在且为目录、绑定 `127.0.0.1:8080`、成功时向 stdout 打印含协议主机端口的地址；任一环节失败时向 stderr 写明原因并返回错误（不进入服务状态）。验证：2.2 的测试覆盖四种参数情形
- [x] 2.2 写 `internal/app` 单测，覆盖 spec 中根目录参数的四个 Scenario：参数可用、参数缺失、路径不存在、路径不是目录；断言退出原因写入 stderr 且未开始监听。验证：`go test ./internal/app/` 全部通过
- [x] 2.3 写 `internal/app` 单测覆盖启动结果报告：断言成功时 stdout 含协议、主机、端口，且 listener 地址为回环地址；断言监听地址不可用时返回错误且未进入服务状态。验证：`go test ./internal/app/` 全部通过
- [x] 2.4 实现 `main.go`：仅调用 `app.Run` 并按其返回值决定 `os.Exit` 码，不含其他逻辑。验证：`go build` 产出单可执行文件；手工以缺失参数运行，进程退出码非零且 stderr 有中文说明

## 3. HTTP 表面与 JSON 错误

- [x] 3.1 实现 `internal/server` 的 API handler：`/api/health` 返回 `{"status":"ok"}`；`/api/` 下未匹配路径返回 JSON 错误信封（design.md D4）而非 mux 默认的 `text/plain` 404。验证：3.2 的测试断言状态码、`Content-Type` 为 JSON、body 可反序列化且含 `error` 字段
- [x] 3.2 写 `internal/server` 的 `httptest` 单测，覆盖健康端点与未知 API 端点两条 Scenario。验证：`go test ./internal/server/` 全部通过
- [x] 3.3 实现路由分区：根 mux 先注册 `/api/` 前缀分支，其余路径交给文件服务，保证 `/api/` 请求不会落到文件服务上。验证：3.4 的测试断言请求 `/api/nope` 得到 JSON 而非 HTML
- [x] 3.4 写 `internal/server` 的 `httptest` 单测，断言 `/api/` 前缀优先于文件服务生效。验证：`go test ./internal/server/` 全部通过

## 4. 内嵌前端与前后端往返

- [x] 4.1 实现 `web/index.html`、`web/app.js`、`web/style.css`：页面调用 `fetch('/api/health')` 并把结果渲染进 DOM。验证：4.3 的测试断言 index.html 引用了 app.js
- [x] 4.2 用 `//go:embed web` 内嵌前端资源，根路径经 `http.FileServer(http.FS(...))` 提供（design.md D2）。验证：4.3 的测试断言根路径返回 HTML、静态资源返回对应内容类型
- [x] 4.3 写 `internal/server` 的 `httptest` 单测，覆盖根路径、前端静态资源、内容类型三条 Scenario。验证：`go test ./internal/server/` 全部通过，且测试无需起真实服务器或写临时目录
- [x] 4.4 在 README「快速开始」补充本 Change 的实际用法与访问地址说明。验证：按 README 描述的命令行启动程序，浏览器打开打印出的地址能看到前端页面显示服务就绪

## 5. 端到端确认

- [x] 5.1 手工端到端验证：以有效目录启动程序，确认 stdout 打印的地址可在浏览器打开、页面显示服务就绪；从另一台主机尝试连接该端口应无法建立连接。验证：两条观察均成立
- [x] 5.2 运行 `go vet ./...` 与 `go test ./...`，确认无告警且全部测试通过；确认 `go build` 产物在未安装 Go 运行时假设下仍可执行（无外部资源目录依赖）。验证：两条命令均通过，且产物在仅有 web/ 被删除的干净环境下仍能返回内嵌页面
