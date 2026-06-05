<script lang="ts">
  import EmptyState from "../shared/EmptyState.svelte";
  import StateBadge from "../shared/StateBadge.svelte";
  import { platformState } from "../../lib/state/platform.svelte";

  const preparable = new Set(["BACKGROUND_SERVICE", "TUN_MODE", "DNS_HIJACK"]);

  function labelFor(name: string) {
    return name.replaceAll("_", " ").toLowerCase();
  }

  function canPrepare(capability: { name: string; state: string }) {
    return preparable.has(capability.name) && capability.state !== "available" && capability.state !== "unsupported";
  }
</script>

<section class="panel diagnostics">
  <div class="panel-title">
    <h2>Platform Capabilities</h2>
    <button type="button" onclick={() => platformState.refresh()} disabled={platformState.loading}>
      {platformState.loading ? "Refreshing..." : "Refresh"}
    </button>
  </div>

  <div class="platform-capability-grid">
    {#each platformState.capabilities as capability}
      <div class="platform-capability">
        <div>
          <strong>{labelFor(capability.name)}</strong>
          {#if capability.reason}
            <span>{capability.reason}</span>
          {/if}
          {#if platformState.prepareResults[capability.name]}
            <span>
              Last prepare: {platformState.prepareResults[capability.name].state}
              {platformState.prepareResults[capability.name].reason ? ` - ${platformState.prepareResults[capability.name].reason}` : ""}
            </span>
          {/if}
        </div>
        <div class="capability-actions">
          <StateBadge state={capability.state} label={capability.state} />
          {#if canPrepare(capability)}
            <button type="button" onclick={() => platformState.prepareFeature(capability.name)} disabled={platformState.preparingFeature === capability.name}>
              {platformState.preparingFeature === capability.name ? "Preparing..." : "Prepare"}
            </button>
          {/if}
        </div>
      </div>
    {:else}
      <EmptyState message="No platform capabilities reported" />
    {/each}
  </div>
</section>
