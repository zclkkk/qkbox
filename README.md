# qkbox

qkbox is a Windows, macOS, and Linux desktop GUI client built around:

```text
Wails GUI + user-scope qkboxd + explicit RuntimeOwner + Platform Capability Boundary
```

qkbox uses embedded sing-box as its runtime core and relies on official SagerNet
components for network substrate behavior. Product code owns profile/snapshot
orchestration, runtime owner selection, IPC, observability, system proxy
ownership, and platform diagnostics.

## Workspace

```text
apps/desktop
  Wails 3 desktop app and Svelte frontend

cmd/qkboxd
  user-scope qkboxd executable entry point

core/qkboxd
  qkboxd service, RuntimeOwner state machine, and IPC server

internal/ipc
  local framed JSON IPC and platform transport

platform
  platform capability boundary and native providers

shared/api
  public qkbox API DTOs and structured errors

shared/model
  public qkbox product models

packaging
  installer and package assets outside Wails build assets
```

## Local Commands

```powershell
npm run frontend:install
npm run check
npm run build
```

`npm run build` builds `bin/qkboxd.exe` first, then builds the Wails desktop app.

## Architecture

Read `docs/ARCHITECTURE.md` before changing runtime, provider, platform, or
public API boundaries.
