# Milestone 2: Frontend Shell And State Architecture

## Objective

Turn the single-file frontend into a maintainable application shell with domain
stores and reusable components. This milestone should not add major product
features; it makes later UI work safe.

## Decisions

- Keep Svelte 5, TypeScript, Vite, npm, and current Wails bindings.
- Use Svelte runes in domain state modules.
- Keep all backend calls behind a typed frontend API client.
- Components consume domain stores; they do not directly own qkboxd truth.
- Wails events remain the bridge from qkboxd runtime events to frontend state.

## Target Shape

```text
apps/desktop/frontend/src/
  App.svelte                    shell and route outlet
  main.ts
  styles/
    global.css
    variables.css
  lib/
    api/client.ts               typed bridge wrapper
    routing.ts                  hash route state
    format.ts                   display formatting helpers
    state/
      engine.svelte.ts
      runtime-events.svelte.ts
      profile.svelte.ts
      asset.svelte.ts
      platform.svelte.ts
      diagnostics.svelte.ts
  components/
    layout/
    shared/
    engine/
    platform/
    diagnostics/
    profile/
  views/
    EngineView.svelte
    ProfilesView.svelte
    PlatformView.svelte
    DiagnosticsView.svelte
```

## Work Items

### 1. API Client

Create `lib/api/client.ts` with domain groups:

- `api.engine`
- `api.profile`
- `api.asset`
- `api.platform`
- `api.diagnostics`
- `api.events`

The API client should translate Wails binding result wrappers into a consistent
frontend error shape while preserving structured error fields.

### 2. Runtime Event Store

Create a single runtime event bridge store:

- starts Wails event listeners.
- calls `StartRuntimeEventBridge`.
- keeps bounded log history.
- stores latest traffic and connection snapshots.
- forwards status events into engine state.
- preserves backend event bridge errors instead of hiding them with refreshes.

### 3. Domain Stores

Create stores for:

- engine status, capabilities, groups, current reload result.
- profiles, selected profile, active profile, active snapshot, snapshots.
- subscriptions and data assets.
- platform capabilities, provider status, system proxy status.
- diagnostics report and debug bundle result.

Stores should own refresh/load actions and expose derived UI state. Components
should remain mostly presentational.

### 4. Shell And Routing

Use hash routes:

- `#engine`
- `#profiles`
- `#platform`
- `#diagnostics`

`App.svelte` should only mount the event bridge, initialize primary stores, and
render the shell.

### 5. Shared Components

Create small reusable components:

- `StateBadge`
- `ErrorNotice`
- `MetricCard`
- `EmptyState`
- `IconButton`
- `Toolbar`
- `ConfirmDialog`

Avoid nested cards and avoid marketing-style layout. This is an operational
desktop client; dense, predictable, scannable UI wins.

## Out Of Scope

- Full profile editor behavior.
- Full engine dashboard behavior.
- Provider installation flow.
- New backend APIs.

## Verification

- `npm run check`
- `npm run build`
- `go test -tags with_clash_api ./...`
- `git diff --check`
- Engine start/stop still works through the new component tree.
- Platform proxy errors remain visible.
- Runtime events update stores without polling-derived fake state.

## Acceptance Criteria

- `App.svelte` is a shell, not the product implementation.
- Each state module has one domain responsibility.
- Components do not call Wails bindings directly except through the API client.
- Existing tabs and visible capabilities remain available after the split.

