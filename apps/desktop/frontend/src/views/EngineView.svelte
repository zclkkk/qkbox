<script lang="ts">
  import { RefreshCw } from "@lucide/svelte";
  import CapabilityList from "../components/shared/CapabilityList.svelte";
  import EmptyState from "../components/shared/EmptyState.svelte";
  import ErrorNotice from "../components/shared/ErrorNotice.svelte";
  import IconButton from "../components/shared/IconButton.svelte";
  import StateBadge from "../components/shared/StateBadge.svelte";
  import { formatBytes } from "../lib/format";
  import { engineState } from "../lib/state/engine.svelte";
  import { runtimeEvents } from "../lib/state/runtime-events.svelte";
</script>

<section class="panel">
  <div class="panel-title">
    <h2>Engine Status</h2>
    <IconButton label="Refresh engine" onclick={() => engineState.refresh()} disabled={engineState.loading} spinning={engineState.loading}>
      <RefreshCw size={16} />
    </IconButton>
  </div>
  <div class="button-row panel-actions">
    <StateBadge state={engineState.status?.state} label={engineState.status?.state ?? "UNKNOWN"} />
    <button
      type="button"
      onclick={() => engineState.start()}
      disabled={engineState.loading || engineState.status?.state === "STARTED" || engineState.status?.state === "STARTING" || engineState.status?.state === "STOPPING" || engineState.status?.state === "FATAL"}
    >
      Start
    </button>
    <button
      type="button"
      onclick={() => engineState.stop()}
      disabled={engineState.loading || engineState.status?.state === "IDLE" || engineState.status?.state === "UNINITIALIZED" || engineState.status?.state === "STARTING" || engineState.status?.state === "STOPPING"}
    >
      Stop
    </button>
  </div>
  {#if engineState.status?.active_snapshot_id}
    <p>Active snapshot {engineState.status.active_snapshot_id}</p>
  {/if}
  <ErrorNotice message={engineState.error} />
  <ErrorNotice message={runtimeEvents.bridgeError} />
  {#if engineState.status?.last_error_message}
    <ErrorNotice message={`${engineState.status.last_error_code}: ${engineState.status.last_error_message}`} />
  {/if}
</section>

<section class="panel">
  <h2>Traffic</h2>
  {#if engineState.started && runtimeEvents.traffic}
    <div class="metrics">
      <div><span class="label">Upload</span><strong>{formatBytes(runtimeEvents.traffic.upload_total)}</strong></div>
      <div><span class="label">Download</span><strong>{formatBytes(runtimeEvents.traffic.download_total)}</strong></div>
      <div><span class="label">Up rate</span><strong>{formatBytes(runtimeEvents.traffic.upload_rate)}/s</strong></div>
      <div><span class="label">Down rate</span><strong>{formatBytes(runtimeEvents.traffic.download_rate)}/s</strong></div>
    </div>
  {:else if engineState.started}
    <EmptyState message="Waiting for traffic source" />
  {:else}
    <EmptyState message="Traffic source unavailable" />
  {/if}
</section>

<section class="panel">
  <div class="panel-title">
    <h2>Connections</h2>
    <button type="button" onclick={() => engineState.closeAllConnections()} disabled={!engineState.started || !runtimeEvents.connections?.connections?.length}>Close all</button>
  </div>
  {#if engineState.started && runtimeEvents.connections}
    <div class="connection-list">
      {#each runtimeEvents.connections.connections as connection}
        <div class="connection-row">
          <strong>{connection.host || connection.destination || connection.id}</strong>
          <span>{connection.network} / {connection.outbound || "unknown"} / {formatBytes(connection.upload)}/{formatBytes(connection.download)}</span>
        </div>
      {:else}
        <EmptyState message="No active connections" />
      {/each}
    </div>
  {:else if engineState.started}
    <EmptyState message="Waiting for connection source" />
  {:else}
    <EmptyState message="Connection source unavailable" />
  {/if}
</section>

<section class="panel">
  <h2>Outbound Groups</h2>
  {#if engineState.started}
    <div class="group-list">
      {#each engineState.groups as group}
        <div class="group-row">
          <div class="panel-title">
            <strong>{group.tag}</strong>
            {#if group.type === "urltest"}
              <button type="button" onclick={() => engineState.urlTest(group.tag)}>URLTest</button>
            {/if}
          </div>
          <span class="label">{group.type} / selected {group.selected}</span>
          <div class="outbound-options">
            {#each group.outbounds as outbound}
              {#if group.type === "selector"}
                <button type="button" class:active={outbound.tag === group.selected} onclick={() => engineState.selectOutbound(group.tag, outbound.tag)}>
                  {outbound.tag}
                </button>
              {:else}
                <span class:active={outbound.tag === group.selected}>{outbound.tag}</span>
              {/if}
            {/each}
          </div>
        </div>
      {:else}
        <EmptyState message="No runtime groups" />
      {/each}
    </div>
  {:else}
    <EmptyState message="Runtime groups unavailable" />
  {/if}
</section>

<section class="panel">
  <h2>Logs</h2>
  <div class="logs">
    {#each runtimeEvents.logs as entry}
      <div><span>{entry.level}</span>{entry.message}</div>
    {:else}
      <EmptyState message="No logs" />
    {/each}
  </div>
</section>

<CapabilityList title="Runtime" items={engineState.capabilities} />
