# Proposal

## Why

Change 04 刻意先发了「扩展名白名单」这一不完整的判定器：无扩展名或扩展名未被收录的文本文件（`Makefile`、`LICENSE`、`.gitignore`、`go.mod`——这个仓库自己就有的一批文件）一律被判为非文本、无法预览。这是当时已知的、最确定会发生的一次用户可见失败，Change 05 因此立项。嗅探方案已在 Change 04 的 design D7 定到可照抄的粒度，本次不重新调研。

## What Changes

- `text-preview` 的判定规则从单一扩展名白名单放宽为两级：**扩展名在白名单内的文件**沿用名字判定；**名称没有扩展名的文件**改为按内容起始窗口判定——内容不含二进制数据字节的视为可读文本。
- 无 BOM 的 UTF-16 文件（ASCII 内容时每个偶数位一个 NUL，会被二进制判据误杀）通过空字节奇偶对齐启发式一并救回，识别为可读文本。
- 白名单顺带补充 Go 工作区扩展名（`mod`、`sum`、`work`），使 `go.mod` / `go.sum` / `go.work` 可预览——清单属 design 细节（Change 04 D6 已授权其可变），零 spec 变化。
- 带扩展名但扩展名不在白名单内的文件（如 `logo.png`）**不嗅探**，维持拒绝——Change 04 已明确接受「文件名与内容不符的责任在文件系统」这一取舍，本次不推翻。
- 判定仍为服务端行为，嗅探占据既有判定槽位（存在性/类型检查之后、大小上限之前）。
- 编码处理不变（一律按 UTF-8 呈现，D9）：UTF-16 文件可被打开，内容以替换字符呈现；BOM 解码不进本 Change。
- 前端零改动；`directory-browsing`、`service-startup` 零 delta。

## Capabilities

### New Capabilities

（无。不产生新 capability。）

### Modified Capabilities

- `text-preview`: MODIFIED `Text file recognition`——判定规则放宽，完整重写该 Requirement（含全部 Scenario）。既有 Scenario「名称没有任何扩展名的文件 → 非文本」被推翻；`Content reading failures` 与 `Text file content response`、`Preview position representation` 不动。

## Impact

- `internal/server/content.go`：判定函数重构（扩展名判定 + 内容嗅探），新增嗅探常量（窗口、NUL 阈值）。
- `internal/server/content_test.go`：MODIFIED 的固有测试债——「无扩展名」用例反转，白名单覆盖率交叉检查跟进，新增约 5 个 Scenario 对应测试。
- 前端 `web/`：零改动（判定在服务端，`not_text` 文案已存在）。
- 无新依赖（WHATWG binary data byte 判据为手写约十二行循环，不引入 `net/http.DetectContentType` / libmagic / filetype / charset，理由见 Change 04 design D7）。
- 归档时注意事项（非本 Change 内动作）：`text-preview` Purpose 中「不包含文件类型识别能力」半句将变假，按 journal 观察 6 纪律届时请用户授权修改并记入 journal。
