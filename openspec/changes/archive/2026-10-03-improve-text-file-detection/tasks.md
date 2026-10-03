# Tasks

## 1. 嗅探判定实现（internal/server）

- [x] 1.1 在 content.go 实现内容嗅探函数：binary data byte 判据（0x00–0x08、0x0B、0x0E–0x1A、0x1C–0x1F）+ 空字节奇偶对齐豁免（起始窗口内空字节全部只落偶数位、或全部只落奇数位的不计为二进制数据字节），窗口常量 4096（design D2/D3/D4）。Verify：表驱动单元测试覆盖纯 ASCII、含 0x09/0x1B、PNG 魔数、UTF-16LE/BE 无 BOM、奇偶混杂 NUL、空输入，`go test` 通过。
- [x] 1.2 判定入口分流：扩展名命中白名单直接通过；名称无扩展名（末个 `.` 不存在、位于名字首位、或其后为空，design D1）打开文件读起始 ≤4096 字节嗅探；其余拒绝。嗅探位于 `too_large` 之前（design D5），同步更新函数注释里「两者都判为非文本」的旧口径。Verify：既有白名单测试不回归，`go test` 通过。
- [x] 1.3 白名单补充 Go 工作区扩展名 `mod`、`sum`、`work`（design D6）。Verify：`go.mod`、`go.sum` 用例断言可预览，`go test` 通过。

## 2. Scenario 对应测试（internal/server/content_test.go）

按项目约定每个 Scenario 一个 API 层测试：

- [x] 2.1 改写既有「无扩展名」用例（Scenario: A file without an extension is requested）：Makefile 式文本 fixture → 200 内容。Verify：`go test` 通过。
- [x] 2.2 新增 Scenario: A file with a non-text extension contains text content：`photo.png` 装纯 ASCII → `not_text`。Verify：`go test` 通过。
- [x] 2.3 新增 Scenario: A file without an extension contains binary data bytes：无扩展名文件头为 PNG 魔数 → `not_text`。Verify：`go test` 通过。
- [x] 2.4 新增 Scenario: A file without an extension and without any content is requested：`.gitkeep` 式空文件 → 200。Verify：`go test` 通过。
- [x] 2.5 新增 Scenario: Binary data bytes appear only after the start of the content：4096 字节文本 + 其后单个 0x01 → 200。Verify：`go test` 通过。
- [x] 2.6 新增 Scenario: A UTF-16 text file without a byte order mark is requested：无 BOM UTF-16 文本 fixture → 200。Verify：`go test` 通过。
- [x] 2.7 新增 Scenario: NUL bytes appear at both even and odd positions：奇偶混杂 NUL 且无其他二进制字节 → `not_text`。Verify：`go test` 通过。

## 3. 回归与集成

- [x] 3.1 优先级与一致性回归：无扩展名稀疏大文件（> 1 MiB）→ `too_large`（design D5：判定先于超限）；带非文本扩展名超大文件 → `not_text` 不变；重复请求一致性用例通过。Verify：`go test` 通过。
- [x] 3.2 静态检查与全量构建：`go vet ./...`、`go build ./...`、全量 `go test ./...` 通过。Verify：三条命令零错误。
- [x] 3.3 端到端冒烟：启动服务，`/api/content?path=go.mod` 返回 200 与内容；请求一个二进制文件返回 `not_text` 错误信封；浏览器打开根目录确认列表渲染与「二进制文件」提示文案正常（前端零改动的验证）。Verify：curl 断言 + 浏览器目测。
