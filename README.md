# qkbox

qkbox is a Windows, macOS, and Linux desktop proxy client built around:

```text
qkbox tray/daemon + private qkbox-window helper + qkbox-provider + RuntimeOwner + Platform Capability Boundary
```

qkbox uses embedded sing-box as its runtime core and relies on official SagerNet
components for network substrate behavior. Product code owns profile/snapshot
orchestration, runtime owner selection, IPC, observability, system proxy
ownership, and platform diagnostics.

`qkbox` is the user-facing entry point. `qkbox-window` is a private helper
spawned by `qkbox` when the user opens the window; direct launch is unsupported
behavior. `qkbox-provider` is the privileged helper used for machine-network
features.

## Workspace

```text
apps/desktop
  Wails 3 qkbox-window helper and Svelte frontend

cmd/qkbox
  user-facing tray, IPC server, and runtime coordinator entry point

cmd/qkbox-provider
  privileged provider executable entry point

core/qkboxd
  qkbox service, RuntimeOwner state machine, and IPC handlers

internal/ipc
  local framed JSON IPC and platform transport

platform
  platform capability boundary and native providers

shared/api
  public qkbox API DTOs and structured errors

shared/model
  public qkbox product models

packaging
  installer and package assets; only qkbox is exposed as a user entry point
```

## Local Commands

```powershell
npm run frontend:install
npm run check
npm run build
```

`npm run build` produces `qkbox`, `qkbox-window`, and `qkbox-provider`
artifacts under `bin/` (with `.exe` suffixes on Windows).
