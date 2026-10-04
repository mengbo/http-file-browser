# Design: improve-root-confinement

## Context

现状：三个端点（list / content / image）共用 `internal/server/browse.go` 的 `resolve()`，全程按字面路径处理、不调用 `filepath.EvalSymlinks`——这是 Change 02 design D3 的决策，spec 三处（`directory-browsing / Root directory confinement` 正文、`text-preview` 与 `image-preview` 的出根软链 Scenario）如实承诺了这一强度。动机与翻案时机见 proposal.md 的 Why。本 Change 只动判定基准，不动任何端点语义、识别规则与错误词汇。

## Goals / Non-Goals

**Goals:**

- 根目录内经符号链接指向根目录之外的路径一律拒绝（`outside_root`），三个端点一致。
- 根内符号链接（指向根目录内另一位置）继续可用——收紧不得过度阻塞。
- 根目录路径自身经符号链接的环境（macOS `/tmp` → `/private/tmp`）零回归。
- spec 错误词汇零新增：拒绝复用 `outside_root`，悬空软链维持 `not_found`。

**Non-Goals:**

- 不把符号链接做成第三类条目：`Entry type distinction` 与 `Entry metadata` 零 delta，安全修复与 UX 改进解耦，不焊在一次 Change 里。已知的关联取舍继续留在想法池（「符号链接的条目类型」「符号链接的大小是链接自身的长度」两条，出处为 Change 02 归档 design 的 D12）：指向目录的符号链接在列表里显示为 `file` 且不可点击，其大小是链接自身的长度而非目标内容的大小；两者的修法都是引入 `type: "symlink"` 第三类条目，属独立的 UX Change，本 Change 不做，其措辞待本 Change 归档时随状态联动复核。
- 不提供任何开关（`-follow-symlinks` 之类）：见 D1，开关等于把洞原价买回。
- 不做 TOCTOU 的内核级封堵：见 D5，威胁模型下不取 Linux-only 方案。
- 不改条目元数据的 lstat 语义：列表里符号链接仍报链接自身长度（`An entry is a symbolic link` Scenario 不变）。

## Decisions

### D1. 一律物理判定，无开关；「回环宽松、远程严格」一并否决

备选与落选理由：

- **`-follow-symlinks` 开关**：默认安全、显式买回，看似两全；实际是把洞原价买回并给它一个体面的名字。真正的用例「想浏览根外的目录」有更自然的答案——根目录是用户自己选的，把根指过去即可。根就是边界，为软链在边界上开预售票违背零配置初衷。
- **按监听地址切换强度**（回环字面、远程物理）：让 `directory-browsing` 的 confinement 变成 `service-startup` 配置的函数，两个 capability 被焊死；spec 要为「同一请求、不同强度」写条件 Scenario，测试矩阵翻倍。一律物理判定让 confinement 永远只是目录浏览自己的事。

### D2. 判定机制：启动时解析根一次，请求时解析路径一次

- 服务启动时对根目录执行一次 `filepath.EvalSymlinks`，得到**物理根**；此后每请求不再重复解析根。
- `resolve()` 在既有字面检查（绝对路径拒绝、`Clean`/`Join`、`Rel` 越界判定——全部保留，它们继续拦截 `..` 与兄弟前缀这类字面逃逸）之后追加物理判定：对拼接出的绝对路径执行 `filepath.EvalSymlinks`，对结果与物理根做 `filepath.Rel`，`..` 开头即 `outside_root`。
- **两侧都解析**，Change 02 D3 的对称性论证在物理层原样重现：只解析一侧会因 `/tmp` → `/private/tmp` 这类环境产生假越界，两侧都解析则判定基准一致。当年的论据没有被推翻，是换了层。

### D3. `EvalSymlinks` 失败时回落到既有 Stat 分类

`EvalSymlinks` 报错（ENOENT / EACCES / ELOOP）意味着路径中存在缺失、不可读或成环的组件——同一组件序列下后续的 `os.Stat` 也必然失败。因此解析失败时**不做越界判定**，直接放行到端点既有的 Stat 分类路径：悬空软链得 `not_found`，无权限得 `permission_denied`，成环落入 `classify` 的 default 分支。回落不重开洞：解析不出来的路径一个字节都提供不出去，而分类词汇与今日完全一致。

### D4. 错误词汇零新增

- 出根（无论经 `..` 还是经软链）统一 `outside_root`：客户端对两种越界没有不同的动作可做，区分不产生可观察价值。
- 悬空软链 → `not_found` 是既有行为（今日 Stat 即失败于此），经 D3 回落自然保持，不加 Scenario、不改 failure 条目。

### D5. TOCTOU 窗口：记录为已知限制，不封堵

`resolve()` 判定与端点后续 `os.Stat` / `os.Open` 之间存在窗口，本机进程在此窗口内替换符号链接可竞赢判定。本 Change 防御的对象是**远程网络请求**（13 之后的同网段主机），不是本机进程——后者本就能直接读文件系统，服务器不给它任何增益。内核级封堵（Linux `openat2` + `RESOLVE_BENEATH`）是 Linux-only，与 ADR-0001 的跨平台交叉编译价值冲突，不取。此限制记入本节与代码注释，不做虚承诺。

### D6. Windows 行为：apply 期实证，不预先假设

NTFS junction 与 8.3 短名在 `filepath.EvalSymlinks` 下的具体行为（junction 是否按符号链接解析、短名是否被展开为长名）不在本设计断言。apply 时在 Windows 上实证（或依据运行时行为调整测试策略）：结论只影响测试的取舍，不影响 spec——spec 只承诺「解析后物理位置在根内」，这一承诺与解析器的具体行为正交。

### D7. spec 落点分工：政策正文只住在 directory-browsing

判定基准（物理位置、根目录的物理位置）写在 `directory-browsing / Root directory confinement` 的正文——「根目录边界」这个主题属于它。`text-preview` 与 `image-preview` 的 SHALL NOT 正文句（「不返回根目录之外的文件的内容/图片内容」）本就不指明判定基准，保持原样；变的是各自那条出根软链 Scenario（翻转为拒绝并点名 `outside_root`）与 failure 条目里 `outside_root` Scenario 的 WHEN 措辞（「经字面解析后」→「解析后」，消除与新政策的自相矛盾）。共 6 条 MODIFIED、3 个 capability，无 ADDED。

### D8. 呈现端零变化，走查一条

错误信封的前端呈现是端点无关的通用逻辑，`web/` 无需任何改动。浏览器走查只做一条：点一个出根软链（或直接请求其路径），确认错误说明正常出现——与 Change 09 的回退说明走查同一强度。

## Risks / Trade-offs

- **[BREAKING：本地出根软链失效]** → 已在 proposal 标注；缓解即产品语义本身：想浏览什么，把根指过去。根内软链（D2 保证）覆盖「目录内整理用链」的主要场景。
- **[TOCTOU 竞争窗口]** → 威胁模型下接受（D5）；注释如实记录，不做超出实现强度的暗示。
- **[Windows 解析行为未知]** → D6 apply 期实证；最坏结果是 Windows 专属用例的取舍调整，spec 不动。
- **[测试环境的软链差异]**（macOS `/var/folders` 本身是真目录，而 `/tmp` 是软链）→ fixture 不依赖环境巧合：用 `os.Symlink` 自建出根/根内/悬空三种链接；「根自身经软链」用包住 `t.TempDir` 的软链作根，跨平台可复现。
- **[既有三处宽松断言]**（`browse_test.go` / `content_test.go` / `image_test.go` 各一）→ 反转为期待 `outside_root`，不删除：同一 fixture 继续钉住「物理出根必拒」。

## Migration Plan

不适用。无数据迁移；单二进制内嵌前后端，无版本错配窗口。行为收紧随部署立即生效，回滚即撤销本 Change 的提交。

## Open Questions

- D6 的 Windows 实证结果（junction / 8.3 短名的解析行为）——影响测试取舍，不影响 spec、方案与任务划分，可在 apply 期回答。
