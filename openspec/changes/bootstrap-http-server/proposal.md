# Proposal

## Why

项目目前只有规划与两份 ADR，没有任何可运行代码。后续每一个 Change（目录浏览、文本预览、编辑、远程访问）都依赖同一个前提：命令行指定目录后能起一个本地 HTTP 服务，并在浏览器里打开一个能工作的页面。这个前提必须由第一个 Change 单独建立，且必须一次建立对——否则后面每个 Change 都要先还骨架的债。

同时，ADR-0002 已把「JSON API + 内嵌零构建静态前端」定为长期架构约束，并写明测试映射细则要在本 Change 的 design.md 落地。本 Change 需要提供一个真实的 JSON 端点，让该约束在归档时就已可验证，而不是推迟到第二个 Change。

## What Changes

- 新增命令行入口：接受一个目录参数启动服务；参数缺失或目录不可用时报错并以非零状态退出
- 服务默认仅监听 `127.0.0.1`，启动时向标准输出打印可访问地址
- 建立 HTTP 响应分区约定：`/` 及静态资源返回内嵌前端页面，`/api/` 前缀返回 JSON
- JSON 端点在出错时同样返回 JSON 响应体，而非 HTML 错误页
- 提供 `/api/health` 端点，并在内嵌前端中通过 `fetch` 消费它、渲染结果，以此证明前后端链路连通
- 建立项目骨架：Go module、HTTP 服务器、`go:embed` 内嵌前端资源、API 层测试约定

本 Change 不包含：目录列表、文件元信息、任何文件读写、监听地址可配置、认证。这些属于后续 Change。

## Capabilities

### New Capabilities

- `service-startup`: 命令行启动契约与 HTTP 响应形态契约。覆盖参数校验、退出码、启动输出、默认监听地址、静态页与 JSON 的响应分区、JSON 错误响应约定。不包含目录浏览语义——目录列表由 `directory-browsing` 拥有。

### Modified Capabilities

无。这是本项目第一个 Change，`openspec/specs/` 目前为空。

## Impact

- 新增 Go module 与源码目录，首次引入构建产物形态
- 新增内嵌静态资源目录（零构建，不引入 npm 与打包器）
- HTTP API 从本 Change 起存在，后续 Change 沿用其响应分区与错误信封约定
- 后续 `directory-browsing` 的 Scenario 断言点依赖本 Change 建立的测试约定
- `add-remote-access` 将以 ADDED 方式扩展本 Change 的默认监听地址 Requirement，而非修改它
