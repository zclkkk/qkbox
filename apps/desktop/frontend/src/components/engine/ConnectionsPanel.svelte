<script lang="ts">
  import EmptyState from "../shared/EmptyState.svelte";
  import { formatBytes, formatTimestamp } from "../../lib/format";
  import { engineState } from "../../lib/state/engine.svelte";
  import { runtimeEvents } from "../../lib/state/runtime-events.svelte";
  import type { RuntimeConnection } from "../../lib/api/client";

  type SortKey = "started" | "download" | "upload" | "host";

  let filter = $state("");
  let sortKey = $state<SortKey>("started");

  let filteredConnections = $derived.by(() => {
    const needle = filter.trim().toLowerCase();
    const connections = [...(runtimeEvents.connections?.connections ?? [])];
    const filtered = needle
      ? connections.filter((connection) => connectionText(connection).includes(needle))
      : connections;
    return filtered.sort((left, right) => {
      if (sortKey === "download") {
        return right.download - left.download;
      }
      if (sortKey === "upload") {
        return right.upload - left.upload;
      }
      if (sortKey === "host") {
        return displayHost(left).localeCompare(displayHost(right));
      }
      return (right.started_at ?? 0) - (left.started_at ?? 0);
    });
  });

  function displayHost(connection: RuntimeConnection) {
    return connection.host || connection.destination || connection.source || connection.id;
  }

  function connectionText(connection: RuntimeConnection) {
    return [
      connection.id,
      connection.network,
      connection.source,
      connection.destination,
      connection.host,
      connection.process,
      connection.inbound,
      connection.outbound,
      connection.rule
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();
  }
</script>

<section class="panel">
  <div class="panel-title">
    <h2>Connections</h2>
    <button type="button" onclick={() => engineState.closeAllConnections()} disabled={!engineState.started || !runtimeEvents.connections?.connections?.length}>
      Close all
    </button>
  </div>

  {#if engineState.started && runtimeEvents.connections}
    <div class="connection-controls">
      <label>
        <span class="label">Filter</span>
        <input bind:value={filter} placeholder="host, process, outbound..." />
      </label>
      <label>
        <span class="label">Sort</span>
        <select bind:value={sortKey}>
          <option value="started">Newest</option>
          <option value="download">Download</option>
          <option value="upload">Upload</option>
          <option value="host">Host</option>
        </select>
      </label>
    </div>
    <div class="metrics">
      <div><span class="label">Total upload</span><strong>{formatBytes(runtimeEvents.connections.upload_total)}</strong></div>
      <div><span class="label">Total download</span><strong>{formatBytes(runtimeEvents.connections.download_total)}</strong></div>
      <div><span class="label">Active</span><strong>{runtimeEvents.connections.connections.length}</strong></div>
      <div><span class="label">Sample</span><strong>{formatTimestamp(runtimeEvents.connections.timestamp)}</strong></div>
    </div>
    <div class="connection-table">
      {#each filteredConnections as connection}
        <div class="connection-row">
          <div>
            <strong>{displayHost(connection)}</strong>
            <span>{connection.network} · {connection.source} → {connection.destination}</span>
            <span>{connection.inbound || "unknown inbound"} · {connection.outbound || "unknown outbound"} · {connection.rule || "no rule"}</span>
            {#if connection.process}
              <span>{connection.process}</span>
            {/if}
          </div>
          <div class="connection-metrics">
            <span>{formatBytes(connection.upload)} up</span>
            <span>{formatBytes(connection.download)} down</span>
            <span>{formatTimestamp(connection.started_at)}</span>
            <button type="button" onclick={() => engineState.closeConnection(connection.id)}>Close</button>
          </div>
        </div>
      {:else}
        <EmptyState message={filter ? "No matching connections" : "No active connections"} />
      {/each}
    </div>
  {:else if engineState.started}
    <EmptyState message="Waiting for connection source" />
  {:else}
    <EmptyState message="Connection source unavailable" />
  {/if}
</section>
