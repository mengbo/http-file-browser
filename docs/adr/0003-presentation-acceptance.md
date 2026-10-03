# ADR-0003：呈现层行为的验收策略——Scenario 按行为性质分派断言位置

- 状态：accepted
- 日期：2026-10-03
- 关联：ADR-0002（收窄其测试约定）；Change 06 `add-syntax-highlighting`（首次适用）；Change 07/08/09/16（后续适用的同类 Change）

## Context

ADR-0002 确定「每个 Scenario 对应一个 API 层的 Go 测试（`net/httptest`）」。该约定在 Change 01–05 顺利运转，因为那些 Change 的用户可观察行为恰好全部落在 HTTP API 上。Change 06 引入语法高亮：行为发生在浏览器 DOM 的渲染结果里，Go 测试从原理上无法断言颜色。此后 Phase 2 的大部分 Change（Markdown 渲染、渲染/源码切换、图片预览、体验优化）同属纯呈现层，同一个问题会反复出现。

同时存在一个含糊地带需要澄清：ADR-0002 的「前端零构建，不引入 npm 与打包器」是否禁止引入任何测试工具。

## Decision

Scenario 照写用户可观察行为不变，验收按行为的性质分派断言位置：

1. **API 层行为**（请求/响应形状、错误标识、资源可服务性）→ `net/httptest` 的 Go 测试，沿用 ADR-0002 约定。
2. **呈现层行为**（DOM 渲染结果、视觉形态、前端分支）→ 浏览器验证：用浏览器自动化工具（本机已有 agent-browser）走查 Scenario，必要时断言 DOM 等价形式（如「预览元素的 `textContent` 等于 content API 返回的内容」），并在该 Change 的 tasks.md 验收清单中勾选记录。
3. 两条都覆盖的行为优先选择机器可断言的等价形式。

「零构建」约束的是**交付物**（`web/` 目录：无打包器、无转译、vendored 库以静态文件进仓库、`go build` 即跑），不约束**开发过程**；测试工具不进 `web/` 目录、不进二进制，不属于该约束的范围。Project 内呈现层 Scenario→测试的映射据此修正：ADR-0002 的「每个 Scenario 对应 API 测试」收窄为「API 层行为如此，呈现层行为按本 ADR 执行」。

## Alternatives

- **维持全 Scenario 走 API 测试**：对呈现层行为只能断言「库资源在场」而断言不了「行为发生」，Spec 沦为空话；或为凑断言把呈现逻辑搬到服务端（违反 ADR-0002 的前后端分离）。
- **前端单元测试（jsdom + node test runner）**：断言不经过真实渲染管线，且向仓库引入 node 工具链，与零构建开发体验的极简主义冲突。
- **只靠人工目测、不写任何浏览器验证**：成本最低，但 Scenario 与验证脱钩，`openspec verify` 无法对呈现层行为给出一致性证据，学习目标（验证 SDD 全流程）受损。

## Consequences

- 纯前端 Change 从此有确定的 spec 写法与验收路径：Requirement 写用户可观察行为，Scenario 的断言位置在 design.md 分派、tasks.md 落勾选项。
- 每个 Phase 2+ Change 不再重复讨论流程问题，proposal 直接引用本 ADR。
- 代价：呈现层 Scenario 的验证不是 `go test` 一键全量，浏览器走查是半自动的；接受该成本换取 Spec 的诚实。
- 后续若引入 Playwright 等更重的 E2E 工具链，只需修订本 ADR 的工具选择，分派原则不变。
