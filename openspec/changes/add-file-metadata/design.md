# Design

## Context

当前状态见 `openspec/specs/directory-browsing/spec.md` 与 Change 02 的归档 design。本 design 只需要重申三条会被本 Change 触及的既有约束：

- **Change 02 的 D3**：越界判定全程按字面路径处理，不解析符号链接。系统只承诺「字面位置在根目录内就按该路径提供内容」。
- **Change 02 的 D12**：条目类型用 `os.ReadDir` 的 `DirEntry.IsDir()`，它对符号链接返回 false，因此指向目录的软链按字面报告为 `file`。
- **Change 02 的 D8**：前端条目名一律用 `textContent` 写入，绝不拼接 `innerHTML`——这是安全不变量，不是风格偏好。
- **ADR-0002**：前端为零构建 vanilla JS，后端只提供 JSON API。

还有一条与 scope 直接相关的事实：**文件条目当前不可点击**（`web/app.js` 把它渲染成 `<span>` 而非链接），因此系统里不存在「脱离目录列表查看某个文件」的场景。这是 D1 的前提。

## Goals / Non-Goals

**Goals:**

- 目录列表的每个条目带上大小与最后修改时间，取值规则由 spec 承诺。
- 不削弱 Change 02 已确立的任何安全边界（纯字面越界判定、`textContent` 不变量）。
- 让「内容读取」这道门从一句 spec 措辞变成一个可测的 Scenario，以便 `text-preview` 必须有意识地跨过它。
- 元信息的加入不改变列表的顺序语义，也不改变错误信封。

**Non-Goals（design 层面，proposal 已列产品范围）:**

- 不引入任何新的接口、参数或响应顶层字段。`GET /api/list` 的顶层形状不变。
- 不引入文件类型分类（文本/图片/Markdown/二进制）。
- 不引入分页、缓存或响应压缩。
- 不为 `size` 引入第二种「无值」表示。
- 不引入任何第三方依赖。

## Decisions

### D1. 元信息随列表返回，不新增详情端点

`listEntry` 直接增加两个字段，不开 `/api/file`。

**为什么**：文件当前不可点击，系统里没有「脱离列表查看文件信息」这个消费场景。为一个还不存在的场景设计端点，等于提前替 Change 04 做决定——Change 04 一旦落地，文件是否可点、`text-preview` 的入口长什么样，都会反过来影响这个端点是否值得存在。

**备选：独立详情端点 `/api/file?path=`**。它的好处是列表响应保持轻量、大目录代价不增。坏处有三：文件得先变成可点（那是导航行为，属于 `directory-browsing` 而不是元信息）；前端要引入「正在取哪个条目详情」的状态；列表视图将没有大小与日期两列，与「像 Finder 一样浏览」这一产品目标正面相悖——Finder 的列表视图里这两列正是最常被扫视的。

**备选：`?meta=1` 开关**。同一端点返回两种响应形状，spec 要同时约束两套，客户端也要分支。比任一固定形状都差，不考虑。

### D2. 不产生新 capability，元信息作为 `directory-browsing` 的 ADDED Requirement

**为什么**：元信息只在目录列表这一处出现，它描述的是「列表条目长什么样」，与 `Directory listing response`、`Entry type distinction`、`List ordering` 是同一件事的四个侧面。拆成独立 capability 会让同一批字段在两处 spec 里各被描述一次，且 ADDED 与 MODIFIED 分落两个 capability 时，sync 阶段漏掉其中一个的风险明显更高。

**代价**：`docs/roadmap.md` 的 Capability 地图里原计划的 `file-metadata` 不成立，需要改写。这是本 Change 唯一一处需要改动 roadmap 的地方。

**备选：立 `file-metadata` capability**。当系统真的有了详情视图（文件可点之后）再立，那时它才有独立于列表的 Requirement 可写。

### D3. 拆开 `Directory listing response` 的排除项，并给「不提供内容」加一条 Scenario

原句是「系统 SHALL 在每个条目中只提供名称与类型，**不提供大小、修改时间或内容**」。这三样捆在一起，但性质不同：「大小、修改时间」是本次要推翻的，「内容」是给 `text-preview` 守住 partitioning 的门。

**为什么必须拆**：MODIFIED 要求带完整 Requirement，如果原地改这一句，最省事的写法是把三个词一起删掉——那就会连带松掉「不提供内容」，Change 04 的边界从此无人看守。拆开后，元信息的承诺由 ADDED 的 `Entry metadata` 承担，`Directory listing response` 只保留「不提供内容」这一半排除。

**额外加一条 Scenario `A listed directory contains readable files`**：让「响应不含条目内容」从一句措辞变成一个可断言的行为。这样 Change 04 想读文件内容时，是**有意识地推翻一条 Scenario**，而不是发现那道门根本没锁。

### D4. 用 `DirEntry.Info()`（lstat 语义）取元信息，不用 `os.Stat`

**这是本 Change 最容易做错、且错法最危险的一个决定。**

`DirEntry.Info()` 在 unix 上基于 `lstat`：它**检测**符号链接但**不解析**目标。而 `os.Stat` 会跟随。

```
                        DirEntry.Info()          os.Stat
符号链接 -> 目录         mode=L, size=链接长度      mode=d, size=目标目录 inode 大小
符号链接 -> 外部大文件   mode=L, size=链接长度      mode=-, size=外部文件的真实字节数
```

**为什么 lstat 是必须的**：Change 02 的 D3 承诺系统只做字面判定。若用 `Stat` 取大小，根目录内一个指向外部的软链会报出**根目录之外那个文件的真实大小**——服务在报告边界外文件的信息，而同一响应里的越界判定还在说「这在根目录内」。lstat 完全没有这个问题，也不需要触碰 D3 的任何承诺：检测不是解析。

**为什么这不推翻 D12**：D12 的第一条理由（「`IsDir()` 免去每个条目一次 stat」）在本 Change 之后失效——要大小和修改时间就必须 stat，省不掉。但 D12 的第二条理由（分类跟随链接会与越界判定不对称）防的正是 `Stat`，用 lstat 不触发它。**D12 本身一个字都不用改**，因为本 Change 不动 `type` 的取值。

**测试落点**：`Entry metadata` 的 `An entry is a symbolic link` 那条 Scenario，断言一个指向根目录外大文件的软链，其大小等于目标路径字符串的字节数（而非目标文件大小）。这条断言是 D4 唯一的守门人。

### D5. `size` 与 `modified_at` 用缺省，不用 `null`

`size` 字段「当且仅当 `type` 为 `directory` 之外时」出现；条目元信息不可得时，`size` 与 `modified_at` 都直接不出现在该条目里。

**为什么不是 `null`**：Change 02 的 D10 拒绝过给 `parent` 引入第二种「无值」表示，但那里的 `""` 意味着**两件不同的事**（根目录 / 一级子目录），是真正的歧义。这里 `null` 只意味一件事，语义上并不需要第二种表示。而在前端，`entry.size == null` 对 `null` 与 `undefined` 行为完全一致——**引入第二种表示的代价是零**，缺省还能省掉大目录的响应体积。既然代价是零，就不引入。

**备选：固定字段 + `null`**。形状更统一，代价是每个消费者都要写条件分支，且大目录响应体更大。为一个零收益的收益付费，不值。

### D6. 修改时间用自 Unix 纪元起的整秒

**为什么**：`List ordering` 有一条 Scenario 是「同一目录多次请求顺序完全相同」，这个 capability 已经在为**可复现**付费。修改时间沿用同一价值观——同一个文件在任何机器、任何时区、任何 locale 下都必须给出逐字节相同的值。

Unix 整秒是唯一满足这一点的表示。RFC3339 带偏移量的话，同一个文件在 `en` 和 `fr` 机器上会返回不同字符串，可复现性当场破掉，而且还得在 spec 里额外约定「用哪个时区」，那本身就是一个新的歧义源。

附带好处是避开 JSON number 的秒/毫秒歧义类——JavaScript 的 `Date` 构造器要毫秒，这个坑会反复咬人，而它恰恰是最容易在实现阶段犯的错。

**备选：Unix 毫秒**。同样可复现，但值大十倍，而亚秒精度对本工具没有实用收益（本 Change 已取整到秒）。
**备选：RFC3339 字符串**。可读性更好，代价见上。

**代价**：`curl` 里看到的是 `1758000000` 而非人话；亚秒精度被丢弃。后者已由 spec 的 `Modification time falls within the same second` 如实承诺，不是隐藏取舍。

### D7. 单条目元信息不可得时保留条目、省略元信息，不让整个列表失败

Go 文档明确写着：`DirEntry.Info` —「If the file has been removed or renamed since the directory read, Info may return an error」。要取元信息就绕不开这条路径，而它在一个繁忙目录（构建产物、`git checkout`、正在下载的文件夹）里会偶发触发。

三个选项里选「保留 + 省略」：

- **丢弃该条目** → 列表对「存在性」说谎，客户端无法区分「文件不存在」和「文件刚好被删了」。
- **整体失败** → 最差：`npm install` 期间那个目录直接打不开。目录列表的首要性质是健壮，不是精确。
- **保留 + 省略**（选中）→ 列表完整且诚实，客户端对「有名字没大小」的行有明确的降级路径。与 D5 的缺省表达是同一个形状，客户端处理方式一致，不需要区分「不适用」与「不可得」。

### D8. 前端用 CSS grid 多列呈现，节点数从 2 涨到 4

名称、大小、修改时间各占一列，容器用 grid 对齐——接近 Finder 列表视图的观感。用不用 grid 不影响节点数，两种做法都是 4 个节点。

**关键约束**：Change 02 的 D8（`textContent`，绝不 `innerHTML`）在本 Change 第一次真正被考验。节点数翻倍之后，「用模板字符串拼一行」是最省事的写法，也正是 D8 要防的那件事。**新增的两个文本节点必须与条目名走同一条 `textContent` 路径**，这一点要作为显式验证项进 tasks，而不是靠 apply 阶段自觉。

**呈现细节（spec 不约束，design 定方向）**：大小渲染为人类可读形式（`1.2 MB`）而非原始字节数；修改时间渲染为本地时区的可读形式（D6 保证的是**取值**可复现，不是**显示字符串**可复现，这两件事不要混）。目录的大小一栏留空。

**零条目时的表头**：条目列表为空时（含目录为空与访问失败两种情形）隐藏列名行。一个描述零行的表头没有意义，而「列名行」本身是本 Change 才引入的东西——Change 02 的单列列表不需要表头，所以这条不构成对既有呈现的改变。

**窄视口**：三列全部保留，只收窄固定列宽并调小字号。某列在窄屏上直接 `display: none` 会让用户在**没有任何提示**的情况下少看到一整类元信息，那是产品取舍而非表现层细节，超出本 Change 的授权范围。

### D9. 为条目元信息取数保留一个可注入的 seam

`Entry metadata` 的 `An entry's metadata cannot be obtained` 这条 Scenario **无法确定性构造**：要让 `DirEntry.Info()` 失败，就得在 readdir 之后、`Info()` 之前删掉那个文件。用「建一堆文件 + 后台 goroutine 删」去撞是 flaky 的，而 flaky 测试比没有测试更糟——它会训练团队忽略红色。

所以 `browser` 结构上保留一个可注入的取数函数，默认实现即 `DirEntry.Info`，测试可替换为「对指定条目返回错误」。

**这不是超出任务范围的私自加参数**：Change 01 的 design D7 已经为 `app.run` 确立了注入式测试 seam 这个模式，本 Change 沿用同一形状，且**在 design 与 tasks 里显式列出**，不靠 apply 阶段顺手加。Change 02 的 journal 观察 3 记的正是「不加声明就加 seam」这一类越界——那次是把冲突摆出来问的，这次是在开工前就写进 artifacts。

**备选：接受这条 Scenario 不可测，`t.Skip` 并说明原因**。可行，但 journal 观察 4 已经记了教训——skip 之后必须回答「这条 Scenario 要防的东西现在由谁负责」。而这里防的东西很明确：防止将来有人把 `Info()` 的错误顺手 `continue` 掉（等于丢弃条目）或让整个列表失败。两者都是真实会犯的错，值得一个确定性测试来钉住。seam 的成本是一个函数字段，远低于这个风险。

### D10. 不引入 `kind` 字段

**为什么**：「是不是文本」「能不能高亮」「是不是 Markdown」是 `text-preview`（Change 04/05/07）的判断。本 Change 抢先定义分类，等于替后续 capability 做决定，性质上与 Change 02 journal 里记录的现象二（同一件事横跨两个 capability）是同一类错误，只是方向反过来。

图标同理不需要服务端字段：条目名已经带着扩展名，客户端自行推导即可。roadmap 把图标留在 `polish-file-browser`。

## Risks / Trade-offs

- **符号链接的大小是链接自身长度，用户会困惑**（指向 100KB 文件的软链显示「11 B」）→ 这是 D12 已知 wart 的延伸，不是新问题。spec 已如实承诺（`An entry is a symbolic link`），不藏。彻底修需要引入 `symlink` 第三类条目类型，那是独立决定，本 Change 明确不做，已登记在想法池。
- **每个非目录条目多一次 lstat** → 大目录延迟上升。本 Change 不分页（想法池已登记该债），因此这笔债从「响应体大」升级为「响应体大 + N 次 lstat」。分页的必要性被推高，但不在本 Change 解决。
- **前端节点数翻倍** → D8 的 `textContent` 不变量压力上升；万级条目的目录渲染成本明显恶化。缓解：D8 已列为显式验证项；虚拟滚动属表现层，留给后续。
- **「有名字没大小/没时间」的行看起来残缺** → D7 的自觉取舍。换来的是繁忙目录始终可浏览。
- **`modified_at` 整秒截断** → 秒内发生的两次修改无法区分。spec 已承诺；对一个文件浏览器无实际影响。
- **归档后 capability 的 Purpose 会与自身 Requirements 矛盾** → `openspec/specs/directory-browsing/spec.md` 的 Purpose 写着「不包含文件元信息与文件内容读取」，本 Change 让前半句变假。archive 只合并 Requirement，不重写 Purpose；而 `AGENTS.md` 禁止直接修改 `openspec/specs/`。**这一句必须留到 archive 阶段处理**，见 tasks 的收尾任务。

## Migration Plan

不适用。无数据迁移，无既有部署。响应条目的新增字段对忽略未知字段的消费者是向后兼容的；错误信封、排序语义、越界判定均不变。回滚方式为撤销本 Change 的提交。

## Open Questions

以下问题可推迟到后续 Change 回答，届时不改变本 Change 的 spec、方案与任务划分：

- **相对时间显示**（「3 天前」）。若要做，同一列表内所有相对时间必须共享同一个「现在」参照，否则基准会随渲染时刻漂移，同一页面刷新前后显示不同。纯呈现层。
- **大目录的虚拟滚动**。本 Change 让节点数翻倍，问题更明显了，但解法与本 Change 的元信息无关。
- **文件/目录图标**。roadmap 第 30 行把它列进 MVP，但 Change 地图 03–16 没有任何一条以图标为目标。本 Change 明确不接手，需要在 `polish-file-browser` 认领或在 Change 地图里补一条。
