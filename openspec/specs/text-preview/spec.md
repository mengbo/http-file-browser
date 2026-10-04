# text-preview Specification

## Purpose

定义服务在命令行指定的根目录范围内向用户提供文本文件内容查看时用户可观察到的行为契约：内容响应包含什么、哪些文件被当作可读文本的文件、查看位置如何表示与重现，以及系统在什么情况下拒绝提供内容。不包含目录浏览本身的行为，不包含文本判定之外的文件类型识别，不包含内容的编码检测与解码，不包含内容的编辑与写回。

## Requirements

### Requirement: Text file content response

系统 SHALL 提供读取文件内容的端点，接受一个表示位置的相对根目录路径参数，返回该文件的规范化相对路径与该文件的内容。系统 SHALL NOT 返回根目录之外的文件的内容。系统 SHALL 以 UTF-8 呈现内容，无法解码为 UTF-8 的字节序列以替换字符呈现。

#### Scenario: A text file is requested

- **WHEN** 客户端请求一个被认定为可读文本的文件的路径
- **THEN** 系统返回 JSON 响应体，其中包含该文件的规范化相对路径与该文件的内容

#### Scenario: A path inside the root traverses a symbolic link outward

- **WHEN** 客户端请求的路径字面位置位于根目录之内，但解析后指向根目录之外
- **THEN** 系统拒绝该请求，返回机器可读错误标识为 `outside_root` 的 JSON 错误响应体，不返回该文件的内容

#### Scenario: The content is not decodable as UTF-8

- **WHEN** 被认定为可读文本的文件包含无法解码为 UTF-8 的字节序列
- **THEN** 系统仍返回成功响应，其中该部分字节以替换字符呈现

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

### Requirement: Preview position representation

系统 SHALL 用相对根目录的路径表示被查看文件的位置。系统 SHALL 使同一个位置表示在重复请求时解析到同一文件并返回相同的内容。

#### Scenario: A file position is reopened

- **WHEN** 客户端此前请求过某个文件位置的内容，随后以同一位置表示再次请求
- **THEN** 系统返回同一文件的内容，且响应中的路径为该位置的规范化结果

#### Scenario: The position contains redundant segments

- **WHEN** 客户端提供的路径参数包含冗余片段，例如多余的当前目录标记、重复的分隔符或结尾分隔符
- **THEN** 系统按规范化后的路径提供内容，且响应中的路径为该规范化结果

### Requirement: Content reading failures

系统 SHALL 在内容读取请求无法完成时返回 JSON 错误响应体，其机器可读错误标识区分失败原因：路径不存在为 `not_found`，路径指向目录为 `not_a_directory`，路径存在但不是普通文件为 `not_a_regular_file`，无访问权限为 `permission_denied`，超出根目录范围为 `outside_root`，不被视为可读文本的文件为 `not_text`，超过可提供内容的最大字节数为 `too_large`。

#### Scenario: The requested path does not exist

- **WHEN** 客户端请求的路径在文件系统中不存在
- **THEN** 系统返回机器可读错误标识为 `not_found` 的 JSON 错误响应体

#### Scenario: The requested path points to a directory

- **WHEN** 客户端请求的路径指向一个目录
- **THEN** 系统返回机器可读错误标识为 `not_a_directory` 的 JSON 错误响应体

#### Scenario: The requested path is not a regular file

- **WHEN** 客户端请求的路径存在但不是普通文件
- **THEN** 系统返回机器可读错误标识为 `not_a_regular_file` 的 JSON 错误响应体

#### Scenario: The requested file cannot be read

- **WHEN** 客户端请求的文件存在但当前进程无读取权限
- **THEN** 系统返回机器可读错误标识为 `permission_denied` 的 JSON 错误响应体

#### Scenario: The requested path is outside the root

- **WHEN** 客户端请求的路径解析后落在根目录之外
- **THEN** 系统返回机器可读错误标识为 `outside_root` 的 JSON 错误响应体

#### Scenario: The requested file is not recognized as text

- **WHEN** 客户端请求的路径不被视为可读文本的文件
- **THEN** 系统返回机器可读错误标识为 `not_text` 的 JSON 错误响应体

#### Scenario: The requested file exceeds the maximum size

- **WHEN** 客户端请求的文件超过系统可提供内容的最大字节数
- **THEN** 系统返回机器可读错误标识为 `too_large` 的 JSON 错误响应体
