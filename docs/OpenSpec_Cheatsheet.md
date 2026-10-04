# OpenSpec 速查表（http-file-browser）

> 放手边的速查：命令、文件、术语、决策。想读来龙去脉看 [实战总结](OpenSpec_Retrospective.md)。
> 命令形态以本项目 `openspec init` 后生成的文件为准（此处为 `/opsx-propose` 写法，别的工具可能显示成 `/opsx:propose`）。

## 0. 一句话

```
specs/     现在是什么
changes/   正在变什么
archive/   怎么变成这样的
```

---

## 1. 文件放哪

| 路径 | 放什么 |
|---|---|
| `openspec/specs/<能力>/spec.md` | 系统当前行为（每个能力一份） |
| `openspec/changes/<名称>/proposal.md` | 这次为什么做、改什么 |
| `openspec/changes/<名称>/specs/<能力>/spec.md` | 这次要改的 delta（不是整份规格） |
| `openspec/changes/<名称>/design.md` | 这次怎么做 |
| `openspec/changes/<名称>/tasks.md` | 可打勾的任务清单 |
| `openspec/changes/<名称>/.openspec.yaml` | 变更元数据（CLI 生成，别手写） |
| `openspec/changes/archive/<日期>-<名称>/` | 归档的历史（不可改） |
| `openspec/config.yaml` | 项目级背景与规则 |
| `docs/adr/NNNN-标题.md` | 长期架构决策 |
| `docs/roadmap.md` | 长期规划与 Change 地图（活文档） |
| `docs/journal.md` | 学习实验观察 |

---

## 2. 关键术语（含解释）

| 术语 | 中文 | 它到底代表啥（人话） |
|---|---|---|
| Spec | 规格 | 一份「这东西现在会做什么」的说明书。只写用户看得见的行为，不写代码怎么实现。 |
| Capability | 能力 | 软件能替用户做的一件独立的事，比如「浏览目录」「看文本」「看图」。一件事一份规格。 |
| Change | 变更 | 一次改动从头到尾的整个过程，比如「让没有扩展名的文本也能打开」。从提出到归档算一个。 |
| Delta spec | 变更规格 | 这次「要改哪几条」，不是把整份规格重抄一遍。做完归档时拿它替换旧条款。 |
| Archive | 归档 | 做完的变更搬进去的档案室，之后不能再改。翻这里能知道系统当初为什么变成今天这样。 |
| Proposal | 提案 | 一次变更开头的立项说明：为什么做、这次到底改什么。 |
| Design | 设计 | 这次具体怎么实现，比如选什么技术、分几个模块。「做什么」归规格，「怎么做」归设计。 |
| Tasks | 任务 | 一份能一条条打勾的待办清单，每条都得能验证做没做。 |
| Requirement | 需求条目 | 规格里的一条行为要求，比如「用户点文件夹，就进入这个文件夹」。 |
| Scenario | 场景 | 挂在需求下面的一条具体例子：在什么情况下、会看到什么结果。能直接拿去当测试用。 |
| Purpose | 用途说明 | 每份规格开头的一小段话，说清这能力是干嘛的、不包括啥。注意：归档时电脑只搬需求条目，不会自动改这段话。 |
| ADDED / MODIFIED / REMOVED / RENAMED | 四种改法 | 分别是：新增一条 / 改一条已有的 / 删一条 / 只改个名字。用错会出乱子。 |
| Explore | 探索 | 动手前先把问题聊清楚。不写代码、不出正式文件，聊出来的东西基本只留在对话里。 |
| Propose | 提变更 | 把想清楚的写成四份文件：提案、规格、设计、任务。这一步只写字，不写代码。 |
| Apply | 实施 | 照着任务清单真正写代码、写测试。 |
| Verify | 核对 | 做完后啥也不改，只回头对账：规格要求的和实际做的对不对得上。 |
| Update | 改计划 | 做到一半发现计划不对时，先回去改提案/规格/设计/任务，再接着做——不是偷偷改代码。 |
| Sync | 同步规格 | 把变更规格并进主规格，但这次变更还不算结束，仍算进行中。 |
| Archive | 归档（动作） | 变更全做完后收尾：把变更规格并进主规格，再把整个变更搬进档案室。 |
| ADR | 架构决策记录 | 记「当初为什么这么选」，比如为什么用 Go、为什么打成单个文件。省得同一个问题以后反复吵。 |
| `/opsx-*` vs `openspec` | 两种命令 | 前者在聊天框里输入、指挥 AI 干活；后者在终端里敲，是命令行工具。 |

---

## 3. `/opsx-*` 命令速查

| 命令 | 一句话 |
|---|---|
| `/opsx-explore` | 只想不写，把问题聊清楚 |
| `/opsx-propose` | 立变更 + 一次写完全部规划文件 |
| `/opsx-new` | 只立骨架、看模板，什么都不写 |
| `/opsx-continue` | 接着写下一份文件 |
| `/opsx-ff` | 把缺的文件一次补齐到可开工 |
| `/opsx-apply` | 照任务写代码、写测试 |
| `/opsx-verify` | 只核对，不改任何东西 |
| `/opsx-update` | 改计划文件，不碰代码 |
| `/opsx-sync` | 规格并进主规格（变更仍进行中） |
| `/opsx-archive` | 合并规格 + 搬进档案室 |
| `/opsx-bulk-archive` | 一次归档多个变更 |
| `/opsx-onboard` | 手把手教学走一遍 |

---

## 4. openspec CLI 速查（含参数）

终端里敲的命令。`--store <id>` 属多仓库（beta）用法，本项目未用，可忽略。

**装与配**

| 命令 | 干什么 | 常用参数 |
|---|---|---|
| `openspec init [path]` | 装 OpenSpec，生成 `openspec/` 与 AI 工具文件 | `--tools all\|none\|claude,cursor`；`--language zh-CN`；`--force`；`--profile core`；`--no-animation` |
| `openspec update [path]` | 把已装的命令/技能文件刷到当前 CLI 版本 | `--force` 即使已最新也重写 |
| `openspec config` | 看/改全局配置 | `list`、`get <键>`、`set <键> <值>`、`unset <键>`、`reset --all`、`edit`、`profile core` |

**看**

| 命令 | 干什么 | 常用参数 |
|---|---|---|
| `openspec list` | 列变更（默认）或规格 | `--specs`、`--changes`、`--archived`、`--all`、`--sort recent\|name`、`--json` |
| `openspec show <名>` | 打印某个变更或规格 | `--json`、`--type change\|spec`、`--diff`（变更：附需求差异）、`--deltas-only`、`--no-scenarios`、`-r <序号>` |
| `openspec view` | 一屏仪表盘：草稿 / 进行中 / 已完成 / 已归档 | — |

**校验与归档**

| 命令 | 干什么 | 常用参数 |
|---|---|---|
| `openspec validate [名]` | 检查结构问题，并预演归档时的合并 | `--all`、`--changes`、`--specs`、`--archived`、`--strict`、`--type`、`--json`、`--report full\|findings`、`--concurrency <n>`、`--no-interactive` |
| `openspec archive [名]` | 归档完成的变更，并把 delta 合并进主规格 | `-y/--yes` 全部确认、`--skip-specs` 不动主规格、`--no-validate`、`--json` |

**工作流（给 AI 的底层命令）**

| 命令 | 干什么 | 常用参数 |
|---|---|---|
| `openspec new change <名>` | 只建变更目录 + `.openspec.yaml` | `--description <文本>`、`--goal <文本>`、`--schema <名>`、`--json` |
| `openspec status` | 看某/所有变更的 artifact 进度与下一步 | `--change <名>`、`--all`、`--schema`、`--json` |
| `openspec instructions <artifact\|apply\|archive>` | 取某一步的写作 / 实施 / 归档指引 | `--change <名>`（必填）、`--schema`、`--json` |
| `openspec templates` / `openspec schemas` | 取模板路径 / 列可用 schema | `--json` |

**其它**：`openspec version`、`openspec completion install`。官方原文：openspec.dev/docs/cli。

---

## 5. 选路决策表

| 你的情况 | 用哪个 |
|---|---|
| 不知道该用哪个 / 下一步干嘛 | 直接问 AI，它懂这套流程（不用背命令） |
| 需求清楚、变更简单 | `/opsx-propose` → review → `/opsx-apply` → `/opsx-archive` |
| 需求不清楚 / 要比较方案 | 先 `/opsx-explore` 聊清楚，再 propose（**同一会话**） |
| 想一个文件一个文件地 review | `/opsx-new` → `/opsx-continue` 反复 |
| 实施中发现计划不对 | `/opsx-update` 改 artifacts，再 `/opsx-apply` |
| 交付前想对账 | `/opsx-verify` |
| 只想先把规格并进主规格 | `/opsx-sync` |
| 完成、要收尾 | `/opsx-archive` |
| 一次归档多个 | `/opsx-bulk-archive` |
| 第一次上手 | `/opsx-onboard` |

---

## 6. 一个新 Change 的标准节奏（可照抄）

1. `/opsx-propose <名称>`（不清楚先 `/opsx-explore`）
2. Review：**提案 → 规格 → 设计 → 任务**，重点是规格。
3. `/opsx-apply`：按任务实现、写测试，完成一项勾一项。
4. 发现计划不对 → `/opsx-update` 改 artifacts，再继续 apply。
5. `/opsx-verify`：只读对账。
6. `openspec validate`
7. `/opsx-archive`
8. 同步 `docs/roadmap.md` 状态与 `README` 当前状态。

---

## 7. 四种 delta 判据

| 情况 | 用 | 硬规则 |
|---|---|---|
| 新增一条行为 | ADDED | 至少一个 Scenario |
| 已有行为发生变化 | MODIFIED | **完整复制旧 Requirement 再改** |
| 删掉一条行为 | REMOVED | 写 Reason + Migration |
| 只是改名，行为没变 | RENAMED | 别和 MODIFIED 混 |
| 推翻的是 design 而非规格 | ADDED | 主规格里没有旧条款，就没有 MODIFIED 的对象 |

---

## 8. 新会话切点

| 接缝 | 磁盘上有交接物吗 | 能否 `/new` |
|---|---|---|
| explore → propose | ❌ explore 不落盘 | **不能**（同一会话） |
| propose → apply | ✅ artifacts 齐了 | 能 |
| apply → verify | ✅ 代码 + 任务 | 能 |
| verify → archive | ✅ | 能 |

要在 explore 后切，先把结论落盘：捕获成 Change 的 artifacts，或写进 roadmap / ADR。

---

## 9. 归档前检查清单

- [ ] `tasks.md` 全部 `[x]`
- [ ] `openspec validate` 通过
- [ ] `/opsx-verify` 对账过，无未处置的 CRITICAL / WARNING
- [ ] delta 的 ADDED / MODIFIED / REMOVED / RENAMED 用对了
- [ ] MODIFIED 完整复制了旧 Requirement
- [ ] 没有把实现细节写进 Requirement
- [ ] 归档后：`specs/` 成为新事实；`archive/` 能解释某行为为什么存在

---

## 10. 常见反模式

- 把「用 XX 库 / XX 技术」写进 Requirement。
- 改已有行为却用 ADDED，另起一个同名新能力。
- MODIFIED 只写半句（「此处也支持 X」）。
- 只是改名却用 MODIFIED。
- 实施中顺手加了任务书之外的功能。
- 直接手改 `openspec/specs/` 或 `openspec/changes/archive/`。
- 归档后又回头改那个历史 Change。
- explore 之后 `/new` 再 propose（讨论会丢）。

---

## 11. 常用 Prompt 模板（精简版）

**Explore**
```
/opsx-explore <需求>
先读相关 specs 与代码，别写代码、别出正式文件。
请：解释当前行为 / 找出需求里不明确处 / 判断是新增能力还是改已有能力 /
给方案与影响 / 指出现在必须决定和可以推迟的事。别替我做未经确认的产品决策。
```

**Propose**
```
/opsx-propose <需求>
先读 openspec/specs/、当前 active changes、相关 capability。
判断新增还是修改已有能力，不要重复造已有能力。
写 proposal / delta specs / design / tasks。
规格只写可观察行为，实现进 design，任务要能逐项验证。
有未决策的问题先指出，不要擅自决定。完成规划后停下，别写代码。
```

**Apply**
```
/opsx-apply
先读 proposal、specs、design、tasks，严格按 tasks 执行，完成一项勾一项。
为关键行为加测试。不自行增加本 Change 未定义的功能。
发现规格有问题就暂停说明；不要用改代码掩盖需求变化。
```

**Verify**
```
/opsx-verify
逐步检查：目标是否达成 / 每条 Requirement 是否有实现 / 每个 Scenario 是否可验证 /
有没有规格外多做 / 有没有 tasks 勾了但行为没做 / 与 design 是否一致 / 安全问题与边界。
不要改代码。输出：已满足 / 不满足 / 额外行为 / 建议。
```

**Update**
```
/opsx-update
实施中发现：<问题>
重新检查 proposal / specs / design / tasks：
判断是实现问题还是需求问题、哪些 artifact 要改、哪些已完成任务仍有效。
先更新 artifacts 使计划自洽，不要直接改业务代码。
```

**Archive**
```
/opsx-archive
先确认 tasks 全部完成、delta 有效、实现已验证。
无阻塞则归档：delta 正确合并进主 specs，Change 完整移入 archive，不改历史。
```
