# Proposal

## Why

文本预览已可用（`text-preview`），但代码文件以素文本呈现，可读性差：结构（关键字、字符串、注释）无法一眼分辨。本项目进入 Phase 2（增强预览）的第一步，以语法高亮提升查看体验。

## What Changes

- 前端在文件视图（`#preview`）中以 vendored highlight.js 对文本内容做语法高亮呈现，不改变任何 API 行为。
- 引入 vendored 静态库 `web/vendor/`（highlight.js），随 `go:embed` 内嵌进单二进制，服务端按静态资源提供。
- 新增 capability `syntax-highlighting`，约束高亮的呈现行为、内容不变承诺与判定失败时的回退行为。
- 后端 Go 代码仅扩展静态资源清单（embed 指令加入新文件），`/api/` 端点零变化——`text-preview`、`directory-browsing`、`service-startup` 零 delta。

## Capabilities

### New Capabilities

- `syntax-highlighting`: 文本预览呈现层的高亮行为契约——什么情况下内容以高亮形式呈现、高亮永远不改变内容的字符序列、语言无法识别时回退为普通文本呈现。

### Modified Capabilities

（无——本 Change 不修改任何既有 capability 的 Requirement。）

## Impact

- `web/`：`index.html`、`app.js`、新增 `web/vendor/`（highlight.js 及样式）；`web/web.go` 的 embed 清单扩展。
- `internal/server/`：静态资源经由既有 `http.FileServer` 自动服务，预期零代码变化，仅测试补 vendored 资源可服务的断言。
- 交付物体积：highlight.js vendored 文件进二进制（预估数十至一二百 KB 量级，选库与语言集时在 design.md 权衡）。
- 验收策略遵循 ADR-0003：API 层 Scenario 走 Go 测试，呈现层 Scenario 走浏览器验证。
