# qkbox 宏观架构设计

这份文档是 qkbox 的结构基线。后续任何实现阶段都必须遵守这里定义的产品边界、权限边界和类型边界。

## 目标

qkbox 是面向 Windows、macOS、Linux 的三端桌面 GUI 客户端：

```text
qkbox = Wails GUI + user-scope qkboxd + embedded sing-box core + Platform Capability Boundary
```

GUI 只是控制面。GUI 不拥有 runtime、secret、route 状态、DNS 状态、TUN 设备或任何特权平台变更。

## 不可变原则

1. GUI 不是 runtime owner。
2. `qkboxd` 是用户态服务，拥有用户数据、profile 编排、snapshot 编排和默认 embedded runtime。
3. 特权平台能力必须隔离在 capability boundary 后面。
4. 用户数据按 OS 用户隔离。
5. v1 的机器级网络模式必须单占。
6. 公开 IPC 暴露 qkbox 产品语义，不暴露 sing-box 内部结构。
7. `shared`、IPC schema、persistence model、GUI 代码不得出现 sing-box 类型。
8. App update 更新完整可安装产品，不更新单独二进制组件。
9. profile、subscription、rule set、geo asset 等数据资产可以独立刷新。
10. runtime observability 必须 capability-aware。GUI 不得假设 traffic、connections、groups、URLTest 永远可用。

## 明确不做

以下形态不得成为正式架构：

```text
GUI -> spawn sing-box CLI
GUI -> FFI/libbox -> sing-box
GUI process -> long-lived sing-box runtime
GUI -> direct TUN / route / DNS / firewall operations
qkboxd / core / helper / platform component 独立热更新
v1 多用户并发占用机器级 TUN / route / DNS runtime
```

CLI spawn 只能用于本地调查或测试辅助，不是产品 runtime 模型。

## 逻辑角色

### Wails GUI

GUI 以普通用户权限运行。

GUI 负责：

```text
Profile UI
Editor UI
Remote subscription UI
Dashboard UI
Logs / connections / groups UI
System proxy toggle UI
TUN feature UI
Permission and diagnostic UI
Update entry points
```

GUI 不负责：

```text
sing-box lifecycle
runtime config generation
config merge / normalize / compile
TUN creation
route mutation
DNS mutation
firewall mutation
system proxy implementation
secret persistence
runtime truth
```

### user-scope qkboxd

`qkboxd` 按 OS 用户运行。它是用户态 coordinator，也是默认 RuntimeOwner。

`qkboxd` 负责：

```text
Profile orchestration
Snapshot orchestration
Encrypted user persistence
Secret references
Config compiler
singboxadapter invocation
Default embedded runtime lifecycle
Runtime status machine
Runtime observability bridge
Structured diagnostics
IPC version and capability handshake
App-update coordination signals
```

`qkboxd` 不应为了持有用户 profile 或 secret 而以 root、LocalSystem 或机器级 service 身份运行。

### RuntimeOwner

RuntimeOwner 是实际持有 sing-box runtime 的组件。

默认形态：

```text
qkboxd = RuntimeOwner
```

例外形态：

```text
macOS Network Extension 或其他平台 runtime container
可以在特定网络模式下成为 RuntimeOwner。
```

GUI 不依赖 RuntimeOwner 的具体进程形态。

### PlatformCapabilityProvider

PlatformCapabilityProvider 承载特权或平台相关能力：

```text
TUN
route management
DNS management
firewall management
privileged repair actions
background service integration
start on boot
process lookup
connection tracking where platform-scoped
```

可能实现：

```text
Windows privileged helper / Windows Service
macOS Network Extension / privileged helper
Linux systemd service / root helper / polkit-mediated helper
```

PlatformCapabilityProvider 不保存 profile、subscription、qkbox domain state 或用户 secret。

## 三端目标形态

### Windows

```text
qkbox.exe
  GUI

qkboxd.exe
  user-scope coordinator and default runtime owner

privileged provider
  TUN / route / DNS / repair

IPC
  Named Pipe with OS ACLs
```

### macOS

```text
qkbox.app
  GUI

qkboxd
  user-scope coordinator and system-proxy runtime owner

Network Extension
  preferred formal TUN / VPN runtime shape

IPC
  Unix socket, with room for XPC where appropriate
```

system-proxy mode 可以先于 TUN mode 实现。TUN mode 不应通过临时 root 改 route 的方式落地。

### Linux

```text
qkbox
  GUI

qkboxd
  user-scope coordinator and default runtime owner

privileged provider
  systemd/root helper with polkit-class authorization

IPC
  Unix socket with UID isolation
```

## 多用户策略

qkbox 支持用户数据隔离，但 v1 不支持多个用户并发拥有机器级网络 runtime。

```text
profiles / snapshots / settings / secrets
  scoped to the OS user

system proxy mode
  user-session scoped where the OS supports it

TUN / route / DNS mode
  machine-level and exclusive
```

如果一个用户已经启用机器级网络模式，其他用户再次启用时必须返回：

```text
NETWORK_MODE_OWNED_BY_ANOTHER_SESSION
```

privileged provider 最多保存最小 owner 状态：

```text
uid
session_id
runtime_id
mode
started_at
```

它不得保存 profile content 或 secret。

## Profile 与 Snapshot 模型

Profile 是 qkbox 产品对象，不是裸 `config.json`。

```text
Draft
  mutable user editing state

Snapshot
  validated, traceable, runnable candidate

Active Snapshot
  snapshot currently used by runtime
```

runtime 永远从 snapshot 启动，不直接读取 mutable draft。

Snapshot 元数据：

```text
snapshot_id
profile_id
encrypted_raw_content_ref
optional encrypted_normalized_content_ref
validation_diagnostics
runtime_summary
required_capabilities
embedded_singbox_version
runtime_hash
created_at
```

raw profile content 是敏感数据，必须加密落盘。diagnostics、summary、debug bundle 默认脱敏。

## Secret 策略

qkbox 将导入的 profile 内容视作敏感文档。

必须满足：

```text
Raw profile content is encrypted at rest.
Snapshot content is encrypted at rest.
Remote profile auth metadata uses SecretRef.
SecretStore stores key material or key references through OS backends.
Debug exports are redacted by default.
```

目标后端：

```text
Windows Credential Manager
macOS Keychain
Linux Secret Service
```

v1 不要求完整解析并抽取所有 sing-box secret 字段。这样做会脆弱，也会被协议字段变化拖累。

## Config Compiler Boundary

配置处理只存在于 `qkboxd` internal：

```text
Raw Profile Content
  -> optional Normalized Document
  -> Internal Compiled Artifact
  -> singboxadapter runtime creation
```

GUI 可以看到：

```text
diagnostics
required capabilities
profile summary
snapshot status
runnable / not runnable reasons
```

GUI 不可以看到：

```text
option.Options
runtime config
route plan
DNS plan
TUN options
compiled artifact
```

## singboxadapter Boundary

`singboxadapter` 是唯一允许依赖 sing-box 语义的边界。

职责：

```text
Validate raw content
Compile internal runtime artifact
Create runtime
Start runtime
Close runtime
Bridge logs
Bridge traffic / connections / groups when available
Map sing-box errors to qkbox diagnostics
Expose qkbox domain models only
```

允许存在私有桥接层，例如：

```text
singboxadapter/platformbridge
```

它可以适配 sing-box platform interface，但不得把 sing-box 类型泄漏到 qkbox 公开层。

以下类型禁止出现在 singboxadapter 私有边界之外：

```text
option.Options
box.Box
adapter.Router
adapter.OutboundManager
adapter.ConnectionManager
clashapi.Server
daemon.StartedService
any sing-box config struct
any sing-box runtime object
any experimental/libbox type
```

## 公开服务面

公开服务是 qkbox 产品契约，不绑定 sing-box 内部 API。

### EngineService

```text
Start
Stop
Reload
Restart
GetStatus
SubscribeStatus
SubscribeLogs
GetRuntimeCapabilities
SubscribeTraffic
SubscribeConnections
ListGroups
SelectOutbound
URLTest
CloseConnection
CloseAllConnections
GetStartedAt
GetDeprecatedWarnings
```

### ProfileService

```text
CreateProfile
UpdateProfileDraft
DeleteProfile
ListProfiles
GetProfile
ImportProfile
UpdateRemoteProfile
ValidateProfileDraft
GetProfileDiagnostics
CreateProfileSnapshot
ActivateProfileSnapshot
GetActiveProfile
GetActiveSnapshot
RollbackToSnapshot
```

### PlatformCapabilityService

```text
GetPlatformCapabilities
GetPermissionStatus
PrepareFeature
InstallPrivilegedComponent
UninstallPrivilegedComponent
GetSystemProxyStatus
SetSystemProxyEnabled
GetNetworkModeStatus
RunRepairAction
```

feature name 是产品层概念：

```text
SYSTEM_PROXY
TUN_MODE
DNS_HIJACK
BACKGROUND_SERVICE
START_ON_BOOT
PROCESS_LOOKUP
CONNECTION_TRACKING
```

禁止公开的平台底层 API：

```text
OpenTun
ApplyRoutePolicy
ApplyDNSPolicy
ConfigureFirewall
arbitrary shell
arbitrary file read
```

## IPC Contract

IPC 必须满足：

```text
local only
authenticated
versioned
capability-aware
stream-friendly
structured-error based
protected by OS ACLs or peer credentials
```

建议 transport：

```text
Windows
  Named Pipe with DACLs

macOS / Linux
  Unix socket with UID ownership and restrictive permissions
```

product build 不允许未鉴权 TCP control port。

结构化错误：

```text
code
message
detail
source
recoverable
user_action
debug_ref
```

错误 code 族：

```text
PROFILE_*
CONFIG_*
ENGINE_*
PLATFORM_*
PERMISSION_*
IPC_*
RUNTIME_*
SINGBOX_ADAPTER_*
ASSET_*
UPDATE_*
```

## Runtime Observability

observability 是 capability，不是 runtime 存在后的天然承诺。

Source：

```text
LogSource
TrafficSource
ConnectionSource
GroupSource
URLTestSource
ClashModeSource
```

Source 状态：

```text
available
unavailable
partial
degraded
unsupported
```

GUI 必须把不可用能力渲染为 unavailable，不得显示假数据或猜测数据。

Clash API 不是 qkbox 的公开主控制面。它可以作为内部 compatibility source 或 observability source。

## 状态机

`qkboxd` 拥有 runtime state machine：

```text
UNINITIALIZED
IDLE
VALIDATING
STARTING
STARTED
RELOADING
STOPPING
FATAL
DEGRADED
```

GUI 只消费状态机，不根据进程状态推断 runtime truth。

## Reload 语义

Reload 是产品层操作，不承诺底层平台变更具备原子事务能力。

v1 流程：

```text
validate target snapshot
resolve capabilities
prepare platform changes
stop old runtime
start new runtime
publish state
cleanup old resources
return structured result
```

结果值：

```text
success
failed_validation
failed_permission
failed_platform_prepare
failed_runtime_start
rolled_back
degraded
cleanup_failed
```

只有新 runtime 和必要平台能力成功启动后，active snapshot 才能切换。平台 rollback 是 best effort。

## 更新模型

App update 指替换完整可安装产品：

```text
GUI
qkboxd
embedded sing-box core
platform components
IPC schema
migration logic
packaging metadata
```

`qkboxd` 可以报告版本并协调停机，但不替换自身或 helper。

Data asset update 独立处理：

```text
remote profile content
subscription content
rule-set files
geo assets
provider cache
```

不支持：

```text
CoreUpdate
RuntimeBundleUpdate
HelperUpdate
component-level rollback
```

## License Baseline

qkbox 采用 GPL-compatible open-source 方向。最终 license 可以后定，但实现阶段不得引入与 bundle 或 embed sing-box 不兼容的依赖和分发方式。

