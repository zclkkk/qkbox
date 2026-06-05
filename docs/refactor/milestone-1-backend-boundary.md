# Milestone 1: Backend Product Boundary Hardening

## Objective

Reshape qkboxd into explicit product domains while preserving the current API
surface and runtime behavior. This milestone is about architecture, not new
features.

## Decisions

- Split by domain service, not by file convenience.
- IPC handlers are thin adapters. Business logic belongs in domain services.
- Keep qkboxd IPC and provider IPC as separate product wrappers.
- Extract shared IPC frame/transport/registry foundations only where it removes
  real duplication without erasing product semantics.
- Keep the rich `StructuredError` shape.
- Keep qkbox-owned observability and the existing RuntimeOwner model.

## Target Shape

```text
core/qkboxd/
  service.go                 composition, Hello, Close
  wire.go                    IPC method and subscription registration
  profile_service.go         profile CRUD and encrypted draft content
  snapshot_service.go        validation, snapshots, activation, rollback
  asset_service.go           subscriptions and data assets
  runtime_service.go         engine lifecycle, reload, observability calls
  platform_service.go        capabilities, provider status, system proxy
  diagnostics_service.go     diagnostics report and debug bundle

internal/ipcframework/
  frame.go                   shared frame codec
  transport.go               shared transport interfaces
  transport_unix.go
  transport_windows.go
  registry.go                typed method/subscription registration helpers
  auth.go                    constant-time token auth helper

internal/ipc/
  qkboxd product client/server wrappers

internal/provideripc/
  provider product client/server wrappers and provider DTOs

internal/eventhub/
  shared status/log/event replay hub
```

## Work Items

### 1. Domain Service Split

Move logic out of `core/qkboxd/service.go` into domain services.

`Service` should keep shared dependencies and compose domain services. It should
not become a large method bag again.

Domain services:

- `ProfileService`: profile CRUD, encrypted draft read/write helpers.
- `SnapshotService`: validation, snapshot create/activate/list/rollback,
  required runtime capability extraction.
- `AssetService`: profile subscriptions, data assets, remote fetch limits,
  content-addressed cache coordination.
- `RuntimeService`: engine start/stop/reload/status, runtime observability,
  reload rollback path, runtime target loading.
- `PlatformService`: dynamic platform capabilities, provider status, repair
  actions, system proxy snapshot/restore ownership.
- `DiagnosticsService`: product diagnostics report, redaction, debug bundle.

`wire.go` registers all methods and subscriptions by calling these services.

### 2. Preserve Correct Cross-Domain Ownership

Do not split away the invariants that currently prevent drift:

- Snapshot mutation remains blocked while runtime is running.
- Runtime start/reload target and rollback target both go through the same
  capability preparation path.
- System proxy cleanup still happens before engine stop when qkbox owns the
  system proxy.
- Daemon close still performs best-effort system proxy restore before runtime
  shutdown.

### 3. Shared IPC Foundation

Extract common frame, transport, auth, and registration helpers into
`internal/ipcframework`.

Do not erase the product wrappers:

- `internal/ipc` remains the qkboxd product IPC.
- `internal/provideripc` remains the provider product IPC.

If multiplexing is implemented in this milestone, the protocol must use explicit
frame kinds:

```text
auth
request
response
event
cancel
```

Do not use implicit "empty request means cancel" semantics.

### 4. Unified Event Hub

Extract the duplicate qkboxd/provider event hub shape into `internal/eventhub`.

Required behavior:

- status subscribers receive the latest status immediately.
- log subscribers receive the fixed-size ring buffer replay.
- bridge/provider errors are publishable without coupling to qkboxd internals.
- slow subscribers are bounded and cannot block publishers forever.

### 5. Keep Rich Errors

Do not simplify `StructuredError` to only `code` and `message`.

Instead, normalize usage:

- `Code` is stable and contract-tested.
- `Message` is concise.
- `Source` identifies qkboxd, provider, platform, or singboxadapter.
- `Recoverable` describes whether retry/repair is meaningful.
- `UserAction` is populated for errors that should drive a UI action.
- `Detail` is typed where practical, especially for validation diagnostics and
  provider owner state.

## Out Of Scope

- New profile editor UI.
- New TUN setup UI.
- New packaging behavior.
- Replacing qkbox-owned observability with Clash HTTP controller.
- Removing current API methods.

## Verification

- `go test -tags with_clash_api ./...`
- `go vet -tags with_clash_api ./...`
- `npm run check`
- `git diff --check`
- Architecture test still prevents SagerNet imports outside
  `internal/singboxadapter`.
- Contract tests still pass for all public DTOs and error shapes.
- Regression tests cover profile CRUD, snapshot lifecycle, reload rollback,
  system proxy ownership, provider status, and event subscriptions.

## Acceptance Criteria

- `core/qkboxd/service.go` is a small composition root, not a product logic file.
- Domain services are independently reviewable and do not recreate a single
  large hidden dependency graph.
- IPC code has less duplication, while qkboxd and provider wrappers remain
  product-specific.
- No existing runtime, system proxy, provider, subscription, or diagnostics
  capability regresses.

