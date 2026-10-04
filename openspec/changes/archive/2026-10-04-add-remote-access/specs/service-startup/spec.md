# Spec Delta — service-startup

## MODIFIED Requirements

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
