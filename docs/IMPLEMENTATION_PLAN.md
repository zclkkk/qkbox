# qkbox 阶段实施计划

这份文档把宏观架构切成可交给后续 Agent 执行的阶段。每个阶段都必须落在终局架构骨架内，不能为了短期推进引入会被废弃的 sidecar、CLI-runtime 或 FFI 主路径。

执行任何阶段前，必须先阅读 `docs/ARCHITECTURE.md`。

## Agent 执行规则

1. 保持架构边界不变。
2. 每次只实现当前阶段的范围。
3. 不把 sing-box 类型引入 GUI、shared model、IPC schema 或 persistence。
4. 不让 GUI 拥有 runtime lifecycle 或特权平台动作。
5. 优先做垂直切片，避免空框架膨胀。
6. 每个阶段必须有验证方式。
7. 如果阶段引入公开契约，必须有聚焦的 contract test 或 schema test。
8. 如果阶段存储用户数据，必须考虑迁移和加密影响。
9. 如果阶段触碰平台状态，必须覆盖 cleanup 和 repair 行为。
10. 不确定时，缩小产品 API，保留内部实现替换空间。

## Phase 0: Repository Baseline

### 目标

建立后续实现需要的最小仓库骨架和工具基线。

### 范围

```text
top-level module/workspace layout
apps/desktop placeholder
core/qkboxd placeholder
shared/api placeholder
shared/model placeholder
platform placeholders
packaging placeholders
basic lint/check/test commands
architecture docs wired into repository
```

### 输出

```text
文档化的 workspace 结构
根目录开发命令
可本地运行的最小 check/test
不包含 runtime 实现
```

### 验收

```text
根目录 check 命令成功。
没有生成临时 runtime 架构。
sing-box 依赖没有出现在允许边界之外。
```

### 禁止

```text
不实现 runtime start。
不安装 helper。
不添加平台状态变更。
不添加 CLI-spawn runtime path。
```

## Phase 1: IPC Handshake And Capability Shell

### 目标

建立最终控制面形状：

```text
GUI -> user-scope qkboxd
```

本阶段不实现 runtime 行为。

### 范围

```text
qkboxd process startup in user scope
GUI connection to qkboxd
Hello / HelloReply handshake
API version
app version
qkboxd version
platform identification
runtime capability shell
platform capability shell
structured error shell
```

### 公开契约

```text
Hello(client_version)
HelloReply(
  api_version,
  app_version,
  qkboxd_version,
  platform,
  runtime_capabilities,
  platform_capabilities
)
```

### 验收

```text
GUI 能连接 qkboxd。
handshake 返回确定的 version/platform 数据。
unsupported capability 渲染为 unavailable，而不是缺字段。
不兼容 API version 返回 structured error。
```

### 禁止

```text
不启动 sing-box。
不暴露平台底层 API。
product path 不使用未鉴权 TCP。
```

## Phase 2: Profile Draft And Encrypted Persistence

### 目标

加入用户态 profile 存储，但不执行 runtime。

### 范围

```text
Profile model
Draft content storage
Encrypted raw content reference
SecretStore abstraction
Profile list/get/create/update/delete
Remote profile metadata shell
Basic redaction utilities
```

### 服务

```text
CreateProfile
UpdateProfileDraft
DeleteProfile
ListProfiles
GetProfile
```

### 数据规则

```text
Raw profile content is encrypted at rest.
Profile ownership is scoped to the current OS user.
No profile data is stored in privileged provider locations.
Debug/display output is redacted by default.
```

### 验收

```text
Profile CRUD tests pass.
Stored raw content is not plaintext.
Profile records survive qkboxd restart.
Redaction tests cover common secret-like fields.
```

### 禁止

```text
不在 singboxadapter 之外 parse 成 sing-box 类型。
不创建 runnable snapshot。
不把 secret 明文保存到普通 persistence。
```

## Phase 3: Validation And Snapshot Lifecycle

### 目标

让 snapshot 成为唯一 runnable unit，但仍不启动 runtime。

### 范围

```text
ValidateProfileDraft
GetProfileDiagnostics
CreateProfileSnapshot
ActivateProfileSnapshot
GetActiveProfile
GetActiveSnapshot
RollbackToSnapshot
runtime summary model
required capability model
```

### 内部边界

```text
Config compiler:
  raw encrypted content -> decrypted working content -> validation -> diagnostics
```

如果本阶段引入 sing-box validation，必须放在 `singboxadapter` 后面。

### 验收

```text
Invalid config produces structured diagnostics.
Snapshot creation is blocked by validation failure.
Runtime never reads draft content.
Rollback changes active snapshot metadata only.
No sing-box types leak into shared model, IPC, or persistence.
```

### 禁止

```text
不启动 runtime。
不向 GUI 暴露 normalized/runtime config。
不持久化 option.Options 或任何 sing-box struct。
```

## Phase 4: Embedded Runtime Start/Stop

### 目标

通过 qkboxd 使用终局 embedded core 形态启动和停止 active snapshot。

### 范围

```text
singboxadapter runtime creation
EngineService Start / Stop / GetStatus
Runtime state machine
log bridge foundation
startedAt
fatal error mapping
basic lifecycle lock
```

### 必须使用的 runtime path

```text
qkboxd -> singboxadapter -> embedded sing-box core
```

### 验收

```text
Start uses active snapshot, not draft.
Stop closes runtime.
State transitions are observable.
Fatal startup errors become structured ENGINE_* or SINGBOX_ADAPTER_* errors.
Repeated start/stop does not leave stale runtime state.
```

### 禁止

```text
不把 spawn sing-box CLI 作为主路径。
不添加 privileged TUN/route/DNS 行为。
不让 GUI 直接调用 singboxadapter。
```

## Phase 5: Logs And Status Streams

### 目标

在不引入完整 dashboard 复杂度的情况下，提供基础 runtime feedback。

### 范围

```text
SubscribeStatus
SubscribeLogs
log ring buffer
daemon log source
runtime log source
platform log source placeholder
stream lifecycle management
```

### 验收

```text
GUI receives status stream.
GUI receives log stream.
Late subscribers receive recent log buffer where intended.
Stream cancellation does not leak goroutines/resources.
```

### 禁止

```text
不伪造 traffic 或 connection 数据。
不暴露包含实现类型的 raw internal log object。
```

## Phase 6: Runtime Observability Capabilities

### 目标

通过 capability-aware source 增加 dashboard 数据。

### 范围

```text
GetRuntimeCapabilities
SubscribeTraffic
SubscribeConnections
ListGroups
SelectOutbound
URLTest
CloseConnection
CloseAllConnections
source states: available / unavailable / partial / degraded / unsupported
```

### 验收

```text
Capabilities reflect actual runtime support.
Unavailable observability returns structured unavailable responses.
Groups and connections map to qkbox domain models.
No sing-box tracker or Clash API types leak to public API.
```

### 禁止

```text
不把 Clash API 作为公开控制面。
不要求每个 profile 支持所有 dashboard feature。
不在 GUI 显示推断/伪造指标。
```

## Phase 7: System Proxy Mode

### 目标

先交付第一个有实际网络价值、但不涉及机器级 TUN 复杂度的模式。

### 范围

```text
SYSTEM_PROXY capability
GetSystemProxyStatus
SetSystemProxyEnabled
system proxy cleanup on runtime stop
user-session scoped behavior where supported
diagnostics for unsupported desktop environments
```

### Ownership Rule

qkbox 里 system proxy 只有一个 owner：

```text
PlatformCapabilityService / platform layer
```

config compiler 不应静默把 system proxy ownership 委托给 sing-box inbound options。除非未来设计明确选择这条路径。

### 验收

```text
Enable sets proxy to the active runtime listener.
Disable clears proxy.
Stop runtime clears proxy if qkbox owns it.
Unsupported platform returns PLATFORM_* diagnostics.
```

### 禁止

```text
不引入 TUN。
如果平台可避免，不为了 user-session proxy 强制引入 privileged helper。
不让 GUI 直接修改 OS proxy settings。
```

## Phase 8: Privileged Provider Boundary

### 目标

在实现 TUN 前，先落地正式 privileged capability boundary。

### 范围

```text
Privileged provider installation/status shell
helper identity and version reporting
user qkboxd -> provider IPC
provider owner state model
PrepareFeature for TUN_MODE / DNS_HIJACK / BACKGROUND_SERVICE
RunRepairAction shell
```

### 安全要求

```text
Provider exposes allowlisted operations only.
Provider does not accept arbitrary shell or file operations.
Provider authenticates qkboxd.
Provider records minimal owner state only.
Provider does not store profile content or secrets.
```

### 验收

```text
qkboxd can detect provider unavailable / installed / mismatched version.
Unauthorized calls are rejected.
Provider owner state is observable through qkboxd diagnostics.
No profile or secret data is written by provider.
```

### 禁止

```text
不实现 route/DNS/TUN mutation。
不让 GUI 直接调用 provider。
product IPC 不使用未鉴权 localhost TCP。
```

## Phase 9: TUN / Route / DNS Exclusive Network Mode

### 目标

增加机器级网络模式，并显式处理 ownership 和 best-effort repair。

### 范围

```text
TUN_MODE capability
machine-level owner lock
NETWORK_MODE_OWNED_BY_ANOTHER_SESSION error
platform-specific capability preparation
runtime diagnostics
cleanup on stop
repair actions for stale state
```

### 平台方向

```text
Windows
  privileged provider owns TUN / route / DNS actions

macOS
  prefer Network Extension for formal TUN/VPN mode

Linux
  privileged provider with systemd/root helper and polkit-class authorization
```

### 验收

```text
Only one user/session can own machine-level network mode.
Owner state is released on clean stop.
Stale owner state can be diagnosed and repaired.
Route/DNS cleanup failures produce cleanup_failed or degraded results.
```

### 禁止

```text
v1 不支持 concurrent per-user TUN runtimes。
不隐藏 cleanup failure。
不承诺 atomic platform rollback。
```

## Phase 10: Reload Semantics

### 目标

基于现有 runtime lifecycle 实现产品层 reload 语义。

### 范围

```text
Reload target snapshot
validation before mutation
capability resolution
best-effort rollback to previous snapshot
structured reload result
cleanup failure reporting
```

### Result Values

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

### 验收

```text
Validation failure does not affect current runtime.
Runtime start failure attempts previous snapshot restore.
Active snapshot changes only after successful new runtime start.
Cleanup failure is visible and repairable.
```

### 禁止

```text
不向 GUI 暴露 stop/start implementation details。
不声称 route/DNS/TUN rollback 是原子事务。
```

## Phase 11: Data Asset Updates

### 目标

支持数据资产独立刷新，不支持二进制组件独立刷新。

### 范围

```text
remote profile content
subscription update metadata
rule-set assets
geo assets
provider cache
asset status
asset diagnostics
```

### 验收

```text
Asset updates do not replace binaries.
Remote profile updates create drafts or snapshots according to product policy.
Failed asset updates do not corrupt current active snapshot.
```

### 禁止

```text
不实现 core/helper/runtime bundle updates。
不绕过 snapshot 语义修改 active runtime config。
```

## Phase 12: App Update Coordination

### 目标

协调完整产品更新，但不让 qkboxd 自己替换自己。

### 范围

```text
GetCurrentAppVersion
GetComponentVersions
CheckAppUpdate
DownloadAppUpdate
ApplyAppUpdate coordination
runtime quiesce before update
post-update migration hooks
```

### Ownership Rule

真正 apply package update 的是 updater 或 installer。`qkboxd` 可以协调 shutdown、报告版本，但不替换自己的 binary 或 helper binary。

### 验收

```text
Component versions are visible for diagnostics.
Update flow can stop runtime cleanly before installer handoff.
No component-level hot update path exists.
```

### 禁止

```text
不添加 CoreUpdate。
不添加 HelperUpdate。
不添加 RuntimeBundleUpdate。
不添加 component-level rollback。
```

## Phase 13: Debug Bundle And Recovery

### 目标

让失败可诊断，同时不泄漏 secret。

### 范围

```text
debug bundle export
redacted profiles/summaries
runtime status
capability status
platform owner state
recent logs
structured error history
repair action recommendations
```

### 验收

```text
Debug bundle redacts common secrets.
Debug bundle does not include plaintext profile content by default.
Repair actions are allowlisted.
```

### 禁止

```text
不导出 arbitrary files。
不在未脱敏情况下导出可能含 secret 的 privileged helper logs。
```

## 跨阶段不变量

每个阶段完成后都必须保持：

```text
GUI does not import sing-box.
shared/api does not contain sing-box types.
shared/model does not contain sing-box types.
persistence does not store sing-box structs.
Runtime starts from snapshots, never mutable drafts.
Privileged provider stores no profile content or secrets.
Machine-level network mode is exclusive in v1.
Public IPC uses structured errors.
Capabilities gate optional GUI features.
```


