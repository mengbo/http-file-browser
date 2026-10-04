# Tasks

## 1. 前端实现（design D1 / D2 / D3）

- [x] 1.1 改写 `web/app.js` 的 `formatModifiedAt`：改用 `getFullYear` / `getMonth() + 1` / `getDate` / `getHours` / `getMinutes` / `getSeconds`，各数值字段 `padStart(2, "0")` 后拼接为 `YYYY-MM-DD HH:mm:ss`；保留 `typeof !== "number" || !isFinite` 返回 `""` 的守卫；验证：代码中不再出现 `toLocaleString`，月份经过 `+ 1`，六个数值字段全部补零
- [x] 1.2 确认缺失值仍留空：`formatModifiedAt` 对非有限数值返回空串，`renderEntries` / `parentRow` 的 `textCell` 路径未变；验证：元信息缺失的条目 `entry-time` 为空、无占位符
- [x] 1.3 窄屏列宽验证与按需调整（design D4）：在 34rem 断点下用固定 19 字符时间确认不溢出、不折行；仅当实测折行才调整 `web/style.css` 的窄屏 `entry-time` 列宽（8.25rem）与 `entries-header` 同步值；验证：视觉检查通过

## 2. 呈现验收（ADR-0003：浏览器走查，覆盖本 Change 三个 Scenario）

- [x] 2.1 走查 Scenario `修改时间以固定格式呈现`：用 `testdata/` 作根，对已知 `modified_at` 的条目断言显示为 `YYYY-MM-DD HH:mm:ss`、各字段补零正确；验证：与 `date -r <文件>` 手算对照
- [x] 2.2 走查 Scenario `呈现格式不随语言变化`：把浏览器语言切到 en-US 后重载，断言格式仍为 `YYYY-MM-DD HH:mm:ss`，不再出现 `10/4/2026, 11:35:25 AM`；验证：两种语言下视觉对比一致
- [x] 2.3 走查 Scenario `修改时间按本地时区展开`：断言显示为本地时区的墙钟时刻；验证：与 `date` 输出对照
- [x] 2.4 记录实际观察结果（沿用 Change 03 的表格惯例）：逐项填写观察项与实际结果，**未实测的项留空不勾选**；验证：表格中每一勾选行都有对应观察

## 3. 集成检查

- [x] 3.1 跑 `go build ./...`、`go vet ./...`、`go test ./...`：确认无 FAIL、无新增告警（本 Change 只动前端，后端应零变化）；验证：命令输出
- [x] 3.2 端到端冒烟：`go run . testdata` 启动后，目录列表、进入子目录、返回上级的时间列均按固定格式显示且导航正常；验证：逐项记录实际观察结果

## 2.4 / 3.2 实际观察结果

环境：macOS；`go run . testdata` 启动服务，根目录 `testdata/`，服务地址 `http://127.0.0.1:8080`；浏览器走查用 agent-browser（Chromium），`date` 为对照基准。为覆盖月份 +1 与单数字段补零，临时新增夹具 `testdata/time-fixture.txt`（`touch -t 202603040705.09`，mtime 2026-03-04 07:05:09，epoch 1772579109），走查后删除。语言与时区切换用 CDP `Emulation.setLocaleOverride` / `Emulation.setTimezoneOverride` 后重载页面实现。

| 观察项 | 实际结果 |
|---|---|
| Scenario `修改时间以固定格式呈现`（2.1） | `README.md`（`modified_at=1791084925`）显示 `2026-10-04 11:35:25`，与 `date -r README.md` 一致；临时夹具 `time-fixture.txt` 显示 `2026-03-04 07:05:09`，覆盖月份 +1（3 月而非 2）与月/日/时/分/秒补零（`03`/`04`/`07`/`05`/`09`）；根列表全部 10 条时间均匹配 `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$` |
| Scenario `呈现格式不随语言变化`（2.2） | 默认浏览器 locale 为 zh-CN，旧实现会显示 `2026/3/4 07:05:09`；用 CDP 覆盖页面 locale 为 en-US 并重载后，`new Date(1772579109*1000).toLocaleString()` 返回 `3/4/2026, 7:05:09 AM`（即旧实现会显示的形式），而时间列仍为 `2026-03-04 07:05:09`，两种语言下格式完全相同 |
| Scenario `修改时间按本地时区展开`（2.3） | 本机默认 `CST +0800`，`time-fixture.txt` 显示 `2026-03-04 07:05:09`，与 `TZ=Asia/Shanghai date -r` 一致；CDP 覆盖时区为 `America/New_York` 重载后显示 `2026-03-03 18:05:09`（`README.md` 显示 `2026-10-03 23:35:25`），与 `TZ=America/New_York date -r` 逐项一致；同一 `modified_at` 取值（1772579109 / 1791084925）不随环境变化 |
| 缺元信息 / `..` 行留空（呼应 1.2） | 上级入口 `..` 行的 `entry-size` 与 `entry-time` 均为空字符串，无占位符 |
| 34rem 断点列宽（1.3，design D4） | 视口 544px（34rem）下窄屏 `entry-time` 列宽 132px（=8.25rem），19 字符时间单行渲染（`lines=1`，高 17.4px），`scrollWidth===clientWidth===132`，`documentElement.scrollWidth===innerWidth===544`，无溢出、无折行——按 design D4 默认不改 `style.css` |
| 控制台与页面错误 | `agent-browser errors` 与 `agent-browser console` 均无输出 |
| 端到端冒烟（3.2） | 根列表时间列按固定格式显示；点击 `docs` 进入 `?path=docs`，`config.yaml` / `data.csv` / `hello.txt` / `notes.md` 时间列同为 `2026-10-04 11:35:25`；点 `..` 返回 `?path=`，根列表恢复且 `..` 行时间/大小为空；目录列表、进入子目录、返回上级三步导航均正常 |
| 集成命令（3.1） | `go build ./...` 与 `go vet ./...` 无输出（通过）；`go test ./...`：`internal/app ok`、`internal/server ok`，无 FAIL、无新增告警 |

范围外观察（不计入本 Change、未修改代码）：320px 视口下 `documentElement.scrollWidth=385 > innerWidth=320`；把时间文本临时替换为旧的短格式（`2026/9/16 13:20:00`）后 scrollWidth 仍为 385，说明该溢出与本次固定格式无关，属既有窄屏布局问题，非本 Change 引入，也不在 1.3 的 34rem 任务范围内。
