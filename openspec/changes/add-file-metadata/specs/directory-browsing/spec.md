# Spec Delta

## ADDED Requirements

### Requirement: Entry metadata

系统 SHALL 在目录列表的每个条目中给出该条目的最后修改时间。系统 SHALL 为非目录条目给出该条目自身的大小，并 SHALL NOT 为目录条目给出大小。系统 SHALL 给出的大小是该条目自身占用的字节数；对于符号链接条目，该值是链接自身的长度，而不是其目标内容的大小。系统 SHALL 以自 Unix 纪元起的整数秒表示最后修改时间，该表示 SHALL NOT 随运行环境的时区或语言变化。系统 SHALL 在某个条目的大小或最后修改时间无法获得时省略该值，而不是给出占位值。系统 SHALL NOT 因单个条目的元信息无法获得而使整个列表请求失败。

#### Scenario: A file entry is listed

- **WHEN** 客户端请求的目录包含一个普通文件
- **THEN** 该条目在列表中给出该文件自身的大小与最后修改时间

#### Scenario: A directory entry is listed

- **WHEN** 客户端请求的目录包含一个子目录
- **THEN** 该条目在列表中给出该子目录的最后修改时间，且不给出该子目录的大小

#### Scenario: An entry is a symbolic link

- **WHEN** 客户端请求的目录包含一个指向其他位置的符号链接
- **THEN** 该条目给出的大小是链接自身的长度，而不是其目标内容的大小

#### Scenario: Modification time falls within the same second

- **WHEN** 某个条目的最后修改时间含有不足一秒的精度
- **THEN** 系统给出的是向下取整到整秒的取值

#### Scenario: Modification time is read under a different environment

- **WHEN** 同一个目录在不同时区或不同语言环境的进程中被列出
- **THEN** 两次响应中每个条目的最后修改时间取值相同

#### Scenario: An entry's metadata cannot be obtained

- **WHEN** 客户端请求的目录在系统读取期间发生变化，某个已列出的条目的大小或最后修改时间无法获得
- **THEN** 系统返回成功响应，该条目仍出现在列表中且其无法获得的元信息被省略

#### Scenario: Repeated listings return the same metadata

- **WHEN** 客户端对同一目录多次请求列表，且期间该目录内容未发生变化
- **THEN** 每次响应中每个条目的大小与最后修改时间取值都相同

## MODIFIED Requirements

### Requirement: Directory listing response

系统 SHALL 提供目录列表端点，接受一个表示位置的相对根目录路径参数，返回该目录的规范化相对路径、上级相对路径与条目列表。系统 SHALL 在路径参数缺失或为空时返回根目录的列表。系统 SHALL 在每个条目中提供名称与类型，以及该条目的大小与最后修改时间，其取值规则见 `Entry metadata`。系统 SHALL NOT 在列表中提供条目的内容。

#### Scenario: A directory is listed

- **WHEN** 客户端请求一个存在且可读的目录的列表
- **THEN** 系统返回 JSON 响应体，其中包含该目录的规范化相对路径、上级相对路径与该目录下的条目列表

#### Scenario: Path parameter is missing or empty

- **WHEN** 客户端在未提供路径参数或提供空路径参数的情况下请求列表
- **THEN** 系统返回根目录的列表，且响应中的路径为空字符串

#### Scenario: A directory has no entries

- **WHEN** 客户端请求一个存在、可读但不包含任何条目的目录
- **THEN** 系统返回条目列表为空的成功响应，不返回错误

#### Scenario: Path parameter contains redundant segments

- **WHEN** 客户端提供的路径参数包含冗余片段，例如多余的当前目录标记、重复的分隔符或结尾分隔符
- **THEN** 系统按规范化后的路径返回列表，且响应中的路径为该规范化结果

#### Scenario: A listed directory contains readable files

- **WHEN** 客户端请求的目录包含一个存在且可读的文件
- **THEN** 该文件在响应中只以元信息的形式出现，响应不包含该文件的任何内容
