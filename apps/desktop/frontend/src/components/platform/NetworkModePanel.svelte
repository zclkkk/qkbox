<script lang="ts">
  import StateBadge from "../shared/StateBadge.svelte";
  import { platformState } from "../../lib/state/platform.svelte";

  let { platformOS }: { platformOS: string | undefined } = $props();

  const machineFeatures = ["TUN_MODE", "DNS_HIJACK"];

  function capability(name: string) {
    return platformState.capabilities.find((item) => item.name === name);
  }

  function featureLabel(name: string) {
    return name.replaceAll("_", " ").toLowerCase();
  }

  function canPrepare(capability: { state: string } | undefined) {
    return capability && capability.state !== "available" && capability.state !== "unsupported";
  }

  let systemProxy = $derived(capability("SYSTEM_PROXY"));
  let tunMode = $derived(capability("TUN_MODE"));
  let dnsHijack = $derived(capability("DNS_HIJACK"));
  let owner = $derived(platformOS === "darwin" ? "Apple NetworkExtension" : "privileged provider");
</script>

<section class="panel diagnostics">
  <h2>Network Modes</h2>
  <div class="mode-grid">
    <div class="mode-card">
      <div class="panel-title">
        <strong>System proxy</strong>
        <StateBadge state={systemProxy?.state} label={systemProxy?.state ?? "unknown"} />
      </div>
      {#if systemProxy?.reason}
        <span>{systemProxy.reason}</span>
      {/if}
    </div>

    <div class="mode-card">
      <div class="panel-title">
        <strong>Machine network</strong>
        <StateBadge state={tunMode?.state} label={tunMode?.state ?? "unknown"} />
      </div>
      <span>{owner}</span>
      {#if tunMode?.reason}
        <span>{tunMode.reason}</span>
      {/if}
      {#if dnsHijack?.reason && dnsHijack.reason !== tunMode?.reason}
        <span>{dnsHijack.reason}</span>
      {/if}
      <div class="repair-actions">
        {#each machineFeatures as feature}
          {@const cap = capability(feature)}
          {#if canPrepare(cap)}
            <button type="button" onclick={() => platformState.prepareFeature(feature)} disabled={platformState.preparingFeature === feature}>
              {platformState.preparingFeature === feature ? "Preparing..." : `Prepare ${featureLabel(feature)}`}
            </button>
          {/if}
        {/each}
      </div>
    </div>
  </div>
</section>
