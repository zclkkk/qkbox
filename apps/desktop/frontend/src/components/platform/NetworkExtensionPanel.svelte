<script lang="ts">
  import ErrorNotice from "../shared/ErrorNotice.svelte";
  import MetricCard from "../shared/MetricCard.svelte";
  import StateBadge from "../shared/StateBadge.svelte";
  import { formatTimestamp } from "../../lib/format";
  import { platformState } from "../../lib/state/platform.svelte";
</script>

{#if platformState.networkExtensionStatus}
  <section class="panel">
    <h2>Apple NetworkExtension</h2>
    <div class="metrics">
      <div>
        <span class="label">Installed</span>
        <StateBadge
          state={platformState.networkExtensionStatus.installed ? "available" : "unavailable"}
          label={platformState.networkExtensionStatus.installed ? "Yes" : "No"}
        />
      </div>
      <div>
        <span class="label">Reachable</span>
        <StateBadge
          state={platformState.networkExtensionStatus.reachable ? "available" : "unavailable"}
          label={platformState.networkExtensionStatus.reachable ? "Yes" : "No"}
        />
      </div>
      <div>
        <span class="label">Authorized</span>
        <StateBadge
          state={platformState.networkExtensionStatus.authorized ? "available" : "unavailable"}
          label={platformState.networkExtensionStatus.authorized ? "Yes" : "No"}
        />
      </div>
      <MetricCard label="Bundle" value={platformState.networkExtensionStatus.bundle_id || "unknown"} />
      <MetricCard label="Version" value={platformState.networkExtensionStatus.version || "unknown"} />
    </div>

    {#if platformState.networkExtensionStatus.reason}
      <ErrorNotice tone="default" message={platformState.networkExtensionStatus.reason} />
    {/if}

    {#if platformState.networkExtensionStatus.owner_state?.owned}
      <div class="owner-state">
        <MetricCard label="Owner mode" value={platformState.networkExtensionStatus.owner_state.mode || "network_extension"} />
        <MetricCard label="Runtime" value={platformState.networkExtensionStatus.owner_state.runtime_id || "unknown"} />
        {#if platformState.networkExtensionStatus.owner_state.snapshot_id}
          <MetricCard label="Snapshot" value={platformState.networkExtensionStatus.owner_state.snapshot_id} />
        {/if}
        {#if platformState.networkExtensionStatus.owner_state.started_at}
          <MetricCard label="Started" value={formatTimestamp(platformState.networkExtensionStatus.owner_state.started_at)} />
        {/if}
        <div>
          <span class="label">State</span>
          <StateBadge
            state={platformState.networkExtensionStatus.owner_state.stale ? "degraded" : "available"}
            label={platformState.networkExtensionStatus.owner_state.stale ? "Stale" : "Owned"}
          />
        </div>
      </div>
    {/if}
  </section>
{/if}
