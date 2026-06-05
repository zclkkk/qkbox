<script lang="ts">
  import EmptyState from "../shared/EmptyState.svelte";
  import { formatTimestamp } from "../../lib/format";
  import { runtimeEvents } from "../../lib/state/runtime-events.svelte";

  let level = $state("all");
  let source = $state("all");
  let query = $state("");
  let autoScroll = $state(true);
  let logList: HTMLDivElement;

  let sources = $derived(Array.from(new Set(runtimeEvents.logs.map((entry) => entry.source).filter(Boolean))).sort());
  let levels = $derived(Array.from(new Set(runtimeEvents.logs.map((entry) => entry.level).filter(Boolean))).sort());
  let filteredLogs = $derived.by(() => {
    const needle = query.trim().toLowerCase();
    return runtimeEvents.logs.filter((entry) => {
      if (level !== "all" && entry.level !== level) {
        return false;
      }
      if (source !== "all" && entry.source !== source) {
        return false;
      }
      if (!needle) {
        return true;
      }
      return `${entry.source} ${entry.level} ${entry.message}`.toLowerCase().includes(needle);
    });
  });

  $effect(() => {
    runtimeEvents.logs.length;
    filteredLogs.length;
    if (!autoScroll || !logList) {
      return;
    }
    queueMicrotask(() => {
      logList.scrollTop = logList.scrollHeight;
    });
  });
</script>

<section class="panel">
  <div class="panel-title">
    <h2>Logs</h2>
    <button type="button" onclick={() => runtimeEvents.clearLogs()} disabled={runtimeEvents.logs.length === 0}>Clear view</button>
  </div>

  <div class="log-controls">
    <label>
      <span class="label">Level</span>
      <select bind:value={level}>
        <option value="all">All</option>
        {#each levels as item}
          <option value={item}>{item}</option>
        {/each}
      </select>
    </label>
    <label>
      <span class="label">Source</span>
      <select bind:value={source}>
        <option value="all">All</option>
        {#each sources as item}
          <option value={item}>{item}</option>
        {/each}
      </select>
    </label>
    <label>
      <span class="label">Search</span>
      <input bind:value={query} placeholder="message text" />
    </label>
    <label class="toggle-row">
      <input type="checkbox" bind:checked={autoScroll} />
      <span>Auto-scroll</span>
    </label>
  </div>

  <div class="logs" bind:this={logList}>
    {#each filteredLogs as entry}
      <div class="log-row">
        <span>{entry.seq}</span>
        <span>{formatTimestamp(entry.timestamp)}</span>
        <span>{entry.source}</span>
        <strong>{entry.level}</strong>
        <p>{entry.message}</p>
      </div>
    {:else}
      <EmptyState message={runtimeEvents.logs.length === 0 ? "No logs" : "No logs match the current filters"} />
    {/each}
  </div>
</section>
