# service-startup Specification

## Purpose

定义命令行启动文件浏览服务时用户可观察到的行为契约：如何指定要浏览的根目录、启动成功或失败时系统如何告知、HTTP 表面上静态页面与 JSON 接口如何划分。不包含目录浏览本身的行为。

## Requirements

### Requirement: Root directory argument validation

系统 SHALL 在启动前校验用于指定根目录的命令行参数，并在该参数缺失或不可用时向标准错误输出说明原因，且以非零状态退出。系统 SHALL NOT 在根目录不可用时启动 HTTP 服务。

#### Scenario: Root directory argument is provided and usable

- **WHEN** 用户以一个存在且可读的目录作为参数启动
- **THEN** 系统以该目录为根目录启动 HTTP 服务，且以零状态退出前的运行状态持续服务

#### Scenario: Root directory argument is missing

- **WHEN** 用户未提供根目录参数就启动
- **THEN** 系统向标准错误输出说明缺少该参数，并以非零状态退出，且不启动 HTTP 服务

#### Scenario: Root directory argument does not exist

- **WHEN** 用户提供的路径不存在
- **THEN** 系统向标准错误输出说明该路径不可用，并以非零状态退出，且不启动 HTTP 服务

#### Scenario: Root directory argument is not a directory

- **WHEN** 用户提供的路径存在但不是目录
- **THEN** 系统向标准错误输出说明该路径不是目录，并以非零状态退出，且不启动 HTTP 服务

### Requirement: Startup outcome reporting

系统 SHALL 在启动阶段向标准输出报告结果：成功时给出浏览器可直接访问的地址；失败时给出可定位的原因。系统 SHALL 在无法绑定监听地址时以非零状态退出，且不进入服务状态。当启用远程访问（监听非回环地址）时，系统 SHALL 在启动输出中额外报告已启用远程访问，并给出访问所需的凭证。

#### Scenario: Service starts successfully

- **WHEN** 服务成功开始监听
- **THEN** 系统向标准输出打印包含协议、主机与端口的访问地址

#### Scenario: Listen address cannot be bound

- **WHEN** 期望的监听地址已被占用或不可用
- **THEN** 系统向标准错误输出说明该地址不可用，并以非零状态退出，且不进入服务状态

#### Scenario: Remote access is enabled

- **WHEN** 服务以非回环监听地址启动
- **THEN** 系统在启动输出中报告已启用远程访问，并给出访问所需的凭证

### Requirement: Loopback-only binding by default

系统 SHALL 默认仅监听回环地址，使服务在未显式配置前不对本机之外的主机提供访问。系统 SHALL 允许通过显式配置指定监听地址，包括非回环地址；仅在显式配置了非回环地址时，系统才对本机之外的主机提供访问。

#### Scenario: Default startup

- **WHEN** 用户以有效参数默认启动，未指定任何监听配置
- **THEN** 系统仅在回环地址上接受连接

#### Scenario: Reachability from a non-loopback interface

- **WHEN** 服务已按默认方式启动，另一台主机尝试连接该服务监听的端口
- **THEN** 连接无法建立

#### Scenario: Explicit non-loopback binding

- **WHEN** 用户显式配置一个非回环监听地址后启动服务
- **THEN** 系统在该地址上接受来自本机之外主机的连接

#### Scenario: Explicit loopback binding with a different port

- **WHEN** 用户显式配置一个回环监听地址后启动服务
- **THEN** 系统仅在该回环地址上接受连接

### Requirement: HTTP surface partitioning

系统 SHALL 将 HTTP 表面划分为两个区域：根路径及前端静态资源返回前端应用页面，以 `/api/` 开头的请求返回 JSON 响应体，但图片内容端点的成功响应例外——返回图片内容的字节序列，响应内容类型为按目标文件扩展名对应的图片内容类型。系统 SHALL NOT 对 `/api/` 区域的错误返回 HTML 页面。

#### Scenario: Root path is requested

- **WHEN** 客户端请求根路径
- **THEN** 系统返回前端应用页面，响应内容类型为 HTML

#### Scenario: Frontend static asset is requested

- **WHEN** 客户端请求前端应用引用的某个静态资源
- **THEN** 系统返回该资源，响应内容类型与其资源类型相符

#### Scenario: API path is requested

- **WHEN** 客户端请求 `/api/` 区域内图片内容端点以外的某个端点
- **THEN** 系统返回 JSON 格式的响应体，响应内容类型为 JSON

#### Scenario: Service health is queried

- **WHEN** 客户端请求服务健康端点
- **THEN** 系统返回 JSON 响应体，指示服务已就绪

#### Scenario: Image content endpoint succeeds

- **WHEN** 客户端请求图片内容端点，且目标文件被认定为图片文件
- **THEN** 系统返回该文件内容的字节序列，响应内容类型为图片内容类型，而非 JSON 响应体

### Requirement: JSON error responses

系统 SHALL 在 `/api/` 区域的请求失败时返回 JSON 格式的错误响应体，包含机器可读的错误标识与人类可读的错误说明。机器可读的错误标识 SHALL 是错误响应体自身携带的、与人类可读说明相区分的字段，供调用方在不解析说明文本、不依赖 HTTP 状态码的前提下区分失败原因。系统 SHALL NOT 在该区域用 HTML 错误页或纯文本响应表示失败。

#### Scenario: Unknown API endpoint is requested

- **WHEN** 客户端请求 `/api/` 区域中不存在的端点
- **THEN** 系统返回指示「未找到」状态的 JSON 错误响应体，内容类型为 JSON

#### Scenario: API request fails for a domain reason

- **WHEN** `/api/` 区域的请求因目标不存在、无权限等原因无法完成
- **THEN** 系统返回表达该失败原因的 JSON 错误响应体，且不返回 HTML 错误页

#### Scenario: Failure causes are distinguishable from the response body

- **WHEN** `/api/` 区域的两次请求因不同原因失败
- **THEN** 两次响应体的机器可读错误标识不同，且各自的人类可读说明表达其各自的失败原因
