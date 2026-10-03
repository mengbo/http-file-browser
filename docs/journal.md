# OpenSpec 学习实验记录

> 教材（[OpenSpec_HTTP_File_Browser.md](OpenSpec_HTTP_File_Browser.md)）定义了要观察什么；本文件记录**实际观察到了什么**。每完成一个 Change，在「实验日志」补一节，并勾选验收清单。

## 验收清单

来自教材第 59 节。

- [ ] 我知道 `openspec/specs` 和 `changes` 的区别
- [ ] 我知道 delta spec 是什么
- [ ] 我知道 ADDED 和 MODIFIED 的区别
- [ ] 我知道 MODIFIED 为什么必须保留完整 Requirement
- [ ] 我知道 Scenario 与测试的关系
- [ ] 我知道 Proposal、Spec、Design、Tasks 各自解决什么问题
- [ ] 我知道什么时候使用 explore
- [ ] 我知道什么时候使用 update
- [ ] 我知道 sync 与 archive 的区别
- [ ] 我知道为什么已经 Archive 的 Change 不应该直接修改
- [ ] 我能够从历史 Change 回溯某个系统行为为什么存在
- [ ] 我能够从当前 Spec 判断系统应该有什么行为
- [ ] 我能够让 AI 在实现过程中停止自行扩大需求
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
