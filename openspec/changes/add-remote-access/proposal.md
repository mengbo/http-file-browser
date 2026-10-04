# Proposal: add-remote-access

## Why

服务当前只监听回环地址（`127.0.0.1:8080`），只能本机访问；要服务于局域网内的其他设备（手机、另一台电脑），需要能显式指定监听地址。但开放非回环监听意味着同网段任何主机都能读取所浏览目录的内容——「开放远程访问」与「引入访问认证」是同一个安全决定，必须一次落地，不能先开放、后补门禁。

原 Change 地图把两者拆成 13（监听地址）与 14（Token 认证），本次合并为一个 Change：避免在两次归档之间留下「远程可访问且无需认证」的不安全状态。原 15（认证行为变化演练）取消——MODIFIED 演练已由 05、08 完成。

## What Changes

- 新增启动参数 `--listen <host:port>`，默认值仍为 `127.0.0.1:8080`（默认行为零变化）；`:8080` 等空主机表示监听所有网卡（非回环）。
- 回环判定：`localhost` / `127.0.0.0/8` / `::1` 视为回环；其余（含 `0.0.0.0`、`::`、空主机、其他地址）视为非回环。
- 仅当监听非回环地址时启用认证：启动生成一次性随机 token（进程内有效、重启即换、不落盘）并打印；对服务的**全部** HTTP 请求（页面、静态资源、`/api/`、图片字节）要求携带有效凭证。默认回环启动完全不生成 token、不要求认证。
- 凭证经 Cookie 传递：首次以 `/?token=<token>` 访问，服务端校验通过后 `Set-Cookie`（HttpOnly、SameSite=Strict）并跳转到不含 token 的干净 URL；此后浏览器自动在每个请求上携带该 Cookie。全站统一，无端点例外——图片 `<img>` 因此天然可用，无需 URL 特例或 blob 变通。
- 未携带或携带无效凭证的请求，失败标识为新增的 `unauthorized`（HTTP 401）；`/api/` 区域仍以 JSON 错误信封返回。
- 前端：缺少凭证时呈现 token 输入；`?token=` 链接与输入框两条进入路径。凭证只存活在会话内，不写入可分享的 URL。
- **BREAKING**（就 spec 而言）：`service-startup` 中「不监听非回环地址」的绝对承诺改为「默认不监听，除非显式配置」——对默认启动的用户零行为变化。
- 新增 ADR-0004：默认只读；开放远程访问与写入均须显式开启。

## Capabilities

### New Capabilities

- `authentication`：远程访问下的凭证契约——何时要求认证（监听非回环时）、凭证的来源（启动生成的一次性随机 token）、取得与携带方式（`?token=` 换取 Cookie、之后浏览器全站自动携带）、失败标识（`unauthorized`）、以及仅监听回环时不启用认证的豁免。

### Modified Capabilities

- `service-startup`：`Loopback-only binding by default`——由「只监听回环」放宽为「默认回环、可显式指定非回环」；`Startup outcome reporting`——非回环启动时额外报告已启用远程访问及访问所需 token。

## Impact

- `internal/app/app.go`：`listenAddress` 常量改为可配置（新增 `--listen` 参数解析），启动输出按是否启用远程访问分支。
- `internal/server/`：新增认证层（Cookie 解析、凭证校验、`unauthorized` 错误码与状态码映射），在路由外统一加门；现有端点内部零改动。
- 新增 token 生成（`crypto/rand`）与常量时间比较（`crypto/subtle`）逻辑。
- `web/`：凭证的取得、存储与携带；401 呈现；token 输入视图；`ERROR_TEXT` 增加 `unauthorized`。
- Spec：新增 `openspec/specs/authentication/spec.md`；修改 `openspec/specs/service-startup/spec.md`。
- 无新增第三方依赖（标准库 `net` / `crypto/rand` / `crypto/subtle` / `encoding/base64` 足够）。
- `docs/roadmap.md`：Change 13/14 合并、15 取消、Host 校验进想法池（已在立 Change 时落盘）。

## Non-goals

- TLS/HTTPS：服务仍是纯 HTTP，token 在网络上明文传输——零配置的自觉代价，明确记录而非假装不存在。
- token 的过期、轮换、持久化；用户系统与多凭证。
- Host 头校验（防 DNS rebinding）：属只读场景的历史遗留，记入想法池独立处理。
- Origin 校验：待文件编辑引入写请求时再加，那时它才真正生效。
- 写入相关的开关与语义：属文件编辑（Change 11）。
