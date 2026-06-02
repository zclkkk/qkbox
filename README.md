# qkbox

qkbox is a Windows, macOS, and Linux desktop GUI client built around:

```text
Wails GUI + user-scope qkboxd + embedded sing-box core + Platform Capability Boundary
```

The current code implements Phase 0 and Phase 1 only. It provides the repository baseline, Wails desktop shell, user-scope `qkboxd` command, local IPC transport, `Hello` handshake, capability shell, and local verification commands.

## Workspace

```text
apps/desktop
  Wails 3 desktop app and Svelte frontend

cmd/qkboxd
  user-scope qkboxd executable entry point

core/qkboxd
  qkboxd service and IPC server

internal/ipc
  local framed JSON IPC and platform transport

platform
  platform capability boundary placeholders

shared/api
  public qkbox API DTOs and structured errors

shared/model
  public qkbox product models

packaging
  packaging placeholders outside Wails build assets
```

## Local Commands

```powershell
npm run frontend:install
npm run check
npm run build
```

`npm run build` builds `bin/qkboxd.exe` first, then builds the Wails desktop app.

## Phase Boundaries

Phase 0/1 do not implement runtime start, profile persistence, privileged helpers, TUN, route, DNS, system proxy, or sing-box integration.
