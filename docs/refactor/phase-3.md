# Phase 3: Frontend Architecture

> Depends on Phase 1 (handler split gives clean API surface).
> Can run in parallel with Phase 2.

## Problem

`apps/desktop/frontend/src/App.svelte` is 800 lines with 20+ `$state` variables, all API calls, event subscriptions, UI rendering, and formatting utilities in a single file. No component decomposition, no routing, no state management layer.

## Action

### 3.1 Directory Structure

Replace the single file with:

```
apps/desktop/frontend/src/
  main.ts
  App.svelte                          — Shell: sidebar + router outlet
  styles/
    global.css                        — Refactored from current styles.css
    variables.css                     — CSS custom properties (colors, spacing)

  lib/
    api/
      client.ts                       — Typed wrapper around BridgeService
      types.ts                        — TS interfaces matching shared/api types
    state/
      engine.svelte.ts                — Engine status, capabilities, groups
      profile.svelte.ts               — Profile list, active profile
      asset.svelte.ts                 — Subscriptions + data assets
      platform.svelte.ts              — Capabilities, provider status, proxy
      events.svelte.ts                — Runtime event bridge (log, traffic, connections)
    routing.ts                        — Hash-based router
    format.ts                         — formatBytes, formatDuration, formatTimestamp

  components/
    layout/
      Sidebar.svelte                  — Navigation with route links
      Topbar.svelte                   — Header: title, refresh button, status indicator
    shared/
      ErrorNotice.svelte              — Structured error display
      StateBadge.svelte               — Color-coded state indicator
      MetricCard.svelte               — Label + value metric display
      EmptyState.svelte               — "No data" placeholder
      LoadingSpinner.svelte           — Loading indicator
    engine/
      EnginePanel.svelte              — Start/Stop/Reload, state badge, error
      TrafficPanel.svelte             — Upload/download rates + totals
      ConnectionPanel.svelte          — Active connections table
      GroupPanel.svelte               — Outbound group cards
      LogPanel.svelte                 — Log entries list
    platform/
      ProviderPanel.svelte            — Privileged provider status
      ProxyPanel.svelte               — System proxy toggle
      CapabilityList.svelte           — Capability badge list
    diagnostics/
      DiagnosticsPanel.svelte         — Diagnostics report
      CheckList.svelte                — Diagnostic check items
      AssetSummary.svelte             — Subscription + data asset overview
    profile/
      ProfileList.svelte              — Profile CRUD list (Phase 4)
      ProfileEditor.svelte            — JSON editor (Phase 4)
      SubscriptionPanel.svelte        — Subscription management (Phase 4)
      DataAssetPanel.svelte           — Data asset management (Phase 4)

  views/
    EngineView.svelte                 — Assembles engine components
    PlatformView.svelte               — Assembles platform components
    DiagnosticsView.svelte            — Assembles diagnostics components
    ProfilesView.svelte               — Profile management (Phase 4)
```

### 3.2 State Management

Svelte 5 runes (`$state`, `$derived`) in `.svelte.ts` modules. Each domain is a class with reactive state + async actions. Exported as singletons.

#### `lib/state/engine.svelte.ts`

```typescript
class EngineState {
    status = $state<EngineStatus | null>(null);
    error = $state<string | null>(null);
    capabilities = $state<Capability[]>([]);
    groups = $state<OutboundGroup[]>([]);

    get isStarted() { return this.status?.state === 'STARTED'; }
    get isBusy() { return this.status?.state === 'STARTING' || this.status?.state === 'STOPPING'; }
    get isIdle() { return this.status?.state === 'IDLE'; }

    async refresh(): Promise<void> { ... }
    async start(): Promise<void> { ... }
    async stop(): Promise<void> { ... }
    async reload(snapshotId: string): Promise<void> { ... }
    async refreshGroups(): Promise<void> { ... }
    async selectOutbound(groupTag: string, outboundTag: string): Promise<void> { ... }
    async urlTest(groupTag: string): Promise<void> { ... }
    async closeAllConnections(): Promise<void> { ... }
}

export const engineState = new EngineState();
```

#### `lib/state/events.svelte.ts`

```typescript
class RuntimeEvents {
    logs = $state<RuntimeLogEntry[]>([]);
    traffic = $state<TrafficSnapshot | null>(null);
    connections = $state<ConnectionSnapshot | null>(null);

    private disposers: (() => void)[] = [];

    start(): void {
        // Register Wails event listeners:
        //   qkbox.engine.status  → engineState.status = data
        //   qkbox.engine.log     → this.logs = [...logs.slice(-127), data]
        //   qkbox.engine.traffic → this.traffic = data
        //   qkbox.engine.connections → this.connections = data
        //   qkbox.engine.eventBridgeError → engineState.error = ...
        // Then call BridgeService.StartRuntimeEventBridge()
    }

    stop(): void {
        this.disposers.forEach(d => d());
        BridgeService.StopRuntimeEventBridge();
    }
}

export const runtimeEvents = new RuntimeEvents();
```

#### `lib/state/profile.svelte.ts`

```typescript
class ProfileState {
    profiles = $state<ProfileSummary[]>([]);
    activeProfile = $state<ProfileSummary | null>(null);
    activeSnapshot = $state<SnapshotSummary | null>(null);
    selectedProfile = $state<ProfileSummary | null>(null);
    selectedProfileContent = $state<string>('');
    snapshots = $state<SnapshotSummary[]>([]);
    error = $state<string | null>(null);

    async refresh(): Promise<void> { ... }
    async create(name: string, content: string): Promise<void> { ... }
    async updateDraft(profileId: string, content: string): Promise<void> { ... }
    async delete(profileId: string): Promise<void> { ... }
    async select(profileId: string): Promise<void> { ... }
    async validate(profileId: string): Promise<Diagnostics> { ... }
    async createSnapshot(profileId: string): Promise<void> { ... }
    async activateSnapshot(snapshotId: string): Promise<void> { ... }
    async rollback(snapshotId: string): Promise<void> { ... }
}

export const profileState = new ProfileState();
```

#### `lib/state/platform.svelte.ts`

```typescript
class PlatformState {
    capabilities = $state<Capability[]>([]);
    providerStatus = $state<PrivilegedProviderStatus | null>(null);
    proxyStatus = $state<GetSystemProxyStatusReply | null>(null);
    error = $state<string | null>(null);

    async refresh(): Promise<void> { ... }
    async refreshProvider(): Promise<void> { ... }
    async refreshProxy(): Promise<void> { ... }
    async toggleProxy(): Promise<void> { ... }
    async runRepair(action: string): Promise<void> { ... }
}

export const platformState = new PlatformState();
```

### 3.3 API Client Layer

`lib/api/client.ts` — typed wrapper organized by domain:

```typescript
import { BridgeService } from '../../bindings/github.com/zclkkk/qkbox/apps/desktop';

export const api = {
    hello: () => BridgeService.Hello(),

    engine: {
        start: () => BridgeService.EngineStart(),
        stop: () => BridgeService.EngineStop(),
        reload: (req: EngineReloadRequest) => BridgeService.EngineReload(req),
        getStatus: () => BridgeService.EngineGetStatus(),
        getCapabilities: () => BridgeService.EngineGetRuntimeCapabilities(),
        listGroups: () => BridgeService.EngineListGroups(),
        selectOutbound: (req: EngineSelectOutboundRequest) => BridgeService.EngineSelectOutbound(req),
        urlTest: (req: EngineURLTestRequest) => BridgeService.EngineURLTest(req),
        closeConnection: (req: EngineCloseConnectionRequest) => BridgeService.EngineCloseConnection(req),
        closeAllConnections: () => BridgeService.EngineCloseAllConnections(),
    },

    profile: {
        create: (req: CreateProfileRequest) => BridgeService.CreateProfile(req),
        updateDraft: (req: UpdateProfileDraftRequest) => BridgeService.UpdateProfileDraft(req),
        delete: (req: DeleteProfileRequest) => BridgeService.DeleteProfile(req),
        list: () => BridgeService.ListProfiles({} as any),
        get: (req: GetProfileRequest) => BridgeService.GetProfile(req),
        getActive: () => BridgeService.GetActiveProfile({} as any),
        getActiveSnapshot: () => BridgeService.GetActiveSnapshot({} as any),
        validate: (req: ValidateProfileDraftRequest) => BridgeService.ValidateProfileDraft(req),
        getDiagnostics: (req: GetProfileDiagnosticsRequest) => BridgeService.GetProfileDiagnostics(req),
        createSnapshot: (req: CreateProfileSnapshotRequest) => BridgeService.CreateProfileSnapshot(req),
        activateSnapshot: (req: ActivateProfileSnapshotRequest) => BridgeService.ActivateProfileSnapshot(req),
        listSnapshots: (req: ListSnapshotsRequest) => BridgeService.ListSnapshots(req),
        rollback: (req: RollbackToSnapshotRequest) => BridgeService.RollbackToSnapshot(req),
    },

    asset: {
        createSubscription: (req: CreateProfileSubscriptionRequest) => BridgeService.AssetCreateProfileSubscription(req),
        listSubscriptions: (req: ListProfileSubscriptionsRequest) => BridgeService.AssetListProfileSubscriptions(req),
        refreshSubscription: (req: RefreshProfileSubscriptionRequest) => BridgeService.AssetRefreshProfileSubscription(req),
        deleteSubscription: (req: DeleteProfileSubscriptionRequest) => BridgeService.AssetDeleteProfileSubscription(req),
        createAsset: (req: CreateDataAssetRequest) => BridgeService.AssetCreateDataAsset(req),
        listAssets: (req: ListDataAssetsRequest) => BridgeService.AssetListDataAssets(req),
        refreshAsset: (req: RefreshDataAssetRequest) => BridgeService.AssetRefreshDataAsset(req),
        deleteAsset: (req: DeleteDataAssetRequest) => BridgeService.AssetDeleteDataAsset(req),
    },

    platform: {
        getCapabilities: () => BridgeService.PlatformGetCapabilities(),
        getProviderStatus: () => BridgeService.PlatformGetPrivilegedProviderStatus(),
        prepareFeature: (req: PrepareFeatureRequest) => BridgeService.PlatformPrepareFeature(req),
        runRepair: (req: RunRepairActionRequest) => BridgeService.PlatformRunRepairAction(req),
        getProxyStatus: () => BridgeService.PlatformGetSystemProxyStatus(),
        setProxyEnabled: (req: SetSystemProxyEnabledRequest) => BridgeService.PlatformSetSystemProxyEnabled(req),
    },

    diagnostics: {
        getReport: () => BridgeService.DiagnosticsGetReport(),
        createDebugBundle: () => BridgeService.DiagnosticsCreateDebugBundle(),
    },

    events: {
        startBridge: () => BridgeService.StartRuntimeEventBridge(),
        stopBridge: () => BridgeService.StopRuntimeEventBridge(),
    },
};
```

### 3.4 Routing

`lib/routing.ts`:

```typescript
export type Route = 'engine' | 'profiles' | 'platform' | 'diagnostics';

export function currentRoute(): Route {
    const hash = window.location.hash.slice(1) as Route;
    if (['engine', 'profiles', 'platform', 'diagnostics'].includes(hash)) return hash;
    return 'engine';
}

export function navigate(route: Route) {
    window.location.hash = route;
}
```

`App.svelte` (target: < 60 lines):

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import { currentRoute, type Route } from './lib/routing';
  import Sidebar from './components/layout/Sidebar.svelte';
  import Topbar from './components/layout/Topbar.svelte';
  import EngineView from './views/EngineView.svelte';
  import PlatformView from './views/PlatformView.svelte';
  import DiagnosticsView from './views/DiagnosticsView.svelte';
  import ProfilesView from './views/ProfilesView.svelte';
  import { runtimeEvents } from './lib/state/events.svelte';
  import { engineState } from './lib/state/engine.svelte';

  let route = $state<Route>(currentRoute());

  onMount(() => {
    const onHashChange = () => { route = currentRoute(); };
    window.addEventListener('hashchange', onHashChange);
    runtimeEvents.start();
    engineState.refresh();
    return () => {
      window.removeEventListener('hashchange', onHashChange);
      runtimeEvents.stop();
    };
  });
</script>

<div class="shell">
  <Sidebar active={route} />
  <section class="content">
    <Topbar />
    {#if route === 'engine'}
      <EngineView />
    {:else if route === 'profiles'}
      <ProfilesView />
    {:else if route === 'platform'}
      <PlatformView />
    {:else if route === 'diagnostics'}
      <DiagnosticsView />
    {/if}
  </section>
</div>
```

### 3.5 Shared Components

#### `components/shared/StateBadge.svelte`

Replaces the inline `data-state` pattern used everywhere:

```svelte
<script lang="ts">
  let { state, label }: { state: string; label?: string } = $props();
</script>

<span class="state" data-state={state}>{label ?? state}</span>
```

#### `components/shared/ErrorNotice.svelte`

```svelte
<script lang="ts">
  import { CircleAlert } from '@lucide/svelte';
  let { error }: { error: string | null } = $props();
</script>

{#if error}
  <section class="notice error">
    <CircleAlert size={18} />
    <span>{error}</span>
  </section>
{/if}
```

#### `components/shared/MetricCard.svelte`

```svelte
<script lang="ts">
  let { label, value }: { label: string; value: string } = $props();
</script>

<div>
  <span class="label">{label}</span>
  <strong>{value}</strong>
</div>
```

### 3.6 Format Utilities

`lib/format.ts`:

```typescript
export function formatBytes(value: number): string {
    if (value < 1024) return `${value} B`;
    if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KiB`;
    return `${(value / 1024 / 1024).toFixed(1)} MiB`;
}

export function formatDuration(ms: number): string { ... }
export function formatTimestamp(ts: number): string { ... }
export function formatError(err: { code: string; message: string }): string {
    return `${err.code}: ${err.message}`;
}
```

## Verification

- [ ] `App.svelte` is < 60 lines
- [ ] Each component is < 200 lines
- [ ] Each state module is < 100 lines
- [ ] `api/client.ts` provides typed access to all IPC methods
- [ ] Navigation works via URL hash (#engine, #profiles, #platform, #diagnostics)
- [ ] Engine start/stop works through the new component tree
- [ ] Real-time events (status, log, traffic, connections) propagate to all components
- [ ] `npm run check` passes (svelte-check)
- [ ] `npm run build` produces working frontend
