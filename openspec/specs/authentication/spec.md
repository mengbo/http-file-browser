# authentication Specification

## Purpose

定义远程访问下的访问凭证契约：何时要求认证、访问凭证的来源与生命周期、客户端如何取得并在后续请求中自动携带认证、未通过认证时的失败标识，以及仅监听回环地址时免于认证的豁免。不包含传输层加密（TLS），不包含用户系统与多凭证管理。

## Requirements

### Requirement: Authentication requirement for remote access

系统 SHALL 仅当监听非回环地址时要求请求携带有效访问凭证；系统 SHALL NOT 在仅监听回环地址时要求认证。

#### Scenario: Remote request without a credential

- **WHEN** 服务监听一个非回环地址，且请求未携带访问凭证
- **THEN** 系统拒绝该请求，不提供所请求的资源

#### Scenario: Remote request with an invalid credential

- **WHEN** 服务监听一个非回环地址，且请求携带的访问凭证无效
- **THEN** 系统拒绝该请求，不提供所请求的资源

#### Scenario: Remote request with a valid credential

- **WHEN** 服务监听一个非回环地址，且请求携带有效访问凭证
- **THEN** 系统提供所请求的资源

#### Scenario: Loopback-only binding does not require authentication

- **WHEN** 服务仅监听回环地址，且请求未携带访问凭证
- **THEN** 系统不要求认证即提供所请求的资源

### Requirement: Access credential

系统 SHALL 在启用认证时生成一个每次启动唯一、仅在本次运行内有效的访问凭证。重启后，先前启动所生成的凭证 SHALL NOT 再被接受。

#### Scenario: Credential differs across startups

- **WHEN** 服务以启用认证的方式重启
- **THEN** 新生成的访问凭证与上一次启动所生成的凭证不同

#### Scenario: A credential from a previous run is refused

- **WHEN** 请求携带上一次运行所使用的访问凭证
- **THEN** 系统按无效凭证拒绝该请求

### Requirement: Credential acquisition and reuse

系统 SHALL 允许客户端以有效访问凭证完成一次请求，并使其后续对页面、静态资源、接口与图片内容的请求无需重复提供访问凭证即获得认证。系统 SHALL NOT 要求客户端在后续浏览位置中持续携带访问凭证。

#### Scenario: Acquiring a session with the credential

- **WHEN** 客户端在请求中提供有效的访问凭证
- **THEN** 系统认可该请求，并在其后的请求中自动认可同一客户端

#### Scenario: Subsequent requests need no credential

- **WHEN** 已获得认可的客户端请求页面、静态资源、接口或图片内容
- **THEN** 系统无需其再次提供访问凭证即提供所请求的资源

#### Scenario: The credential does not remain in the browsed location

- **WHEN** 客户端以访问凭证完成访问
- **THEN** 系统呈现的浏览位置不再包含访问凭证

### Requirement: Authentication failure reporting

系统 SHALL 在请求未通过认证时返回失败。对 `/api/` 区域的未认证请求，系统 SHALL 返回 JSON 错误信封，其机器可读错误标识为 `unauthorized`；对页面的未认证请求，系统 SHALL 提供输入访问凭证的入口，而非所请求页面的内容。

#### Scenario: API request without authentication

- **WHEN** 未认证的请求指向 `/api/` 区域
- **THEN** 系统返回机器可读错误标识为 `unauthorized` 的 JSON 错误响应体

#### Scenario: Page request without authentication

- **WHEN** 未认证的请求指向页面
- **THEN** 系统提供输入访问凭证的入口，不提供所请求页面的内容

#### Scenario: Authentication failure is distinguishable

- **WHEN** 一次请求因认证失败而失败，另一次请求因其他原因而失败
- **THEN** 两次响应体的机器可读错误标识不同