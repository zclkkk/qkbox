# qkbox v1.0 Roadmap

## 1. Vision

qkbox 是一个基于 sing-box 的跨平台桌面代理客户端。

**v1.0 目标**：在不妥协架构的前提下，将完整能力以用户友好的 UX 分层暴露，交付一个成熟的代理客户端。

### 约束

| 约束 | 说明 |
|------|------|
| 不做 Clash 格式转换 | sing-box 与 Clash 能力不对齐，转换降级引入大量问题。仅支持 URI 导入 + sing-box 原生 JSON |
| 不做开机自启 | — |
| 不做自动更新 | 尚无 Release |
| 不做节点参数可视化编辑 | FlClash 验证了此路径可行：节点参数修改通过 JSON 编辑器完成 |
| 不做完整路由规则可视化编辑器 | sing-box route rules 是异构结构，GUI 化会迅速膨胀。推迟到 v1.1+ |
| macOS TUN 推迟到 v1.x | 需要 Apple 开发者账号 + NetworkExtension，工程量等同于 2-3 个 Phase |

### 不做的事（及原因）

| 放弃项 | 原因 |
|--------|------|
| Clash 订阅转换 | 能力不对齐，降级策略会成为永久维护负担 |
| 节点表单编辑器 | FlClash 证明不需要——`Proxy` 模型只有 name/type/now，参数修改通过 raw editor |
| 路由规则表编辑 | sing-box rules 异构结构，注册表 + 双向 AST 编辑器的复杂度超出 v1.0 范围 |
| 开机自启 | 非核心功能，后续按需添加 |
| 自动更新 | 尚无 Release，发布后再做 |
| 双引擎（sing-box + Clash） | 架构复杂度翻倍，与 sing-box 隔离原则矛盾 |

---

## 2. Process Model

### 目标模型

```
qkbox            ← 唯一用户入口（托盘 + IPC server + runtime coordinator + local runtime owner when applicable）
qkbox-window     ← 按需打开的 GUI 工具（WebView，关了就释放内存）
qkbox-provider   ← 提权进程（用户无感，TUN/DNS 劫持用）
```

> **runtime coordinator** 而非 engine owner：在 provider-hosted runtime 或未来 macOS NetworkExtension 下，sing-box 运行在别的进程里，qkbox 协调各 runtime owner 的生命周期。只有在本地 HTTP/mixed 代理模式下，qkbox 才直接持有 sing-box runtime。

二进制产出：**3 个**，但用户只感知到 1 个。

### 生命周期

```
用户双击 qkbox
  ├→ 获取用户锁
  ├→ 打开 SQLite
  ├→ 启动 IPC 服务器
  ├→ 注册系统托盘
  │    ├─ 状态图标 (运行中/停止/错误)
  │    ├─ 启动/停止引擎
  │    ├─ 模式切换 (规则/全局/直连)
  │    ├─ 打开主窗口 → spawn qkbox-window
  │    └─ 退出 → 优雅关闭一切
  └→ 等待

用户关闭 qkbox-window 窗口
  └→ qkbox-window 进程退出，WebView 销毁，内存释放
  └→ qkbox 继续运行（托盘 + 引擎不受影响）

用户右键托盘 → 退出
  ├→ 停止引擎（含系统代理恢复）
  ├→ 通知 qkbox-provider 关闭
  ├→ 销毁托盘
  ├→ 关闭 SQLite
  └→ 释放用户锁
```

### qkbox-window 单实例

- qkbox 通过 IPC 追踪是否有活跃的 qkbox-window 连接
- 托盘"打开窗口"：已有连接 → 发送 `ShowWindow` 命令；无连接 → spawn 新进程
- qkbox-window 关闭 = IPC 断开 = qkbox 感知到窗口已关闭

### qkbox-provider 生命周期

- qkbox 存活时：心跳(2s) 通信
- qkbox 正常退出：主动通知 provider 关闭
- qkbox 异常退出：provider 心跳超时(8s) → 自清理 TUN/iptables → 退出
- 双保险机制，无孤儿进程

### qkbox-window 私有入口约束

`qkbox-window` 是 private executable，不进入 PATH / Start Menu / Dock / `.desktop`，只由 `qkbox` spawn。

如果开发者直接运行 `qkbox-window`，且 `qkbox` 不可达：
- 这是 unsupported behavior
- qkbox-window 只做 IPC handshake，失败即报错/退出
- 不自动拉起 qkbox，不做用户兜底

发布验收关注 packaging 是否仍暴露 `qkbox-window`：
- 如果 Start Menu / Dock / `.desktop` / PATH 能直接打开 `qkbox-window`，这是 packaging bug
- 如果 `qkbox-window` 直启没有 fallback，不是 bug

---

## 3. Product Model

### 3.1 两类 Profile

| | Managed Profile | Raw Profile |
|---|---|---|
| 来源 | URI 导入、订阅导入 | sing-box 原生 JSON、手动创建 |
| 结构感知 | qkbox 掌握 outbounds 列表 | qkbox 只做存储 |
| 节点列表 | ✅ 只读展示 + 搜索 + 延迟测试 | ❌ 不解析节点 |
| 代理组切换 | ✅ | ✅（引擎运行时） |
| 模板切换 | ✅ 重新生成 route/DNS/inbounds | ❌ |
| 导入导出 | ✅ URI 格式 | ✅ JSON 文件 |
| 编辑方式 | JSON 编辑器 | JSON 编辑器 |
| 订阅刷新 | ✅ | ❌ |

Profile 模型新增 `mode` 字段：`"managed" | "raw"`。

Managed Profile 的元数据存储在 profile 记录中（节点列表、协议类型、标签等），由解析器在导入时提取。运行时不依赖元数据——引擎只看 JSON。

#### Managed Profile 的双真相防护

Managed Profile 同时持有"生成出的 JSON"和"提取的节点元数据"。如果用户通过 JSON 编辑器修改了 JSON，元数据立刻变旧，破坏用户信任。

**规则：用户通过 JSON 编辑器修改 Managed Profile → 自动 fork 为 Raw Profile。**

```
Managed Profile + JSON 编辑保存
  ↓
mode: "managed" → "raw"
  ↓
节点元数据保留（仅做展示参考，不再可用）
  ↓
节点列表功能不可用，进入纯 JSON 编辑模式
```

一旦用户手动编辑 JSON，qkbox 不再声称自己"理解"这个配置的结构。这是最干净的边界。

### 3.2 生成优先，编辑兜底

**核心原则：配置通过"生成"获得，不通过"编辑"获得。**

```
生成路径（结构化输入 → 完整配置）：
  URI 导入         → uriparse → configbuild → 完整 sing-box JSON
  模板选择         → configbuild → 完整 sing-box JSON
  订阅 URL        → fetch → uriparse → configbuild → 完整 sing-box JSON

编辑路径（已有配置 → 修改）：
  JSON 编辑器      → 手动修改 → validate → 快照 → 启动
```

用户不需要在"节点列表"和"路由规则"之间做结构化编辑。需要改配置时，用 JSON 编辑器 + 快照回滚保障安全。

### 3.3 URI 解析支持等级

| 等级 | 含义 | v1.0 协议 |
|------|------|----------|
| **Supported** | 完整解析，已知格式全覆盖 | ss (SIP002), trojan, hysteria2 |
| **Experimental** | 常见格式覆盖，边缘参数可能降级 | vmess (标准格式), vless (基础参数) |
| **Unsupported** | 生成 diagnostic 告知用户，不导入 | vless (reality/xtls), tuic |
| **Ignored** | 静默跳过，计入导入摘要 | 空行、注释、非 URI 文本 |

导入结果返回 `ImportReport`：
```
成功: 12 个节点 (8 ss, 3 vmess, 1 trojan)
跳过: 3 行（空行/注释）
不支持: 1 个（tuic://... — tuic 暂不支持）
```

### 3.4 配置校验策略

#### 校验职责分层

| 层 | 职责 | 不做的事 |
|---|------|---------|
| **configbuild** | 生成 JSON 时确保结构合法 | — |
| **validate.go** | 轻量 pre-check：JSON 合法、顶层对象、inbounds/outbounds 存在 | 不做深度 schema 校验，不翻译 box.New() error |
| **singboxadapter** | 权威校验：`box.New()` 的 error，以及 error → diagnostic 翻译 | — |

`validate.go` 不膨胀。它只做快速反馈（前端即时校验），不承担 `box.New()` 错误翻译的职责。错误翻译放在 singboxadapter 的 snapshot 创建流程中。

#### Snapshot 创建流程

**核心规则：只有 singboxadapter 权威校验通过，才能创建 runnable snapshot。**

```
configbuild 生成 / 用户编辑 JSON
  ↓
validate.go 轻量 pre-check（JSON 合法、顶层结构）
  ↓ 失败 → 返回前端即时反馈（不创建 snapshot）
  ↓ 通过
创建 snapshot → singboxadapter.Parse() + box.New() 权威校验
  ↓ 通过 → snapshot 标记 valid + 提取 RuntimeSummary
  ↓ 失败 → snapshot 标记 invalid + diagnostic 来自 box.New() error（由 singboxadapter 翻译）
  ↓
用户 activate snapshot → 引擎启动（已知配置有效）
```

不存在"snapshot valid 但 start 失败"的情况。snapshot 的 valid 状态由 singboxadapter 权威校验保证。

---

## 4. UX Structure

### 4.1 五页面结构

```
┌─────────────────────────────────────┐
│  代理   订阅   规则   设置   诊断    │
├─────────────────────────────────────┤
│  代理: 节点列表+搜索+延迟测试        │
│        模式切换(规则/全局/直连)       │
│        代理组切换                    │
│  订阅: 订阅管理+URI导入              │
│        Profile CRUD+快照管理         │
│  规则: 模板选择(规则/全局/直连)       │
│        DNS策略+入站端口              │
│        JSON编辑器(fallback)          │
│  设置: 系统代理+托盘行为              │
│        Provider/TUN(进阶)            │
│  诊断: 日志+流量+连接                │
│        调试包+数据资产(进阶)          │
└─────────────────────────────────────┘
```

### 4.2 内容分层

每个页面分**基础**和**进阶**两层。基础层覆盖 80% 用户需求，进阶层提供完整能力。

| 页面 | 基础层 | 进阶层 |
|------|--------|--------|
| 代理 | 节点列表、代理组切换、延迟测试、快速启停 | 连接列表管理 |
| 订阅 | 订阅 CRUD、URI 导入、节点列表（只读） | 快照管理、Profile JSON 编辑 |
| 规则 | 模板切换、DNS 策略、入站端口 | JSON 编辑器 |
| 设置 | 系统代理开关、窗口关闭行为 | Provider 状态、TUN 准备 |
| 诊断 | 日志、流量、连接 | 调试包、数据资产、能力矩阵 |

### 4.3 托盘菜单

```
qkbox
├─ [图标] 引擎运行中 / 已停止 / 错误
├─ 启动引擎 / 停止引擎
├─ ─────────
├─ 模式: ● 规则模式 / ○ 全局代理 / ○ 直连
├─ ─────────
├─ 打开主窗口
└─ 退出 qkbox
```

---

## 5. Phases

### Dependency Graph

```
Phase 0A (Entry + Lifecycle)        Phase 0B (Infrastructure)
  │                                    │
  ├──────────────────┐                 │
  │                  │                 │
  ▼                  ▼                 │
Phase 1            Phase 2             │
(URI Parser +      (Navigation         │
 Config Builder)    Restructure)       │
  │                  │                 │
  └────────┬─────────┘                 │
           │                           │
           ▼                           │
       Phase 3 (Proxy + Subscribe)  ◄─┘ (0B 不阻塞产品闭环，但 Phase 3 开始前应完成)
           │
           ▼
       Phase 4 (Rules + Settings)
           │
           ▼
       Phase 5 (Diagnostics Polish)
           │
           ▼
       Phase 6 (i18n + Theme)
           │
           ▼
       Phase R (Release Hardening)
           │
           ▼
       Phase 7 (macOS TUN) [v1.x]
```

Phase 0A 是架构阻塞项——Phase 1 和 Phase 2 依赖它。Phase 0B 是发布基础设施，可以和 Phase 1/2 并行，Phase 3 开始前应完成。

---

### Phase 0A: Entry + Lifecycle

**目标：** 完成进程模型重构——单一入口、系统托盘、窗口生命周期。这是架构阻塞项。

| 变更 | 说明 |
|------|------|
| `cmd/qkboxd/` → `cmd/qkbox/` | 重命名 + 引入 `getlantern/systray` 系统托盘 |
| `cmd/qkbox/` 新增 `tray.go` | 托盘菜单：引擎控制、模式切换、打开窗口、退出 |
| `cmd/qkbox/` 新增 `ShowWindow` IPC 方法 | 通知 qkbox-window 弹到前台 |
| `apps/desktop/` 产出重命名 | `qkbox` → `qkbox-window` |
| `apps/desktop/bridge.go` | 移除 `launchQKBoxD()` fallback；只做 IPC handshake，qkbox 不可达即失败 |
| IPC 协议 | 新增 GUI 连接状态追踪（`WindowSession`）、`ShowWindow` 方法 |
| `packaging/*` | 所有平台更新二进制名、路径；Start Menu / Dock / .desktop 指向 `qkbox`，不得暴露 `qkbox-window` |

**完成标准：** qkbox 启动 → 托盘出现 → 点击"打开窗口"→ qkbox-window 弹出 → 关闭窗口 → 托盘仍在 → 点击"退出"→ 一切关闭。

**不涉及：** 不改变业务逻辑、不改变 IPC 协议语义（只新增方法）、不改变状态管理。

---

### Phase 0B: Infrastructure

**目标：** 建立自动化质量基线。与 Phase 0A 可以并行，不阻塞产品闭环，但 Phase 3 开始前应完成。

| 变更 | 说明 |
|------|------|
| CI/CD | GitHub Actions: Go test + frontend check + 跨平台构建 matrix (Win/Mac/Linux) |
| 版本管理 | `VERSION` 文件 → Go ldflags + package.json + Wails config.yml 统一注入 |
| 应用图标 | `.ico` / `.icns` / `.png`，嵌入打包流程 |
| 窗口状态 | 记忆窗口位置/大小，持久化到 settings 表 |
| 前端测试 | vitest 基础设施，覆盖 routing + format |

---

### Phase 1: URI Parser + Config Builder

**目标：** 构建"URI → 可用 sing-box 配置"的完整管道。

这是整条 Roadmap 的关键路径——Phase 3 的订阅导入和节点列表都依赖此 Phase。

#### 模块拆分

**`internal/uriparse/`** — URI 解析

| 职责 | 说明 |
|------|------|
| 格式检测 | sing-box JSON → 直通；base64 → decode → URI 列表；单行 → URI |
| URI 解析 | `ss://` `vmess://` `trojan://` `hysteria2://` `vless://` `tuic://` |
| 输出 | `[]ParsedOutbound`（结构化节点信息：tag、type、server、port、协议参数） |
| 不做的事 | 不生成完整配置、不依赖 sing-box 库 |

每个协议解析器按支持等级实现：
- Supported：完整解析所有已知参数
- Experimental：解析常见参数，忽略的参数记录在 diagnostic
- Unsupported：返回 diagnostic，不生成 outbound

**`internal/configbuild/`** — 配置组装

| 职责 | 说明 |
|------|------|
| outbound → JSON | 将 `[]ParsedOutbound` 转为 sing-box outbounds JSON 数组 |
| 默认模板 | 生成完整 sing-box config：inbounds（mixed://127.0.0.1:7890）+ outbounds + route + DNS |
| selector group | 自动创建 `proxy` selector outbound，包含所有导入节点 |
| tag 命名 | 去重 + 序号策略，处理 emoji/特殊字符/重复 tag |
| 模板切换 | 规则模式 / 全局代理 / 直连模式 → 不同的 route + DNS 配置 |
| 不做的事 | 不做节点参数编辑、不做 AST 级 JSON 修改 |

#### 配置生成管道

```
URI string / base64 string
  ↓
uriparse: detect → parse → []ParsedOutbound
  ↓
configbuild: wrap outbounds + default inbounds + template route + template DNS
  ↓
完整 sing-box JSON string
  ↓
profile_service: encrypt → store as Managed Profile
```

#### 变更范围

| 层 | 变更 |
|---|------|
| **新** `internal/uriparse/` | 格式检测 + 各协议解析器 |
| **新** `internal/configbuild/` | outbounds 组装 + 默认模板 + tag 命名 |
| `shared/model/profile.go` | Profile 新增 `Mode` 字段（`managed` / `raw`） |
| `shared/model/` | 新增 `ParsedOutbound`、`ImportReport` 模型 |
| `core/qkboxd/asset_service.go` | `RefreshProfileSubscription` 流程接入 uriparse → configbuild |
| `core/qkboxd/profile_service.go` | 新增 `ImportNodes` 方法（URI 文本 → Managed Profile） |
| `core/qkboxd/validate.go` | 保持轻量 pre-check 不变（不膨胀） |
| `shared/api/` | 新增 `ImportNodesRequest`/`ImportNodesReply`、`ImportReport` |

#### 不涉及

- 不改变 UI
- 不改变 IPC 传输层
- 不改变 profile/snapshot 存储模型（只新增字段）

---

### Phase 2: Navigation Restructure

**目标：** 将 4 页面重构为 5 页面，建立新交互骨架。

与 Phase 1 完全并行，不依赖 URI parser。

#### 变更范围

| 层 | 变更 |
|---|------|
| `routing.svelte.ts` | `Route` 类型 → `proxy / subscribe / rules / settings / diagnostics` |
| `AppShell.svelte` | `navItems` → 5 项，图标重新选型 |
| `App.svelte` | view conditional → 5 视图 |
| **新视图** | `ProxyView` / `SubscribeView` / `RulesView` / `SettingsView` / `DiagnosticsView` |
| CSS | 侧栏 5 项布局调整 |

#### 页面拆分来源

| 新页面 | Phase 2 内容 | Phase N 填充 |
|--------|-------------|-------------|
| 代理 | 占位 / 迁移 EngineView 控制面板 | Phase 3 |
| 订阅 | 占位 / 迁移 ProfilesView CRUD+订阅 | Phase 3 |
| 规则 | 占位 | Phase 4 |
| 设置 | 占位 / 迁移 PlatformView | Phase 4 |
| 诊断 | 迁移 DiagnosticsView + EngineView 日志/流量/连接 | Phase 5 |

#### 架构影响

- 状态单例与路由完全解耦——state 文件不需要改动
- `AppShell` 的 `children` snippet 模式使 shell 路由无关
- 需要改动的文件：`routing.svelte.ts`、`App.svelte`、`AppShell.svelte` + 新增 5 个 view 文件

---

### Phase 3: Proxy + Subscribe Pages

**目标：** 实现核心操作流：导入订阅 → 浏览节点 → 选择节点 → 连接上网。

依赖 Phase 1（URI parser）和 Phase 2（导航骨架）。这是工作量最大的 Phase。

#### 代理页面（ProxyView）

| 功能 | 组件 | 数据源 |
|------|------|--------|
| 节点列表 | `NodeList`（新） | Managed Profile 的 ParsedOutbound 列表 |
| 节点搜索/过滤 | 内嵌 NodeList | 前端过滤 |
| 延迟测试（单节点） | 延迟按钮 + 结果 | `engineState.urlTest()` 扩展 |
| 延迟测试（批量） | 全局测试按钮 | 新 API 或现有 URLTest 批量 |
| 代理组切换 | 迁移 `OutboundGroupsPanel` | `engineState.groups` |
| 快速连接/断开 | 精简 `EngineControlPanel` | `engineState.start/stop()` |
| 流量速览 | 精简 `TrafficPanel` | `runtimeEvents.traffic` |

节点列表是**只读**的——展示 tag、类型、服务器、端口、延迟。不做节点参数编辑。

#### 订阅页面（SubscribeView）

| 功能 | 组件 | 数据源 |
|------|------|--------|
| 订阅列表 | 迁移 ProfilesView 订阅部分 | `assetState.subscriptions` |
| 添加订阅（URL） | 表单 | `assetState.createSubscription()` |
| 导入（剪贴板/文件/URI） | `ImportDialog`（新） | Phase 1 解析器 |
| 订阅刷新 | 刷新按钮 | `assetState.refreshSubscription()` |
| Profile 管理 | 迁移 ProfilesView CRUD | `profileState` |
| 快照管理 | 迁移 ProfilesView 快照（进阶折叠） | `profileState.snapshots` |

#### 后端变更

| 层 | 变更 |
|---|------|
| `shared/api/` | `ImportNodesRequest`/`Reply`、`ListParsedNodesRequest`/`Reply` |
| `core/qkboxd/asset_service.go` | 接入 uriparse → configbuild |
| `core/qkboxd/profile_service.go` | `ListParsedNodes(profileID)` — 从 Managed Profile 提取节点列表 |
| `shared/model/` | `ParsedOutbound` 用于前端展示 |

#### 节点列表数据流

```
Managed Profile (stored JSON)
  ↓
uriparse: 从 JSON outbounds 提取 → []ParsedOutbound（只读）
  ↓
前端 NodeList 组件展示（tag / type / server / port / latency）
  ↓
用户选择代理组出站 → engineState.selectOutbound()
  ↓
（不修改 profile JSON，只修改引擎运行时状态）
```

---

### Phase 4: Rules + Settings Pages

**目标：** 暴露模板切换和平台配置能力。

#### 规则页面（RulesView）— v1.0 范围

**基础层（v1.0）：**

| 功能 | 说明 |
|------|------|
| 模板选择器 | 规则模式 / 全局代理 / 直连模式 → 一键生成新配置 |
| DNS 策略 | fake-ip / direct / remote — 简化选择 |
| 入站基础配置 | mixed 端口、绑定地址（127.0.0.1 / 0.0.0.0） |
| JSON 编辑器 | CodeMirror 作为 fallback，迁移现有组件 |

**推迟到 v1.1+：**

| 功能 | 推迟原因 |
|------|---------|
| 路由规则表可视化编辑 | sing-box rules 异构结构，需要规则类型注册表 |
| DNS servers 表格编辑 | 属于高级 DNS 配置 |
| fakeip 配置面板 | 细粒度配置，v1.0 用模板默认值 |
| 规则类型注册表 | 前端 TypeScript 注册表，工程量大 |

模板选择器的后端实现已在 Phase 1 的 `configbuild` 中完成。Phase 4 只需前端 UI 触发。

#### 设置页面（SettingsView）

| 功能 | 说明 |
|------|------|
| 系统代理开关 | 迁移 `SystemProxyPanel` |
| 窗口关闭行为 | 直接关闭 / 询问确认（见进程模型） |
| 主题切换 | 开关 + Phase 6 实现 |
| 语言切换 | 开关 + Phase 6 实现 |

**进阶层：**

| 功能 | 说明 |
|------|------|
| Provider 状态 | 迁移 `ProviderStatusPanel` |
| TUN/DNS 劫持准备 | 迁移 `CapabilityMatrixPanel` TUN 部分 |

#### 窗口关闭行为（对齐新进程模型）

qkbox-window 关闭 = 关闭窗口进程，qkbox 继续运行。不存在"最小化到托盘"——窗口和托盘是两个进程。

```
关闭窗口时：
  ○ 直接关闭窗口
  ○ 询问确认
```

不提供"关闭窗口 = 退出 qkbox"。真正退出通过托盘菜单的 Quit 按钮。

---

### Phase 5: Diagnostics Polish

**目标：** 打磨可观测性面板，增强诊断能力。

| 功能 | 变更 |
|------|------|
| 日志面板 | 迁移到诊断页，增加文件 sink（按天轮转）+ 导出 |
| 流量面板 | 迁移到诊断页，扩展历史窗口 |
| 连接面板 | 迁移到诊断页，增加连接详情弹窗 |
| 出站组管理 | 迁移到代理页（代理操作，非诊断） |
| 诊断报告 | 增强 health check 覆盖面 |
| 数据资产 | 迁移到诊断页进阶区域 |
| 能力矩阵 | 迁移到诊断页进阶区域 |

#### 日志持久化

在 `internal/eventhub/` 层增加可选文件 sink。日志文件按天轮转，存放在 stateDir。不在 DB 中存储——SQLite 不适合追加写入密集的日志场景。

---

### Phase 6: i18n + Theme

**目标：** 国际化 + 深色模式。

#### 国际化

| 层 | 变更 |
|---|------|
| 前端 | 引入 `typesafe-i18n`（轻量、TypeScript 友好） |
| 语言文件 | `zh-CN.json` / `en-US.json` |
| 后端错误 | 结构化 error code → 多语言 message 映射 |

#### 深色模式

| 层 | 变更 |
|---|------|
| `variables.css` | 扩展为 `[data-theme="dark"]` 作用域 |
| `global.css` | 所有硬编码颜色 → CSS 变量引用 |
| 主题切换 | 读取 OS 偏好 + 手动覆盖，持久化到 settings |

---

### Phase R: Release Hardening

**目标：** 发布前的安全和质量保障。在 Phase 6 之后、Phase 7 之前。

| 项目 | 说明 |
|------|------|
| 代码签名 (Windows) | EV 证书 + signtool 集成到打包流程 |
| 代码签名 (macOS) | Apple Developer ID + `xcrun notarytool` 公证 |
| CI 最终验证 | Win/Mac/Linux × amd64/arm64 构建矩阵 |
| 安装包 smoke test | 安装 → 启动 → 连接 → 卸载 全流程 |
| Checksums | SHA-256 校验和 |
| `SECURITY.md` | 安全策略文档 |
| Release notes 模板 | CHANGELOG 格式 |

---

### Phase 7: macOS TUN (NetworkExtension) — v1.x

**目标：** 补全 macOS 的 TUN 模式。

| 项目 | 说明 |
|------|------|
| Apple Developer 账号 | NetworkExtension entitlement |
| NetworkExtension target | 独立的 System Extension |
| sing-box 嵌入运行时 | 作为 NE 的代理核心 |
| 主应用 ↔ 扩展 IPC | 运行时通信通道 |
| 安装/卸载/状态管理 | System Extension 生命周期 |
| App Sandbox 兼容 | Hardened Runtime |

放在 v1.x 的原因：需要 Apple 开发者账号、工程量等同于 2-3 个 Phase、system proxy 模式对大多数使用场景足够。

---

## 6. Module Architecture

### 新增模块

| 模块 | 路径 | 职责 | Phase |
|------|------|------|-------|
| 系统托盘 | `cmd/qkbox/tray.go` | 托盘菜单、qkbox-window 生命周期管理 | 0 |
| URI 解析 | `internal/uriparse/` | 格式检测、各协议解析器、支持等级分类 | 1 |
| 配置组装 | `internal/configbuild/` | outbounds → JSON、默认模板、tag 命名、模板切换 | 1 |

### 模块职责边界

```
internal/uriparse/
  输入: URI string / base64 string / JSON string
  输出: []ParsedOutbound + ImportReport
  不依赖: singboxadapter、configbuild

internal/configbuild/
  输入: []ParsedOutbound
  输出: sing-box JSON string (完整配置)
  不依赖: singboxadapter、uriparse

internal/singboxadapter/
  输入: sing-box JSON string
  输出: 运行中的 box 实例
  权威校验: box.New() error → diagnostic 翻译（错误翻译的唯一归属）
  不依赖: uriparse、configbuild

core/qkboxd/
  编排层: 调用 uriparse → configbuild → validate → encrypt → store
```

### 现有模块变更

| 模块 | 变更 | Phase |
|------|------|-------|
| `cmd/qkboxd/` | 重命名为 `cmd/qkbox/`，新增 tray.go | 0 |
| `apps/desktop/` | 产出重命名；qkbox-window 作为 private helper，仅由 qkbox spawn | 0 |
| `core/qkboxd/asset_service.go` | 接入 uriparse + configbuild | 1 |
| `internal/singboxadapter/` | snapshot 创建流程中承担 error → diagnostic 翻译 | 1 |
| `core/qkboxd/profile_service.go` | 新增 ImportNodes、ListParsedNodes；Managed→Raw fork 逻辑 | 1, 3 |
| `shared/model/profile.go` | Profile 新增 Mode 字段 | 1 |
| `internal/eventhub/` | 日志文件 sink | 5 |
| `packaging/*` | 二进制名更新 | 0 |

---

## 7. Code Impact Matrix

### 后端

| 文件 | Ph.0 | Ph.1 | Ph.3 | Ph.4 | Ph.5 |
|------|------|------|------|------|------|
| `cmd/qkboxd/` | **重命名** + tray | | | | |
| `apps/desktop/main.go` | **重命名**产出 | | | | |
| `apps/desktop/bridge.go` | **移除** launch fallback；仅 IPC handshake | | | | |
| `asset_service.go` | | **接入** uriparse | 增强 | | |
| `singboxadapter/` | | snapshot 创建 error 翻译 | | | |
| `profile_service.go` | | 新增 ImportNodes + fork | 新增 ListParsedNodes | | |
| `shared/model/profile.go` | | 新增 Mode | | | |
| `shared/api/*.go` | | 新增 Import DTO | 新增 ListParsedNodes DTO | | |
| `internal/eventhub/` | | | | | 新增文件 sink |
| `packaging/*` | **更新** 二进制名 | | | | |

### 前端

| 文件 | Ph.0 | Ph.2 | Ph.3 | Ph.4 | Ph.5 |
|------|------|------|------|------|------|
| `routing.svelte.ts` | | **重写** 5路由 | | | |
| `App.svelte` | | **重写** 5视图 | | | |
| `AppShell.svelte` | | **重写** 5导航 | | | |
| `EngineView.svelte` | | 拆分 | 重组到代理页 | | |
| `ProfilesView.svelte` | | 拆分 | 重组到订阅页 | | |
| `PlatformView.svelte` | | 拆分 | | 重组到设置页 | |
| `DiagnosticsView.svelte` | | 重组 | | | **增强** |
| `engine.svelte.ts` | | | 新增批量测试 | | |
| `global.css` | | 样式调整 | 新组件样式 | | |

---

## 8. Risks

| 风险 | 严重性 | Phase | 缓解 |
|------|--------|-------|------|
| Wails v3 alpha 稳定性 | 高 | 0 | 托盘不依赖 Wails（用 getlantern/systray）；WebView 只用于 qkbox-window |
| sing-box v1.14.0-alpha 配置格式变化 | 中 | 1 | configbuild 生成 JSON，不硬编码 schema；singboxadapter 做权威校验 |
| URI 格式多样性超预期 | 中 | 1 | 支持等级分级 + diagnostic 报告；Unsupported 不阻断导入 |
| getlantern/systray Linux DE 兼容性 | 低 | 0 | 已有大量生产项目验证；fallback 为无托盘模式 |
| 五页面重构影响现有功能 | 低 | 2 | 状态单例与路由解耦；AppShell 路由无关；纯视层改动 |

---

## 9. Summary

| 维度 | 决策 |
|------|------|
| 进程模型 | qkbox（单入口）+ qkbox-window（按需 GUI）+ qkbox-provider（提权） |
| 配置策略 | 生成优先（URI + 模板），编辑兜底（JSON 编辑器 + 快照） |
| Profile 类型 | Managed（URI 导入，有节点元数据）+ Raw（原生 JSON）。JSON 编辑 Managed → 自动 fork 为 Raw |
| 节点交互 | 只读列表 + 搜索 + 延迟测试 + 代理组切换。不做参数编辑 |
| 规则交互 | 模板切换（规则/全局/直连）+ DNS 策略。不做规则表编辑 |
| 校验策略 | configbuild 生成时保证 + validate.go 轻量 precheck + singboxadapter 权威校验（含 error 翻译）。snapshot valid = singboxadapter 通过 |
| 格式兼容 | sing-box JSON + URI（ss/vmess/trojan/hysteria2/vless/tuic）。不做 Clash |
