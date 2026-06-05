# Milestone 4: Engine Observability UX

## Objective

Turn existing runtime lifecycle and observability APIs into a mature engine
workspace: lifecycle, reload, logs, traffic, connections, outbound groups,
selector actions, URLTest, and connection close actions.

## Decisions

- GUI consumes only qkbox APIs and Wails events.
- No Clash HTTP controller is exposed or depended on.
- No fake counters, fake rows, or polling-derived stream semantics.
- Engine truth comes from qkboxd runtime events and engine status.

## Work Items

### 1. Engine Control Panel

Implement:

- state badge.
- start, stop, reload.
- active snapshot display.
- reload target picker.
- reload result display.
- uptime.
- last error with structured detail.

### 2. Traffic Panel

Implement:

- upload/download totals.
- upload/download rates.
- offline/unavailable state.
- waiting-for-source state.
- compact history chart if it can be implemented without a chart dependency.

### 3. Connections Panel

Implement:

- active connection table.
- host/destination display.
- network, inbound, outbound, rule, process.
- upload/download totals.
- filter and sort.
- close connection.
- close all connections.

### 4. Logs Panel

Implement:

- bounded log list.
- level filter.
- source filter.
- text search.
- auto-scroll control.
- clear local view without clearing daemon ring buffer.

### 5. Outbound Groups Panel

Implement:

- group cards or dense list.
- selected outbound.
- selector outbound switch.
- URLTest action and result display.
- failed latency result display.

### 6. Event Lifecycle

Ensure event bridge lifecycle is stable:

- one frontend bridge session at a time.
- event listeners are disposed on app shutdown.
- backend bridge errors stay visible.
- late subscribers receive expected replay.

## Out Of Scope

- Profile editor changes.
- Platform provider installation.
- Rule editor.
- New runtime data persistence.

## Verification

- `npm run check`
- `npm run build`
- `go test -tags with_clash_api ./...`
- Start engine with a usable profile.
- Observe live status/log/traffic/connection events.
- Select outbound in selector group.
- Run URLTest in URLTest group.
- Close individual and all connections.
- Confirm unavailable states do not show fake data.

## Acceptance Criteria

- Engine page is useful for daily operation, not only diagnostics.
- Runtime events drive UI state without hidden polling as a substitute for
  subscriptions.
- Structured runtime/provider/platform errors remain visible.
- qkbox product API remains the only GUI control plane.

