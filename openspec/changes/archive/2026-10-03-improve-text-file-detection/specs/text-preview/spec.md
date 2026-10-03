# Spec Delta — text-preview

## MODIFIED Requirements

### Requirement: Text file recognition

系统 SHALL 将名称的扩展名属于已知文本扩展名的文件视为可读文本的文件。对于名称没有任何扩展名的文件，系统 SHALL 依据其内容的起始部分判定：起始部分不包含二进制数据字节的，视为可读文本的文件；起始部分包含二进制数据字节的，SHALL NOT 视为可读文本的文件。系统 SHALL NOT 将其余文件视为可读文本的文件；名称的扩展名不属于已知文本扩展名的文件，即使其内容不包含二进制数据字节，也 SHALL NOT 被视为可读文本的文件。二进制数据字节指取值属于 0x00–0x08、0x0B、0x0E–0x1A、0x1C–0x1F 的字节；内容起始部分中的空字节全部只出现于偶数位置、或全部只出现于奇数位置的，空字节不计为二进制数据字节。对名称没有任何扩展名的文件，系统 SHALL 仅依据其内容的起始部分进行判定，SHALL NOT 依据其内容的其余部分进行判定。系统 SHALL 使同一文件的认定结果在多次请求之间保持一致。

#### Scenario: A file with a known text extension is requested

- **WHEN** 客户端请求一个名称的扩展名属于已知文本扩展名的文件的路径
- **THEN** 系统将该文件视为可读文本的文件并提供其内容

#### Scenario: A file with a non-text extension is requested

- **WHEN** 客户端请求一个名称的扩展名不属于已知文本扩展名的文件的路径
- **THEN** 系统不将该文件视为可读文本的文件，返回机器可读错误标识为 `not_text` 的 JSON 错误响应体

#### Scenario: A file with a non-text extension contains text content

- **WHEN** 客户端请求一个名称的扩展名不属于已知文本扩展名的文件的路径，且该文件的内容不包含二进制数据字节
- **THEN** 系统仍不将该文件视为可读文本的文件，返回机器可读错误标识为 `not_text` 的 JSON 错误响应体

#### Scenario: A file without an extension is requested

- **WHEN** 客户端请求一个名称没有任何扩展名的文件的路径，且该文件内容的起始部分不包含二进制数据字节
- **THEN** 系统将该文件视为可读文本的文件并提供其内容

#### Scenario: A file without an extension contains binary data bytes

- **WHEN** 客户端请求一个名称没有任何扩展名的文件的路径，且该文件内容的起始部分包含二进制数据字节
- **THEN** 系统不将该文件视为可读文本的文件，返回机器可读错误标识为 `not_text` 的 JSON 错误响应体

#### Scenario: A file without an extension and without any content is requested

- **WHEN** 客户端请求一个名称没有任何扩展名且内容为空的文件的路径
- **THEN** 系统将该文件视为可读文本的文件并提供其内容

#### Scenario: Binary data bytes appear only after the start of the content

- **WHEN** 客户端请求一个名称没有任何扩展名的文件的路径，该文件内容的起始部分不包含二进制数据字节，而起始部分之后存在二进制数据字节
- **THEN** 系统仍将该文件视为可读文本的文件并提供其内容

#### Scenario: A UTF-16 text file without a byte order mark is requested

- **WHEN** 客户端请求一个名称没有任何扩展名的文件的路径，该文件内容为不带字节序标记的 UTF-16 编码文本，其起始部分中的空字节全部只出现于偶数位置、或全部只出现于奇数位置，且除空字节外不包含其他二进制数据字节
- **THEN** 系统将该文件视为可读文本的文件并提供其内容

#### Scenario: NUL bytes appear at both even and odd positions

- **WHEN** 客户端请求一个名称没有任何扩展名的文件的路径，该文件内容的起始部分中的空字节既出现于偶数位置、又出现于奇数位置，且除空字节外不包含其他二进制数据字节
- **THEN** 系统不将该文件视为可读文本的文件，返回机器可读错误标识为 `not_text` 的 JSON 错误响应体

#### Scenario: The same file is requested repeatedly

- **WHEN** 客户端多次请求同一路径且该路径对应的认定结果未发生变化
- **THEN** 每次响应中该路径的认定结果都相同
