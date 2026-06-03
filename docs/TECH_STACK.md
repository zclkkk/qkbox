# qkbox 技术栈定稿

这份文档固定 qkbox 的技术栈和工具链选择。后续 Phase 执行时，必须同时遵守：

```text
docs/ARCHITECTURE.md
docs/IMPLEMENTATION_PLAN.md
docs/TECH_STACK.md
```

如果三者冲突，优先级为：

```text
ARCHITECTURE > TECH_STACK > IMPLEMENTATION_PLAN
```

## 已确定技术栈

### Desktop Shell

```text
Wails 3
```

Wails 负责桌面窗口、Go 与前端绑定、基础打包任务和平台 WebView 集成。

Wails 不是 runtime owner。Wails Go 侧只能作为 GUI bridge，不拥有 profile 编排、runtime truth、secret、TUN、route、DNS 或 system proxy 状态。

### Frontend

```text
Svelte 5
TypeScript
Vite
bits-ui
lucide-svelte
npm
local Node.js
```

规则：

```text
npm is the only frontend package manager.
Do not introduce pnpm, yarn, bun, corepack, or a Node version manager as project baseline.
Use the local Node.js installation available on the development machine.
```

`bits-ui` 只作为 headless primitive 来源。业务页面不得直接依赖 bits-ui。

推荐边界：

```text
apps/desktop/frontend/src/lib/ui/*
  may wrap bits-ui primitives

apps/desktop/frontend/src/routes or feature views
  consume qkbox UI wrappers
  do not import bits-ui directly
```

### Backend

```text
Go
```

Go 负责：

```text
qkboxd
system tray (qkboxd owns the tray, pure Go, no WebView)
Wails bridge
IPC client/server
profile and snapshot services
persistence
singboxadapter
platform capability boundaries
```

`qkboxd` 是 user-scope coordinator，也是默认 RuntimeOwner，同时拥有系统托盘图标。它不应以 root、LocalSystem 或机器级 service 身份保存用户 profile 或 secret。

GUI 关闭窗口时进程完全退出，不留后台。qkboxd 通过托盘保持用户可见性。详见 `docs/ARCHITECTURE.md` qkboxd 生命周期章节。

## GUI 到 qkboxd IPC

### Recommended Shape

```text
Svelte UI
  -> Wails generated TypeScript bindings
  -> Wails Go bridge
  -> local IPC client
  -> user-scope qkboxd IPC server
  -> qkboxd internal services
```

前端不直接打开 socket、pipe 或本地端口。Wails Go bridge 是薄桥，只做参数转发、错误映射和事件转发。

### Transport

```text
Windows
  Named Pipe

macOS / Linux
  Unix Domain Socket
```

协议使用 length-prefixed JSON frames。Product path 不使用未鉴权 localhost TCP control port。

IPC 认证采用 OS 访问控制 + pre-shared token 两层，跨平台对称。详见 `docs/ARCHITECTURE.md` IPC 章节。

### Protocol Style

采用 JSON-RPC-like 的产品协议，而不是完整照搬 JSON-RPC 2.0。

请求：

```json
{
  "id": "req_123",
  "method": "engine.getStatus",
  "params": {},
  "deadline_ms": 5000
}
```

响应：

```json
{
  "id": "req_123",
  "result": {
    "state": "IDLE"
  }
}
```

错误：

```json
{
  "id": "req_123",
  "error": {
    "code": "ENGINE_NOT_STARTED",
    "message": "Engine is not started",
    "detail": null,
    "source": "qkboxd",
    "recoverable": true,
    "user_action": "Start the engine first",
    "debug_ref": null
  }
}
```

流式数据通过订阅方法和服务端事件表达。Wails bridge 再把 qkboxd events 转成前端事件。

```text
engine.subscribeStatus
engine.subscribeLogs
```

### qkboxd Lifecycle

Phase 1 目标形态：

```text
GUI starts
  -> try connect to current user's qkboxd
  -> if unavailable, launch bundled qkboxd
  -> wait for pipe/socket readiness
  -> Hello handshake
  -> capability bootstrap
```

`qkboxd` 必须使用 per-user single-instance lock。GUI 退出窗口不等于 runtime truth 消失。

## API 契约

### Recommended Shape

```text
Go DTOs are the contract source of truth.
Wails generated TypeScript bindings expose frontend types.
Contract tests guard JSON shape and method registry.
```

不在 Phase 0/1 引入 Protobuf、buf 或 OpenAPI。

### Module Boundary

```text
shared/model
  qkbox product domain models
  no transport concern
  no sing-box type

shared/api
  request/reply DTOs
  structured errors
  capability enums
  method names
  no sing-box type

core/qkboxd
  imports shared/api and shared/model
  implements services

apps/desktop
  imports shared/api and shared/model
  exposes Wails bridge methods
```

禁止：

```text
shared/api imports sing-box
shared/model imports sing-box
frontend models mirror sing-box config structs
Wails bridge exposes sing-box runtime objects
```

### Versioning

`HelloReply` 至少包含：

```text
api_version
min_supported_api_version
schema_revision
app_version
qkboxd_version
platform
runtime_capabilities
platform_capabilities
```

版本不兼容必须返回 structured error。

### Contract Tests

Phase 0/1 至少建立：

```text
DTO JSON marshal/unmarshal golden tests
method registry test
structured error code test
forbidden import test for shared/api and shared/model
```

## Persistence

### SQLite Driver

```text
database/sql + modernc.org/sqlite
```

选择理由：

```text
Pure Go driver.
Avoids CGO as the default persistence path.
Simplifies Windows/macOS/Linux development and packaging.
Performance is sufficient for profile/snapshot metadata.
```

如果未来发现 modernc sqlite 在目标平台存在不可接受问题，可以在 persistence boundary 内替换 driver。公开 API、profile model、snapshot model 不应因此变化。

### Storage Model

```text
SQLite
  profile metadata
  snapshot metadata
  settings
  asset metadata
  migration version
  active snapshot pointer

Encrypted blobs
  raw profile content
  snapshot source material
  optional normalized content
```

raw profile content 和 snapshot content 不以明文写入 SQLite。

## Secret And Encryption

推荐形态：

```text
Go envelope encryption
AES-GCM content encryption
OS SecretStore stores key material or key references
```

目标 SecretStore 后端：

```text
Windows Credential Manager
macOS Keychain
Linux Secret Service
```

Phase 2 可以先建立 SecretStore interface 和最小可验证后端，但不得把 raw profile content 明文保存到普通 persistence。

## Packaging

### Windows

```text
NSIS only
```

默认开发和早期发布路径推荐 per-user install：

```text
%LOCALAPPDATA%\Programs\qkbox
```

但安装设计必须保留安装到 Program Files 的能力：

```text
%ProgramFiles%\qkbox
```

这意味着：

```text
installer scripts must not assume per-user install only
runtime data must not be stored under install dir
qkboxd user data remains per-user even when binaries are machine-wide
privileged provider installation remains a separate explicit flow
uninstall should not delete user data by default
```

Windows package contents:

```text
qkbox.exe
qkboxd.exe
frontend assets bundled by Wails
NSIS uninstaller
Start Menu shortcut
optional Desktop shortcut
```

不做：

```text
MSIX
WiX
Inno Setup
driver/helper install in the main app installer
mandatory admin install for normal GUI usage
```

### macOS

Wails app bundle is the baseline.

正式分发方向：

```text
signed .app
notarized .app
DMG produced by project-owned packaging task when needed
```

TUN/VPN mode 的正式方向是 Network Extension。它不属于 Phase 0/1。

### Linux

```text
DEB only for the first Linux packaging target
```

目标形态：

```text
modern Debian/Ubuntu family
GTK4 + WebKitGTK 6.0 baseline
```

推荐安装布局：

```text
/usr/bin/qkbox
/usr/lib/qkbox/qkboxd
/usr/share/applications/qkbox.desktop
/usr/share/icons/hicolor/...
```

不做：

```text
AppImage
RPM
AUR
systemd user service in Phase 0/1
root helper in Phase 0/1
legacy GTK3/WebKit2GTK path unless later required
```

## Local Verification

CI 暂时定义为 local-only repeatable checks。

根目录必须提供一个明确入口，用于本地验证：

```text
check
test
build
```

推荐命令族：

```text
npm run check
npm run build
go test ./...
go vet ./...
wails3 doctor
wails3 build
```

可以使用 Wails/Taskfile 组织命令，但不引入远端 CI 作为 Phase 0 要求。

## Deferred Tooling

以下工具或方案暂不作为 Phase 0/1 baseline：

```text
pnpm
yarn
bun
corepack
Node version manager
Protobuf
buf
OpenAPI
Rust
SQLCipher
Docker-based CI
GitHub Actions
MSIX
WiX
Inno Setup
AppImage
RPM
AUR
code signing certificate setup
macOS Network Extension implementation
Windows privileged helper implementation
Linux privileged helper implementation
```

这些不是永久禁止项。只有当对应 Phase 的产品目标真正需要时，才允许引入。

## Phase 0 Toolchain Baseline

Phase 0 开始前，本机至少需要：

```text
Go available on PATH
%USERPROFILE%\go\bin available on PATH
Node available on PATH
npm available on PATH
wails3 available on PATH
WebView2 Runtime installed on Windows
```

Phase 0 不要求：

```text
Rust
MSVC Build Tools
NSIS
Windows SDK signing tools
Docker
protoc
buf
```

NSIS 在 Windows packaging phase 前安装即可。
