# Spec Delta — service-startup

## MODIFIED Requirements

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
