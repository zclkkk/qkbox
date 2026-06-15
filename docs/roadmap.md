# qkbox v1.0 Roadmap

本文档是 qkbox v1.0 的执行计划。架构决策以 [architecture-decisions.md](architecture-decisions.md) 为准。

这份 Roadmap 采用 fresh-start 视角：不沿用旧 Phase 切分，不保留 Profile Snapshot、Draft/Snapshot、Managed/Raw 或长期兼容桥接。实现可以通过一次性迁移进入新模型，但最终代码必须只呈现新模型。

---

## 1. Product Contract

qkbox 是基于 sing-box 的跨平台桌面代理客户端。

v1.0 的目标是交付一个用户可用、架构干净、可继续演进的客户端：

- 用户通过 `qkbox` 进入应用。
- 用户可以导入、创建、编辑、保存和激活 Profile。
- Profile 是完整 sing-box JSON 配置。
- Profile 写入前必须通过 sing-box 权威验证。
- 运行时能启动、停止、切换模式、选择节点、观测流量和连接。
- 包装产物只暴露用户入口，不暴露私有 helper。

### 不支持的行为

| 行为 | 决策 |
|------|------|
| 直接运行 `qkbox-window` | unsupported behavior |
| 直接运行 `qkbox-provider` | unsupported behavior |
| Clash 订阅格式转换 | 不做，能力不对齐 |
| 节点参数表单编辑器 | 不做，参数编辑走 JSON editor |
| 完整路由规则可视化编辑器 | v1.0 不做 |
| 开机自启 | v1.0 不做 |
| 自动更新 | 首个 Release 后再评估 |
| 对外暴露 Clash HTTP API | 不做 |
| macOS TUN / NetworkExtension | v1.x 后续阶段 |

---

## 2. Architecture Contract

### Process Model

```
qkbox            用户入口；托盘、IPC server、runtime coordinator
qkbox-window     私有 GUI helper；只由 qkbox spawn
qkbox-provider   私有提权 helper；只由 qkbox runtime 编排启动
```

`qkbox-window` 关闭只释放 GUI 进程；`qkbox` 继续运行托盘和 runtime。

`qkbox` 退出必须按顺序停止 runtime、恢复系统代理、关闭 provider、销毁托盘、关闭 SQLite、释放用户锁。

### Config Model

```
输入方式
  URI import
  subscription refresh
  JSON editor save
  template create

统一边界
  singboxadapter.Validate()
  profiles.content

使用方式
  activateProfile(profileID)
```

Profile 是配置边界。不存在 Profile Snapshot、Draft/Snapshot 分离、Managed/Raw Profile、Profile source URL。

### Runtime Model

Activation 是 service 层编排：

```
load profile
validate content
extract runtime capabilities
stop old runtime
platform prepare
start new runtime
persist active_profile_id
rollback on failure
```

engine 只管理 runtime owner 生命周期，不读取数据库，不持有 Profile content，不理解导入来源。

### Runtime Controls

运行时控制不修改 Profile content：

- `SelectOutbound(groupTag, outboundTag)`
- `SetClashMode(mode)`
- `URLTest(groupTag)`
- traffic snapshot
- connection snapshot

节点列表由 content 静态解析和 active runtime groups 动态状态按 tag 合并。

### UX Structure

主窗口使用五个顶层页面：

| Page | Primary responsibility |
|------|------------------------|
| Proxy | runtime status, start/stop, node list, group selection, mode switch |
| Subscribe | profiles, subscriptions, imports, JSON editor |
| Rules | rule templates and routing-oriented controls |
| Settings | platform, app, window, language, theme settings |
| Diagnostics | diagnostics report, logs, traffic, connections, recovery actions |

五页面导航是信息架构骨架，可以和后端 profile/config reset 并行推进。它不是旧 Phase 切分的一部分，也不依赖旧 Profile Snapshot 模型。

---

## 3. Removed Legacy Shapes

旧形状只允许出现在本节、迁移测试或删除清单中。它们不是未来实现路径。

| Removed shape | Replacement |
|---------------|-------------|
| Profile Snapshot | Profile content 直接作为 runnable config |
| Draft/Snapshot split | editor dirty text 是前端内存态，Save 后才持久化 |
| Managed/Raw Profile | 单一 Profile 类型 |
| `profiles.source_url` | `profile_subscriptions.profile_id` |
| `active_snapshot_id` | `settings.active_profile_id` |
| `runtime_state` table | `settings` |
| `encrypted_content` | `profiles.content` |
| `ContentCodec` | 无字段级配置加密 |
| `FileKeyStore` / `master.key` | 无字段级配置加密 |
| `SnapshotService` | Profile service + activation service |
| snapshot IPC methods | profile content + activation IPC methods |
| persisted parsed node metadata | content parser + runtime groups |

OS proxy snapshot、traffic snapshot 和 connection snapshot 属于其他领域，不在删除范围内。

---

## 4. Target Storage

### Tables

```sql
CREATE TABLE profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE profile_subscriptions (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    update_policy TEXT NOT NULL,
    last_status TEXT NOT NULL,
    last_error TEXT,
    last_checked_at INTEGER,
    last_updated_at INTEGER,
    content_sha256 TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE data_assets (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    source_url TEXT NOT NULL,
    status TEXT NOT NULL,
    cache_key TEXT,
    version TEXT,
    content_sha256 TEXT,
    size_bytes INTEGER,
    last_error TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

`settings.active_profile_id` 保存当前 active Profile。不存在独立 runtime state singleton table。

### Write Rules

任何写入 `profiles.content` 的路径都必须满足：

1. 生成或接收完整 sing-box JSON。
2. 调用 `singboxadapter.Validate()`。
3. 验证失败则不写入。
4. 验证成功才进入事务。

---

## 5. Implementation Plan

### Milestone A: Document Reset

目标：让文档只有一套架构叙事。

交付物：

- 新增 `docs/architecture-decisions.md`。
- 重写 `docs/roadmap.md`。
- 删除讨论稿性质的 profile/config proposal。

验收：

- Roadmap 不再沿用旧 Phase 切分。
- 旧 Profile Snapshot 模型只作为 removed shape 出现。
- Roadmap 的执行计划能直接指导代码重构。

### Milestone B: Storage and Domain Reset

目标：数据库和 domain model 进入单一 Profile content 模型。

交付物：

- `profiles` 表增加或重建为 `content TEXT NOT NULL`。
- `profile_subscriptions` 通过 `profile_id` 指向 Profile。
- `settings.active_profile_id` 替代 profile snapshot active state。
- 删除 profile snapshot 表、encrypted content 表和 runtime state 表。
- 删除字段级加密存储代码。
- 重写 Profile repository。

一次性迁移：

- 如果存在 active profile snapshot，将其 content 迁移到对应 Profile。
- 若无法确定唯一 active content，迁移失败并给出明确 diagnostic。
- 迁移后删除旧表，不保留 runtime 兼容路径。

验收：

- profile/config 业务表只包含 `profiles`、`profile_subscriptions`、`data_assets`、`settings`。
- Profile model 不含 draft、snapshot、mode、active snapshot 字段。
- 订阅来源不在 `profiles` 表。

### Milestone C: API and IPC Reset

目标：公开 API 只表达新模型。

交付物：

- 删除 profile draft 和 profile snapshot IPC。
- 新增或保留以下 Profile API：
  - `createProfile`
  - `updateProfile`
  - `deleteProfile`
  - `listProfiles`
  - `getProfile`
  - `saveProfileContent`
  - `validateProfileContent`
  - `activateProfile`
  - `getActiveProfile`
- Runtime API 使用 profile identity：
  - engine status exposes `active_profile_id`
  - runtime start target uses `profile_id`
- 更新 Wails bridge、IPC client、IPC server、method registry 和 contract tests。

验收：

- Profile snapshot IPC 方法不存在。
- API DTO 不含 profile snapshot identity。
- 前端无法通过旧方法调用 draft/snapshot 生命周期。

### Milestone D: Validation and Config Build Boundary

目标：建立所有写入路径共用的权威验证边界。

交付物：

- `internal/singboxadapter.Validate(configJSON)`。
- validation diagnostic 结构。
- `internal/configbuild` 生成完整 sing-box config：
  - 默认 mixed inbound。
  - outbounds。
  - DNS template。
  - route template。
  - clash mode rule 支持。
- URI parse -> configbuild -> Validate -> persist。
- template create -> configbuild -> Validate -> persist。
- JSON save -> Validate -> persist。

验收：

- 轻量 JSON 检查不能写入持久化配置。
- 所有 `profiles.content` 写入都集中穿过 validation boundary。
- 无效 sing-box config 不进入数据库。

### Milestone E: Runtime Activation

目标：activation 成为 service 层的业务编排。

交付物：

- `ActivateProfile(profileID)` service。
- `RuntimeStartTarget` 改为 profile 语义：
  - `ProfileID`
  - `ConfigJSON`
  - `RequiredCapabilities`
- engine start/stop 接收 target，不读取 Profile。
- activation 失败时恢复系统代理，并 best-effort 启动旧 active Profile。
- `settings.active_profile_id` 只在新 runtime start 成功后更新。

验收：

- engine 不读取数据库。
- engine 不持有 ConfigJSON 作为业务状态。
- rollback 不依赖 profile snapshot。
- active runtime status 使用 profile identity。

### Milestone F: Subscription, Import, and Assets

目标：导入和订阅都成为产生 Profile content 的输入通道。

交付物：

- URI import 创建 Profile。
- subscription create 执行 fetch/parse/build/validate，并在同一事务中创建 Profile 和 subscription row。
- subscription refresh 更新同一个 Profile 的 content。
- refresh 失败不覆盖原 Profile content。
- data assets 保持独立实体，可使用 `data_assets.source_url`。

验收：

- 不存在空 content Profile 中间态。
- `profiles.source_url` 不存在。
- 订阅 refresh 不改变 Profile identity。

### Milestone G: Runtime Controls

目标：运行时控制可用且不污染配置。

交付物：

- content parser 提供静态 group/outbound 列表。
- active runtime groups 提供动态状态。
- `ListProfileNodes(profileID)` 合并静态和动态数据。
- `SelectOutbound(groupTag, outboundTag)`。
- `SetClashMode(mode)` 通过 embedded ClashServer 本地 controller。
- `URLTest(groupTag)`。

验收：

- runtime control 不写 `profiles.content`。
- 不暴露 Clash HTTP API。
- inactive Profile 能显示节点结构。
- active Profile 能显示当前选择和 runtime 状态。

### Milestone H: Navigation Restructure

目标：建立五页面主窗口信息架构，让后续功能进入正确页面，而不是继续堆叠旧视图。

交付物：

- `routing.svelte.ts` 路由收敛为：
  - `proxy`
  - `subscribe`
  - `rules`
  - `settings`
  - `diagnostics`
- `AppShell.svelte` 导航更新为五项。
- `App.svelte` 按五页面渲染。
- 新建或重组五个顶层视图：
  - `ProxyView`
  - `SubscribeView`
  - `RulesView`
  - `SettingsView`
  - `DiagnosticsView`
- 迁移现有视图内容到正确页面：
  - Engine controls -> Proxy。
  - Profile/subscription/editor -> Subscribe。
  - Platform controls -> Settings。
  - diagnostics/logs/traffic/connections -> Diagnostics。

验收：

- 主窗口只有五个顶层导航项。
- 页面路由和状态单例解耦。
- 旧视图不再作为顶层页面继续扩张。
- 页面骨架不依赖 Profile Snapshot、Managed/Raw 或旧 draft state。

### Milestone I: Frontend Feature Reset

目标：前端状态和 UI 只反映新模型。

交付物：

- 重写 Profile state：
  - profiles
  - selected profile
  - editor dirty text
  - validation diagnostics
  - active profile
  - subscriptions
  - runtime groups
- Subscribe page 收敛为：
  - Profile list/import/subscription。
  - JSON editor/save/validate。
- Proxy page 收敛为：
  - Runtime status/start/stop。
  - Node list/group selection/mode switch。
- 删除 snapshot panel、draft state、managed/raw 分支。

验收：

- UI 没有 snapshot、draft、managed/raw 操作。
- Save 表示 validate + persist。
- Activate 表示启动 Profile。
- 运行时选择和模式切换不修改 editor content。

### Milestone J: Release Hardening

目标：进入可发布客户端质量线。

交付物：

- 版本号单源。
- Windows/macOS/Linux 包装只暴露 `qkbox`。
- helper 以私有资源打包。
- smoke test 覆盖：
  - first launch。
  - open window。
  - create profile。
  - save invalid content blocked。
  - save valid content。
  - activate profile。
  - stop runtime。
  - quit cleanup。
- CI gate 覆盖 Go tests、frontend build、format、package script dry run。

验收：

- 用户能安装并启动一个可用客户端。
- 私有 helper 不出现在 Start Menu、Dock、desktop entry 或 PATH 入口。
- app 退出后无残留 runtime owner。

---

## 6. Codebase Refactor Map

### Delete

| Area | Targets |
|------|---------|
| Profile snapshot model | `shared/model/snapshot.go`, profile snapshot DTOs |
| Snapshot service | `core/qkboxd/snapshot_service.go` |
| Snapshot persistence | profile snapshot repository and migrations |
| Draft persistence | draft content repository paths |
| Field encryption | `ContentCodec`, `FileKeyStore`, `master.key` paths |
| Frontend old state | draft/snapshot state and UI panels |

Only delete profile-config snapshot concepts. Do not delete OS proxy snapshot or traffic/connection snapshot types.

### Rewrite

| Area | Targets |
|------|---------|
| Persistence | migrations, profile repository, subscription repository |
| API/IPC | method names, DTOs, server/client/bridge registration |
| Profile service | CRUD, validate, save content, activate |
| Engine integration | runtime target identity and status |
| Navigation shell | routing, AppShell, App, top-level views |
| Frontend Profile UX | state, view, validation display, activation controls |

### Add

| Area | Targets |
|------|---------|
| Validation | `internal/singboxadapter.Validate()` |
| Config build | `internal/configbuild` |
| Import parse | `internal/uriparse` if not already sufficient |
| Runtime nodes | content parser + runtime group merge |
| Clash mode | embedded ClashServer controller capture |

### Keep

| Area | Reason |
|------|--------|
| Process model | Entry/lifecycle work remains valid |
| Private helper boundary | Still required |
| Provider runtime owner pattern | Still required for privileged runtime |
| OS proxy snapshot | Different domain, needed for restore |
| Traffic/connection snapshot | Observability naming, not Profile Snapshot |

---

## 7. Global Acceptance Gates

Before starting product feature expansion beyond the reset, the following must be true:

- Documentation has one architecture source of truth.
- Database has one Profile content model.
- API has no profile draft or profile snapshot lifecycle.
- Service layer owns activation orchestration.
- Engine does not own config.
- All config writes validate through sing-box.
- Subscriptions are separate channels.
- Runtime controls do not mutate config.
- Navigation exposes the five top-level pages.
- Frontend has no old profile lifecycle UI.
- Packaging exposes only supported user entry points.

Useful searches during review:

```powershell
rg -n "ProfileSnapshot|SnapshotService|CreateProfileSnapshot|ActivateProfileSnapshot|RollbackToSnapshot|ValidateProfileDraft|UpdateProfileDraft|active_snapshot_id|runtime_state|encrypted_content|ContentCodec|FileKeyStore|Managed Profile|Raw Profile" .
```

Matches are allowed only when they are:

- code scheduled for deletion in the current reset commit,
- migration tests proving removal,
- OS proxy or observability snapshot terms that are not Profile Snapshot,
- historical git output outside the working tree.

---

## 8. Release Definition

v1.0 is releasable when a fresh user can:

1. Install qkbox.
2. Launch `qkbox`.
3. Open the window from tray.
4. Navigate Proxy, Subscribe, Rules, Settings, and Diagnostics.
5. Create or import a Profile.
6. Save only valid sing-box config.
7. Activate the Profile.
8. See runtime status.
9. Select node and mode where supported by the active config.
10. Stop runtime.
11. Quit cleanly.

No unsupported helper entry point needs to be user-friendly. Unsupported paths should fail clearly and stay private.
