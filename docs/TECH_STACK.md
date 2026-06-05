# qkbox Technical Stack

This document fixes the implementation stack and tool choices. Architecture
rules live in `docs/ARCHITECTURE.md`; milestone sequencing lives in
`docs/IMPLEMENTATION_PLAN.md`.

## Product Stack

```text
Desktop shell: Wails 3
Frontend: Svelte + TypeScript + Vite
Frontend package manager: npm with local Node.js
UI primitives: bits-ui where it fits the existing Svelte surface
Daemon/control plane: Go
Runtime core: embedded sing-box
Transparent network substrate: sing-box using sing-tun
DNS runtime: sing-box using SagerNet DNS components
Persistence: SQLite through modernc.org/sqlite
Local IPC: authenticated local transports
Windows installer: NSIS
Linux package: DEB
CI policy: local verification first
```

## Go Runtime And Boundaries

`qkboxd` is the user-scope product coordinator. It owns user data, profile
orchestration, snapshot lifecycle, reload semantics, and default local runtime
ownership.

The active runtime is held by a `RuntimeOwner`:

```text
local embedded owner
provider-hosted owner
Apple NetworkExtension owner
```

The current local embedded owner wraps `internal/singboxadapter`. Future provider
or extension owners must reuse the same qkbox runtime DTO semantics and must not
invent a second sing-box integration layer.

## SagerNet Component Policy

qkbox uses upstream network components directly or through sing-box:

```text
sing-box
  runtime core

sing-tun
  TUN, route, strict route, DNS hijack, Wintun, WFP, nftables, platform network mechanics

sing-dns
  DNS runtime behavior through sing-box

sing-geosite / sing-geoip
  geo and rule matching assets

srsc
  candidate rule-set conversion tool for the data asset plane
```

qkbox does not implement its own proxy core, DNS runtime, TUN packet stack,
Wintun wrapper, route engine, WFP layer, nftables layer, or geo database format.

## Type Boundary

Only `internal/singboxadapter` may import sing-box or sing packages.

Forbidden outside that boundary:

```text
github.com/sagernet/sing-box imports
github.com/sagernet/sing imports
sing-box option structs
sing-tun structs
Clash API DTOs
runtime object handles
```

Public APIs expose qkbox product DTOs only.

## Frontend

The frontend is a product control surface, not a runtime surface.

Frontend responsibilities:

```text
profile and subscription workflows
runtime status and observability views
system proxy controls
network mode status and repair entry points
diagnostics and update entry points
```

Frontend must not:

```text
own runtime lifecycle
parse sing-box structs as product model
call provider IPC directly
perform platform mutation
invent fake traffic, connection, group, or log data
```

## IPC

Product IPC is qkbox semantic API:

```text
GUI -> qkboxd
```

Provider IPC is an authenticated platform boundary:

```text
qkboxd -> provider/runtime container
```

Both use structured errors and versioned method registries. Neither exposes
arbitrary shell, arbitrary file operations, or sing-box internal APIs.

## Persistence

SQLite stores qkbox product state:

```text
profiles
drafts
snapshots
encrypted content
active snapshot pointer
remote metadata
asset metadata
system proxy owner record
diagnostic metadata
```

Provider storage is minimal:

```text
owner lock
runtime owner identity
repair metadata
stale state marker
```

Provider storage must not contain user profile content, subscription content,
secrets, or decrypted runtime config.

## Platform Tooling

Windows:

```text
Wails desktop app
user-scope qkboxd
privileged provider for machine network runtime
Named Pipe for provider IPC
NSIS installer
osslsigncode or signtool-compatible signing path when signing is introduced
```

macOS:

```text
Wails desktop app
user-scope qkboxd
NetworkExtension runtime container for VPN/TUN mode
Unix socket product IPC
native packaging and signing path when release work starts
```

Linux:

```text
Wails desktop app
user-scope qkboxd
provider-hosted runtime through systemd/root helper or polkit-class authorization
Unix socket product IPC
DEB package first
```

## Build And Verification

Local verification baseline:

```text
go test -tags with_clash_api ./...
go vet -tags with_clash_api ./...
npm run check
```

`with_clash_api` remains part of the local Go verification path because the
pinned sing-box runtime path requires it for current observability integration.
qkbox does not expose Clash HTTP/API as its product control plane.

## Packaging Defaults

Windows installer:

```text
NSIS only
install to Program Files remains supported
provider installation remains an explicit privileged flow
```

Linux package:

```text
DEB first
no RPM/AppImage/Flatpak baseline until explicitly selected
```

macOS package:

```text
native app bundle direction
NetworkExtension entitlement/signing work belongs to Apple runtime milestone
```

## Dependency Rules

Allowed:

```text
GPL-compatible open-source dependencies
official SagerNet components
small platform libraries that preserve product boundaries
```

Not allowed:

```text
local .temp replace sources
closed-source runtime dependencies
sidecar runtime as product path
FFI/libbox product path
runtime hot-swapping separate from product update
```
