# Proposal

## Why

目录列表当前只给出条目的名称与类型，用户无法在不逐个打开文件的前提下判断「这个文件多大」「上次什么时候改过」。这两个信息恰恰是 macOS Finder 风格列表视图里最常被扫视的两列，也是「像 Finder 一样浏览」这一产品目标最直接的落差。

Change 02 在 `Directory listing response` 里写下了一句显式排除：「系统 SHALL 在每个条目中只提供名称与类型，不提供大小、修改时间或内容」。那是当时为了让 capability 边界干净而主动收窄的结果，本次 Change 正是要把它推翻——因此本 Change 的 delta 是一次**推翻既有 Change 自身所写排除项**的 MODIFIED。

## What Changes

- **ADDED** `directory-browsing / Entry metadata`：目录列表的每个条目除名称与类型外，还给出大小与最后修改时间；并规定两者在不可得时的表达方式。
- **MODIFIED** `directory-browsing / Directory listing response`：移除「不提供大小、修改时间」这一半排除，**保留「不提供内容」**。内容读取属于 `text-preview`（Change 04），那道门不能在本 Change 松掉。
- 前端在列表中把大小与修改时间与名称一并呈现，呈现形态（单行内联或多列对齐）属表现层，spec 不约束。
- **不产生新 capability。** 元信息只在目录列表这一处出现，尚不存在「脱离列表查看文件信息」的场景，因此它是 `directory-browsing` 的一部分，而不是一个独立能力。`docs/roadmap.md` 的 Capability 地图需要相应调整。

本 Change 明确**不包含**以下内容，均已在探索阶段单独决定并记录：

- **不改符号链接的条目类型。** Change 02 的 D12 保持原样：指向目录的符号链接仍按字面报告为 `file`、仍不可点击。因此大小字段对符号链接取的是链接自身长度而非目标大小——这一语义会被 spec 如实承诺，而不是留给实现自行解释。
- **不做符号链接图标与可导航性。**
- **不做文件类型分类（文本/图片/Markdown/二进制）。** 那属于 `text-preview`（Change 04/05/07）的判断，本 Change 抢先定义会替后续 capability 做决定。
- **不做文件/目录图标。** 图标选型是纯呈现层，客户端可由条目名自行推导，不需要服务端新增字段。roadmap 把它留在 `polish-file-browser`。
- **不做大目录分页。** 响应体与前端节点数都会因此增长，分页是独立 Change。

## Capabilities

### New Capabilities

无。

### Modified Capabilities

- `directory-browsing`：新增 `Entry metadata` Requirement；`Directory listing response` 的字段范围发生改变（ADDED + MODIFIED 同一个 capability）。`Entry type distinction`、`List ordering`、`Position representation and reproducibility`、`Parent directory reference`、`Root directory confinement`、`Directory access failures` 均不变。

## Impact

- **服务端** `internal/server/browse.go`：`listEntry` 结构新增字段；`list()` 需要对每个条目取一次 `DirEntry.Info()` 以获得大小与修改时间；新增「条目在 readdir 之后消失」这一竞态的错误处理路径。
- **服务端** `internal/server/browse_test.go`：既有断言列表响应形状的测试需相应调整。
- **前端** `web/app.js`、`web/style.css`：条目渲染从 2 个节点增加到 4 个节点；Change 02 的 D8「绝不用 `innerHTML`」安全不变量在节点数翻倍后需要重新验证。
- **API 契约**：`GET /api/list` 的响应条目新增两个字段。属于**非破坏性**扩展——既有消费者忽略未知字段即可继续工作。
- **Spec**：归档后 `openspec/specs/directory-browsing/spec.md` 的 `Directory listing response` 会被改写。
- **依赖**：无新增。仍为零第三方依赖、零构建、单二进制。
