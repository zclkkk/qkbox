<script lang="ts">
  import EmptyState from "../shared/EmptyState.svelte";
  import ErrorNotice from "../shared/ErrorNotice.svelte";
  import MetricCard from "../shared/MetricCard.svelte";
  import StateBadge from "../shared/StateBadge.svelte";
  import { formatTimestamp } from "../../lib/format";
  import { platformState } from "../../lib/state/platform.svelte";
</script>

<section class="panel">
  <h2>Privileged Provider</h2>
  {#if platformState.privilegedProviderStatus}
    <div class="metrics">
      <div>
        <span class="label">Installed</span>
        <StateBadge
          state={platformState.privilegedProviderStatus.installed ? "available" : "unavailable"}
          label={platformState.privilegedProviderStatus.installed ? "Yes" : "No"}
        />
      </div>
      <div>
        <span class="label">Reachable</span>
        <StateBadge
          state={platformState.privilegedProviderStatus.reachable ? "available" : "unavailable"}
          label={platformState.privilegedProviderStatus.reachable ? "Yes" : "No"}
        />
      </div>
      <div>
        <span class="label">Authenticated</span>
        <StateBadge
          state={platformState.privilegedProviderStatus.authenticated ? "available" : "unavailable"}
          label={platformState.privilegedProviderStatus.authenticated ? "Yes" : "No"}
        />
      </div>
      <MetricCard label="Version" value={platformState.privilegedProviderStatus.version || "unknown"} />
      {#if platformState.privilegedProviderStatus.expected_version}
        <MetricCard label="Expected" value={platformState.privilegedProviderStatus.expected_version} />
      {/if}
      {#if platformState.privilegedProviderStatus.endpoint}
        <MetricCard label="Endpoint" value={platformState.privilegedProviderStatus.endpoint} />
      {/if}
    </div>

    {#if platformState.privilegedProviderStatus.reason}
      <ErrorNotice tone="default" message={platformState.privilegedProviderStatus.reason} />
    {/if}

    {#if platformState.privilegedProviderStatus.owner_state?.owned}
      <div class="owner-state">
        <MetricCard label="Owner mode" value={platformState.privilegedProviderStatus.owner_state.mode || "runtime"} />
        <MetricCard label="Runtime" value={platformState.privilegedProviderStatus.owner_state.runtime_id || "unknown"} />
        {#if platformState.privilegedProviderStatus.owner_state.profile_id}
          <MetricCard label="Profile" value={platformState.privilegedProviderStatus.owner_state.profile_id} />
        {/if}
        {#if platformState.privilegedProviderStatus.owner_state.started_at}
          <MetricCard label="Started" value={formatTimestamp(platformState.privilegedProviderStatus.owner_state.started_at)} />
        {/if}
        {#if platformState.privilegedProviderStatus.owner_state.last_heartbeat_at}
          <MetricCard label="Heartbeat" value={formatTimestamp(platformState.privilegedProviderStatus.owner_state.last_heartbeat_at)} />
        {/if}
        <div>
          <span class="label">State</span>
          <StateBadge
            state={platformState.privilegedProviderStatus.owner_state.stale ? "degraded" : "available"}
            label={platformState.privilegedProviderStatus.owner_state.stale ? "Stale" : "Owned"}
          />
        </div>
      </div>
      {#if platformState.privilegedProviderStatus.owner_state.reason}
        <ErrorNotice tone="default" message={platformState.privilegedProviderStatus.owner_state.reason} />
      {/if}
      {#if platformState.privilegedProviderStatus.owner_state.repair_actions?.length}
        <div class="repair-actions">
          {#each platformState.privilegedProviderStatus.owner_state.repair_actions as action}
            <button type="button" onclick={() => platformState.runProviderRepair(action)} disabled={platformState.repairingAction === action}>
              {platformState.repairingAction === action ? "Repairing..." : action}
            </button>
            {#if platformState.repairResults[action]}
              <StateBadge state={platformState.repairResults[action].outcome || "available"} label={platformState.repairResults[action].outcome || "done"} />
            {/if}
          {/each}
        </div>
      {/if}
    {:else}
      <div class="notice compact-notice">
        <span>No provider-owned runtime state.</span>
      </div>
    {/if}
  {:else if platformState.error}
    <ErrorNotice message={platformState.error} />
  {:else}
    <EmptyState message="Loading provider status..." />
  {/if}
</section>
