# Design: add-remote-access

## Context

服务当前由 `internal/app/app.go` 硬编码监听 `127.0.0.1:8080`，`internal/server/server.go` 用 `NewHandler` 组装根 mux（`/api/` 交给 API 分区，其余交给内嵌静态资源）。整个 HTTP 表面是**只读**的，不做任何身份检查。`run(args, stdout, stderr, assets, listen)` 是启动层的测试 seam（`listen` 可注入），`NewAPIHandler` / `apiHandler` 是 handler 层的 seam。

本 Change 首次在 HTTP 表面引入一道门。动机与范围见 `proposal.md`，行为契约见 `proposal.md` 声明的两个 delta spec（新增 `authentication`、修改 `service-startup`）。以下只记录「怎么做」及其取舍。

## Goals / Non-Goals

**Goals:**

- 在**不改默认启动行为**的前提下，允许显式指定非回环监听地址。
- 用**单一、无端点例外**的凭证机制保护 Listen 非回环时的全部请求。
- 认证逻辑与现有端点解耦：端点代码零改动。
- 默认回环模式下不生成凭证、不要求认证，现有测试与行为零回归。

**Non-Goals（设计层，proposal 已列的不重复）：**

- 不引入服务端会话状态：认证必须是无状态的，避免多一份需要过期与清理的数据。
- 不引入任何第三方依赖（沿用 ADR-0001 单二进制、零运行时的约束）。

## Decisions

### D1. 认证触发按**监听地址**，不按请求来源 IP

规则简化为「暴露到网络 = 要凭证」。实现只需在启动时对 `--listen` 的值做一次回环判定，此后认证层是纯开关，不需要解析每个请求的 `RemoteAddr`，也回避了「反代后 RemoteAddr 不可信」的问题。代价是监听非回环时本机访问也要凭证——可接受，且与 `authentication` spec 的 `Loopback-only binding does not require authentication` 一致。

替代方案：按来源 IP 判定。更顺手，但引入逐请求 IP 解析与代理信任问题，且规则不再是一句话。放弃。

### D2. 凭证用 **Cookie** 承载，token 值即 Cookie 值（无服务端状态）

这是本设计的核心。浏览器请求分两类：**我们自己的 JS 发起的**（可自定义 `Authorization` 头）与**浏览器自己发起的**（页面导航、`<script>`/`<link>`、`<img>`，无法设头）。若用 `Authorization` 头，后一类必须靠 URL 特例或 `fetch`+blob 打补丁——两者都在破坏「单一机制」。

Cookie 的浏览器**自动携带**特性从根上消除这个分裂：一次换取、全站生效，`<img>` 天然可用。为保持无状态，**Cookie 值直接等于启动生成的 token**，服务端只做常量时间比较，不维护任何会话表、不做过期清理。

替代方案：(a) `Authorization` 头 + 图片 blob/URL 特例——已否决，不一致；(b) 随机 session id + 服务端 map——需要过期与并发管理，无必要；(c) 全程 URL `?token=`——token 进历史/日志/Referer，且静态资源无法动态携带。放弃。

### D3. `--listen` 参数与回环判定

新增 `--listen <host:port>`，默认 `127.0.0.1:8080`；解析与判定放启动层（`internal/app`）。判定用标准库：`net.SplitHostPort` 拆主机与端口，主机为空视为「所有网卡」（非回环），`localhost` 特判为回环，其余 `net.ParseIP(host).IsLoopback()`；解析不出 IP 且非 `localhost` 视为非回环（安全默认：不认识就当远程）。非法值（缺端口、端口非数字）按启动错误报错退出。

参数解析需要从「纯位置参数」升级为「flag + 一个位置参数」：保持 `<目录>` 仍是唯一位置参数，`--listen` 可前可后。

替代方案：分开 `--host` / `--port`。参数更多、组合校验更多，无收益。放弃。

### D4. token 生成、生命周期与比较

- 生成：`crypto/rand` 取 32 字节，`base64.RawURLEncoding` 编码（URL 安全，适合放进 `?token=`）。
- 时机：仅在解析出非回环地址时生成；回环启动完全不生成。
- 生命周期：进程内有效，重启即换，不落盘——对应 `authentication` spec 的 `Access credential`。
- 比较：`crypto/subtle.ConstantTimeCompare`，避免长度/前缀时序差异。

### D5. 认证层位置与路由

在**根 handler 最外层**包一层认证 middleware（`app` 把 `auth` 配置传给 `server.NewHandler`），所有请求先过门，再进现有 mux。现有端点内部零改动，满足 Goals。

middleware 逻辑：

```
若未启用认证（回环）      -> 直接放行
若请求带 ?token=<有效>     -> Set-Cookie + 302 到去掉 token 的同 URL（D6）
若请求携带有效 Cookie      -> 放行
否则                      -> 拒绝：
     /api/ 前缀           -> 401 + JSON 信封 {error:{code:"unauthorized",...}}
     其余（页面/静态资源） -> 401 + 登录页 HTML（D7）
```

按路径前缀区分失败表现，对应 `authentication` spec 的 `Authentication failure reporting`。

### D6. Cookie 属性与 token 交换

`Set-Cookie` 属性：`HttpOnly`（JS 读不到，降低 XSS 泄露面）、`SameSite=Strict`（跨站一律不带，从机制上顶住 CSRF）、`Path=/`。**不设** `Secure`（纯 HTTP，设了浏览器不保存）与 `Max-Age`（会话 Cookie，关浏览器即失效，与「运行期凭证」语义一致）。

首次以 `/?token=<token>` 访问时，middleware 校验后 `Set-Cookie` 并 302 到去掉 `token` 查询参数的同一 URL（保留 `path`、`q` 等其它参数），对应 spec 的 `The credential does not remain in the browsed location`。

### D7. 登录入口由服务端提供，且自包含

未认证的**页面**请求返回一个最小登录页（401 + HTML），含一个表单 `GET /`、字段 `name="token"`。实现为内嵌资源 `web/login.html`，由 middleware 读取后作为 401 响应体写出。

要点：登录页**不得引用任何外部资源**（`<link>` / `<script>`），否则那些请求同样未认证、拿不到资源。样式内联。这样无需为登录资源开任何白名单例外，也不会形成重定向环。

前端 `web/app.js`：`ERROR_TEXT` 增加 `unauthorized` 一条；`fetchJSON` 收到 401 时跳回 `/`（由服务端给出登录页），而不把它当普通内容错误展示。

### D8. 启动输出

- 回环启动：输出不变（现有 `服务已就绪，访问 http://... 浏览 ...`）。
- 非回环启动：在现有地址行之外追加——已启用远程访问的提示、`凭证：<token>`、以及一条同机可直接点的 `http://localhost:<port>/?token=<token>` 链接（监听主机为 `0.0.0.0` 时它本身不可浏览，故另给 localhost 形式）。对应 `service-startup` 的 `Remote access is enabled`。

### D9. ADR-0004

新建 `docs/adr/0004-default-read-only-explicit-opt-in.md`：把「默认只读；开放远程访问与写入均须显式开启」正式确立为长期决策，作为 Phase 4 安全姿态的落点。

## Risks / Trade-offs

- **token 在网络上明文传输**（无 TLS）→ 记录为明确的非目标与已知限制，不中途引入 TLS（零配置约束）。局域网内防的是「无心访问」而非「有能力的窃听者」。
- **Cookie 值即 token**，Cookie 一旦泄露等于凭证泄露 → `HttpOnly` + `SameSite=Strict` 缩小暴露面；纯 HTTP 下无法再加 `Secure`。
- **DNS rebinding 仍无防御**（Host 不校验）→ 回环无凭证模式尤其暴露；本 Change 不处理，已记入 `docs/roadmap.md` 想法池，待独立 Change。
- **`SameSite=Strict` 的副作用**：从外部链接首次进入不带 Cookie，需要重新输入 token → 属预期，登录页是兜底入口。
- **测试不能绑定真实非回环端口**（会真的对外暴露）→ 启动层用注入的 `listen` seam 断言「按 `--listen` 解析出的地址」；认证行为在 handler 层用构造好的配置直接测，不依赖真实 socket。这延续 `app_test.go` 既有的「不绑定真实端口」纪律。
- **服务端渲染登录页进 `web/`** 与 proposal「前端呈现 token 输入」措辞略有出入 → design 精化为「系统呈现」：登录页是 `web/` 下的内嵌资源，由服务端在未认证时输出，前端 JS 只负责把 API 401 导回入口。

## Migration Plan

无数据迁移。默认启动行为与现有 spec 场景零变化；回滚即删除认证层与 `--listen` 参数。上线顺序：先 `--listen` 与回环判定（默认不变）→ 再认证层与 token → 最后前端登录/401 与 ADR。
