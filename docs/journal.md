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

- 日期：
- 现象一（AI 把 Spec 写成实现方案）：
- 现象二（需求变化被误做成 ADDED）：不适用（本 Change 为新增 capability）
- 现象三（Apply 偷偷扩大范围）：
- 其他观察：
