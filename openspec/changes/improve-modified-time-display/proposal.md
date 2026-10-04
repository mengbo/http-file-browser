# Proposal

## Why

目录列表的修改时间目前由 `Date.prototype.toLocaleString()` 渲染，输出随浏览器/系统的 locale 变化：在 en-US 环境下显示为 `10/4/2026, 11:35:25 AM`。这带来两个问题——一是同一个中文界面里混入 US 风格的日期，观感不一致；二是 `10/4` 这类写法存在月/日歧义。需要改成固定、无歧义、与 locale 无关的呈现格式。

## What Changes

- 目录列表条目的修改时间改为固定格式 `YYYY-MM-DD HH:mm:ss`，不再随 locale 变化。
- 仅固定排版，时区语义不变：仍按运行环境本地时区展开（与既有「取值跨时区可复现、显示字符串本地化」的区分保持一致，只是显示字符串从随 locale 变为固定排版）。
- 该呈现行为纳入 spec 承诺（新增 Requirement），不再停留在「spec 不约束、design 定方向」的状态。
- 显式推翻归档 `2026-10-03-add-file-metadata` design 中「修改时间渲染为本地时区的可读形式」这一呈现决定；旧记录作为历史不修改，由本 Change 的 design 说明来由。
- 如固定格式在窄视口溢出，则一并调整目录表的修改时间列宽（宽屏 10rem / 窄屏 8.25rem）。

无 **BREAKING**：不改变 `/api/list` 响应形状与取值，仅改前端呈现。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `directory-browsing`：新增 Requirement `修改时间呈现格式`——系统 SHALL 以固定 `YYYY-MM-DD HH:mm:ss` 呈现目录列表条目的最后修改时间，且该呈现不随运行环境的语言变化。

## Impact

- 代码：`web/app.js` 的 `formatModifiedAt()`（唯一调用点为目录列表条目渲染）；如需，`web/style.css` 的 `.entry` / `.entries-header` 列宽与窄视口断点。
- Spec：`openspec/specs/directory-browsing/spec.md`（archive 时合入）。
- 后端、API 形状、依赖：均无变化。
- 验收：纯呈现层行为，按 [ADR-0003](../../../docs/adr/0003-presentation-acceptance.md) 走浏览器走查，不新增 Go 测试。
