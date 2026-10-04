# OpenSpec 实战总结：用 http-file-browser 走通 SDD

> [OpenSpec_HTTP_File_Browser.md](OpenSpec_HTTP_File_Browser.md) 是**项目最初计划**（与 ChatGPT 讨论后成文），讲「出发前打算怎么做」，静态、不随开发更新。本文是**走完之后**的复盘：把 15 个归档 Change 的真实经历，提炼成一份精简方法手册，`http-file-browser` 只作例证。要立刻查命令、文件放哪、怎么选路，用 [速查表](OpenSpec_Cheatsheet.md)。

---

## 1. 心智模型：三层就够了

OpenSpec 把「系统现在是什么」和「系统怎么变成这样的」分开存，中间用 Change 连接：

```
openspec/specs/             系统现在是什么（当前行为的事实）
      ↑ 合并 delta（archive 时）
openspec/changes/<名称>/     正在发生的一次变化（这个 Change 的完整生命周期）
      ↓ 做完后归档
openspec/changes/archive/    怎么变成现在这样的（不可变的历史）
```

- **specs/**：每个能力一份，写系统**应当**有的行为。本项目现有 8 份（见第 12 节）。
- **changes/**：一次行为变化的全部材料；没做完的 Change 在这儿。
- **archive/**：做完的 Change 原样搬进来，永远不再改。想知道某条行为当初为什么存在，翻它。

一句话：**specs 是现在时，changes 是进行时，archive 是过去式。**

---

## 2. 关键术语

21 条术语的中文解释（Spec / Capability / Change / Delta / Requirement / Scenario / Purpose / ADR / 各种动作……）见 [速查表](OpenSpec_Cheatsheet.md) 第 2 节。

---

## 3. `/opsx-*` 命令一览

本项目 OpenCode 生成的命令写法是 `/opsx-propose` 这种（见 `.opencode/commands/opsx-*.md`）；有的工具显示成 `/opsx:propose`。以 `openspec init` 后项目里实际出现的为准。

| 命令 | 干什么（人话） | 本项目 |
|---|---|---|
| `/opsx-explore` | 只想不写：把问题聊清楚，不碰代码、不出正式文件。聊出来的东西只留在对话里。 | ✅ 用过（开工前架构探索；图片、远程访问的边界探索） |
| `/opsx-propose` | 一步到位：立一个变更，并把提案、规格、设计、任务四份文件一次写完。只写字，不写代码。 | ✅ 主用（15 个变更基本都走它） |
| `/opsx-new` | 只立变更骨架、给你看第一份文件的模板，什么都不写就停下等你发话。想一个文件一个文件地走时从它开始。 | ✗ 未用 |
| `/opsx-continue` | 接着 `/opsx-new`，一次写下一份文件。和 new 配对逐步推进。 | ✗ 未用 |
| `/opsx-ff` | 快进：把当前变更还缺的文件一次补齐到「可以开工」。结果和 propose 基本一样。 | ✗ 未用 |
| `/opsx-apply` | 照着任务清单真正写代码、写测试。 | ✅ 主用 |
| `/opsx-verify` | 做完后什么也不改，只回头对账：规格要求的和实际做的对不对得上，给出「已满足 / 没满足 / 多做了」的清单。 | ✅ 用过 |
| `/opsx-update` | 做到一半发现计划不对，先回去改提案/规格/设计/任务，再接着做。不碰代码。 | ✅ 用过（把设计文档回写、追平实现） |
| `/opsx-sync` | 把变更里的规格合并进主规格，但这次变更还不算结束，仍算进行中。 | ✅ 用过（归档时顺带） |
| `/opsx-archive` | 变更全做完后收尾：合并规格，再把整个变更搬进档案室。 | ✅ 用过 |
| `/opsx-bulk-archive` | 一次归档好几个变更。 | ✗ 未用 |
| `/opsx-onboard` | 手把手带你走一遍完整流程的教学命令。 | ✗ 未用 |

终端里的 `openspec` 命令（`list` / `validate` / `archive` 等，含参数）见 [速查表](OpenSpec_Cheatsheet.md)。

---

## 4. 一个 Change 的标准节奏

```
explore（可选）→ propose → review → apply → verify → archive
                              ↑
                        发现问题时 update
```

1. **Explore**：问题不清楚时先想清楚。只讨论，不写代码、不出正式文件。
2. **Propose**：生成四份文件——提案（为什么/改什么）、规格（delta，行为）、设计（怎么做）、任务（可执行步骤）。
3. **Review**：按 **提案 → 规格 → 设计 → 任务** 的顺序看一遍。最重要的是规格；其次设计；最后看任务有没有覆盖规格。
4. **Apply**：按任务清单实现、写测试，完成一项勾一项。
5. **Verify**：只读复核，逐条对账实现和 artifacts。
6. **Archive**：把 delta 合并进主规格，Change 搬进档案室。

本项目 15 个 Change 基本都走这条线；`update` 与 `sync` 在需要时插入。

> **不用背命令。** OpenCode 里装了 OpenSpec 的 skills 与 commands，AI 本身就知道这套流程。不知道该走哪一步、下个文件写什么、某个需求该 explore 还是 propose 时，直接用自然语言问它（「现在该干嘛？」「这个是新增还是修改已有能力？」），它会给出下一步。命令是确定时用的快捷方式，不是必须记住的咒语。

---

## 5. 核心一课：四种 delta 的判据

一次变更相对当前系统，只有四种改法：

| 改法 | 什么时候用 | 硬规则 |
|---|---|---|
| **ADDED** | 新增一条行为 | 每条至少一个 Scenario |
| **MODIFIED** | 已有行为发生变化 | **必须把旧 Requirement 完整抄出来再改**；归档时用它整块替换旧条款 |
| **REMOVED** | 删掉一条行为 | 要写 Reason（为什么删）和 Migration（怎么办） |
| **RENAMED** | 只是名字变了，行为没变 | 别和 MODIFIED 混 |

**为什么 MODIFIED 必须完整复制？** 因为归档是拿 delta 去**替换**主规格里的旧条款，不是「补充说明」。只写半句、写「此处也支持 X」，归档时无法定位、无法替换。

### 两个最值得记的案例

**案例 A：Change 17「预约的翻案」。** 系统早年刻意允许「根目录内的软链按字面路径放行」。后来引入远程访问，这个宽松成了安全洞——同网段任何人都能顺着软链读到根外文件。Change 17 把越界判定从「字面路径」改成「物理路径」，正式翻案。它没有新增一条行为，而是在**同一条 Scenario 标题下**把 THEN 从「按该路径提供内容」原地反转成「拒绝并返回 `outside_root`」。需求变化被表达成对同一条行为的重写，而不是叠加一条新行为。更关键的是：当初立下宽松政策时，就同时预约了翻案时机（「13 的前置」），所以这次 BREAKING 有出处、零争议。**写下一个将来可能要翻的行为时，顺手把翻案时机也写上。**

**案例 B：Change 19「推翻 design ≠ MODIFIED」。** 项目要把目录列表的时间显示从「随语言环境」改成固定格式。字面上是「推翻」一个旧决定，但那项决定当初被归为「spec 不约束、design 定方向」，**从没进过主规格**。既然主规格里没有旧条款可改，本次就不能用 MODIFIED，而是用 **ADDED** 把这个一直被规格忽略的呈现行为第一次写成契约。**决定 ADDED 还是 MODIFIED 的，是「主规格里有没有旧条款」，不是措辞里有没有「推翻」二字。**

---

## 6. Spec / Design / ADR 的分工

| 问题 | 答案写在哪 |
|---|---|
| 系统应该有什么**行为**？ | 规格（Requirement / Scenario） |
| 这次**怎么实现**？ | 该 Change 的 design.md |
| 一个**长期**架构决策为什么这么选？ | `docs/adr/` |
| 项目整体的长期约束与规划？ | `docs/roadmap.md` |

要点：

- **Requirement 只写可观察的行为。** 反例：「使用 React + Prism 渲染」。正例：「以该语言的语法高亮形式呈现」。
- **可调旋钮不进规格。** 文本预览的 1 MiB 上限、内容嗅探的 4096 字节窗口、图标颜色值、渲染库的名字——全在 design。规格里连「1 MiB」都不写，只说「超过系统可提供内容的最大字节数」。好处是调参不动任何 Scenario。
- **本项目 15 个 Change，「AI 把规格写成实现方案」一次都没发生。** 靠的不是运气，是三样东西：`openspec/config.yaml` 里的语言与规则、每次 review 的固定检查法、以及 design 专门承接实现细节。

---

## 7. Scenario → 验收的闭环

- **Scenario 应当能直接转成测试。** 本项目坚持规格里的 Scenario 列表和测试函数列表同构，可以逐条勾对（Change 10 是 22 条 Scenario 对 22 个测试函数）。
- **按行为性质分派断言位置**（ADR-0003）：请求/响应、错误标识这类 **API 层行为**走 `go test`；DOM 渲染、视觉形态这类 **呈现层行为**走浏览器走查，结果记回 tasks。纯前端变更不再为「怎么验收」反复争论。
- **测试写在 apply，verify 只复核。** apply 的任务里就有「补某测试」「跑 go test / go vet」「端到端走查并记录观察结果」；verify 什么也不改，只对账。
- **守门人测试必须被亲手证伪一次。** 设计里声明「这条测试守住某条关键决策」，就要临时把实现改错、看它是否真的红（比如把 `DirEntry.Info` 换成 `os.Stat`，软链用例应立刻失败）。没被证伪过的测试，只是「看起来会抓住」。

---

## 8. 新会话的切点

换会话（`/new`）能买到的是**干净的上下文**：实现方必须把 artifacts 当唯一契约来读，复核方不会替自己刚写的东西辩护。但能不能切，只看一件事——**磁盘上有没有自足的交接物**。

| 接缝 | 磁盘上有交接物吗 | 能否切 |
|---|---|---|
| explore → propose | ❌ explore 不落盘 | **不能**，默认同一会话 |
| propose → apply | ✅ 提案/规格/设计/任务都在 | 能 |
| apply → verify | ✅ 代码 + 测试 + 勾好的任务 | 能 |
| verify → archive | ✅ | 能（同会话也行） |

**为什么 explore 不能和 propose 分开：** explore 的设计就是「决策留在对话、不落盘」，而 propose 只从磁盘读（`openspec context` → `new change` → `instructions`）。一刀切下去，讨论全没，propose 只能拿一句描述重推，很可能和刚敲定的相反。要在两者间切，必须先把 explore 的结论落盘——要么让 explore 直接把它捕获成 Change 的 artifacts，要么落进 roadmap / 想法池 / ADR。本项目就是这么做的：ADR-0001 / 0002 本身就是一次架构探索的落盘。

一句话：**explore+propose 同一个会话；propose→apply、apply→verify 两个接缝切会话。**

---

## 9. 需求变化的两条路

| 情况 | 怎么走 |
|---|---|
| **Change 还没归档** | 用 `/opsx-update` 改这个 Change 的 artifacts，再继续 apply。 |
| **Change 已经归档** | 另立一个新 Change，用 MODIFIED / ADDED / REMOVED / RENAMED 改主规格。绝不回头改 archive。 |

- **计划先行。** 发现计划不对时，先 `update`（改文件）再 `apply`（改代码），而不是「代码先变、文档追认」。反过来会留下一个「代码已经不对、文档还不知道」的危险窗口。本项目 Change 16 / 18 两轮换设计都是这么走的。
- `update` 只改 planning artifacts，**不碰代码**。
- archive 是不可变历史：已归档的行为要变，只能新开 Change。

---

## 10. 归档与 sync

- **归档前**：跑 `/opsx-verify` 对账，再跑 `openspec validate`，确认 tasks 全部打勾、delta 有效。
- **`/opsx-archive` 会先问你要不要 sync。** 归档流程在动手前会逐个 delta 核对它与对应主规格的同步状态，然后弹选项：需要同步时给「Sync now（推荐）/ Archive without syncing」；已经同步时给「Archive now / Sync anyway / Cancel」；某能力同步被阻塞时只给「Archive without syncing / Cancel」。选 Sync now 会**内联**跑一遍 `/opsx-sync`（智能合并），再回头核对合并结果，确认无误才归档——所以选它，同步与归档一次完成，不必再手动敲一次 sync。
- **sync vs archive**：`sync` 把 delta 合进主规格但 Change 仍 active（适合只想先固化一部分规格）；`archive` 是「合并 + 搬进档案室」的完整收尾。归档通常顺带完成同步。
- **合并规则**：ADDED 落位到对应能力；MODIFIED 是「拿 delta 整块替换主规格里的同名 Requirement」——本项目多次退化成「逐字节搬运 + 断言」（合并的智能体现在「知道这次不需要智能」）。归档前要数落地结果：Requirement 数量对不对、旧措辞有没有残留、有没有把 `## ADDED` 这类 delta 头泄漏进主规格。
- **Purpose 不会被自动重写。** archive 只搬 Requirement。当某次变更让某份规格开头的「用途说明」变假时，工具不会改它。本项目为此做过**三次经用户明确授权的破例**，每次都只改那一句、并记进 journal。纪律是：**破例要重新问，不能因为上次破过就顺手破。**

---

## 11. 配套文档与长期需求管理

OpenSpec 只管两件事：**系统当前行为**（`openspec/specs/`）和**正在发生的变化**（`openspec/changes/`，完成后进 `archive/`）。它**故意不管**「还没成为 Change 的想法」和「跨变更的长期决策」。这些由项目自己在 `docs/` 下补齐，几份文档各管一段时间态：

| 文档 | 管什么 | 什么时候更新 |
|---|---|---|
| `openspec/specs/` | 系统现在是什么（事实） | 归档时由 delta 合并 |
| `openspec/changes/` | 正在变什么（进行中） | 立 Change / 实施 / 归档时 |
| `openspec/changes/archive/` | 过去怎么变的（历史，不可改） | 归档时 |
| `docs/roadmap.md` | 长期需求：愿景、长期约束、Change 地图、能力地图、**想法池** | 立 Change（💡→🚧）与归档（🚧→✅） |
| `docs/adr/` | 跨变更的长期架构决策为什么这么选 | 出现长期决策时 |
| `docs/journal.md` | 学习实验观察：实际发生了什么 | 每完成一个 Change |
| `docs/OpenSpec_HTTP_File_Browser.md` | 项目最初计划（与 ChatGPT 讨论后成文） | 静态，不更新 |

三条归属规则：

- **一次变更的实现方案 → 该 Change 的 design.md；跨变更的长期决策 → ADR。** 判据是「它会不会在本次 Change 之外继续成立」。
- **系统行为 → 规格；还没成为 Change 的意愿 → roadmap 想法池**，不进 `openspec/`。
- **破例的授权与实验观察 → journal**（比如 Purpose 那三次经授权的修改）。

为什么这一节属于本文：OpenSpec 是工作流的骨架，但「长期需求」这件事它不替你管——backlog 放哪、架构决策记哪、实验观察写哪，都是 SDD 落地必须自己补的一层。**roadmap 的想法池，就是这套流程里的长期需求入口**：想法先在这里沉淀，成熟了才认领成一个 Change、进入 `openspec/changes/`。

---

## 12. 本项目证据链

### 计划 vs 实际

最初计划给了 16 个 Change 的排布；实际归档 15 个，形状差了很多：

| 最初计划 | 实际 |
|---|---|
| 01 bootstrap-http-server | ✅ 照做 |
| 02 directory-browsing | ✅ 照做 |
| 03 file-metadata | ✅ 但不成立为独立能力，落成 `directory-browsing` 的一条 ADDED 需求 |
| 04 text-preview | ✅ 照做 |
| 05 improve-text-file-detection | ✅ 第一次 MODIFIED 演练 |
| 06–08 高亮 / Markdown / 源码切换 | ✅ 照做；07 顺带改了 `syntax-highlighting`，08 是 07 留位的兑现 |
| 09 image-preview | ✅ 顺带改了 `service-startup` 的 API 分区承诺 |
| 10 add-file-search | ✅ |
| 11 / 12 编辑与冲突 | ❌ 未做（本项目到此为止） |
| 13 / 14 远程 + Token | ✅ 合并成一个 Change |
| 15 认证行为变化演练 | ✅ 取消（演练已由前面几次完成） |
| — | ✅ 计划外新增 17 improve-root-confinement（安全前置，插队到 10 前） |
| — | ✅ 计划外新增 16 polish-file-browser、18 add-directory-tree、19 improve-modified-time-display |

**结论：计划是假设，Change 序列会随现实重排。** 该合的合（13+14）、该取消的取消（15）、该长出来的长出来（16/17/18/19）。OpenSpec 的价值不在于严格照计划执行，而在于每次偏离都留下可追溯的理由。

### 最终产出

**8 个能力**（`openspec/specs/`）：

`service-startup`（启动契约与 HTTP 分区）、`directory-browsing`（目录列表与导航）、`text-preview`（文本查看）、`syntax-highlighting`（代码高亮）、`markdown-preview`（Markdown 渲染与源码切换）、`image-preview`（图片）、`search`（文件名搜索）、`authentication`（远程访问凭证）。

**4 条架构决策**（`docs/adr/`）：0001 Go 单二进制；0002 前后端 JSON API + 内嵌零构建静态前端；0003 呈现层按性质分派验收；0004 默认只读、扩大暴露面须显式开启。

**15 个归档 Change**，git 历史是一致的 `feat: <change>` / `archive: <change>` 配对。

唯一天然缺口：文件编辑（11 / 12）未做。

---

## 13. 可复用经验清单

1. **规格只写行为，实现进 design。** 判断标准：这句话用户看得见、能观察到吗？
2. **可调旋钮不进规格。** 数值、库名、色值放 design，改它们不该动任何 Scenario。
3. **改已有行为用 MODIFIED，且完整复制旧 Requirement。** 只写半句，归档时替换不了。
4. **推翻 design 不等于 MODIFIED。** 先看主规格里有没有旧条款；没有就是 ADDED。
5. **写 BREAKING 候选行为时，顺手写下翻案时机。** Change 17 零争议，全靠 Change 02 当年预约了「13 的前置」。
6. **测试写在 apply，verify 只复核。** 别让 verify 重写测试。
7. **守门人测试要被证伪一次** 才算数。
8. **apply 里发现范围外的东西就停下摆出，别顺手做。** 记进 roadmap 想法池或另立 Change。
9. **计划有误先 update artifacts 再 apply。** 不让代码跑在文档前面。
10. **未归档用 update，已归档开新 Change。** archive 不可改。
11. **换会话的合法性看磁盘交接物。** explore 必须和 propose 同会话。
12. **破例要重新问、并留记录。** 上次破过不代表这次可以顺手破。

---

真正建立起来的，不是「会用几条命令」，而是这条习惯：

```
想法 → 探索 → 提议 → 规格 → 设计 → 任务 → 实施 → 核对 → 归档 → 下一次变化
```

系统当前是什么，看 `openspec/specs/`；它为什么变成这样，看 `openspec/changes/archive/`。OpenSpec 让「需求变化」这件事本身，也变成有据可查的历史。
