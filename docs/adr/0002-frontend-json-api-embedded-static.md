# ADR-0002：前后端分离——JSON API + 内嵌零构建静态前端

- 状态：accepted
- 日期：2026-10-03
- 关联：ADR-0001；Change 01 `bootstrap-http-server`；Change 02 `directory-browsing`

## Context

Finder 风格的交互是密集型：进入/返回目录、预览切换、渲染/源码视图切换、搜索过滤，未来还有编辑保存。这些行为会随 Change 演进持续增长。单二进制（ADR-0001）要求前端资源可内嵌。此外，本项目以 SDD 为首要目标，Spec 中的 Scenario 需要清晰、稳定的自动化断言点。

## Decision

前后端分离：Go 只提供 JSON API（目录列表、文件元数据、文件内容等）；前端为 vanilla JS 静态应用，由 `go:embed` 内嵌，服务端只保留一个入口 HTML，无模板逻辑。前端零构建：不引入 npm 与打包器，第三方渲染库（如 highlight.js、marked）以 vendored 静态文件进仓库，随二进制内嵌。

## Alternatives

- html/template 服务端渲染：每个 Change 出活最快，但交互密集后模板与零散 JS 的边界会逐渐模糊，且缺少 API 级测试断言点，Scenario 与测试的映射不清晰。

## Consequences

- 测试约定顺势确定：每个 Scenario 对应一个 API 层的 Go 测试（`net/httptest`）；具体映射细则在 Change 01 的 design.md 落地。
- 此后 Change 的任务大体按「API 行为 + 前端消费」两半拆分。
- 初期成本略高（Change 01/02 要先搭 API 与前端骨架），换来后续每个 Change 的实现链路稳定。
- Phase 2 引入渲染库时 vendor 进仓库，不新增工具链；「零配置」的开发体验保住：`go build` 即跑。
