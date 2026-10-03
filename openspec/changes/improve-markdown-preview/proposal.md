# Proposal

## Why

被查看的 Markdown 文件目前只能以渲染形式呈现，用户看不到 Markdown 源码本身。Change 07 立渲染呈现时，在 `syntax-highlighting` 的 MODIFIED 中刻意把豁免条件挂在「呈现形式」（被 Markdown 渲染呈现的文件）而非文件种类上，为源码视图留位——本 Change 兑现该留位（roadmap Change 08）。

## What Changes

- Markdown 文件的呈现形式可由用户切换：默认以渲染形式呈现，切换后当前查看以源码形式呈现。
- 切换不跨查看保留：离开后再次查看同一文件，重新以渲染形式呈现（无记忆语义，探索已定）。
- 渲染形式的既有行为（形似 HTML 字面呈现、相对链接导航、图片呈现、代码块高亮、渲染失败回退素文本）全部挂上「以渲染形式呈现时」条件。
- 源码形式的呈现形态不由 `markdown-preview` 规定：spec 沉默，由 `syntax-highlighting` 既有行为自然接管（`markdown` 语言别名命中 → 高亮源码）。`syntax-highlighting` 零 delta——这是「条件挂呈现形式」措辞决策的兑现时刻。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `markdown-preview`：MODIFIED「Markdown rendered presentation」——无条件 SHALL 改为「默认渲染 + 用户可切换」，渲染相关 Scenario 条件化；ADDED「呈现形式切换」Requirement——切换的可观察行为与无记忆语义。

## Impact

- `web/app.js`：`renderContent` 的 Markdown 分支引入呈现形式状态与切换控件逻辑，渲染管线 / 高亮管线按形式分岔（高亮管线即 Change 07 之前的既有路径）。
- `web/index.html`、`web/style.css`：切换控件与样式。
- 服务端零变化：`internal/server` 无改动，无新增 vendored 依赖，`/api/` 契约零 delta。
- 纯前端呈现层变化，Scenario 验收按 ADR-0003 分派给浏览器走查（agent-browser）。
