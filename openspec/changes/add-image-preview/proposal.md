# Proposal

## Why

图片文件（png/jpg 等）今天点击后被 `text-preview` 以扩展名不在文本白名单拒绝（`not_text`），界面只能给出「这是二进制文件」——图片是文件浏览场景中最高频的可预览类型之一，却完全看不了。这是 Phase 2（增强预览）的下一站（roadmap Change 09），也是路线图中排在文件编辑、远程访问之前最后一块纯只读的呈现能力。

## What Changes

- 新增图片内容端点：按扩展名白名单识别图片文件，成功响应返回该文件内容的**原始字节序列**（按扩展名对应的图片 Content-Type），供前端 `<img>` 加载。这是本项目第一个非 JSON 的成功响应。
- `service-startup` 携带一条 MODIFIED：`HTTP surface partitioning` 的「以 `/api/` 开头的请求返回 JSON 响应」为图片内容端点的成功响应开口（错误响应仍一律 JSON 信封，`JSON error responses` 零 delta）。
- 图片识别为纯扩展名判定：命中白名单即按名字认定、不读内容、不验魔数；不命中（含无扩展名）一律拒绝并返回新错误标识 `not_an_image`。这是 Change 04「按名字判定 + 不为带扩展名文件嗅探」取舍的同构重演，由浏览器解码兜底。
- **不设大小上限**：图片内容以流式发送，服务端内存与文件大小无关；`text-preview` 的 1 MiB 上限不翻案（其传输路径理由仍然成立）。
- 前端对被查看文件按扩展名分派呈现：图片扩展名走 `<img>`（装饰非门，与 Markdown 识别同形）；图片加载失败时呈现回退消息，不留静默破图。
- 路径级失败语义原样复用：`not_found` / `not_a_directory` / `not_a_regular_file` / `permission_denied` / `outside_root`。
- **不做 SVG**：它是文本与图像的边界物，拖着 text 白名单变更、呈现形式推广、直接打开时的脚本执行面三件事，记入想法池独立立 Change。

## Capabilities

### New Capabilities

- `image-preview`: 图片文件查看的行为契约——哪些文件被当作图片文件、图片内容端点的字节响应与 Content-Type、路径级与识别级失败的可读错误标识，以及文件视图中图片的呈现与加载失败回退。

### Modified Capabilities

- `service-startup`: `HTTP surface partitioning` 一条 Requirement 变更——`/api/` 区域在图片内容端点的成功响应上以图片字节序列返回，不再是「一律 JSON 响应体」；两个区域的划分、静态资源区域与 JSON 错误信封全部不变。

## Impact

- `internal/server/`：新增图片内容端点（识别表、Content-Type 映射、流式发送）；复用既有 `resolve`/`classify`/`writeError` 路径基础设施；新增 `not_an_image` 错误标识。
- `web/app.js`：呈现分派新增图片分支（扩展名装饰映射、`<img>` 构建与 onerror 回退）；`ERROR_TEXT` 增补 `not_an_image`。
- `web/index.html`、`web/style.css`：图片视图的挂载点与样式；不引入任何新 vendored 库。
- 其余 capability 零 delta：`text-preview`（图片扩展名不进文本白名单）、`directory-browsing`（条目一律是链接的既有决策不变）、`markdown-preview`、`syntax-highlighting`。
- 验收策略遵循 ADR-0003：API 层 Scenario 走 Go 测试，呈现层 Scenario 走浏览器验证（agent-browser）。
- 记账（apply 阶段随手项）：roadmap #09 💡→🚧；想法池新增「SVG 图片预览」条目。
