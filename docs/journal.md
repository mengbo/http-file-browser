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
