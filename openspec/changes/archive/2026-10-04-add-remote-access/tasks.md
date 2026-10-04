# Tasks

## 1. 监听地址配置（默认行为不变）

- [x] 1.1 在 `internal/app` 解析 `--listen <host:port>`：默认 `127.0.0.1:8080`；实现回环判定（空主机与 `0.0.0.0`/`::` 为非回环，`localhost`/`127.0.0.0/8`/`::1` 为回环，解析不出 IP 亦按非回环处理），非法值按启动错误报错退出。验证：`internal/app/app_test.go` 新增用例覆盖默认无参仍为 `127.0.0.1:8080`、显式回环、显式非回环（`0.0.0.0:8080` 与 `:8080`）、非法值四种情形，`go test ./internal/app/` 通过。
- [x] 1.2 新建 `docs/adr/0004-default-read-only-explicit-opt-in.md`：确立「默认只读；开放远程访问与写入均须显式开启」为长期决策（复用 0000 模板）。验证：文件存在，含决策、背景与后果，`docs/roadmap.md` 架构决策表中登记为 accepted。

## 2. 认证核心（token 与认证门）

- [x] 2.1 新增 token 生成与比较逻辑（如 `internal/server/auth.go`）：`crypto/rand` 32 字节 + `base64.RawURLEncoding`，`crypto/subtle.ConstantTimeCompare` 比较。验证：单元测试断言生成结果非空且两次不同、正确 token 判定通过、错误/前缀 token 判定拒绝。
- [x] 2.2 新增机器可读错误标识 `unauthorized`（HTTP 401）并登记到状态码映射。验证：`server_test.go` 或在 2.3 的用例中断言 `unauthorized` 映射为 401 且走 JSON 信封。
- [x] 2.3 实现认证 middleware：启用时校验请求 Cookie，未携带或无效时对 `/api/` 前缀返回 401 JSON 信封（`unauthorized`）；未启用（回环）时完全放行。验证：handler 层用 `httptest` 构造启用/未启用两种配置，断言未认证 `/api/list` 得 401 + `unauthorized`、携带有效 Cookie 得正常响应、回环配置下无 Cookie 亦正常响应。
- [x] 2.4 把认证层接到根 handler 外层而不改动现有端点：调整 `server.NewHandler` 接收认证配置并包裹现有 mux。验证：`go test ./internal/server/` 全绿，既有端点测试无需修改即通过（证明端点零改动）。

## 3. 凭证交换、登录入口与启动输出

- [x] 3.1 实现 `?token=<token>` 交换：校验通过则 `Set-Cookie`（`HttpOnly`、`SameSite=Strict`、`Path=/`，不设 `Secure`/`Max-Age`）并 302 到去掉 `token`、保留其余查询参数的同一 URL；校验失败按未认证处理。验证：`httptest` 断言有效 token 得 302 且 `Set-Cookie` 属性正确、重定向 Location 不含 `token` 且保留 `path`、无效 token 不种 Cookie。
- [x] 3.2 新增自包含登录页 `web/login.html`（内联样式、无 `<link>`/`<script>` 外部引用，表单 `GET /` 字段 `name="token"`），并让 middleware 在未认证的页面请求上以 401 输出它。验证：测试断言未认证 `/` 返回 401 且响应体含 token 表单、不含对外部资源的引用。
- [x] 3.3 启动输出分支：非回环启动时在现有地址行外追加「已启用远程访问」提示、`凭证：<token>` 与同机 `http://localhost:<port>/?token=<token>` 链接；回环输出保持不变。验证：`internal/app/app_test.go` 新增用例断言非回环启动 stdout 含 token 与警告、回环启动 stdout 与既有断言一致。
- [x] 3.4 在 `internal/app.run` 按解析出的监听地址决定是否启用认证、生成 token 并传给 `server.NewHandler`。验证：启动层测试以注入的 `listen` seam 断言非回环时 handler 处于认证启用状态、回环时不生成 token 且行为与现状一致。

## 4. 前端凭据处理

- [x] 4.1 `web/app.js`：`ERROR_TEXT` 增加 `unauthorized` 文案；`fetchJSON` 收到 401 时跳回 `/`（由服务端给出登录页），不当作普通内容错误展示。验证：浏览器手工走查（见 5.1）中会话失效后能回到登录入口而非卡在空视图。

## 5. 端到端验证

- [x] 5.1 用 `testdata/` 做手工走查：默认 `go run . testdata` 回环启动行为与现状一致（无 token、无登录）；`go run . --listen 0.0.0.0:8080 testdata` 启动后从另一设备访问看到登录页，用启动输出中的 token 链接进入后可浏览目录、查看图片与搜索，错误 token 被拒。验证：走查各项结果符合预期并记入 `docs/journal.md`。
- [x] 5.2 全量校验：`go vet ./...`、`go test ./...`、`openspec validate add-remote-access` 三项全部通过。验证：三条命令退出码为 0。
