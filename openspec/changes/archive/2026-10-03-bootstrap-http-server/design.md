# Design

技术栈已由 ADR-0001（Go 单二进制）与 ADR-0002（JSON API + 内嵌零构建静态前端）确定，本文不重复论证，只补充本 Change 特有的决策。动机见 proposal.md，行为契约见 `specs/service-startup/spec.md`。

## Context

- 仓库中无 Go module、无源码，这是第一个可运行的 Change
- ADR-0002 承诺「每个 Scenario 对应一个 API 层的 Go 测试（`net/httptest`）」，并把映射细则显式留给本 Change 的 design.md 落地
- ADR-0002 已确定前端零构建：无 npm、无打包器，第三方库日后以 vendored 静态文件进仓库
- 后续 Change（目录浏览、文本预览、编辑）都会在本 Change 建立的 HTTP 表面上叠加端点，因此响应分区与错误形态一旦定错，修正成本随 Change 数量线性上升

## Goals / Non-Goals

**Goals:**

- 用一个可验证的端点证明 `go:embed` 内嵌、静态页返回、`fetch` 往返三件事都成立
- 让 `specs/service-startup/spec.md` 中每条 Scenario 都有对应的自动化测试落点
- 确立 JSON 错误信封形态，使后续 Change 直接沿用而不各自发明
- 保持零构建：`go build` 得到的产物即完整程序

**Non-Goals:**

- 不做目录列表、路径导航、文件元信息——这些是 `directory-browsing` 的内容
- 不做监听地址可配置、token、认证——`--addr` 属 Change 13
- 不引入 CLI 框架、路由框架、JSON 库、前端框架
- 不引入前端构建流水线与 `package.json`

## Decisions

### D1. 目录布局

```
.
├── go.mod
├── main.go                 包 main，仅负责调用 internal/app
├── internal/
│   ├── app/                 启动流程：解析参数 → 校验 → 监听 → 报告 → 服务
│   └── server/              HTTP handler 与路由
└── web/                     前端源文件，随二进制内嵌
    ├── web.go               package web，承载 //go:embed 指令
    ├── index.html
    ├── app.js
    └── style.css
```

选择 `internal/` 而非扁平包结构：骨架阶段就要让「启动流程」与「HTTP 处理」分属两个包，否则 Change 02 加入路径解析后 `main` 包会同时承担参数校验、路由、文件系统访问三件事。分包的边界正好对应 Change 02 要接入的位置。

替代方案：全部放 `main` 包。更短，但 Change 02 起就要重构，且教材 §12 强调的「一个 Requirement 一个边界」在代码结构上就失去了对应物。

### D2. 前端资源用 `go:embed` 内嵌，零构建

`web/` 目录的内容以 `go:embed` 嵌入二进制，运行期通过 `http.FileServer(http.FS(...))` 提供。构建流程只有 `go build`，没有第二阶段。

`web/` 自身是一个 Go 包（`package web`），由它承载 embed 指令并导出 `web.FS`；`internal/app` 把这个 `fs.FS` 传给 `server.NewHandler`。`internal/server` 只认 `fs.FS`，不 import `web` 包，因此静态资源的位置对它是透明的。

这条是被 Go 的 embed 规则逼出来的：`//go:embed` 只能内嵌**本包目录及其子目录**。若按最初设想由 `internal/server` 写 `//go:embed web`，它会去 `internal/server/web` 找文件，而 web/ 在仓库根——`internal/` 下的任何包都无法引用它。想保留「web/ 在仓库根」和「main.go 只做 `os.Exit`」两条，只能让 web/ 自己成为包，代价是 embed 指令从目录通配（`//go:embed web`）变成列举文件（`//go:embed index.html app.js style.css`）：新增前端文件时必须同步补进指令，否则编译期就不报错、运行期 404。列举文件在这里是更安全的失败方向，故接受。

替代方案：CGO 或外部资源目录。都会破坏 ADR-0001 的单二进制约束。

替代方案：把 `web/` 移进 `internal/server/web`，用 `//go:embed web` 目录通配。可行，但 D1 的目录树要改，且日后若有非 server 的包要用前端资源就得再搬一次。

### D3. 路由分区用标准库 `ServeMux` 的路径前缀

```
请求路径
   │
   ├─ 前缀 /api/  ──▶ server.NewAPIHandler()   → JSON（含错误）
   │
   └─ 其余（含 /） ──▶ 文件服务（内嵌 web/）   → HTML / 静态资源
```

两支各自独立构造 handler，再由根 mux 分派。`/api/` 分支必须先注册，否则 `/api/` 会落到文件服务上，返回一个「找不到」的 HTML 页——那正是本 Change 要消灭的情形。

具体做法：在 API 分支内为未匹配路径注册显式的 not-found handler，输出 JSON 错误信封，而不是依赖 mux 的默认 404（默认 404 是 `text/plain`）。这条对应 spec 中「未知 API 端点返回 JSON 错误」这条 Scenario。

模式字符串必须是 `"/api/"`（带尾斜杠）而不是 `"/api"`：Go 的 `ServeMux` 中不带尾斜杠的是**精确匹配**，只有带尾斜杠才是前缀匹配。写错的表现是 `/api/health` 命中而 `/api/nope` 落到文件服务上返回 HTML 404——恰好是本 Change 要消灭的失败模式，且测试若只覆盖已知端点就发现不了。因此 `TestAPIPrefixTakesPrecedenceOverFileService` 特意对 `/api/nope`、`/api/`、`/api/unknown/deep` 三条都断言。

替代方案：引入第三方路由库。标准库足够，且 ADR-0001 要求依赖数量克制。

### D4. JSON 错误信封用单一 `error` 字段

```json
{ "error": "not found: /api/nope" }
```

这是一个 **design 决策而非 spec 行为**：spec 只约束「机器可读的错误标识 + 人类可读的说明」与「内容类型为 JSON」，字段命名属于实现约定。

选择单字段而非 `{"error":{"code":"...","message":"..."}}`：本阶段唯一消费者是同仓库的前端，人类可读说明已足够定位于日志与调试；结构化 code 的价值在于多客户端或前端需要分支处理时，那属于后续 capability 出现后的演进，届时用 MODIFIED 表达，而不是现在预留一个无人使用的字段。

替代方案：现在就上 `code` + `message`。为想象中的需求付出字段膨胀成本，且空 code 集合会诱使前端写出永远走不到的分支。

所有 JSON 响应体由标准库 `encoding/json` 手写 map 或小结构体序列化，不引入第三方 JSON 库。

### D5. 健康端点为 `/api/health`

```json
{ "status": "ok" }
```

存在的理由有两条，缺一不可：一是它给 Go 测试一个与目录无关的稳定断言点；二是内嵌前端 `fetch` 它并把结果渲染进 DOM，用最小代码证明前后端链路连通。

明确承认它带「为测试而生」的成分。保留而非删除，是因为健康检查本身是正当的 HTTP 端点形态，且不引入任何额外概念。若日后确认无外部消费者，可在 polish 阶段作为独立决策移除。

替代方案：前端 Change 01 不发请求，只渲染静态文案。更干净，但那样「前后端打通」这件事的首次证明会推迟到 Change 02——那时骨架已被目录列表逻辑覆盖，一旦链路有问题，排查要在两层逻辑里做。半天的成本省在错误的地方。

### D6. 监听地址在本 Change 硬编码为 `127.0.0.1:8080`

端口被占用时**报错退出**，不自动顺延到下一个端口。

理由：自动顺延会让「用户实际访问哪个端口」只能从启动输出推断，而启动输出恰恰是零配置场景里唯一的信息来源。让端口冲突显式失败，用户立刻知道发生了什么；顺延成功则可能让用户访问了错误的地址并以为是服务的问题。

替代方案：自动顺延 + 清晰输出。对本项目（单人本地工具）收益不大，且违背后续 Change 13「显式配置监听地址」的方向。

### D7. Scenario 到测试的映射细则（兑现 ADR-0002）

```
spec 中的 Scenario                     测试落点
────────────────────────────────────────────────────────────────────
根目录参数校验 / 退出码              app 包的启动函数单测
                                       注入参数、捕获 stdout/stderr、
                                       断言退出码与监听是否被调用

启动结果报告 / 地址打印              同上，断言输出内容含协议主机端口

默认仅监听回环                       app 包单测：用注入的 listen 捕获
                                       run 请求的地址，断言主机是回环 IP
                                       且端口为 8080。不断言真实 socket，
                                       理由同上——绑 8080 会与本地运行的
                                       程序抢端口；真实 socket 的回环属性
                                       由端到端观察承担（lsof 应显示
                                       127.0.0.1:8080 LISTEN）

根路径 / 静态资源 / 内容类型         server 包 httptest 单测，
                                       GET 后断言状态码与 Content-Type

健康端点                              server 包 httptest 单测，断言 JSON

未知 API 端点返回 JSON 错误          server 包 httptest 单测，
                                       断言状态码 404、
                                       Content-Type 为 JSON、
                                       body 可反序列化且含 error 字段
```

三条约定：

1. **CLI 行为可测**：启动逻辑拆成两层——导出的 `Run(args []string, stdout, stderr io.Writer) error` 供 `main` 调用，退出码由它的返回值决定；内部的 `run` 多带两个可注入参数 `assets fs.FS` 与 `listen listenFunc`，单测直接调它，不启动子进程，避免每个 Scenario 都付进程启动与端口占用竞态的成本。`main` 只做 `os.Exit`。

   两个注入参数各自解决一个具体障碍，不是可选的便利：

   - `listen`：tasks 2.3 要求断言「listener 地址为回环地址」而 design 又要求「测试不绑定真实端口」。`net.Listen` 不是接口，无法在测试中替换；不注入就只能真的绑 8080，违反后者。测试注入返回**已关闭**的 `net.Listener`，`run` 于是打印地址后立即从 `http.Serve` 返回。
   - `assets`：让 `internal/app` 的单测不必依赖 `web` 包的内容，app 层的测试用 `fstest.MapFS` 即可。
2. **前端文件服务可测**：因为资源来自 `embed.FS` 而非磁盘，测试通过 `http.FileServer(http.FS(...))` 构造 handler，不需要起真实服务器、不需要写临时目录。
3. **前端自身不做自动化测试**：零构建意味着没有测试运行器（除非日后 vendored 引入）。所有前端行为由对应 API 层 Scenario 间接覆盖。这与 ADR-0002 已接受的代价一致，记录在此以免日后误以为是遗漏。

## Risks / Trade-offs

- **骨架被后续 Change 大幅重写** → 这是本 Change 最主要的风险。缓解方式是把承诺压到最小：只要求内嵌、静态页返回、fetch 往返三件事可验证，不预建路由与渲染骨架。
- **spec 中「响应内容类型与资源类型相符」缺少枚举** → 目前只知道静态资源可能是 CSS/JS/图片，把枚举写进 spec 会让它随前端演进而频繁 MODIFIED。缓解：该断言的强度由测试承担（对已知的 css/js 分别断言），spec 只保留可观察的定性表述。
- **前端零自动化测试** → 前端回归只能靠人工点击。缓解：把可观察行为尽量表述为 API 层行为，使尽可能多的 Scenario 落在有测试的一侧。
- **硬编码端口在并行开发时冲突** → 单人项目下影响有限。缓解：测试不绑定真实端口，冲突只发生在手工运行程序时。

## Migration Plan

不适用。首次引入可运行产物，无既有部署需要迁移或回滚。删除 `main.go` 与 `internal/` 即回到当前状态。

## Open Questions

以下问题可以推迟到后续 Change 回答，届时不改变本 Change 的 spec、方案与任务划分：

- 内嵌静态资源是否需要 `Cache-Control` 策略（文件名不带内容 hash，变更即同名）
- 前端在 Change 02 之后是否需要路由与历史记录管理，取决于目录浏览的交互复杂度
