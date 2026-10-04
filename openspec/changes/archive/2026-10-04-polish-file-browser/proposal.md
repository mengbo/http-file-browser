# Proposal

## Why

目录浏览视图当前的视觉呈现过于朴素——目录与文件均以纯文字加蓝色链接呈现，与 macOS Finder 的视觉对比（文件夹 / 文件图标一眼可辨）有明显距离。用户在浏览包含大量条目的目录时，肉眼扫描的成本随条目数线性上升。roadmap 已把整体视觉精修（Change 16 `polish-file-browser`）列为最终润色 Change，且在长期约束里锚定"UI 风格借鉴 macOS Finder"。现在做这件事，既兑现这条长期约束，也借机为后续可能的视觉迭代（工具栏化、卡片化）建立正确的 spec 边界——哪些是观察得到的行为变化（进 spec），哪些是 CSS 实现细节（只进 design）。

## What Changes

- `directory-browsing` 携带一条 ADDED Requirement：目录浏览视图的每个条目 SHALL 在名称前显示一个视觉图标，该图标与该条目的类型相对应——目录对应目录图标，文件对应文件图标。图标 SHALL 在浅色与深色模式下均可识别。
- 视觉精修层面的非 spec 级调整（不构成可观察行为变化）作为本次 Change 同步进行的 CSS 调优，落在 `design.md` 与 `tasks.md` 中：
  - **GitHub Primer 风**（apply 阶段经用户两轮实机评审定稿：初稿 macOS 系统蓝 → 极简中性 → 现 GitHub 风）：白/深画布（`#ffffff` / `#0d1117`）、链接用 GitHub 强调蓝（`#0969da` / `#4493f8`）、辅助信息用 GitHub 弱化灰（`#656d76` / `#8b949e`）、边框 `#d0d7de` / `#30363d`、hover 行底色 `#f6f8fa` / `#161b22`、统一 6px 圆角、focus 带浅蓝光圈
  - 行高、条目间距沿用，保留 grid 列宽与窄视口响应
  - 字体栈沿用系统栈并补 `"Segoe UI", "Noto Sans"`（GitHub 栈）
- 图标资源以 vendored SVG 形式内嵌进产物（与现有 `/vendor/highlight.min.js`、`/vendor/markdown-it.min.js` 同路径：`web/vendor/icons/*.svg`，登记进 `web/web.go` 的 embed FS）。
- 图标风格、颜色、形状不进 spec——属于 design 决策。

## Capabilities

### New Capabilities

（无——所有可观察行为变化落在对 `directory-browsing` 的 ADDED 之上，且不构成新 capability 的独立边界。）

### Modified Capabilities

- `directory-browsing`: 新增 `Added `Requirement: 类型化视觉图标`——为目录浏览视图条目新增「按类型显示视觉图标」的可观察承诺。其它既有 Requirement（`Directory listing response`、`Entry type distinction` 等）一字未改。

## Impact

- `web/web.go`：embed FS 多登记 `vendor/icons/` 子目录，预期两个 SVG（`dir.svg`、`file.svg`）。
- `web/index.html`：`<li class="entry">` 结构内新增视觉图标的渲染位置（与 `.entry-link` 平级）。
- `web/style.css`：图标尺寸、对齐、当前色（currentColor）随外观切换；配色（GitHub Primer）、背景、行高、圆角、hover 反馈的视觉调优。
- `web/app.js`：列表渲染时为每个条目追加视觉图标元素；图标资源路径解析与切换；不引入新的分派门。
- 服务端零变化——纯前端呈现层改动。
- 验收策略遵循 ADR-0003：spec 全部为呈现层 Scenario（agent-browser 走查），无 API 层测试新增。
- 约束：ADR-0001（单二进制）、ADR-0002（JSON API + 内嵌零构建前端，vendored SVG 是其内嵌路径的延伸）、ADR-0003（呈现层 Scenario 验收分派）。
- 记账（apply 阶段随手项）：roadmap #16 💡→🚧。