<script lang="ts">
  import { onMount } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { Activity, CircleAlert, Cpu, PlugZap, RefreshCw, ShieldCheck } from "@lucide/svelte";
  import { BridgeService } from "../bindings/github.com/zclkkk/qkbox/apps/desktop";
  import {
    type Capability,
    type EngineStatus,
    type GetSystemProxyStatusReply,
    type HelloReply,
    type OutboundGroup,
    type PrivilegedProviderStatus
  } from "../bindings/github.com/zclkkk/qkbox/shared/api/models";

  type View = "engine" | "platform" | "diagnostics";
  type RuntimeLogEntry = { seq: number; timestamp: number; source: string; level: string; message: string };
  type TrafficSnapshot = { timestamp: number; upload_total: number; download_total: number; upload_rate: number; download_rate: number };
  type RuntimeConnection = {
    id: string;
    network: string;
    source: string;
    destination: string;
    host?: string;
    outbound?: string;
    upload: number;
    download: number;
  };
  type ConnectionSnapshot = { timestamp: number; upload_total: number; download_total: number; connections: RuntimeConnection[] };
  type ProfileSubscription = {
    id: string;
    profile_id: string;
    name: string;
    url: string;
    update_policy: string;
    last_status: string;
    last_error_code?: string;
    last_error_message?: string;
    last_checked_at?: number;
    last_updated_at?: number;
    content_sha256?: string;
  };
  type DataAsset = {
    id: string;
    kind: string;
    name: string;
    source_url: string;
    status: string;
    version?: string;
    content_sha256?: string;
    size_bytes?: number;
    last_error_code?: string;
    last_error_message?: string;
  };
  type DiagnosticCheck = {
    name: string;
    state: string;
    reason?: string;
    recovery?: string;
  };
  type ProductDiagnosticsReport = {
    generated_at: number;
    db_schema_version: number;
    engine_status: EngineStatus;
    checks: DiagnosticCheck[];
  };

  let loading = $state(true);
  let reply = $state<HelloReply | null>(null);
  let error = $state<string | null>(null);
  let activeView = $state<View>("engine");
  let lastChecked = $state<string>("Never");
  let engineStatus = $state<EngineStatus | null>(null);
  let engineError = $state<string | null>(null);
  let runtimeCapabilities = $state<Capability[]>([]);
  let platformCapabilities = $state<Capability[]>([]);
  let logs = $state<RuntimeLogEntry[]>([]);
  let traffic = $state<TrafficSnapshot | null>(null);
  let connections = $state<ConnectionSnapshot | null>(null);
  let groups = $state<OutboundGroup[]>([]);
  let proxyStatus = $state<GetSystemProxyStatusReply | null>(null);
  let proxyError = $state<string | null>(null);
  let proxyLoading = $state(false);
  let privilegedProviderStatus = $state<PrivilegedProviderStatus | null>(null);
  let platformError = $state<string | null>(null);
  let subscriptions = $state<ProfileSubscription[]>([]);
  let dataAssets = $state<DataAsset[]>([]);
  let assetError = $state<string | null>(null);
  let diagnosticsReport = $state<ProductDiagnosticsReport | null>(null);
  let diagnosticsError = $state<string | null>(null);
  let debugBundlePath = $state<string | null>(null);

  function formatStructuredError(err: { code: string; message: string }) {
    return `${err.code}: ${err.message}`;
  }

  function formatBytes(value: number) {
    if (value < 1024) {
      return `${value} B`;
    }
    if (value < 1024 * 1024) {
      return `${(value / 1024).toFixed(1)} KiB`;
    }
    return `${(value / 1024 / 1024).toFixed(1)} MiB`;
  }

  function engineStarted() {
    return engineStatus?.state === "STARTED";
  }

  async function refreshEngineStatus() {
    const stResult = await BridgeService.EngineGetStatus();
    if (stResult.error) {
      engineError = formatStructuredError(stResult.error);
      return;
    }
    if (stResult.reply) {
      engineStatus = stResult.reply.status;
    }
  }

  async function refreshRuntimeCapabilities() {
    const result = await BridgeService.EngineGetRuntimeCapabilities();
    if (result.error) {
      engineError = formatStructuredError(result.error);
      return;
    }
    runtimeCapabilities = result.reply?.capabilities ?? [];
  }

  async function refreshPlatformCapabilities() {
    const result = await BridgeService.PlatformGetCapabilities();
    if (result.error) {
      platformError = formatStructuredError(result.error);
      return;
    }
    platformCapabilities = result.reply?.capabilities ?? [];
  }

  async function refreshPrivilegedProviderStatus() {
    const result = await BridgeService.PlatformGetPrivilegedProviderStatus();
    if (result.error) {
      platformError = formatStructuredError(result.error);
      privilegedProviderStatus = null;
      return;
    }
    privilegedProviderStatus = result.reply?.status ?? null;
  }

  async function runProviderRepair(action: string) {
    platformError = null;
    const result = await BridgeService.PlatformRunRepairAction({ action });
    if (result.error) {
      platformError = formatStructuredError(result.error);
      return;
    }
    await refreshPrivilegedProviderStatus();
    await refreshPlatformCapabilities();
  }

  async function refreshGroups() {
    const result = await BridgeService.EngineListGroups();
    if (result.error) {
      if (result.error.code !== "ENGINE_NOT_STARTED") {
        engineError = formatStructuredError(result.error);
      }
      groups = [];
      return;
    }
    groups = result.reply?.groups ?? [];
  }

  async function refreshAssetState() {
    assetError = null;
    const subResult = await BridgeService.AssetListProfileSubscriptions({ profile_id: "" });
    if (subResult.error) {
      assetError = formatStructuredError(subResult.error);
      subscriptions = [];
    } else {
      subscriptions = (subResult.reply?.subscriptions ?? []) as ProfileSubscription[];
    }
    const assetResult = await BridgeService.AssetListDataAssets({ kind: "" });
    if (assetResult.error) {
      assetError = formatStructuredError(assetResult.error);
      dataAssets = [];
    } else {
      dataAssets = (assetResult.reply?.assets ?? []) as DataAsset[];
    }
  }

  async function refreshDiagnosticsReport() {
    diagnosticsError = null;
    const result = await BridgeService.DiagnosticsGetReport();
    if (result.error) {
      diagnosticsError = formatStructuredError(result.error);
      diagnosticsReport = null;
      return;
    }
    diagnosticsReport = result.reply?.report as ProductDiagnosticsReport;
  }

  async function createDebugBundle() {
    diagnosticsError = null;
    debugBundlePath = null;
    const result = await BridgeService.DiagnosticsCreateDebugBundle();
    if (result.error) {
      diagnosticsError = formatStructuredError(result.error);
      return;
    }
    debugBundlePath = result.reply?.bundle_path ?? null;
    diagnosticsReport = result.reply?.report as ProductDiagnosticsReport;
  }

  async function bootstrap() {
    loading = true;
    error = null;
    try {
      const result = await BridgeService.Hello();
      if (result.error) {
        error = `${result.error.code}: ${result.error.message}`;
        reply = null;
      } else {
        reply = result.reply as HelloReply;
        runtimeCapabilities = reply.runtime_capabilities;
        platformCapabilities = reply.platform_capabilities;
        lastChecked = new Date().toLocaleTimeString();
      }
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      reply = null;
    }

    try {
      await refreshEngineStatus();
      await refreshRuntimeCapabilities();
      await refreshGroups();
      await refreshPlatformCapabilities();
      await refreshPrivilegedProviderStatus();
      await refreshProxyStatus();
      await refreshAssetState();
      await refreshDiagnosticsReport();
    } catch (e) {
      engineError = e instanceof Error ? e.message : String(e);
    }

    loading = false;
  }

  async function startEngine() {
    loading = true;
    engineError = null;
    try {
      const result = await BridgeService.EngineStart();
      if (result.error) {
        engineError = formatStructuredError(result.error);
      }
      await refreshEngineStatus();
      await refreshRuntimeCapabilities();
      await refreshGroups();
    } catch (err) {
      engineError = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  async function stopEngine() {
    loading = true;
    engineError = null;
    try {
      const result = await BridgeService.EngineStop();
      if (result.error) {
        engineError = formatStructuredError(result.error);
      }
      await refreshEngineStatus();
      await refreshRuntimeCapabilities();
      await refreshGroups();
    } catch (err) {
      engineError = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  function show(view: View) {
    activeView = view;
  }

  async function selectOutbound(groupTag: string, outboundTag: string) {
    engineError = null;
    const result = await BridgeService.EngineSelectOutbound({ group_tag: groupTag, outbound_tag: outboundTag });
    if (result.error) {
      engineError = formatStructuredError(result.error);
      return;
    }
    await refreshGroups();
  }

  async function urlTest(groupTag: string) {
    engineError = null;
    const result = await BridgeService.EngineURLTest({ group_tag: groupTag, timeout_ms: 10000 });
    if (result.error) {
      engineError = formatStructuredError(result.error);
      return;
    }
    await refreshGroups();
  }

  async function closeAllConnections() {
    engineError = null;
    const result = await BridgeService.EngineCloseAllConnections();
    if (result.error) {
      engineError = formatStructuredError(result.error);
    }
  }

  async function refreshProxyStatus() {
    proxyError = null;
    try {
      const result = await BridgeService.PlatformGetSystemProxyStatus();
      if (result.error) {
        proxyError = formatStructuredError(result.error);
        proxyStatus = null;
        return;
      }
      proxyStatus = result.reply ?? null;
    } catch (err) {
      proxyError = err instanceof Error ? err.message : String(err);
      proxyStatus = null;
    }
  }

  async function toggleProxy() {
    if (!proxyStatus?.available || !proxyStatus?.supported) return;
    proxyLoading = true;
    proxyError = null;
    try {
      const enable = !proxyStatus.qkbox_owned;
      const result = await BridgeService.PlatformSetSystemProxyEnabled({ enabled: enable });
      if (result.error) {
        proxyError = formatStructuredError(result.error);
      }
      await refreshProxyStatus();
    } catch (err) {
      proxyError = err instanceof Error ? err.message : String(err);
    } finally {
      proxyLoading = false;
    }
  }

  onMount(() => {
    void bootstrap();
    const offStatus = Events.On("qkbox.engine.status", (event) => {
      engineStatus = event.data as EngineStatus;
    });
    const offLog = Events.On("qkbox.engine.log", (event) => {
      logs = [...logs.slice(-127), event.data as RuntimeLogEntry];
    });
    const offTraffic = Events.On("qkbox.engine.traffic", (event) => {
      traffic = event.data as TrafficSnapshot;
    });
    const offConnections = Events.On("qkbox.engine.connections", (event) => {
      connections = event.data as ConnectionSnapshot;
    });
    const offBridgeError = Events.On("qkbox.engine.eventBridgeError", (event) => {
      const err = event.data as { code?: string; message?: string };
      if (err?.code && err.code !== "ENGINE_NOT_STARTED" && err.code !== "OBSERVABILITY_UNAVAILABLE") {
        engineError = `${err.code}: ${err.message ?? ""}`;
      }
    });
    void BridgeService.StartRuntimeEventBridge();
    return () => {
      offStatus();
      offLog();
      offTraffic();
      offConnections();
      offBridgeError();
      void BridgeService.StopRuntimeEventBridge();
    };
  });
</script>

<main class="shell">
  <aside class="sidebar">
    <div class="brand">
      <ShieldCheck size={24} strokeWidth={1.8} />
      <span>qkbox</span>
    </div>
    <nav class="nav">
      <button class="nav-item" class:active={activeView === "engine"} aria-pressed={activeView === "engine"} type="button" onclick={() => show("engine")}><Activity size={18} />Engine</button>
      <button class="nav-item" class:active={activeView === "platform"} aria-pressed={activeView === "platform"} type="button" onclick={() => show("platform")}><PlugZap size={18} />Platform</button>
      <button class="nav-item" class:active={activeView === "diagnostics"} aria-pressed={activeView === "diagnostics"} type="button" onclick={() => show("diagnostics")}><Cpu size={18} />Diagnostics</button>
    </nav>
  </aside>

  <section class="content">
    <header class="topbar">
      <div>
        <h1>Control Plane</h1>
        <p>{loading ? "Refreshing qkboxd handshake..." : `Last checked ${lastChecked}`}</p>
      </div>
      <button class="icon-button" type="button" aria-label="Refresh qkboxd handshake" onclick={bootstrap} disabled={loading}>
        <span class:spin={loading}>
          <RefreshCw size={18} />
        </span>
      </button>
    </header>

    {#if error}
      <section class="notice error">
        <CircleAlert size={18} />
        <span>{error}</span>
      </section>
    {:else if reply}
      <section class="summary">
        <div>
          <span class="label">API</span>
          <strong>{reply.api_version}</strong>
        </div>
        <div>
          <span class="label">Schema</span>
          <strong>{reply.schema_revision}</strong>
        </div>
        <div>
          <span class="label">qkboxd</span>
          <strong>{reply.qkboxd_version}</strong>
        </div>
        <div>
          <span class="label">Platform</span>
          <strong>{reply.platform.os}/{reply.platform.arch}</strong>
        </div>
      </section>

      <section class="columns">
        {#if activeView === "engine"}
          <section class="panel">
            <h2>Engine Status</h2>
            <div style="display: flex; gap: 1rem; align-items: center; margin-bottom: 1rem;">
              <span class="state" data-state={engineStatus?.state}>{engineStatus?.state || "UNKNOWN"}</span>
              <button type="button" onclick={startEngine} disabled={loading || engineStatus?.state === "STARTED" || engineStatus?.state === "STARTING" || engineStatus?.state === "STOPPING" || engineStatus?.state === "FATAL"}>Start</button>
              <button type="button" onclick={stopEngine} disabled={loading || engineStatus?.state === "IDLE" || engineStatus?.state === "UNINITIALIZED" || engineStatus?.state === "STARTING" || engineStatus?.state === "STOPPING"}>Stop</button>
            </div>
            {#if engineStatus?.active_snapshot_id}
              <p>Active snapshot {engineStatus.active_snapshot_id}</p>
            {/if}
            {#if engineError}
              <div class="notice error">
                <span>{engineError}</span>
              </div>
            {/if}
            {#if engineStatus?.last_error_message}
              <div class="notice error">
                <strong>{engineStatus.last_error_code}</strong>: {engineStatus.last_error_message}
              </div>
            {/if}
          </section>
          <section class="panel">
            <h2>Traffic</h2>
            {#if engineStarted() && traffic}
              <div class="metrics">
                <div><span class="label">Upload</span><strong>{formatBytes(traffic.upload_total)}</strong></div>
                <div><span class="label">Download</span><strong>{formatBytes(traffic.download_total)}</strong></div>
                <div><span class="label">Up rate</span><strong>{formatBytes(traffic.upload_rate)}/s</strong></div>
                <div><span class="label">Down rate</span><strong>{formatBytes(traffic.download_rate)}/s</strong></div>
              </div>
            {:else if engineStarted()}
              <p class="empty">Waiting for traffic source</p>
            {:else}
              <p class="empty">Traffic source unavailable</p>
            {/if}
          </section>
          <section class="panel">
            <div class="panel-title">
              <h2>Connections</h2>
              <button type="button" onclick={closeAllConnections} disabled={!engineStarted() || !connections?.connections?.length}>Close all</button>
            </div>
            {#if engineStarted() && connections}
              <div class="connection-list">
                {#each connections.connections as connection}
                  <div class="connection-row">
                    <strong>{connection.host || connection.destination || connection.id}</strong>
                    <span>{connection.network} / {connection.outbound || "unknown"} / {formatBytes(connection.upload)}/{formatBytes(connection.download)}</span>
                  </div>
                {:else}
                  <p class="empty">No active connections</p>
                {/each}
              </div>
            {:else if engineStarted()}
              <p class="empty">Waiting for connection source</p>
            {:else}
              <p class="empty">Connection source unavailable</p>
            {/if}
          </section>
          <section class="panel">
            <h2>Outbound Groups</h2>
            {#if engineStarted()}
              <div class="group-list">
                {#each groups as group}
                  <div class="group-row">
                    <div class="panel-title">
                      <strong>{group.tag}</strong>
                      {#if group.type === "urltest"}
                        <button type="button" onclick={() => urlTest(group.tag)}>URLTest</button>
                      {/if}
                    </div>
                    <span class="label">{group.type} / selected {group.selected}</span>
                    <div class="outbound-options">
                      {#each group.outbounds as outbound}
                        {#if group.type === "selector"}
                          <button type="button" class:active={outbound.tag === group.selected} onclick={() => selectOutbound(group.tag, outbound.tag)}>
                            {outbound.tag}
                          </button>
                        {:else}
                          <span class:active={outbound.tag === group.selected}>{outbound.tag}</span>
                        {/if}
                      {/each}
                    </div>
                  </div>
                {:else}
                  <p class="empty">No runtime groups</p>
                {/each}
              </div>
            {:else}
              <p class="empty">Runtime groups unavailable</p>
            {/if}
          </section>
          <section class="panel">
            <h2>Logs</h2>
            <div class="logs">
              {#each logs as entry}
                <div><span>{entry.level}</span>{entry.message}</div>
              {:else}
                <p class="empty">No logs</p>
              {/each}
            </div>
          </section>
          {@render capabilityList("Runtime", runtimeCapabilities)}
        {:else if activeView === "platform"}
          <section class="panel">
            <h2>Privileged Provider</h2>
            {#if privilegedProviderStatus}
              <div class="metrics">
                <div>
                  <span class="label">Installed</span>
                  <span class="state" data-state={privilegedProviderStatus.installed ? "available" : "unavailable"}>
                    {privilegedProviderStatus.installed ? "Yes" : "No"}
                  </span>
                </div>
                <div>
                  <span class="label">Reachable</span>
                  <span class="state" data-state={privilegedProviderStatus.reachable ? "available" : "unavailable"}>
                    {privilegedProviderStatus.reachable ? "Yes" : "No"}
                  </span>
                </div>
                <div>
                  <span class="label">Authenticated</span>
                  <span class="state" data-state={privilegedProviderStatus.authenticated ? "available" : "unavailable"}>
                    {privilegedProviderStatus.authenticated ? "Yes" : "No"}
                  </span>
                </div>
                <div>
                  <span class="label">Version</span>
                  <strong>{privilegedProviderStatus.version || "unknown"}</strong>
                </div>
                {#if privilegedProviderStatus.expected_version}
                  <div>
                    <span class="label">Expected</span>
                    <strong>{privilegedProviderStatus.expected_version}</strong>
                  </div>
                {/if}
                {#if privilegedProviderStatus.endpoint}
                  <div>
                    <span class="label">Endpoint</span>
                    <strong>{privilegedProviderStatus.endpoint}</strong>
                  </div>
                {/if}
              </div>
              {#if privilegedProviderStatus.reason}
                <div class="notice" style="margin-top: 1rem;">
                  <span>{privilegedProviderStatus.reason}</span>
                </div>
              {/if}
              {#if privilegedProviderStatus.owner_state?.owned}
                <div class="owner-state">
                  <div>
                    <span class="label">Owner</span>
                    <strong>{privilegedProviderStatus.owner_state.mode || "runtime"}</strong>
                  </div>
                  {#if privilegedProviderStatus.owner_state.snapshot_id}
                    <div>
                      <span class="label">Snapshot</span>
                      <strong>{privilegedProviderStatus.owner_state.snapshot_id}</strong>
                    </div>
                  {/if}
                  <div>
                    <span class="label">State</span>
                    <span class="state" data-state={privilegedProviderStatus.owner_state.stale ? "degraded" : "available"}>
                      {privilegedProviderStatus.owner_state.stale ? "Stale" : "Owned"}
                    </span>
                  </div>
                </div>
                {#if privilegedProviderStatus.owner_state.reason}
                  <div class="notice" style="margin-top: 1rem;">
                    <span>{privilegedProviderStatus.owner_state.reason}</span>
                  </div>
                {/if}
                {#if privilegedProviderStatus.owner_state.repair_actions?.length}
                  <div class="repair-actions">
                    {#each privilegedProviderStatus.owner_state.repair_actions as action}
                      <button type="button" onclick={() => runProviderRepair(action)}>{action}</button>
                    {/each}
                  </div>
                {/if}
              {/if}
            {:else if platformError}
              <div class="notice error"><span>{platformError}</span></div>
            {:else}
              <p class="empty">Loading provider status...</p>
            {/if}
          </section>
          <section class="panel">
            <h2>System Proxy</h2>
            {#if proxyStatus}
              <div class="proxy-status">
                {#if !proxyStatus.available}
                  <div class="notice"><span>System proxy is not available on this platform.</span></div>
                {:else if !proxyStatus.supported}
                  <div class="notice"><span>{proxyStatus.reason || "System proxy is not supported on this configuration."}</span></div>
                {:else}
                  <div class="metrics">
                    <div>
                      <span class="label">OS Proxy</span>
                      <span class="state" data-state={proxyStatus.os_enabled ? "available" : "unavailable"}>
                        {proxyStatus.os_enabled ? "Enabled" : "Disabled"}
                      </span>
                    </div>
                    <div>
                      <span class="label">qkbox Owned</span>
                      <span class="state" data-state={proxyStatus.qkbox_owned ? "available" : "unavailable"}>
                        {proxyStatus.qkbox_owned ? "Yes" : "No"}
                      </span>
                    </div>
                    {#if proxyStatus.qkbox_owned && proxyStatus.address}
                      <div>
                        <span class="label">Proxy Address</span>
                        <strong>{proxyStatus.address}:{proxyStatus.port}</strong>
                      </div>
                    {/if}
                  </div>
                  <div style="margin-top: 1rem;">
                    <button type="button" onclick={toggleProxy} disabled={proxyLoading || !engineStarted()}>
                      {proxyStatus.qkbox_owned ? "Disable System Proxy" : "Enable System Proxy"}
                    </button>
                    {#if !engineStarted()}
                      <span class="label" style="margin-left: 0.5rem;">Start the engine first</span>
                    {/if}
                  </div>
                {/if}
              </div>
            {:else if proxyError}
              <div class="notice error"><span>{proxyError}</span></div>
            {:else}
              <p class="empty">Loading proxy status...</p>
            {/if}
          </section>
          {@render capabilityList("Platform", platformCapabilities)}
        {:else}
          {@render diagnostics(reply)}
        {/if}
      </section>
    {:else}
      <section class="notice">
        <span>Connecting to qkboxd...</span>
      </section>
    {/if}
  </section>
</main>

{#snippet capabilityList(title: string, items: Capability[])}
  <section class="panel">
    <h2>{title}</h2>
    <div class="capabilities">
      {#each items as item}
        <div class="capability">
          <div>
            <strong>{item.name}</strong>
            {#if item.reason}
              <span>{item.reason}</span>
            {/if}
          </div>
          <span class="state" data-state={item.state}>{item.state}</span>
        </div>
      {/each}
    </div>
  </section>
{/snippet}

{#snippet diagnostics(data: HelloReply)}
  <section class="panel diagnostics">
    <h2>Diagnostics</h2>
    <dl>
      <div>
        <dt>App version</dt>
        <dd>{data.app_version}</dd>
      </div>
      <div>
        <dt>qkboxd version</dt>
        <dd>{data.qkboxd_version}</dd>
      </div>
      <div>
        <dt>API compatibility</dt>
        <dd>{data.min_supported_api_version} - {data.api_version}</dd>
      </div>
      <div>
        <dt>Schema revision</dt>
        <dd>{data.schema_revision}</dd>
      </div>
      <div>
        <dt>Platform</dt>
        <dd>{data.platform.os}/{data.platform.arch}</dd>
      </div>
    </dl>
    <div class="panel-title" style="margin-top: 1rem;">
      <h2>Support Bundle</h2>
      <div class="button-row">
        <button type="button" onclick={refreshDiagnosticsReport}>Refresh diagnostics</button>
        <button type="button" onclick={createDebugBundle}>Create debug bundle</button>
      </div>
    </div>
    {#if diagnosticsError}
      <div class="notice error"><span>{diagnosticsError}</span></div>
    {/if}
    {#if debugBundlePath}
      <div class="notice"><span>{debugBundlePath}</span></div>
    {/if}
    {#if diagnosticsReport}
      <div class="check-list">
        {#each diagnosticsReport.checks as check}
          <div class="check-row">
            <div>
              <strong>{check.name}</strong>
              {#if check.reason}
                <span>{check.reason}</span>
              {/if}
              {#if check.recovery}
                <span>{check.recovery}</span>
              {/if}
            </div>
            <span class="state" data-state={check.state}>{check.state}</span>
          </div>
        {/each}
      </div>
    {/if}
    <div class="panel-title" style="margin-top: 1rem;">
      <h2>Data Plane</h2>
      <button class="icon-button" type="button" aria-label="Refresh asset state" onclick={refreshAssetState}>
        <RefreshCw size={16} />
      </button>
    </div>
    {#if assetError}
      <div class="notice error"><span>{assetError}</span></div>
    {/if}
    <h3>Subscriptions</h3>
    <div class="asset-list">
      {#each subscriptions as sub}
        <div class="asset-row">
          <div>
            <strong>{sub.name}</strong>
            <span>{sub.url}</span>
          </div>
          <span class="state" data-state={sub.last_status}>{sub.last_status}</span>
        </div>
        {#if sub.last_error_message}
          <div class="notice error"><span>{sub.last_error_code}: {sub.last_error_message}</span></div>
        {/if}
      {:else}
        <p class="empty">No profile subscriptions</p>
      {/each}
    </div>
    <h3>Assets</h3>
    <div class="asset-list">
      {#each dataAssets as asset}
        <div class="asset-row">
          <div>
            <strong>{asset.name}</strong>
            <span>{asset.kind} / {asset.source_url}</span>
          </div>
          <span class="state" data-state={asset.status}>{asset.status}</span>
        </div>
        {#if asset.content_sha256}
          <p class="asset-meta">{asset.content_sha256.slice(0, 12)} / {formatBytes(asset.size_bytes ?? 0)}{asset.version ? ` / ${asset.version}` : ""}</p>
        {/if}
        {#if asset.last_error_message}
          <div class="notice error"><span>{asset.last_error_code}: {asset.last_error_message}</span></div>
        {/if}
      {:else}
        <p class="empty">No cached data assets</p>
      {/each}
    </div>
  </section>
{/snippet}
