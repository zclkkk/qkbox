<script lang="ts">
  import CapabilityList from "../components/shared/CapabilityList.svelte";
  import EmptyState from "../components/shared/EmptyState.svelte";
  import ErrorNotice from "../components/shared/ErrorNotice.svelte";
  import MetricCard from "../components/shared/MetricCard.svelte";
  import StateBadge from "../components/shared/StateBadge.svelte";
  import { engineState } from "../lib/state/engine.svelte";
  import { platformState } from "../lib/state/platform.svelte";
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
        <MetricCard label="Owner" value={platformState.privilegedProviderStatus.owner_state.mode || "runtime"} />
        {#if platformState.privilegedProviderStatus.owner_state.snapshot_id}
          <MetricCard label="Snapshot" value={platformState.privilegedProviderStatus.owner_state.snapshot_id} />
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
            <button type="button" onclick={() => platformState.runProviderRepair(action)}>{action}</button>
          {/each}
        </div>
      {/if}
    {/if}
  {:else if platformState.error}
    <ErrorNotice message={platformState.error} />
  {:else}
    <EmptyState message="Loading provider status..." />
  {/if}
</section>

<section class="panel">
  <h2>System Proxy</h2>
  {#if platformState.proxyStatus}
    <div class="proxy-status">
      {#if !platformState.proxyStatus.available}
        <ErrorNotice tone="default" message="System proxy is not available on this platform." />
      {:else if !platformState.proxyStatus.supported}
        <ErrorNotice tone="default" message={platformState.proxyStatus.reason || "System proxy is not supported on this configuration."} />
      {:else}
        <div class="metrics">
          <div>
            <span class="label">OS Proxy</span>
            <StateBadge
              state={platformState.proxyStatus.os_enabled ? "available" : "unavailable"}
              label={platformState.proxyStatus.os_enabled ? "Enabled" : "Disabled"}
            />
          </div>
          <div>
            <span class="label">qkbox Owned</span>
            <StateBadge
              state={platformState.proxyStatus.qkbox_owned ? "available" : "unavailable"}
              label={platformState.proxyStatus.qkbox_owned ? "Yes" : "No"}
            />
          </div>
          {#if platformState.proxyStatus.qkbox_owned && platformState.proxyStatus.address}
            <MetricCard label="Proxy Address" value={`${platformState.proxyStatus.address}:${platformState.proxyStatus.port}`} />
          {/if}
        </div>
        <div class="panel-actions">
          <button type="button" onclick={() => platformState.toggleProxy()} disabled={platformState.proxyLoading || !engineState.started}>
            {platformState.proxyStatus.qkbox_owned ? "Disable System Proxy" : "Enable System Proxy"}
          </button>
          {#if !engineState.started}
            <span class="label">Start the engine first</span>
          {/if}
        </div>
      {/if}
    </div>
  {:else if platformState.proxyError}
    <ErrorNotice message={platformState.proxyError} />
  {:else}
    <EmptyState message="Loading proxy status..." />
  {/if}
</section>

<CapabilityList title="Platform" items={platformState.capabilities} />
