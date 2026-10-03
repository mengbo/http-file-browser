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
