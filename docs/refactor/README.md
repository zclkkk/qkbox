# qkbox Refactoring Plan

> Full refactoring from prototype to production-grade architecture.
> All phases must leave the project in a working, testable state.

## Decisions

| Decision | Choice | Rationale |
|---|---|---|
| GUI framework | Wails 3 (keep) | Go-native, WebView, fits current stack |
| Frontend | Svelte 5 (keep) | Lightweight, compile-time, runes are clean |
| IPC protocol | Optimize current frame protocol | Zero new deps, same wire format, persistent multiplexed connections |
| Scope | Full refactoring | Architecture + all features + UI, no compromises |

### IPC Protocol Rationale

| Option | Verdict |
|---|---|
| **Optimize current frame protocol** | Chosen. 4-byte length-prefix + JSON is already clean. Add persistent connections with request ID routing. Zero dependencies. |
| gRPC | Rejected. Overkill for local IPC. Protobuf + heavy dep tree buys nothing when both ends are Go and co-released. |
| JSON-RPC 2.0 | Rejected. Text-only, no binary streaming. Marginal standardization benefit not worth a new dependency. |

## Phase Dependency Graph

```
Phase 1 (Backend Architecture)  ← foundation, do first
  |
  +---> Phase 2 (sing-box Deepening)   ← depends on unified IPC
  |
  +---> Phase 3 (Frontend Architecture) ← depends on handler split
          |
          +---> Phase 4 (Profile UI)
          |
          +---> Phase 5 (Engine UI)
          |       |
          |       +---> Phase 6 (Platform Features)
          |       |
          |       +---> Phase 7 (Provider & TUN) ← also depends on Phase 2
          |
          +---> Phase 8 (Packaging) ← depends on all above
```

Phase 1 first. Phases 2+3 can proceed in parallel after Phase 1. Phases 4+5 depend on 3. Phase 6+7 depend on 5. Phase 8 last.

## Phases

| Phase | Title | Status | Plan |
|---|---|---|---|
| 1 | Backend Architecture Refactoring | Not started | [phase-1.md](phase-1.md) |
| 2 | sing-box Integration Deepening | Not started | [phase-2.md](phase-2.md) |
| 3 | Frontend Architecture | Not started | [phase-3.md](phase-3.md) |
| 4 | Profile & Configuration Management UI | Not started | [phase-4.md](phase-4.md) |
| 5 | Engine & Observability UI | Not started | [phase-5.md](phase-5.md) |
| 6 | Platform Features | Not started | [phase-6.md](phase-6.md) |
| 7 | Provider & TUN Mode | Not started | [phase-7.md](phase-7.md) |
| 8 | Packaging & Distribution | Not started | [phase-8.md](phase-8.md) |

## Post-Refactoring Package Structure

```
github.com/zclkkk/qkbox/
  cmd/
    qkboxd/main.go                          -- unchanged
    qkbox-provider/main.go                  -- unchanged
  apps/
    desktop/
      main.go                               -- tray + autostart integration added
      bridge.go                             -- uses ipcframework.Client
      bridge_unix.go                        -- unchanged
      bridge_windows.go                     -- unchanged
      frontend/src/                         -- fully decomposed (Phase 3+)
  core/
    qkboxd/
      daemon.go                             -- simplified, uses ipcframework
      service.go                            -- trimmed to ~30 lines (Hello + Close)
      handler_profile.go                    -- NEW
      handler_asset.go                      -- NEW
      handler_snapshot.go                   -- NEW
      handler_engine.go                     -- NEW
      handler_platform.go                   -- NEW
      handler_diagnostic.go                 -- NEW
      wire.go                               -- NEW: assembles handler registry
      engine.go                             -- uses eventhub.Hub
      runtime_owner.go                      -- uses eventhub.Hub
      provider_runtime_owner.go             -- uses eventhub.Hub
      proxy_owner.go                        -- unchanged
      lock.go / lock_unix.go / lock_windows.go -- unchanged
      runtime_platform.go                   -- unchanged
  internal/
    ipcframework/                           -- NEW (replaces ipc/ + provideripc/)
      frame.go, transport.go, transport_unix.go, transport_windows.go,
      conn.go, server.go, client.go, auth.go, subscription.go, errors.go
    eventhub/                               -- NEW (replaces two event hubs)
      eventhub.go
    singboxadapter/
      adapter.go                            -- modified: uses ClashBridge
      clash_bridge.go                       -- NEW
      validate.go                           -- NEW
      errors.go                             -- unchanged
    configtemplate/                         -- NEW
      templates.go, render.go, builtin.go
    tray/                                   -- NEW
    autostart/                              -- NEW
    updater/                                -- NEW
    crypto/, persistence/, redact/, runtimeapi/, assetcache/,
    providerruntime/                        -- events.go deleted, rest unchanged
  platform/
    capability/                             -- TUN detection added
  shared/
    api/                                    -- simplified StructuredError, simplified version
    model/                                  -- unchanged
```

**Deleted:** `internal/ipc/`, `internal/provideripc/`, `internal/singboxadapter/observability.go`
