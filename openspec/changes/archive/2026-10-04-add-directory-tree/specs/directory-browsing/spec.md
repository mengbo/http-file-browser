# Spec Delta

## ADDED Requirements

### Requirement: 目录树导航

系统 SHALL 在浏览界面中提供一棵以根目录为起点的目录树，并 SHALL 允许用户收起与展开该目录树。目录树 SHALL 以可展开与折叠的节点呈现目录条目，并 SHALL 在每个节点显示该条目的名称。系统 SHALL 在视觉上区分目录节点与文件节点。系统 SHALL 标出当前浏览位置在目录树中的对应节点，且 SHALL 使该位置的全部祖先节点处于展开状态。展开一个目录节点 SHALL 使其直接子条目出现在目录树中，折叠一个已展开的目录节点 SHALL 使其子条目不再出现。点击目录节点 SHALL 将当前浏览位置切换为该目录并在主区显示其内容；点击文件节点 SHALL 在主区打开该文件的内容。系统 SHALL 在文件视图中保持目录树可用。

#### Scenario: Opening the browser shows the tree

- **WHEN** 客户端打开浏览界面
- **THEN** 界面显示一棵以根目录为起点、包含根目录直接子条目的目录树

#### Scenario: The tree can be collapsed and expanded

- **WHEN** 用户收起目录树
- **THEN** 目录树不再显示于浏览界面
- **WHEN** 用户再次展开目录树
- **THEN** 目录树重新显示

#### Scenario: Expanding a directory reveals its children

- **WHEN** 用户展开目录树中一个目录节点
- **THEN** 该目录的直接子条目出现在该节点之下

#### Scenario: Collapsing an expanded directory hides its children

- **WHEN** 用户折叠目录树中一个已展开的目录节点
- **THEN** 该目录的子条目不再出现在目录树中

#### Scenario: Current position is marked with ancestors expanded

- **WHEN** 客户端位于根目录内某个目录或文件
- **THEN** 目录树标出对应该位置的节点，且该位置的全部祖先节点处于展开状态

#### Scenario: Clicking a directory node navigates

- **WHEN** 用户点击目录树中的目录节点
- **THEN** 系统将该目录设为当前浏览位置，并在主区显示该目录的内容

#### Scenario: Clicking a file node opens the file

- **WHEN** 用户点击目录树中的文件节点
- **THEN** 系统在主区打开该文件的内容

#### Scenario: Directory and file nodes are distinguishable

- **WHEN** 目录树中同时存在目录节点与文件节点
- **THEN** 两类节点在外观上可区分

#### Scenario: Tree remains available in file view

- **WHEN** 用户在主区查看一个文件的内容
- **THEN** 目录树仍可用于导航，并标出该文件的位置
