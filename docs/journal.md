# OpenSpec 学习实验记录

> 教材（[OpenSpec_HTTP_File_Browser.md](OpenSpec_HTTP_File_Browser.md)）定义了要观察什么；本文件记录**实际观察到了什么**。每完成一个 Change，在「实验日志」补一节，并勾选验收清单。

## 验收清单

来自教材第 59 节。

- [x] 我知道 `openspec/specs` 和 `changes` 的区别
- [x] 我知道 delta spec 是什么
- [x] 我知道 ADDED 和 MODIFIED 的区别
- [x] 我知道 MODIFIED 为什么必须保留完整 Requirement
- [x] 我知道 Scenario 与测试的关系
- [x] 我知道 Proposal、Spec、Design、Tasks 各自解决什么问题
- [ ] 我知道什么时候使用 explore
- [ ] 我知道什么时候使用 update
- [x] 我知道 sync 与 archive 的区别
- [x] 我知道为什么已经 Archive 的 Change 不应该直接修改
- [x] 我能够从历史 Change 回溯某个系统行为为什么存在
- [x] 我能够从当前 Spec 判断系统应该有什么行为
- [x] 我能够让 AI 在实现过程中停止自行扩大需求
- [ ] 我能够让 AI 根据新的需求创建新的 Change
- [ ] 我能够让 AI 在需求变化后重新同步 Spec、Design 和 Tasks

## 实验日志

> 每个 Change 一节，重点回答教材第 57 节的三个现象。

### 01 bootstrap-http-server

- 日期：2026-10-03
- 现象一（AI 把 Spec 写成实现方案）：**未发生**。spec 全程用可观察行为表述，回环约束写成「仅监听回环地址」而非「绑定 127.0.0.1:8080」，端口细节留在 design D6。
- 现象二（需求变化被误做成 ADDED）：不适用（本 Change 为新增 capability）
- 现象三（Apply 偷偷扩大范围）：**发生了一次，已按规矩处理**。tasks 2.1 写的是 `run(args []string, stdout, stderr io.Writer) error`，实际实现给 `run` 加了 `assets fs.FS` 与 `listen listenFunc` 两个任务书里没有的参数。这确实越了字面范围，但 tasks 2.3 要求「断言 listener 为回环地址」而 design D7 要求「测试不绑定真实端口」，两者在不加注入 seam 的前提下无解。处理方式是开工前把这个冲突摆出来问，而不是默默加。
- 其他观察：
  1. **artifacts 之间的矛盾只有实现能撞出来。** design D1 说「web/ 在仓库根、main.go 只调 internal/app」，D2 说「`//go:embed web`」——但 Go 的 embed 只能内嵌本包及子目录，`internal/` 下任何包都引用不到根目录的 web/。propose 阶段两份文档自洽地读起来毫无问题，apply 写第一行代码就撞墙。修法是让 web/ 自己成为 Go 包。教训：design 的「替代方案」段落如果当初写了「embed 目录通配」这一项，冲突会在 propose 阶段就被识破。
  2. **测试自己也会违反 design。** `TestDefaultListenerIsLoopbackOnly` 原本 `net.Listen("tcp", "127.0.0.1:8080")` 真绑端口——正是 D7 明令禁止的。它在 verify 阶段才暴露（当时本地跑着服务，测试红了）。若没有 verify 这一步，这个测试会安静地留在仓库里，直到某天开发者跑着程序跑测试才发现。**测试代码同样需要被 design 约束。**
  3. **勾选纪律。** 任务 5.1 的验证条款含「从另一台主机尝试连接」，我在只有本机等价近似证据时就打了勾。正确做法是留白、等实测。这次是用户实际跑了跨主机验证（Connection refused + 绑 `0.0.0.0` 的 8081 对照组连通）才真正闭合——对照组的作用是排除「整台机器被防火墙挡住」这个替代解释，否则「连不上」证明不了什么。
  4. **Scenario 数量比预想多。** 5 条 Requirement 展开出 14 条 Scenario，测试落地时发现「未知 API 端点」与「API 域内失败返回 JSON」共用同一条 not-found 代码路径，测试断言也合并了。spec 写 Scenario 的成本远低于补测试的成本，这条不对称值得记住。

### 02 directory-browsing

- 日期：2026-10-03
- 现象一（AI 把 Spec 写成实现方案）：**未发生**。7 条 Requirement 全部用可观察行为表述——「以确定的顺序返回条目」「系统 SHALL 在每个条目中只提供名称与类型」，`filepath.Rel`、比较键三元组、`textContent` 全部留在 design D4/D5/D8。
- 现象二（需求变化被误做成 ADDED）：**发生了一次，处理正确**。proposal 原本想把「结构化错误 code」作为 `directory-browsing` 的新 Requirement 加进去——这会让同一件事横跨两个 capability。apply 前改成对 `service-startup / JSON error responses` 发 MODIFIED。**判据是可复用的**：如果一条新行为只有在某个新消费者出现之后才有意义、且它修正的是既有 Requirement 对同一件事的定义，它就是 MODIFIED 而不是 ADDED。Change 01 的 design D4 早已预写了这条 delta 的触发条件（「结构化 code 的价值在于前端需要分支处理时……届时用 MODIFIED 表达」），proposal 照着兑现即可。
- 现象三（Apply 偷偷扩大范围）：**发生了一次，已按规矩处理**。tasks 没写「`path` 参数是绝对路径时怎么办」，我第一版让 `filepath.Join(root, "/etc")` 静默重解释成 `root/etc`。没有直接改代码了事，而是停下来标出来，最终定成按 `outside_root` 拒绝并补进 design D11。**「spec 没写」不等于「可以随便选」**，但也不是必须回头改 spec——先判断哪个选择与已声明的契约一致，再把它记成 design 决策。
- 其他观察：
  1. **spec 的字面正确不等于契约可用。** `Parent directory reference` 要求「子目录给出上级相对路径、根目录不给出」，但根目录的相对路径本身就是空字符串，于是「一级子目录」和「根目录」的 `parent` 取值撞成了同一个 `""`。spec 逐字成立，客户端却无法据此判断自己在哪一层。真正的 bug 不在服务端，在前端：它按 `parent === ""` 隐藏上级入口，把一级子目录变成了死胡同。教训是**契约的每个取值都问一句「它唯一确定一件事吗」**。修法选了不改响应结构（`path` 已足够），但把「客户端一律用 `path` 判断根位置」写进 design D10 并登记进想法池——将来谁想按 `parent` 推断根目录，就该正式 MODIFIED 一次。
  2. **CSS 能让 `hidden` 属性失效，而断言 `el.hidden` 查不出来。** `.parent-link { display: inline-block }` 的优先级高于 UA 样式表的 `[hidden] { display: none }`，元素明明设了 `hidden` 却仍然可见。首轮手工验证我只查了 `el.hidden === true` 就判定通过，截图才发现根目录也显示着「上级目录」。**对「不可见」的断言必须查计算样式（`getComputedStyle`），查属性等于没查。**
  3. **零构建的前端验证是真验证，但不是自动的。** design D9 声明不做前端自动化测试，于是 7 条前端任务全靠手工。每条都写明了「怎么算通过」——`?path=` 编码后的 href、hover 前后背景色的 `getComputedStyle` 值、`#entries` 内 `img`/`script` 元素数为 0、320px 视口下 `scrollWidth === innerWidth`。这样手工记录才有证据强度，`tasks.md` 8.2 存的是观察结果不是「已验证」三个字。带 `<img src=x onerror=...>` 的真实文件名是这里最有价值的一次构造：它一次性验掉了「用 textContent 而非 innerHTML」这个决策。
  4. **测试构造会被文件系统能力卡住，而 skip 必须说清原因。** 「名称仅大小写不同」那条 Scenario 在 macOS 默认的大小写不敏感 FS 上根本无法构造（三个文件折叠成一个）。按 design D9 已有的 skip 惯例处理，但光 skip 不够——那条 Scenario 真正要防的是 `sort.Slice` 不稳定导致顺序抖动，于是补了 `TestSortEntriesFallsBackToNameForCaseOnlyDifferences` 直接断言比较键第三段。**环境不满足时，skip 之后要回答「那这条 Scenario 要防的东西现在由谁负责」。**
  5. **MODIFIED 的成本确实在归档那一刻才结清。** `TestUnknownAPIEndpointReturnsJSONError` 从 Change 01 起就断言 `error` 是字符串；本 Change 把它改成嵌套结构，那条断言必须改。proposal 与 design 都提前把这条列为显式代价（而不是让 apply 顺手改测试蒙混过去），所以它是计划内的改动，不是「AI 偷偷改了测试」。**MODIFIED 要求必须带完整 Requirement，代价是它连带着既有测试一起欠债。**

### 03 add-file-metadata

- 日期：2026-10-03（归档目录 `openspec/changes/archive/2026-10-03-add-file-metadata/`）
- 现象一（AI 把 Spec 写成实现行为）：**未发生**。`Entry metadata` 的正文只写可观察行为（给出什么值、以什么单位、什么情况下不给出），`DirEntry.Info`、`int64` 指针、`omitempty`、Unix() 这些全部留在 design D4/D5/D6 与 tasks 1.1–1.4。特别值得注意的是 `modified_at` 的形状：spec 承诺「以自 Unix 纪元起的整数秒表示」，但「用整数秒而不是 RFC3339 字符串」这条理由（可复现性）写在 design D6，spec 里一个字都没提格式选择的动机——这正是 SDD 该有的分工。
- 现象二（需求变化被误做成 ADDED）：**未发生，但踩在边缘**。本 Change 的核心动作就是推翻 Change 02 自己写下的一句排除（「不提供大小、修改时间或内容」）。proposal 把它明确拆成 ADDED `Entry metadata` + MODIFIED `Directory listing response`，并且——这是关键——**只推翻前半句**。如果原地改那一句，最省事的写法是三个词一起删，「不提供内容」就被顺手松掉了，`text-preview` 那道门从此无人看守。加一条 Scenario `A listed directory contains readable files` 把「响应不含内容」从一句措辞变成可断言的行为之后，Change 04 要读内容就是**有意识地推翻一条 Scenario**，而不是发现那道门根本没锁。
- 现象三（Apply 偷偷扩大范围）：**未发生**（一条自查后收回的除外，见下）。本 Change 的 seam（tasks 1.4）在 propose 阶段就作为独立任务显式存在，而不是 apply 时顺手加的——这正是 Change 02 journal 观察里后悔过的那类越界。
- 其他观察：
  1. **设计决策可以有一个「唯一的守门人」，前提是那条测试真的守得住。** D4（lstat 而非 `os.Stat`）被写成了「本 Change 最容易做错、且错法最危险的一个决定」，但设计文档本身无法被测试。这次收尾时把 `fillMetadata` 临时改成 `os.Stat` 跑了一遍，`TestAnEntryIsASymbolicLink` 立刻红在 `size = 65536，期望链接自身的长度 112` 上。**一条「应该能抓住」的测试，必须真的被证伪过一次**才算守门人，否则它只是一段看起来很有道理的断言。
  2. **造数据时要保证「错的实现」和「对的实现」在断言处不会偶然相等。** 那条软链用例最初的目标文件与目标路径字符串长度可能接近，此时 `Stat` 与 `Info` 会取到同一个值，用例永远绿。因此在用例里加了一道前置：目标内容必须显著大于路径字符串（64KB vs 85 字节），否则用例自己 `t.Fatalf`。**守卫用例的构造本身也需要被审。**
  3. **`Chtimes` 改目录时间必须放在写完内容之后。** 头一版 `TestADirectoryEntryIsListed` 先定目录时间再造文件，新建条目会把目录的 mtime 改掉，断言随之失败。同一个坑在 `TestRepeatedListingsReturnTheSameMetadata` 里又踩了一次。**「固定某个时间戳」与「在它之后改变目录内容」不能共存，顺序是一条硬约束，值得写进用例注释。**
  4. **前端「窄屏降级」很容易变成静默丢数据。** 三列布局在 320px 下必然紧张，我第一版直接用媒体查询把「修改时间」整列 `display: none`，理由是「表现层细节」。截图后发现那其实是一个产品取舍：用户在没有提示的情况下少看到一整类元信息，而这不在 design D8 授权的「呈现细节」范围内。改成三列全部保留、只收窄固定列宽并调小字号后实测 320px 下 `scrollWidth === innerWidth === 320`、长文件名换行、时间字符串仍单行。**「spec 不约束」不等于「随便定」——表现层里仍然有会改变用户所见数据的决定。**
  5. **`content: 'x'` 与 `<img src=x onerror=...>` 的区别。** 验证「响应不含文件内容」时，最初准备的文件内容是一个字母 `x`——它在 `{"name":` 这种 JSON 里必然出现，断言 `!strings.Contains(body, content)` 等于永远失败。换成 `READMECONTENTMARKER` 才成为一条可能失败的断言。**用来证明「某样东西不在场」的标记，必须是别处不会偶然出现的。**
  6. **一次经用户授权的破例：改 capability 的 Purpose。** `openspec/specs/directory-browsing/spec.md` 的 Purpose 写着「不包含文件元信息与文件内容读取」，本 Change 让前半句变假。archive 只合并 Requirement、不重写 Purpose，而 `AGENTS.md` 写着「禁止直接修改 `openspec/specs/`，前者由 sync/archive 更新」。但 archive 只合并 Requirement，**没有任何工具会重写 Purpose**——这半句从 Change 02 归档起就注定了要在某次归档时变假。

三种走法里选了第三种：**把冲突摆出来，由用户授权改这一句，并留下记录**。理由有两条。其一，CLI 自己的 `openspec instructions specs` 输出明确写着「To change an existing capability's Purpose — including a leftover `TBD` placeholder — edit `openspec/specs/<capability>/spec.md` directly」——**这正是 OpenSpec 对这一处设计的官方路径**，`AGENTS.md` 的禁令本意是「别让 AI 悄悄重写 capability 定义」，而 Purpose 是散文不是定义。其二，「不包含文件元信息」与新 Requirements 直接矛盾时，留着一句假话的代价是：下一个 Change（`text-preview`）读这份 capability 契约，会以为自己不该碰元信息。

**但规则的价值在于被遵守，不在于被论证得通。** 因此这次破例被显式记在这里：授权范围是「那一句」，不是「以后都可以改」；下次若再遇到同类情形，仍要重新问一遍。**为了让 accountability 成立，必须把破例写下来——只在心里知道自己破过例，下次就会顺手破第二次。**

顺带一条，纠正我自己在 verify 阶段的误判：同一份 capability 里还有一对同名 Scenario（`A directory contains both directories and files` 同时挂在 `Entry type distinction` 与 `List ordering` 下）。我在 verify 报告里把它当成「Change 02 遗留的瑕疵」，**这是错的**——Change 02 的 tasks 4.5 原文写着「注意 spec 中……各有一条同名 Scenario……两个测试必须用可区分的名字，不得合并或重名」。它是当时撞到同一个碰撞后**有意保留**的：WHEN 措辞对两条 Requirement 都自然，于是差异放到测试名上区分（`TestEntryTypeIsDistinguishedInAMixedDirectory` 与 `TestDirectoriesPrecedeFilesInAMixedDirectory`）。

本 Change 自己引入的那对（`The same directory is listed repeatedly`）确实该改，改在归档之前，因此 archive 副本与 spec 仍然一致。**两件事同形，处理方式却不同，差别在归属**：一个 Change 有权清理自己造成的麻烦，但不该顺手推翻另一个已归档 Change 记录在案的判断。判断一条 spec 文本是不是缺陷，要先去看当初的 Change 有没有为它做过决定——本次差点凭「看起来别扭」就动手。

### 04 text-preview

- 日期：2026-10-03（归档目录 `openspec/changes/archive/2026-10-03-text-preview/`）
- 现象一（AI 把 Spec 写成实现方案）：**未发生**。四条 Requirement 全部是可观察行为——「返回该文件的规范化相对路径与该文件的内容」「以替换字符呈现」；扩展名白名单的具体清单、1 MiB 上限、先 `Stat` 后打开的判定顺序、`ToValidUTF8` 全部留在 design D6/D9/D10 与 tasks。spec 连「1 MiB」这个数字都没写（只说「超过系统可提供内容的最大字节数」），design 风险一节预言的「调整常量不动任何 Scenario」在实现里如约成立。
- 现象二（需求变化被误做成 ADDED）：不适用（本 Change 全部 ADDED，无需求变化）。但 Change 03 journal 观察 2 的预言在本 Change 被正式推翻：它预言「Change 04 要读内容就是有意识地推翻一条 Scenario」，而 design D1 用独立端点让 `directory-browsing` 零 delta——tasks 2.9 的验证实际跑过（`git diff` 中 `browse_test.go` 无一处改动），八条既有 Requirement 与全部既有测试一行未改。**预言中的 MODIFIED 演练没有发生，因为更优的方案让那道门根本不用推**；第一次真正的 MODIFIED 演练顺延给 Change 05。
- 现象三（Apply 偷偷扩大范围）：**发生了三起小的，全部当场报告、未静默吸收**：① 补了 tasks 未列的 `TestAnEmptyTextFileIsRequested`（零字节 `.txt` 返回成功与空内容）——断言的是 spec 已承诺行为的边界，不是新行为；② 前端 `permission_denied` 文案由「没有读取该目录的权限」改为「没有读取该位置的权限」——`ERROR_TEXT` 一张表现在同时服务目录与文件两个视图，原文案在文件视图里是错的；③ 8080 被用户自己的旧进程占用，端到端验证改用 `.verify-tmp` 下的临时宿主程序在 18099 起同一份 handler + 内嵌前端，验完即删、不落仓库。
- 其他观察：
  1. **守门人测试各被证伪一次，才敢说它守得住**（延续 Change 03 观察 1 的纪律）。FIFO 用例：把类型判定临时挪到 `ReadFile` 之后，用例在 5 秒超时处红掉，报错文本直指 design D10；白名单集合断言：往 `textExtensions` 塞一个没有对应用例的 `graphql`，`TestTextRecognition` 立刻红。两条守卫都不是「看起来会抓住」，是「抓到过一次」。
  2. **tasks.md 里的引用错误只有实现能撞出来**：2.5 标注「design D8」，UTF-8 替换字符实际是 D9 的决策；1.5 说「2.5 的 `not_a_regular_file` 用例」，该用例实际构造在 2.7。propose 阶段读起来完全自洽的两处笔误，apply 第一动手就暴露——与 Change 01 观察 1 同族，但轻一个量级：错的是指针，不是内容。处理方式是按正确出处写代码注释、tasks 原文不动（archive 不可变）。
  3. **勘误：tasks 4.4 手工验证记录里「仓库自己的 `Makefile`」措辞不准**——仓库里并没有 Makefile，那个文件在验证 fixture 里；仓库自己的无扩展名文件是 `AGENTS.md` 与 `go.mod`，失败原因相同（无扩展名 → `not_text`）。archive 已不可变，纠错记在这里。
  4. **端到端验证的环境本身就是变量**。本机 `open()` 无法创建文件名含 `</` 的文件（报 ENOENT，疑似端点安全过滤），`<script>alert(2)</script>.txt` 造不出来，退化为 `<script>alert(2).txt`。断言于是不依赖文件名形态，改以 `#preview` 内 `img`/`script` 元素数与「关闭自动 dismiss 后 dialog 是否出现」为判据——**测试数据的构造要绕开环境的脾气，把判据放在环境碰不到的地方**。
  5. **agent-browser 让手工验证有了代步工具，但判据仍逐条人工设计**。Change 02 观察 3 说「零构建的前端验证是真验证，但不是自动的」；这次每一项仍先写明「怎么算通过」（320px 下 `scrollWidth === innerWidth`、`#preview` 内元素数为 0、后退后 `preview.hidden === true`、各错误码对应文案），再让浏览器执行。工具替代的是点击与读值，不是判定标准——深链接、返回上级、前进后退、XSS、六种错误文案、窄视口共 11 项观察全部实测落表（tasks 4.4）。
  6. **design D6 预言的那次失败在验证中如约发生**：fixture 里的 `Makefile` 点进去就是「这是二进制文件，无法以文本预览」。这不是回归，是本 Change 刻意保留的缺口（「先窄后宽」）。**Change 05 的动机第一次由真实点击撞出来，而不是由 roadmap 预约**——D6 说的「需求变更被现实撞出来」在归档这一刻有了实证。
  7. **归档时确认 Purpose 仍然为真**：`text-preview` 的 Purpose 写「不包含文件类型识别能力」，本次归档后依旧成立（扩展名白名单是判定规则，不是类型识别）；Change 05 归档后它才会变假。想法池那条待办（tasks 4.5）留待届时处理，本次未动 `openspec/specs/` 下任何既有文件——本次归档唯一的 spec 写入是新建 `text-preview/spec.md`。

### 05 improve-text-file-detection

- 日期：2026-10-03（归档目录 `openspec/changes/archive/2026-10-03-improve-text-file-detection/`）
- 现象一（AI 把 Spec 写成实现方案）：**未发生**。MODIFIED 后的 `Text file recognition` 正文只有可观察行为——判据区间（0x00–0x08 / 0x0B / 0x0E–0x1A / 0x1C–0x1F）作为「判据的定义本身」进 spec（design D2 的精度切分），窗口 4096 留在 design，NUL 阈值被 D4 砍掉后 spec 里连「全部只落偶数位」都纯用位置语言表述，一个可调数字都不剩。Change 03 立下的「1 MiB 先例：可调旋钮不进 spec」第二次成立。
- 现象二（需求变化被误做成 ADDED）：**未发生——这正是预约了两个 Change 的第一次 MODIFIED 演练，如约发生**。delta 带完整 Requirement 重写（正文 + 10 条 Scenario），被推翻的旧 Scenario（无扩展名 → `not_text`）不是被删掉，而是被内容判据下的新 Scenario 族替代。Change 04 design D7 把嗅探方案定到「可照抄的粒度」的预付在本 Change 兑现：实现阶段零调研。
- 现象三（Apply 偷偷扩大范围）：**一起，已报告**。名字判定表（`TestTextRecognition`）里的无扩展名用例（`Makefile`、`LICENSE`、`.gitignore`、`notes.`）在分流之后不再「名字说了算」——留在表里只会测「相对路径下没有这个文件」，是假断言。处理：移出名字表、由嗅探判据单测与 API 层用例接管，在会话总结里明说。spec 零变化。
- 其他观察：
  1. **守门人证伪两条当场红掉**（延续 Change 03 观察 1 / Change 04 观察 1 的纪律）：白名单集合断言塞进无对应用例的 `graphql` → 立刻红（清单与用例必须一一对应）；把 `isTextFile` 调用挪到超限判定之后 → `huge.bin` 用例红（`not_text` 变 `too_large`）。证伪 2 顺带暴露一个盲点：`bigfile`（既是文本又超限）在两种顺序下都返回 `too_large`，对顺序完全不敏感——**两条规则的「先后」只有靠两条规则给出不同结果的用例才分得出来；判定与超限同向的 fixture 是冗余守卫**。写用例时该问一句「打乱我要守的东西，我哪条断言会变色」。
  2. **Purpose 半句闭环**（Change 03 观察 6 纪律的第二次执行）：apply 收尾时把冲突摆给用户，用户回复「修改吧，具体什么时候改你定」——授权明确、时机下放。时机选在归档同步时一并落盘：`openspec/specs/` 的写入本就只被 sync/archive 授权，Purpose 修改作为同步动作的一部分最顺。改法沿用 Change 03 的「排除清单保持诚实」：识别入列（内容嗅探让它真是本 capability 的行为），编码检测与解码保留在排除侧（design D9 的取舍未变）。授权范围一句话为限，记档于此。
  3. **sync 的 MODIFIED 合并退化成了「逐字节搬运 + 断言」**：单块整替场景下从主 spec 与 delta 各按 `### Requirement:` 标题切出块、替换后断言两份字节一致，比「智能合并」的手抄更可靠——合并的智能不必体现在每次都重新创作，体现在知道这次不需要。
  4. **稀疏大文件 fixture 有个反直觉点**：全零的稀疏文件嗅探结果是「奇偶混杂 NUL → 非文本」，根本走不到超限判定，「判定先于超限」根本没被测到。fixture 必须头 4096 字节是文本、再 `os.Truncate` 到 1 MiB+1。**测两条规则的先后时，两条规则对同一输入的结果必须不同且都合法。**
  5. **端到端冒烟没有现成的二进制文件可点**（仓库根目录全是文本），`.git/index` 顶上——既是真实二进制、又在根目录内。环境给不了 fixture 时先找系统里现成的，与 Change 04 观察 4「绕开环境的脾气」同一思路。

### 06 add-syntax-highlighting

- 日期：2026-10-03（归档目录 `openspec/changes/archive/2026-10-03-add-syntax-highlighting/`）
- 现象一（AI 把 Spec 写成实现方案）：**未发生**。四条 Requirement 全部是可观察行为——「以该语言的语法高亮形式呈现」「得到的字符序列与该文件的内容一致」「以普通文本形式呈现」「返回成功响应，响应体为该资源内容」；语言识别的具体机制（getLanguage 别名表、`highlightAuto`、`AUTO_MIN_RELEVANCE = 5`）、形状校验的正则、vendored 文件命名与版本注释格式，全部留在 design D2/D3/D4 与代码注释。spec 里连「highlight.js」这个库名都没出现（只说「语法高亮支持的语言」）。
- 现象二（需求变化被误做成 ADDED）：不适用（本 Change 全部 ADDED）。但 Change 02 design D11 立的安全不变量（「这个元素永远不交给 HTML 解析器」）被本 Change 有意识地收窄：不是偷偷绕开，而是 proposal/design/tasks 三处明写「D2 显式推翻 D11 的字面表述」，`index.html` 与 `app.js` 的注释同步改写——**收窄不变量的动作要和不变量当初立起来时一样正式**。既有 capability 零 delta 兑现：三个既有 spec 一行未动。
- 现象三（Apply 偷偷扩大范围）：**四起小的，全部当场报告**：① server_test 的静态资源表补了 `/vendor/highlight.min.css` 条目（tasks 1.4 只点名 `.js`）——同一 spec Scenario 的完整覆盖，非新行为；② Plain text 别名命中视同无映射——design D3「`.txt` 一律走回退」的落实，不是新策略；③ 任务 2.2 验证措辞「无扩展名二进制文件」的落点修正为「无扩展名且语言无法识别的文本文件」——真二进制在 `text-preview` 判定即被拒，到不了高亮层，已在 tasks 走查注记写明；④ 「纸面固定」样式（深色外观下 preview 保持白底）——2.4 授权范围内的协调决策，依据读自上游主题 CSS。无功能被静默增删。
- 其他观察：
  1. **呈现层走查抓到了 API 层测试与校验函数单测都抓不到的 bug**：`safeHighlightHTML(null)` 的 TypeError（低置信回退把 null 送进形状校验）——校验函数 13 个构造用例全绿、`go test` 全绿，但管线把它接错位置，只有浏览器真实点开 `notes` 才暴露。ADR-0003 把呈现层 Scenario 分派给浏览器走查，本 Change 是该决策的第一次兑现：**分派表不是形式，每一行都要有人接**。
  2. **测试工具的导航模型本身就是被测系统的一部分**：agent-browser 的 `open` 是整页导航（window 状态清零）、`parent-link` 无 preventDefault（原生跳转）、文件视图清空 `#entries`（先离开才能再进入）——「同一文件重复查看」的两次渲染天然跨文档，最后靠 localStorage 接住。断言设计前先摸清工具与被测系统的导航语义，否则比对的根本不是两次渲染。
  3. **版本验证项自己也需要版本感知**：tasks 3.4 写「注释中的版本号与运行时 `hljs.version` 一致」，而 11.12.0 的属性名是 `hljs.versionString`（`hljs.version` 不存在）。跨版本的 API 断言先在运行时探一下属性名，别把文档记忆当契约。
  4. **样式协调的判断依据是读上游产物，不是猜**：github 主题只有 `.hljs` 一条基础色规则、不设字号，所以「preview 固定白底 + 主题 token 着色」无冲突；删掉 style.css 深色媒体查询里的 `.preview` 覆盖是唯一需要的动作。vendored 文件升级时这一步要重做——主题若哪天带上 `font-size`，协调方式就得变。
  5. **归档时确认 Purpose 仍然为真**（Change 03 观察 6 纪律的第三次执行）：`syntax-highlighting` 的 Purpose 写「不包含语言识别的具体方法」，实现用了别名表 + auto + 阈值而 spec 无一处提及，仍然成立。

### 07 add-markdown-preview

- 日期：2026-10-03（归档目录 `openspec/changes/archive/2026-10-03-add-markdown-preview/`）
- 现象一（AI 把 Spec 写成实现方案）：**未发生**。五条 Requirement 全部是可观察行为——「以 Markdown 渲染形式呈现」「以字面形式呈现」「导航到解析出的目标位置」「在新的浏览器标签页中打开」「得到的字符序列与该代码块在文件中的内容一致」；markdown-it 与 `html: false`、`SCHEME_RE`、路径逐段解码与 `..` 弹栈、`AUTO_MIN_RELEVANCE` 阈值、fence 钩子返回 `<pre` 前缀串的机制，全部留在 design D1-D8 与代码注释。spec 里连「markdown-it」这个库名都没出现。
- 现象二（需求变化被误做成 ADDED）：**第二次 MODIFIED 演练如约发生**。syntax-highlighting 的「Syntax highlighted presentation」被整体让渡——条件挂在「呈现形式」上（被 Markdown 渲染呈现的文件）而非文件种类上，为 Change 08 的渲染/源码切换留好了位；delta 带完整 Requirement 重写（正文 + 3 条 Scenario，两条旧 Scenario 原样保留）。sync 合并第二次退化成「单块整替 + 断言」（Change 05 观察 3 的做法直接复用），新增 capability 则是「Purpose 逐字搬运 + ADDED 落位」——合并的智能依旧体现在知道这次不需要智能。
- 现象三（Apply 偷偷扩大范围）：**一起小的，已报告**。`index.html` 里 preview 元素的注释随 tasks 2.2 一并改写（tasks 只点名 `app.js`）——该注释复述的「字符串只能来自 hljs 输出」在双管线并行后已不完整，D7「避免注释撒谎」的动机对它同样成立，零行为变化。无功能被静默增删。
 - 其他观察：
   1. **D9 分派表的「注入异常」一行接住了一条真实防线缺口**：`markdownHTMLFor` 初版没有 try/catch——注释声称「抛错返回 null」，实现却让异常沿 promise 链落进「无法连接服务」的错误提示，正好违反「SHALL NOT 向用户报告错误」。API 层测试、`openspec validate`、正常路径走查都绿，只有按分派表构造故障输入才红。**「渲染未成功」这类罕见路径的可观察行为，靠等渲染器自己抛错是测不到的，注入故障是唯一的入射角。**
   2. **第二次安全收窄的防线形状与第一次不同，成本也悬殊**：Change 06 的形状校验是「信任库输出、校验拦截」，写成 30 行正则；这次的配置性封闭（`html: false`）是一个构造参数——解析器根本不为 HTML 开门，无需输出校验。两条防线在注释里并列陈述、互不替代；「安全靠配置不靠组件」第一次落地，连 sanitizer 这个信任面都没引入。
   3. **多标签断言要先确认 eval 落在哪个 tab**：外部链接点击后 agent-browser 的活动标签自动切到新标签，`eval` 随之落在 example.com 上，差点误判成「当前页变了」。实际行为正确（t1 原页未动、t2 新开）；工具的 tab 模型把 spec 里「新的浏览器标签页中打开，当前呈现保持不变」这句话显性化成了两个 tab 的状态比对，这比单页断言更贴近语义本义。
   4. **init script 注入让故障成为可构造输入**：用 `defineProperty` setter 陷阱把 `markdownit` 全局替换为构造即抛错的函数，比寻找「病态输入」更可复现——渲染器对什么输入抛错是库的实现细节，故障注入不依赖它。
   5. **归档时确认 Purpose 仍然为真**（Change 03 观察 6 纪律第四次执行）：`markdown-preview` 的 Purpose 写「不包含文件内容的修改与写回」「不包含渲染呈现之外的语法高亮呈现」，实现服务端零代码变化、代码块高亮挂在本 capability 之下，均成立。`syntax-highlighting` 的 Purpose「不包含 Markdown、图片等其他类型的渲染呈现」在让渡后反而更准了。

### 08 improve-markdown-preview

- 日期：2026-10-03（归档目录 `openspec/changes/archive/2026-10-03-improve-markdown-preview/`）
- 现象一（AI 把 Spec 写成实现方案）：**未发生**。MODIFIED 正文只有可观察行为——「默认以渲染形式呈现」「SHALL NOT 保留此前的源码形式选择」；切换控件放哪、状态怎么存、重置点选在 `load()` 入口，全部留在 design D1/D3。spec 里连「按钮」这个词都没出现。
- 现象二（需求变化被误做成 ADDED）：**第三次 MODIFIED 演练如约发生，且是 Change 07 预约的兑现时刻**。proposal 把本 Change 定位为「留位的兑现」：Change 07 把 syntax-highlighting 的豁免条件挂在「呈现形式」而非文件种类上，本次源码形式由 `syntax-highlighting` 零 delta 自然接管——那句措辞决策的回报在两个 Change 后到账。markdown-preview 的 MODIFIED 带完整 Requirement 重写，渲染相关 Scenario 的 WHEN 全部补上「以渲染形式呈现」条件。
- 现象三（Apply 偷偷扩大范围）：**一起小的，已报告**。tasks 2.3 点名三处注释，实际改了四处——`isMarkdownPath` 上方「扩展名 md/markdown 即渲染」复述了分岔前的结论，按 D4 援引的 Change 07 现象三先例一并改写，零行为变化。无功能被静默增删。
- 其他观察：
  1. **工具的注入语义跨版本会变，故障注入前先验证注入真的到了主世界**。journal 07.4 的 `defineProperty` init-script 手法在本机当前版本的 agent-browser 上静默失效——init script 在隔离世界执行，主世界的 `getOwnPropertyDescriptor` 查到的是 vendor 自己的数据属性，渲染照常成功。加 `window.__trapRan` 探针才定位到根因，改用 `network route --body` 让 vendor 文件本身返回「构造即抛错」的 `markdownit` 全局——故障点相同，主世界确定生效。**「注入了」与「注入生效」是两个断言，后者才配当证据。**
  2. **D1 的重置点选择在走查里兑现为一条强断言**：popstate 回到「曾切过源码的那条历史条目」仍以渲染形式呈现——呈现形式从未进 History，于是同文档内经 `load()` 的重置（D5 特意要求的 SPA 路径而非平凡的整页导航）与「历史条目不含呈现形式」在一条走查里同时被验证。把重置点选在唯一汇聚点，验证成本最低。
  3. **sync 的 MODIFIED 合并第三次退化成「逐字节搬运 + 断言」**（延续 05 观察 3），但这次红的是断言自己——提取函数把 delta 侧的 `## ADDED Requirements` section 头算进了块尾，首跑误报 MISMATCH。**合并可以退化，断言不能想当然：断言失败时先审断言本身，再审被断言物。**

### 09 add-image-preview

- 日期：2026-10-04（归档目录 `openspec/changes/archive/2026-10-04-add-image-preview/`）
- 现象一（AI 把 Spec 写成实现方案）：**未发生**。五条 Requirement（4 ADDED + 1 MODIFIED）全部是可观察行为——「返回该文件内容的字节序列」「SHALL NOT 返回根目录之外的文件的图片内容」「成功响应 SHALL NOT 以 JSON 响应体返回」；端点命名 `/api/image`、八项白名单、`http.ServeContent` 流式拷贝、`dot > 0` 大小写折叠全部留在 design D1-D3 与代码注释，spec 里没有出现过任何实现机制。
- 现象二（需求变化被误做成 ADDED）：**第四次 MODIFIED 演练，这次改的是别的 capability 里的一句既有承诺**。service-startup 的「HTTP surface partitioning」收窄——`/api/` 分区的 JSON 承诺在图片端点的成功响应上开口，MODIFIED 带完整 Requirement 重写（正文 + 4 条旧 Scenario 原样保留 + 新增 Image content endpoint succeeds）；`JSON error responses` 的 Scenario 端点无关，继续适用、零 delta。sync 第四次退化成「单块整替 + 新建 capability 的 Purpose 逐字搬运」（延续 05 观察 3、07 观察 2）。
- 现象三（Apply 偷偷扩大范围）：**一起小的，已报告**。`content_test.go` 的 `TestEveryKnownErrorCodeIsMappedToAStatus` 增补了 `codeNotAnImage`（tasks 未点名这条测试）——新标识登记进 `codeStatus` 后，守门测试的清单若不同步就会名不副实；零行为变化，与 Change 07/08 的注释改写同类。无功能被静默增删。
- 其他观察：
  1. **承诺与观察手段对齐，走查的断言才立得住**：D5 刻意让 spec 只承诺「回退说明出现」而不区分失败原因——路径级错误（`not_found` 等）经 `<img>` 本来就全部塌缩成 onerror，观察不到的区分写进 spec 只会逼出走查时无法兑现的断言。与 Change 06 观察 1 相对：那次是走查抓到了测试没抓到的 bug，这次是分派表先问了「这个区分，浏览器走查能不能观察到」。
  2. **「无上限」没有负测试可写**：spec 不设大小上限，测试无法证明「不存在的限制不存在」；D3 的实际保障是传输路径本身（流式拷贝、内存与文件大小无关），落在测试里的只有路径正确性断言（响应体逐字节等于文件内容）。设计决策承担的部分，别指望测试清单背书。
  3. **前后端双清单的同步靠注释互引 + 双向无害论证**：零构建单二进制约束下没有共享配置的便宜方案，`IMAGE_EXTENSIONS`（app.js）与 `imageExtensions`（image.go）各自独立声明、注释互引提醒同步；漂移两个方向都无害的论证（design D4）让「不同步也不撒谎」成立——同步是本意，不是安全依赖。
  4. **归档时确认 Purpose 仍然为真**（Change 03 观察 6 纪律第五次执行）：`image-preview` 的 Purpose 写「不包含文本型图像格式（如 SVG）的呈现」「不包含图片内容的修改与写回」——实现无 SVG 分支、端点 GET 只读，成立。`service-startup` 的 Purpose「静态页面与 JSON 接口如何划分」在图片字节开口后有一丝张力，但「划分」仍是该 capability 的主题、例外是划分的一部分，不改；如实记录备查。

### 17 improve-root-confinement

- 日期：2026-10-04（归档目录 `openspec/changes/archive/2026-10-04-improve-root-confinement/`）
- 现象一（AI 把 Spec 写成实现方案）：**未发生**。三条 delta 的 MODIFIED 正文全是可观察行为——「请求路径解析其全部符号链接后得到的物理位置位于根目录的物理位置之内」「解析后指向根目录之外」；`filepath.EvalSymlinks`、`filepath.Rel`、启动时解析一次、`physicalRoot` 字段、回落分类等机制全部留在 design D2/D3 与代码注释，spec 里一个机制词都没有出现。
- 现象二（需求变化被误做成 ADDED）：**第五次 MODIFIED 演练，也是第一次「预约的翻案」如约兑现**。被推翻的承诺（根内软链字面出根仍提供）是 Change 02 design D3 当年刻意立下、并同步预约了翻案时机的；proposal 的 Why 直接引用该预约作动机，BREAKING 变更因此有出处、有既定时机。delta 无 ADDED，6 条 MODIFIED 全部带完整 Requirement 重写；最尖锐的一处翻转发生在同一个 Scenario 标题下——`A path inside the root traverses a symbolic link outward` 的 THEN 从「按该路径提供内容」原位反转为「拒绝并返回 `outside_root`」：需求变化被表达为对同一条行为的重写，而不是新增一条行为。
- 现象三（Apply 偷偷扩大范围）：**未发生（以提交内容为证）**。feat 提交的文件清单与 proposal Impact 逐项对应：`browse.go`（resolve 追加物理判定）、`server.go`（`physicalRoot` 字段与构造）、`app.go`（启动错误路径）、三个测试文件各一处断言反转与新增用例；`content.go` 与 `image.go` 一字未动——「端点代码零变化」不是口号，是 tasks 2 节的验收标准，由「越界判定只有一份代码」直接兑现。
- 其他观察：
  1. **BREAKING 的体面取决于翻案条款立得早不早。** Change 02 立字面政策的同时写下「翻案时机预约为 13 的前置」，本 Change 兑现时零争议：为什么改（13 开放远程监听后，字面判定等于向同网段开放任意文件读取）、何时改（13 之前）、改成什么强度（物理判定、无开关）都在几个 Change 之前写好了。「承诺与翻案条款同时立」值得成为写下任何 BREAKING 候选行为时的默认动作。
  2. **反转断言而不是删除测试。** 三处宽松断言（browse / content / image 各一）全部原地反转为拒绝断言，同一 fixture 从「钉住宽松承诺」变成「钉住物理出根必拒」。spec 翻转后测试跟着翻转，覆盖不缩水；删测试则会把「这个行为曾被认为值得钉住」的痕迹一并抹掉。
  3. **design 的汇总句也要与 delta 对账。** D7 写「共 5 条 MODIFIED」，delta 实际是 6 条（3 个 capability × 2）——proposal 与 delta 一致，只有 design 的计数句算错。sync 的字节断言全部通过之后，与 design 对账时才撞见；归档前修正（5→6），归档副本不带笔误。教训与 Change 04 观察 2 同族：错的是指针（汇总计数），不是内容（逐条描述都对）。
  4. **sync 第五次退化成「单块整替 + 断言」**（延续 05 观察 3、07 观察 2、08 观察 3）：脚本按 `### Requirement:` 标题切块、自后向前做字节替换，三连断言——每个 delta 块在合并结果中字节一致、Requirement 总数不变、旧政策措辞（「经字面解析后」「只依据字面路径」）零残留。MODIFIED 整替场景下，智能合并没有用武之地；断言的形状（计数不变 + 旧措辞缺席）比合并算法本身更能兜底。
  5. **未验证也要记录保障边界。** Windows 行为实证（design D6 / tasks 3.4）：**未验证**。用户确认本机不使用 Windows，机器上残留的 Win11 UTM 虚拟机已废弃多年，不作为实证环境；junction 与 8.3 短名下 `filepath.EvalSymlinks` 的实际解析行为维持「不预先假设」。如实记录的是边界而非空白：全部符号链接用例在无法创建软链的环境（含 Windows）自动 skip，不会假失败；`GOOS=windows go build ./...` 交叉编译通过（ADR-0001 不回退）；spec 只承诺「解析后物理位置在根内」，与解析器的具体行为正交，未验证不影响 spec 与方案。将来有真实 Windows 环境时，跑一条「根内 junction 指向根外 → 请求被判 `outside_root`」即可闭合此问号。
  6. **归档时确认 Purpose 仍然为真**（Change 03 观察 6 纪律第六次执行）：`directory-browsing` 的 Purpose 写「不包含文件内容读取」，本 Change 只动越界判定基准，成立；`text-preview` 与 `image-preview` 的 Purpose 未提及判定基准，无需改动。想法池的两条符号链接 wart（条目类型、链接长度）按 design Non-Goals 的预约随状态联动复核措辞——「字面路径可提供」这半句前提已随本 Change 变假，池中条目改写为「可列出性已分裂」的新表述。

### 10 add-file-search

- 日期：2026-10-04（归档目录 `openspec/changes/archive/2026-10-04-add-file-search/`）
- 现象一（AI 把 Spec 写成实现方案）：**未发生**。八条 ADDED Requirement 全部是可观察行为——「以基准位置子树内按名称匹配的条目列表返回」「仅以条目自身的名称进行匹配」「SHALL NOT 在命中条目中提供条目内容」「以确定的顺序返回」「遍历 SHALL NOT 进入符号链接指出的目录」；手写 DFS、`DirEntry.IsDir()`、`strings.Contains(strings.ToLower(...))`、`path.Join`、两段循环等机制全部留在 design D1-D9 与代码注释，spec 里一个机制词都没有。
- 现象二（需求变化被误做成 ADDED）：**未发生**。delta 零 MODIFIED、零 RENAMED；现有六个 capability 的 Requirement 一字未动——搜索结果只是「名字+路径」元数据，不触碰 `Directory listing response` 的「SHALL NOT 提供内容」的门，也不触碰内容判定管道。proposal 提前声明零 MODIFIED，design D4 解释「不交叉引用」的取舍，落地一致。
- 现象三（Apply 偷偷扩大范围）：**未发生**。feat 提交的文件清单与 proposal Impact 逐项对应：后端 `internal/server/search.go` + `search_test.go`、路由在 `server.go` 一行 mux.HandleFunc；前端 `web/index.html`、`web/app.js`、`web/style.css`；`browse.go` 一字未动（`resolve`、`classify`、`sortEntries` 全复用）、`content.go`/`image.go` 一字未动；既有测试（list/content/image）零回归。无功能被静默增删。
- 其他观察：
  1. **机制被复用的边界 = 「复用即机制」需要专门论证**。D7 把「基准失败复用 `resolve`+`classify`、子树失败静默跳过」拆成两条独立决定——基准按既有错误码走，子树按局部降级走，方向相反。两者共用了同一个 `resolve` 函数入口，但语义分裂：基准不放松、子树不扩展。论证「同一份代码不表示同一份语义」的边界要写在 design 里，否则读者会以为复用=语义一致。
  2. **方向相反的两段循环比一段带条件判断的循环更短**。`searchTree` 第一段输出本目录命中、第二段递归子目录，单循环会把先排到的子目录把子树命中插进来——树序契约要求「本目录命中全部先于子树命中」。两段而不是一段，不是为了好看，是为了让顺序契约直接对应到代码骨架（design D5）。换言之：spec 的顺序要求直接决定了循环结构，省掉的就省掉了，多写就多写。
  3. **手写遍历与测试 fixture 的双重隔离**。悬空软链 + 无权限子目录的确定性测试靠 `browse_test.go` 既存的 fixture 模式（tempdir + httptest + 显式 chmod 0）；搜索侧不需要引入新测试基建——同一份 fixture 模板只是新增若干 helper（`makeUnreadableDir` 等）。可复现性来自 fixture 模式而非被测代码本身，与 Change 02 design D7 的 fixture 原则一脉相承。
  4. **瘦条目让遍历天然免疫元信息取不到的边角**。`searchTree` 从头到尾不取条目元信息——`listEntry` 在这里只是「名字 + 类型」排序用，不调 `entryInfo`。结果：悬空软链无 Info/Stat 可失败、照常命中；软链目录的子树不下钻但软链本身按名字参与匹配。瘦是契约层的瘦（D4），也是遍历层的免疫——后者是前者自然带来的副产品，不必刻意去「修」这个边角。
  5. **22 个 spec Scenario 对应 22 个测试函数**：1:1 对得齐没有隐式合并——每条 Scenario 的「WHEN/THEN」都有独立的确定性测试函数（部分场景如「同一目录内列表顺序」与「树序」共享同一 fixture 的多个断言，但仍是独立的 test 函数）。这是 Change 02 起的纪律延续：spec 的 Scenario 列表与测试函数列表同构，可逐项勾对，验收不再靠记忆。
  6. **归档时确认 Purpose 仍然为真**（Change 03 观察 6 纪律第七次执行）：六个现有 capability 的 Purpose 一字未改（proposal 与 delta 一致零 MODIFIED），新增 capability `search` 的 Purpose「不包含依据文件内容的搜索」与代码（不读 `entryInfo`、不读文件内容、` searchMatch` 无内容字段）一致。想法池新增一条「内容搜索」按 proposal Non-Goal 与 design D1 的预约，与「遗留中文编码按 UTF-8 处理」的乱码问题在此会放大被一并点到（roadmap 想法池同步写入）。
