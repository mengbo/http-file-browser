# Spec Delta

## MODIFIED Requirements

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
