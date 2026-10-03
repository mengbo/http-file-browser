# Proposal

## Why

Markdown 文件（`.md`/`.markdown`）已在文本白名单内，可以查看，但呈现的是高亮源码——标题、列表、链接、代码块的结构不可见，可读性差。这类文件在文件浏览场景中高频出现（README、笔记、文档），是 Phase 2（增强预览）的下一站（roadmap Change 07）。

## What Changes

- 前端将被查看的 `.md`/`.markdown` 文件以 Markdown 渲染形式呈现，不再以语法高亮源码呈现。渲染为纯客户端行为：vendored markdown-it（内嵌 HTML 关闭），content API 零变化。
- 渲染视图中：
  - 文件内形似 HTML 的标记文本以字面形式呈现，不解释为可执行标记（不支持内嵌 HTML）；
  - 链接可激活——相对路径链接在应用内导航（复用现有浏览位置模型），外部链接在新标签页打开；
  - 图片按浏览器语义加载呈现（相对路径图片无法解析到可加载资源，呈现加载失败，不报错）；
  - 代码块接 vendored highlight.js：语言被识别则高亮，否则素文本。
- `syntax-highlighting` 携带 MODIFIED：以 Markdown 渲染形式呈现的文件不适用语法高亮呈现，其呈现由 `markdown-preview` 定义（对 `.md` 的整体让渡；后续源码视图若需要高亮，由 `markdown-preview` 自行定义，不回流本 capability）。
- 渲染所需前端资源 vendored 内嵌，单二进制自包含。

## Capabilities

### New Capabilities

- `markdown-preview`: Markdown 文件的呈现行为契约——什么文件以渲染形式呈现、渲染对内嵌 HTML 标记的字面处理、渲染失败时的素文本回退、渲染视图中链接的导航行为、图片的加载行为、代码块的高亮呈现，以及渲染资源自包含。

### Modified Capabilities

- `syntax-highlighting`: `Syntax highlighted presentation` 一条 Requirement 变更——被 Markdown 渲染呈现的文件不再适用语法高亮呈现（条件挂在呈现形式上而非文件种类上，为后续源码视图保留高亮的适用空间）。

## Impact

- `web/`：`app.js` 渲染管线扩展（Markdown 分支、链接重写、代码块高亮钩子）、`style.css` 渲染样式、新增 `web/vendor/`（markdown-it 单文件 UMD）；`web/web.go` 预期零变化（`vendor` 目录已在 embed 清单内）。
- `internal/server/`：零代码变化，仅补 vendored 资源可服务的测试断言。
- `/api/` 端点零变化——`text-preview`、`directory-browsing`、`service-startup` 零 delta。
- 交付物体积：markdown-it 单文件（约百 KB 量级）进二进制。
- 验收策略遵循 ADR-0003：API 层 Scenario 走 Go 测试，呈现层 Scenario 走浏览器验证（agent-browser）。
