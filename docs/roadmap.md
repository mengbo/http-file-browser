# Roadmap

> 本文件是**活文档**，跟踪项目长期规划。静态教材是 [OpenSpec_HTTP_File_Browser.md](OpenSpec_HTTP_File_Browser.md)，只作参考、不随开发更新；本文件随每次立 Change 和归档而更新。
>
> 时间态分工：**想法（本文件）→ 进行中（`openspec/changes/`）→ 事实与历史（`openspec/specs/` + `archive/`）**。尚未成为 Change 的意愿只放这里，不进 `openspec/`。

## 状态标记

| 标记 | 含义 |
|---|---|
| 💡 | 想法，未开始 |
| 🚧 | 进行中，存在 active Change |
| ✅ | 已归档，行为已进入 `openspec/specs/` |

## 愿景

零配置 HTTP 文件浏览器：命令行指定目录即启动服务，浏览器中像 macOS Finder 一样浏览文件系统。同时本项目是 OpenSpec SDD 学习实验场——优先验证完整工作流（Explore → Propose → Apply → Verify → Archive → 需求变更演练），不追求一次把产品做完。

长期约束（2026-10-03 架构探索确定，详见 [docs/adr/](adr/)）：跨平台分发（macOS/Linux/Windows），UI 风格借鉴 macOS Finder；单二进制交付——Go、前端资源内嵌、不依赖语言 runtime；前端为零构建 vanilla JS 静态应用，后端只提供 JSON API；安全默认——默认只听 `127.0.0.1`、默认只读，开放远程访问/写入必须显式指定（Phase 4 时正式 ADR 化）。

## 架构决策

| ADR | 决策 | 状态 |
|---|------|------|
| [0001](adr/0001-go-single-binary.md) | 技术栈选 Go，单二进制交付 | accepted |
| [0002](adr/0002-frontend-json-api-embedded-static.md) | 前后端分离：JSON API + 内嵌零构建静态前端 | accepted |

## 阶段规划

1. **浏览与预览（MVP）**：目录浏览、文件/目录图标、进入/返回上级、文件基本信息、文本文件查看
2. **增强预览**：代码高亮、Markdown 渲染、图片预览、二进制文件提示、搜索/过滤
3. **编辑**：文件编辑、保存、冲突处理、大文件限制、只读/可写模式
4. **远程访问**：监听非 localhost、启动时生成随机 token、Token 认证、安全边界

## Change 地图

来自教材第 21 节。**立 Change 时把 💡 改为 🚧；归档后改为 ✅ 并注明归档目录名。** 不要求全部完成，前 5～8 个即足以掌握 OpenSpec。

| # | Change | 目标 | 备注 | 状态 |
|---|--------|------|------|------|
| 01 | `bootstrap-http-server` | 最小 HTTP 服务 | 建立项目骨架；新增 capability `service-startup` | ✅ [`2026-10-03-bootstrap-http-server`](changes/archive/2026-10-03-bootstrap-http-server/) |
| 02 | `directory-browsing` | Finder 风格目录浏览 | 第一个核心 capability | ✅ [`2026-10-03-directory-browsing`](changes/archive/2026-10-03-directory-browsing/) |
| 03 | `file-metadata` | 文件信息展示 | | 💡 |
| 04 | `text-preview` | 文本文件查看 | | 💡 |
| 05 | `improve-text-file-detection` | 无扩展名文本文件可预览 | 第一次需求变更，MODIFIED 演练 | 💡 |
| 06 | `add-syntax-highlighting` | 代码语法高亮 | | 💡 |
| 07 | `add-markdown-preview` | Markdown 渲染 | | 💡 |
| 08 | `improve-markdown-preview` | 渲染/源码视图切换 | 第二次需求变更，MODIFIED 演练 | 💡 |
| 09 | `add-image-preview` | 图片预览 | | 💡 |
| 10 | `add-file-search` | 文件搜索 | | 💡 |
| 11 | `add-file-editing` | 文件编辑与保存 | 先 Explore 保存语义 | 💡 |
| 12 | `add-edit-conflict-detection` | 编辑期间检测外部修改 | | 💡 |
| 13 | `add-remote-access` | 监听地址 | 先 Explore | 💡 |
| 14 | `add-token-authentication` | Token 认证 | | 💡 |
| 15 | `improve-authentication` | 认证行为变化 | MODIFIED 演练 | 💡 |
| 16 | `polish-file-browser` | 最终体验优化 | | 💡 |

## Capability 地图

产品能力边界（仅是地图，不预先创建，立 Change 时按需产生）：

`directory-browsing` · `file-metadata` · `text-preview` · `syntax-highlighting` · `markdown-preview` · `image-preview` · `search` · `file-editing` · `remote-access` · `authentication`

立 Change 时按需增补。已产生的：

- `service-startup`（Change 01）：命令行启动契约 + HTTP 响应分区（静态页 vs `/api/` JSON、JSON 错误信封、默认仅监听回环）。不含目录浏览语义。
- `directory-browsing`（Change 02）：根目录内的目录列表契约——响应形状（`path`/`parent`/`entries`）、条目类型区分、列表顺序、相对路径导航模型、上级目录语义、根目录边界。**不含**文件元信息与文件内容读取。

## MVP 明确不做

文件编辑、远程访问、Token、用户系统、上传、删除、复制、移动、压缩、全文搜索、实时文件监听、多用户。过程中冒出的好点子 → 记入想法池，不扩入当前 Change。

## 想法池

尚未规划为 Change 的点子。立 Change 时从这里认领或新增。

- **符号链接越界策略**（Change 02 起悬置）：`directory-browsing` 明确只做**字面**路径的越界判定，根目录内指向外部的符号链接可被跟随，spec 已如实承诺这一强度。回环单用户下风险可接受，但 `add-remote-access`（Change 13）开放非回环监听**之前**必须先用 MODIFIED 正式化该策略，否则等于开放远程任意文件读取。
- **大目录分页**：目录列表一次性返回全部条目，十万级文件目录会产生很大响应体。若要分页会改变 `Directory listing response` 的响应形状，属新增 Requirement，需独立 Change。
- **`parent` 空值的二义性**（design D10 记录）：根目录与根目录的一级子目录，其 `parent` 都是空字符串，客户端必须靠 `path` 判断是否在根目录。当前刻意不为它引入 `null` 第二种表示；若 `file-metadata` 的面包屑或后续 Change 发现按 `parent` 推断根目录更方便，再用 MODIFIED `Parent directory reference` 把根目录的 `parent` 正式化为 `null` 或缺省。
- **符号链接的条目类型**（design D12 记录）：指向目录的符号链接目前显示为 `file` 且不可点击，但按其字面路径请求能列出目标内容——「字面路径可提供」与「字面报告为文件」是同一个 D3 决策的两面。等 `file-metadata` 要展示「种类」时一并决定是否把符号链接作为第三类条目，届时 MODIFIED `Entry type distinction`。
- **错误态缺少返回上级入口**：目录访问失败时前端清空条目并隐藏上级入口，用户只能靠浏览器后退离开。属纯呈现层（design Open Questions 已声明 spec 未约束错误态呈现），未在本 Change 处理；若实测中确实碍事，可在后续 Change 的前端润色里补上客户端自行推导的上级入口。
