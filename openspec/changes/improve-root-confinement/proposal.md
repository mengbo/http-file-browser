# Proposal: improve-root-confinement

## Why

`directory-browsing` 自 Change 02 起承诺**只按字面路径**判断越界：根目录内经符号链接指向根目录之外的路径按其字面路径提供内容。该承诺在回环单用户下自洽（能发请求的人本就能读本机文件），但 `add-remote-access`（Change 13）开放非回环监听后，同一行为等于向同网段任何主机开放任意文件读取。Change 02 的 design D3 已把「翻案时机」预约为 13 的前置；本 Change 兑现该预约，插队至 Change 10 之前执行，使 13 只需处理监听与认证本身。

## What Changes

- 越界判定基准由**字面路径**改为**物理路径**：请求路径解析其全部符号链接后，物理位置落在（解析后的）根目录物理位置之内才提供内容，否则拒绝。**无任何开关**。
- **BREAKING**：此前被 spec 明确承诺的行为被移除——根目录内经符号链接指向根目录之外的路径，从「按该路径提供内容」变为「拒绝并返回 `outside_root`」。
- 根目录路径自身经由符号链接时，以解析后的物理位置为判定基准（macOS 上根目录置于 `/tmp` 之下等环境不回归）。
- 拒绝越界复用既有机器可读错误标识 `outside_root`，不新增词汇；悬空符号链接维持既有 `not_found` 行为不变。
- 错误原因条目中「经字面解析后落在根目录之外」的 WHEN 措辞随判定基准同步更新（三个 capability 各一处）。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `directory-browsing`：`Root directory confinement`——判定基准字面改物理，出根符号链接 Scenario 翻转为拒绝，新增「根内符号链接仍可用」「根目录路径自身含符号链接」两条 Scenario；`Directory access failures`——`outside_root` 对应 Scenario 的 WHEN 措辞随判定基准更新。
- `text-preview`：`Text file content response`——出根符号链接 Scenario 翻转为拒绝；`Content reading failures`——`outside_root` 对应 Scenario 的 WHEN 措辞更新。
- `image-preview`：`Image content response`——出根符号链接 Scenario 翻转为拒绝；`Content reading failures`——`outside_root` 对应 Scenario 的 WHEN 措辞更新。

## Impact

- `internal/server/browse.go`：`resolve()` 增加物理位置判定；根目录物理位置在服务启动时解析一次（三个端点共用同一 `resolve()`，端点层代码零分支变化）。
- 测试：`browse_test.go` / `content_test.go` / `image_test.go` 各一处宽松断言反转为拒绝断言；新增根内符号链接、根路径含符号链接、中间段符号链接出根、悬空符号链接等用例。
- 前端 `web/`：零变化（错误呈现为通用的 JSON 错误信封展示）。
- 无新增依赖；`path/filepath` 标准库足以覆盖（不引入第三方路径守卫库）。
- `docs/roadmap.md`：Change 地图新增第 17 行（已在立 Change 时落盘），想法池「符号链接越界策略」条目标记认领。
