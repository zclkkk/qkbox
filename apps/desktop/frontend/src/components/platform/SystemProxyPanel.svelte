<script lang="ts">
  import EmptyState from "../shared/EmptyState.svelte";
  import ErrorNotice from "../shared/ErrorNotice.svelte";
  import MetricCard from "../shared/MetricCard.svelte";
  import StateBadge from "../shared/StateBadge.svelte";
  import { engineState } from "../../lib/state/engine.svelte";
  import { platformState } from "../../lib/state/platform.svelte";
</script>

<section class="panel">
  <h2>System Proxy</h2>
  {#if platformState.proxyStatus}
    <div class="proxy-status">
      {#if !platformState.proxyStatus.available}
        <ErrorNotice tone="default" message={platformState.proxyStatus.reason || "System proxy is not available on this platform."} />
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
          {#if platformState.proxyStatus.address}
            <MetricCard label="Proxy Address" value={`${platformState.proxyStatus.address}:${platformState.proxyStatus.port}`} />
          {/if}
        </div>

        {#if platformState.proxyStatus.os_enabled && !platformState.proxyStatus.qkbox_owned}
          <div class="notice compact-notice">
            <span>OS proxy is enabled by another owner; qkbox will not overwrite it unless you enable qkbox ownership.</span>
          </div>
        {/if}

        <div class="panel-actions">
          <button type="button" onclick={() => platformState.toggleProxy()} disabled={platformState.proxyLoading || !engineState.started}>
            {platformState.proxyStatus.qkbox_owned ? "Disable System Proxy" : "Enable System Proxy"}
          </button>
          {#if !engineState.started}
            <span class="label">Engine must be started first</span>
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
