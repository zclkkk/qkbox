<script lang="ts">
  import EmptyState from "../shared/EmptyState.svelte";
  import { formatBytes, formatRate, formatTimestamp } from "../../lib/format";
  import { engineState } from "../../lib/state/engine.svelte";
  import { runtimeEvents } from "../../lib/state/runtime-events.svelte";

  let peakRate = $derived(
    Math.max(1, ...runtimeEvents.trafficHistory.map((sample) => Math.max(sample.upload_rate, sample.download_rate)))
  );
</script>

<section class="panel">
  <h2>Traffic</h2>
  {#if engineState.started && runtimeEvents.traffic}
    <div class="metrics">
      <div><span class="label">Upload</span><strong>{formatBytes(runtimeEvents.traffic.upload_total)}</strong></div>
      <div><span class="label">Download</span><strong>{formatBytes(runtimeEvents.traffic.download_total)}</strong></div>
      <div><span class="label">Up rate</span><strong>{formatRate(runtimeEvents.traffic.upload_rate)}</strong></div>
      <div><span class="label">Down rate</span><strong>{formatRate(runtimeEvents.traffic.download_rate)}</strong></div>
    </div>
    <div class="traffic-chart" aria-label="Traffic rate history">
      {#each runtimeEvents.trafficHistory as sample}
        <div class="traffic-bar">
          <span class="upload-bar" style={`height: ${Math.max(2, (sample.upload_rate / peakRate) * 100)}%`}></span>
          <span class="download-bar" style={`height: ${Math.max(2, (sample.download_rate / peakRate) * 100)}%`}></span>
        </div>
      {/each}
    </div>
    <span class="label">Last sample {formatTimestamp(runtimeEvents.traffic.timestamp)}</span>
  {:else if engineState.started}
    <EmptyState message="Waiting for traffic source" />
  {:else}
    <EmptyState message="Traffic source unavailable" />
  {/if}
</section>
