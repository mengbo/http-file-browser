# Proposal

## Why

Change 01 建立的只是一个「能启动、能返回前端页面、`/api/health` 往返通」的骨架，前端至今只显示一行「服务就绪」。产品承诺的「像 macOS Finder 一样浏览文件系统」目前没有任何行为支撑：用户能启动服务，但看不到任何文件。

目录浏览是本项目第一个真正的核心 capability，也是后续每一个 Change 的地基——`file-metadata`、`text-preview`、`syntax-highlighting`、`image-preview`、`file-editing` 全部叠在「一个能列出并进入目录的界面」之上。先做它们等于在没有地基的地方盖二楼。

## What Changes

- 新增目录列表端点：按请求的相对路径返回该目录的规范化路径、上级路径与条目列表
- 前端在浏览器中渲染目录列表，用户可进入子目录、返回上级，并看到当前所在位置
- 浏览位置镜像到浏览器地址栏：刷新保持在原目录、浏览器前进/后退可用、当前目录可作为深链接直接打开
- 列表顺序确定：目录排在文件之前，其后按名称排序，同名时顺序稳定可复现
- 服务不提供根目录之外的内容：以字面路径越界的请求被拒绝并返回 JSON 错误
- **MODIFIED** `service-startup` 的 JSON 错误响应约定：错误体显式携带机器可读的 `code` 字段，使前端能区分不同失败原因（详见下方 Capabilities）

本 Change 不包含：文件元信息（大小、修改时间、类型推断）、任何文件内容读取或预览、排序方式可由用户调整、隐藏文件过滤、上传/删除/重命名、侧边栏与分栏视图。这些分别属于后续 Change 或已记入想法池。

## Capabilities

### New Capabilities

- `directory-browsing`: 目录浏览的用户可观察行为契约。覆盖列表响应的形状与内容、条目类型区分、列表顺序、相对路径导航模型、上级目录语义，以及服务不提供根目录之外内容这一边界。不覆盖文件元信息与文件内容——那属于 `file-metadata` 与 `text-preview`。

### Modified Capabilities

- `service-startup`: 仅修改 `JSON error responses` 这一条 Requirement，把「机器可读的错误标识」明确为错误响应体自身携带的字段。**这不是对 Change 01 的返工**：Change 01 的 design D4 当时有意选用单字段 `error`，并预先声明「结构化 code 的价值在于前端需要分支处理时，那属于后续 capability 出现后的演进，届时用 MODIFIED 表达，而不是现在预留一个无人使用的字段」。本 Change 正是那个触发条件——目录浏览是第一个需要按失败原因分支的消费者，它要区分「不存在 / 不是目录 / 无权限 / 越界」四类失败且前端对四者反应各不相同。其余四条 Requirement（参数校验、启动报告、默认回环绑定、响应分区）不变。

## Impact

- 服务端：向 `/api/` 分区新增一个端点；`errorResponse` 结构由单字段字符串改为带 `code` 的结构；根目录需要从 `internal/app` 传入 API 层（当前 API 层不持有根目录）
- 前端：`web/index.html` 与 `web/app.js` 从「单行状态」改为列表视图，并新增地址栏与列表之间的状态同步；`web/style.css` 相应扩充
- 契约：`/api/health` 响应体追加一个字段用于前端显示绝对路径（追加字段不违反该 Requirement 既有 Scenario）
- 测试：沿用 ADR-0002 约定的「每个 Scenario 一个 `net/httptest` API 层测试」；前端无自动化测试设施（零构建约束），URL 同步等前端行为列为手工验证项
- 已归档内容：本 Change 不修改任何 `openspec/changes/archive/` 下的历史文件；对 `service-startup` 的修正只通过本 Change 的 MODIFIED delta 表达
