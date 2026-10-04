# AGENTS.md — AI 协作约定

本文件给 AI 读；README.md 给人读；两者非必要不重复。

## 项目背景

`http-file-browser` 是零配置 HTTP 文件浏览器：命令行指定目录启动本地 HTTP 服务，浏览器中以 Finder 风格浏览文件系统。它同时是 OpenSpec SDD 学习项目——首要目标是走通 Spec 驱动开发全流程（含刻意的需求变更演练），而不是尽快把产品做完。

## 开始工作前

1. `openspec/config.yaml` — 工作流规则与语言约定（OpenSpec 生成，已配置中文）
2. `docs/roadmap.md` — 本次工作在整体规划中的位置
3. 涉及行为变化时，读相关 capability 的 `openspec/specs/<capability>/spec.md`
4. 流程细节（delta spec 格式、MODIFIED 要求等）以教材 `docs/OpenSpec_HTTP_File_Browser.md` 为准
5. OpenSpec 工作流命令为 `/opsx-propose` 形式（见 `.opencode/commands/`），不要假设其他写法

## 硬规则

- 用户可感知的行为变化必须走 OpenSpec Change：explore（问题不清时）→ propose → apply → verify → archive。
- 工作流选择：需求清晰的简单 Change 用 propose 一步到位；复杂或需要逐 artifact 学习 review 的 Change 用 new → continue 逐步生成、逐个 review；归档前用 verify 检查实现与 artifacts 一致（教材 50-52 节）。
- 禁止直接修改 `openspec/specs/` 与 `openspec/changes/archive/`：前者由 sync/archive 更新，后者是不可变历史。
- Apply 中发现范围外功能：停下说明，记入 `docs/roadmap.md` 想法池或另立 Change；不扩大当前 Change。
- Apply 中发现计划有误：先 update Change artifacts 使其一致，再继续实现；不用改代码掩盖需求变化。
- 已归档 Change 的行为要变化：创建新 Change，用 MODIFIED / ADDED / REMOVED / RENAMED 表达；不修改历史归档。
- Requirement 只写可观察行为；实现细节进该 Change 的 design.md，长期架构决策进 ADR。

## 测试环境

- `testdata/` 是本地手工测试环境：图片（png/jpg/gif/bmp）、Markdown 相对链接与内嵌图片、语法高亮样例、根内/出根/悬空软链、超限与二进制等边界用例。不进版本控制（`.gitignore`）。
- 命名取 Go 惯例：`testdata/` 是测试 fixture 的标准去处，且 go 工具链完全忽略该目录（构建、测试、vet 都不下钻），目录里的软链与怪文件名天然与工具链隔离。
- 启动：仓库根执行 `go run . testdata`；重建：`python3 testdata/generate.py`。
- tasks 中的浏览器走查、端到端冒烟类验证可用它作根目录；各用例的预期行为见 `testdata/README.md`。
- 调整环境内容属测试基建，不是用户可感知行为变化，不需要 OpenSpec Change。

## 文档写入规则

| 要写的内容 | 写到哪 |
|---|---|
| 新想法、规划调整 | `docs/roadmap.md`（想法池 / Change 地图状态） |
| 有长期架构影响的决策 | `docs/adr/NNNN-标题.md`（复制 0000 模板） |
| 一次 Change 的实现方案 | 该 Change 的 `design.md` |
| 系统行为 | Change 的 delta spec（archive 后合入 `openspec/specs/`） |
| 学习实验观察 | `docs/journal.md` |

## 语言约定

- 文档与 Spec 正文一律中文。
- AI 与用户的对话也使用中文，与文档/Spec 保持一致。
- OpenSpec 结构标记保留英文格式：`### Requirement:` / `#### Scenario:` / `## ADDED|MODIFIED|REMOVED|RENAMED Requirements` / `**WHEN**` / `**THEN**`（与 `openspec/config.yaml` 一致）。

## Git 约定

- Change 实现提交：`feat: <change-name>`（代码与该 Change 的 openspec 文件同一提交）。
- 归档提交：`archive: <change-name>`。
- `openspec/` 与 `.opencode/` 一并提交，属于项目知识。

## 状态联动

- 立 Change：`docs/roadmap.md` 中 💡 → 🚧
- 归档：roadmap 🚧 → ✅（注明归档目录名）；README「当前状态」同步；`docs/journal.md` 补观察
- 决策被推翻：旧 ADR 标 superseded，新写一条，不删除
