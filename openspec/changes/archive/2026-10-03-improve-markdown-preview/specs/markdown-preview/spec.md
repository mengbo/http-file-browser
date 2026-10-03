# Spec Delta

## MODIFIED Requirements

### Requirement: Markdown rendered presentation

对于扩展名为 `md` 或 `markdown` 的被查看文本文件，系统 SHALL 默认以 Markdown 渲染形式呈现。当该文件的呈现形式已被用户切换为源码形式（见 Presentation form switching）时，系统 SHALL NOT 对其适用 Markdown 渲染呈现。渲染呈现 SHALL NOT 将文件中形似 HTML 的标记解释为可执行标记，此类标记文本 SHALL 以字面形式呈现。以渲染形式呈现且渲染未成功时，系统 SHALL 以普通文本形式呈现该文件内容，SHALL NOT 因渲染未成功而拒绝提供内容或向用户报告错误。

#### Scenario: A markdown file is viewed

- **WHEN** 用户查看一个扩展名为 `md` 或 `markdown` 的文本文件，且其呈现形式未被切换为源码形式
- **THEN** 内容以 Markdown 渲染形式呈现

#### Scenario: Content resembling HTML markup is viewed

- **WHEN** 以渲染形式呈现的一个 Markdown 文件，其内容包含形似 HTML 标记的文本
- **THEN** 该标记文本以字面形式呈现，不作为可执行标记呈现

#### Scenario: Rendering does not succeed

- **WHEN** 系统以渲染形式呈现一个 Markdown 文件，且其内容未被成功渲染
- **THEN** 内容以普通文本形式呈现，且系统不向用户报告错误

## ADDED Requirements

### Requirement: Presentation form switching

对于被查看的 Markdown 文件，系统 SHALL 使用户能将其呈现形式在渲染形式与源码形式之间切换。切换只影响当前查看：离开被查看文件后再次查看同一文件时，系统 SHALL 以渲染形式呈现该文件，SHALL NOT 保留此前的源码形式选择。本 Requirement 只规定呈现形式的切换与默认，不规定源码形式的呈现形态（见 `syntax-highlighting`）。

#### Scenario: The presentation form is switched to source

- **WHEN** 用户将一个以渲染形式呈现的 Markdown 文件切换为源码形式
- **THEN** 该文件以源码形式呈现，不再以 Markdown 渲染形式呈现

#### Scenario: The presentation form is switched back to rendered

- **WHEN** 用户将一个以源码形式呈现的 Markdown 文件切换为渲染形式
- **THEN** 该文件以 Markdown 渲染形式呈现

#### Scenario: The presentation form does not persist across views

- **WHEN** 用户将一个 Markdown 文件切换为源码形式，随后离开该文件，并在新的查看中再次打开同一文件
- **THEN** 该文件以 Markdown 渲染形式呈现
