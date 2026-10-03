# Design

## Context

三条会被本 Change 触及的既有约束：

- **Change 02 的 D3**：越界判定全程按字面路径处理，不解析符号链接。系统只承诺「字面位置在根目录内就按该路径提供内容」。
- **Change 02 的 D8**：前端条目名一律用 `textContent` 写入，绝不拼接 `innerHTML`。
- **Change 03 的 D4**：条目元信息走 `DirEntry.Info()`（lstat 语义）取数，因为 `os.Stat` 会跟随符号链接，让根目录内指向外部的软链报出根目录之外那个目标的真实大小，而同一响应里的越界判定还在说「这在根目录内」。
- **Change 03 的 D6**：修改时间用自 Unix 纪元起的整秒，这个 capability 一直在为**可复现**付费。
- **ADR-0001**：零第三方依赖、零构建、单二进制。**ADR-0002**：后端只提供 JSON API，前端零构建 vanilla JS。

一条与 scope 直接相关的事实：**文件条目当前不可点击**（`web/app.js` 把它渲染成 `<span>`），因此系统里不存在「脱离目录列表查看某个文件」的场景。这是 D3 的前提。

## Goals / Non-Goals

**Goals:**

- 文本文件内容经一个独立端点提供，其响应形状、判定规则与失败原因由 spec 承诺。
- `directory-browsing` 零 delta——既有八条 Requirement 与它们的 Scenario 一字不改，既有测试无需修改。
- 沿用 Change 02 已确立的「浏览位置镜像到地址栏」不变量，刷新停原位、前进后退可用、文件位置可作深链接。
- 让 Change 03 留下的那道门（「列表不提供条目内容」）保持关着，同时让内容仍然可读。
- 把内容嗅探的方案一次定清并落到可照抄的粒度，使 Change 05 不必重新调研（见 D7）。

**Non-Goals（design 层面，产品范围见 proposal）:**

- 不做任何类型识别（文本 / 图片 / Markdown / 二进制）的分类字段。
- 不做编码检测与转码。
- 不做截断预览。
- 不引入任何第三方依赖，也不引入任何子进程调用（`file(1)`）。
- 不引入 service worker、缓存、响应压缩。

## Decisions

### D1. 内容走独立端点；`directory-browsing` 零 delta，那道门不推

`GET /api/content?path=`。列表响应继续不携带任何条目内容。

**为什么这道门不需要推**：Change 03 的 design D3 花了很大力气把「大小、修改时间、内容」三样拆开，并加了一条 Scenario `A listed directory contains readable files` 让「响应不含条目内容」从一句措辞变成可断言的行为，读起来像是在给 Change 04 埋一道必须推的门。

但那条 Scenario 承诺的是**列表响应里不含条目内容**。走独立端点，这道门根本不用推——**要推翻它的反而是把内容内联进列表响应的做法**：一个含 500 个文本文件的目录会变成一次返回 500 份内容的请求。所以 Change 03 实际守住的是响应体体积，而独立端点让那扇门保持原样、毫无成本。

**代价**：Change 03 的 journal 观察 2 预言「Change 04 要读内容就是有意识地推翻一条 Scenario」——本 design 明确推翻这个预言本身。预言错了，而错的代价只是 Change 04 少一次 MODIFIED 演练，这正是我们要的。

**为什么不内联进列表**：一份目录 500 个文件就是 500 份内容进一个响应。Change 03 的 D7 已经确立了这个 capability 对「单个条目的问题不该让整体失败」的敏感度，内联内容会把那条敏感度一并破坏。

### D2. 浏览位置复用 `?path=`，端点语义解耦

`?path=` 在两个端点上指向不同类型的对象：

```
/api/list?path=a        -> a 是目录 -> 200 列表
/api/list?path=a/b.txt  -> b.txt 是文件 -> not_a_directory（现有 spec 已承诺）
/api/content?path=a     -> a 是目录 -> not_a_directory
/api/content?path=a/b.txt -> b.txt 是文件 -> 200 内容
```

前端不预判 `?path=` 指向什么，拿到响应按成败分派。

**为什么复用而不是引入第二个参数**：Change 02 确立了「浏览位置镜像到地址栏，刷新停原位、当前目录可作深链接、前进后退可用」。文件预览若不遵守同一条，刷新就丢位置——那是对一个已归档 Change 承诺的一致性打折。而 `web/app.js` 已经有 `currentPath()` / `urlFor()` / `popstate` 三个入口处理位置态，新增一路状态机是纯增量复杂度。

**关于「一个参数两种语义」的歧义**：这个不对称其实是类型化的——`not_a_directory` 已经是一个被 `Directory access failures` 承诺的机器可读 code，系统早就在用类型而非路径形态表达这件事。真正的歧义只存在于前端，而前端不需要预先回答它：调 `/api/list`，成功渲染列表，`not_a_directory` 则转调 `/api/content`。

**备选：`?file=` 或路径段式 `/file/a/b.txt`**。正交参数让两个概念显式分离，但地址栏要在「目录视图」与「文件视图」之间换一套 URL 规则，且 `/file/` 前缀会与 `HTTP surface partitioning` 的「根路径及静态资源 vs `/api/`」分区打架——静态路由要额外加一条规则，而那条规则不在本 Change 的授权范围内。

### D3. 「文件条目可进入」刻意不写成 Requirement

文件条目在前端变成可点击，但 spec 里**没有**任何一条 Requirement 承诺这件事。

**理由**：在 D2 的方案下，它没有任何服务端可观察行为。`/api/list` 的响应形状一字未变（这是 D1 零 delta 的直接结果），服务端不因「文件可点」与「文件不可点」而有任何差别。一个没有可断言对象的 Requirement 只会是装饰。

**归属判据**：Change 03 的 design D2 给了 capability 归属的一把尺子——**这条行为只在哪个场景出现**。元信息只在列表这一处出现，所以并入 `directory-browsing`；按同一把尺子，文件内容只在内容端点这一处出现，所以它属于新 capability。同一个判据，两个相反的结论，这比「我觉得它像导航」结实。

**Change 03 的 D1 怎么读**：D1 原文是「文件得先变成可点（那是导航行为，属于 `directory-browsing` 而不是元信息）」。这是一条**排除理由**（「所以本 Change 不必顺手解决」），不是**归属主张**（「所以它归 `directory-browsing`」）。D1 从未声称拥有它。本 Change 兑现的是排除。

**必须留痕的两处**，否则下一个读 spec 的人会以为它漏了：

1. `text-preview` 的 Purpose 明写本 capability 覆盖「从目录列表进入文本文件并查看内容」。
2. 本 design 这一节的标题本身就是声明。

（依据是 Change 03 的 journal 观察 6：Purpose 是散文不是定义，且 OpenSpec 官方路径允许直接编辑它。）

### D4. Change 03 的 `/api/file` 详情端点备选：排除理由失效，结论不变

Change 03 的 D1 明确否掉了「独立详情端点 `/api/file?path=`」，理由第一条是「文件得先变成可点（那是导航行为，属于 `directory-browsing` 而不是元信息）」。

**本 Change 让那个条目变成可点了，所以这条排除理由在字面上失效。** 但结论不必跟着失效：Change 03 需要的是元信息，而元信息已由列表携带；Change 04 需要的是内容，两者不重合。**详情端点在 Change 03 的语境下多余，在 Change 04 的语境下内容端点就是它**——只是它属于另一个 capability。

**为什么不顺手把 `/api/file` 复活成「元信息 + 内容」双端点**：那会让元信息有第二个来源，而 `Entry metadata` 的取值规则由 spec 承诺给列表响应一份。两处来源将来必然漂移。

### D5. 判定放在服务端

文件是否可读文本由**服务端**判定，不在前端。

**理由一（测试约定）**：ADR-0002 已把测试约定定成「每个 Scenario 对应一个 API 层的 Go 测试」。判定规则如果不在 API 上，就没有 Scenario 可写，也就没有测试可写——Change 05 将无从 MODIFIED。

**理由二（不漂移）**：前端硬编码一份扩展名清单，这份清单必然与服务端漂移。

**理由三（让 Change 05 有靶子）**：这是 Change 05 的 MODIFIED 演练得以成立的前提，见 D6。

### D6. 判定规则用扩展名白名单——一次明知故犯的不完整判定

```
本次的判定          扩展名 ∈ 已知文本扩展名清单  ->  可读文本
不是本次的判定      内容看起来像文本            ->  可读文本
```

**为什么明知故犯**：无扩展名的文本文件（`Makefile`、`LICENSE`、`.gitignore`、`go.mod`）是这个仓库自己就有的一批文件。内容嗅探比扩展名白名单**在工程上更对**——用户第一次打不开 `Makefile` 就是这一刀的真实代价。

**代价是自觉的，不是疏漏**：本次先发一个不完整的判定器，让它在真实使用中被撞到，Change 05 `improve-text-file-detection` 再用 MODIFIED 放宽。**需求变更是被现实撞出来的，不是被 roadmap 预约出来的**——而这个项目首要目标就是走通 SDD 全流程，「先窄后宽」是 spec 驱动最有说服力的演示之一：spec 怎么从窄变宽、为什么 MODIFIED 必须带完整 Requirement、既有测试怎么跟着欠债，这些在 D6 → Change 05 的路径上会真实发生。

**为什么清单是白名单而不是黑名单**：黑名单（列出所有非文本扩展名）在 Change 05 里几乎无法 MODIFIED——要枚举所有不该当文本的后缀，而 `.txt` 里装着二进制、`.bin` 里装着 JSON 这类反例会把枚举撑爆。白名单是有限可枚举的，Change 05 的放宽是「再纳入一个条件」，而不是「穷举剩下的」。

**清单本身进 design 不进 spec**：spec 承诺「扩展名属于已知文本扩展名」这个规则，具体清单是可变的实现细节。与 Change 03 的 `Entry metadata` 同一分工。

初始清单（可增删，实现为一个集合字面量）：

```
纯文本    .txt .text .md .markdown .log .csv .tsv
结构化    .json .yaml .yml .toml .xml .ini .conf .cfg .properties .env
标记      .html .htm .css .scss .sql
脚本      .sh .bash .zsh .py .rb .pl .php .lua
源码      .go .rs .java .c .h .cc .cpp .hpp .js .mjs .cjs .ts .tsx .jsx .vue .svelte
明确排除  无扩展名的一切文件（Makefile、LICENSE、.gitignore）——Change 05 处理
          图片、音视频、压缩包、字体、可执行文件的一切扩展名
          二进制数据类扩展名（.bin .dat .db .sqlite .parquet）
```

比较规则：扩展名取最后一个 `.` 之后的部分，**大小写不敏感**（`README.TXT` 与 `README.txt` 同等对待，与 `List ordering` 的大小写折叠价值观一致）。以 `.` 开头的文件名（如 `.gitignore`）**没有任何扩展名**——它整名就是名字，不存在「最后一个点之后」的部分。

### D7. 内容嗅探的方案在本次定下，供 Change 05 照抄

Change 06/07/09 之前，Change 05 就要用它。**本次已把方案定到可实施的粒度**，Change 05 不必重新调研。

```
判据        WHATWG MIME Sniffing 的 "binary data byte"：
            0x00–0x08、0x0B、0x0E–0x1A、0x1C–0x1F
            落在这些区间内的字节出现任何一个，该文件即不是可读文本

窗口        建议 4096 字节
            参照：git 8000 / WHATWG 规范 1445 / mimetype 4096 / libmagic 64 KiB
            net/http.DetectContentType 只读 512（规范写的是 1445，Go 自己的选择）
            512 太窄：实测 NUL 落在偏移 512 之后的 PNG 会被判成文本

不引入      net/http.DetectContentType —— 它先跑 17 条 HTML 签名 + 图片 + 音视频
            + 字体 + 压缩包签名，最后才兜底到 text。判据会被拖成「MIME 类型以
            text/ 开头」，spec 的一句话因此挂在一整张类型识别表上，
            与「一个 Requirement 对一个测试」的映射对不上
            libmagic —— 编译后 7.3 MB 魔数库 + 333 个源文件 + cgo，
            与 ADR-0001 明确看重的交叉编译价值冲突
            h2non/filetype —— 实测它完全不做文本判定，只匹配二进制魔数，
            对 ASCII / UTF-8 中文 / GBK / HTML 一律返回 Unknown
            x/net/html/charset —— +497 KB / 20 个包（拖进整个 HTML5 解析器），
            且 WHATWG 编码机制永远不返回 GBK / Shift_JIS / Big5，
            对本项目最需要的 CJK 场景无用

已知误判    无 BOM 的 UTF-16（每个偶数位一个 NUL）会被判成二进制 —— 而这
            正是 Change 05 要修的那一刀：它正是「扩展名白名单 + 内容嗅探」
            仍然不够时的剩余缺口
参照物      file 与 WHATWG 在三个字节上判定相反（0x0B / 0x1C file 说文本、
            标准说二进制；0x7F 反过来）。file 另有窗口 64 KiB、
            且把所有 0x80–0xFF 一律视为「可能是文本」。别照抄 file
```

**依据**：本项目是本地文件系统上的文件浏览器，HTML 规范在讲这段启发式时点名的正是这个场景——「Files from the local file system that contain bytes with values greater than 0x7F which match the UTF-8 pattern are very likely to be UTF-8」。

**同时提醒 Change 05 的一位后来者**：Change 05 是对 `Text file recognition` 的 MODIFIED，必须带完整 Requirement（含全部 Scenario），不是加一条新 Requirement。

### D8. 所有文件条目一律可点，非文本返回 `not_text`

前端把**每一个**文件条目都渲染成链接。点开非文本文件时，服务端返回 `not_text`，前端显示「这是二进制文件，无法以文本预览」。

**为什么不加列表字段**：另一种做法是在 `listEntry` 上加一个「可否预览」的布尔字段，只有文本文件是链接。它体验更干净，但代价是 `directory-browsing` 要 MODIFIED `Directory listing response`——**Change 04 就欠上 Change 03 那种 MODIFIED 债了**（连带既有测试断言一起改）。

**为什么不干脆不可点**：不可点与「这是二进制文件」是两种不同的信息。前者让用户面对一个不给任何解释的死条目，后者至少说明了一次尝试的结果。而这句提示本身就是 roadmap Phase 2 里「二进制文件提示」的粗版。

**备选：端点返回内容 + 一个「这是二进制」标记**。语义更松（端点替客户端猜），且 Change 09 图片预览要来推翻「非文本怎么办」。当前方案下 Change 09 只需要新增自己的分支，不必推翻既有 Requirement。

**`not_text` 是新增的错误标识**。既有四个（`not_found` / `not_a_directory` / `permission_denied` / `outside_root`）由 `Directory access failures` 承诺，语义绑定目录列表；内容端点的失败原因进 `text-preview` 自己的 `Content reading failures`，既有 Requirement 不动。

### D9. 编码一律按 UTF-8 处理，不做检测

无法解码为 UTF-8 的字节序列以替换字符呈现，spec 如实承诺。

**为什么不做检测**：WHATWG 的编码机制**结构性地**只会给出 `utf-8` / `utf-16be` / `utf-16le` / `windows-1252`，永远不返回 GBK / Shift_JIS / Big5——这符合规范，但对这个项目最需要的场景（中文目录里的 GBK 文件）无用。真正的 CJK 检测是统计性的（uchardet、ICU），是另一个重量级工具类别，没有零依赖的 Go 等价物。

**代价**：GBK 文件会显示成乱码或替换字符。这是自觉的取舍——**不检测比检测错更诚实**：一个自信地宣称「这是 GBK」但猜错的实现，比明摆着的乱码更难排查。

**登记**：这个取舍进 `docs/roadmap.md` 想法池，等有真实需求（用户抱怨中文乱码）时立独立 Change。

### D10. 1 MiB 上限，超限返回 `too_large` 而不是截断

上限 1 MiB（1048576 字节）。超过则返回 `too_large`，不给内容。

**为什么不静默截断**：截断之后用户以为看完了整个文件，而实际上没有。Change 03 的 D7 已经在为「宁可省略也不撒谎」付代价——但注意方向相反：D7 是**保留条目、省略元信息**，用户能看见那行残缺；截断内容则看不出来。返回错误让这件事完全可见。

**为什么需要上限**：不加上限，一次请求可以把一个 5 GB 的日志整个读进内存。这是可用性问题不是安全问题，但足以让服务进程崩掉。

**为什么 1 MiB**：源码文件几乎都在这个量级以内，而数据类大文件（CSV / JSON dump）本来也不是「预览」的用途。GitHub 的行内展示上限同为 1 MB。这个数字是可调的，实现时若与实测不符，改一个常量即可，不影响任何 Scenario。

**先 `Stat` 再判定**：存在性、是不是目录、是不是普通文件、大小都在打开文件之前用一次 `Stat` 拿到。这顺带避免了**打开 FIFO 会永久阻塞**的问题——`/tmp` 下的命名管道在 macOS 上并不罕见，一个永不返回的请求比任何错误响应都糟。

### D11. 前端 `textContent` 不变量扩展到文件内容；JSON 通道免疫 XSS

文件内容以 `<pre>` 呈现，`textContent` 写入。

**一个已评估但不存在于本设计的风险**：`net/http.DetectContentType` 会对任何以 `<!-- `、`<title `、`<p>` 开头的文本文件返回 `text/html; charset=utf-8`（HTML 签名表有 17 条）。若服务端按嗅探出的类型渲染预览，就是存储型 XSS。

**本设计不存在这个风险**，理由与 ADR-0002 直接相关：内容以 JSON 字符串字段返回，浏览器从不把它交给 HTML 解析器；`textContent` 写入不解释标记。**ADR-0002 在此处从「架构偏好」变成了「安全性质」**，这是它在本项目里第一次真正发挥作用，值得在 tasks 的验证项里点名。

**代价（留给 Change 09）**：图片不能这样传。图片预览需要 base64 data URI 或一个独立于 `/api/` 的端点，那是后面的债，本 Change 不付。

## Risks / Trade-offs

- **扩展名白名单打不开无扩展名文本文件**（`Makefile`、`LICENSE`、`.gitignore`）→ 这是本 Change 最确定会发生的一次用户可见失败。**缓解**：Change 05 已立项，方案见 D7，roadmap 想法池有指针。刻意不缓解——缓解它就是提前把 Change 05 做掉。
- **`is_text` 的判定与实际内容不符**：一个名为 `a.png` 的文本文件会被拒绝，一个名为 `a.txt` 的二进制文件会被当作文本呈现并显示乱码 → 变更 03 已有的取舍谱系（符号链接的大小是链接自身长度）同一性质：**spec 如实承诺判定规则，不藏**。文件名与内容不符的责任在文件系统，不在浏览器。
- **1 MiB 上限可能偏小**：用户可能有几百 KB 的 JSON 想看一眼 → 上限是一个常量，实现时若实测不符可直接调；spec 里没有写死这个数字，所以调整不动任何 Scenario。
- **GBK 文件显示乱码** → D9 的自觉取舍，已登记想法池。
- **`not_a_regular_file` 这个错误标识可能是过度设计**：绝大多数用户永远见不到它 → 但它是唯一能避免「打开 FIFO 永久阻塞」的标识，而 D10 的先 `Stat` 后判定已经让它的成本接近零。
- **Change 05 的 MODIFIED 会连带改既有测试**：`Text file recognition` 的 Scenario 会全部重写。这是 MODIFIED 的固有代价（Change 03 的 journal 观察 5 已记录），是计划内的改动。
- **`text-preview` 的 Purpose 写着「不包含文件类型识别能力」**：Change 05 归档后这半句会变假，与 Change 03 归档时 `directory-browsing` 的 Purpose 遭遇完全相同。**按 journal 观察 6 的既定做法处理**：archive 只合并 Requirement、不重写 Purpose，届时把冲突摆出来、由用户授权改那一句，并记进 journal。**不预先代改**——journal 观察 6 已确立那条规则的纪律：破例要重新问，不能因为上次破过就顺手破。
- **前端状态复杂度上升**：`?path=` 现在有两种渲染分支，`popstate` 需要按响应分派 → 缓解见 D2，前端任务里单列一条验证项（前进/后退在目录与文件之间来回）。

## Migration Plan

不适用。无数据迁移，无既有部署。新增端点对既有消费者是纯扩展；`GET /api/list` 与 `GET /api/health` 的响应形状不变；错误信封新增三个标识，既有四个取值不变。回滚方式为撤销本 Change 的提交。

## Open Questions

以下问题可推迟到后续 Change 回答，届时不改变本 Change 的 spec、方案与任务划分：

- **相对路径面包屑**。当前只有「上级目录」一个入口，位置显示是拼出来的整串路径。Finder 风格的可点击面包屑是纯呈现层，留给 `polish-file-browser`。
- **大目录虚拟滚动**。想法池已登记该债，与本 Change 无关。
- **图片预览的传输通路**。D11 已记下 JSON 通道不适用于二进制图片，Change 09 需要自己决定 base64 还是独立端点。
