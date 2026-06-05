<script lang="ts">
  import CapabilityMatrixPanel from "../components/platform/CapabilityMatrixPanel.svelte";
  import NetworkExtensionPanel from "../components/platform/NetworkExtensionPanel.svelte";
  import NetworkModePanel from "../components/platform/NetworkModePanel.svelte";
  import ProviderStatusPanel from "../components/platform/ProviderStatusPanel.svelte";
  import SystemProxyPanel from "../components/platform/SystemProxyPanel.svelte";
  import ErrorNotice from "../components/shared/ErrorNotice.svelte";
  import { appState } from "../lib/state/app.svelte";
  import { platformState } from "../lib/state/platform.svelte";
</script>

{#if platformState.error}
  <section class="panel diagnostics">
    <ErrorNotice message={platformState.error} />
  </section>
{/if}

<NetworkModePanel platformOS={appState.hello?.platform.os} />
<ProviderStatusPanel />
{#if appState.hello?.platform.os === "darwin"}
  <NetworkExtensionPanel />
{/if}
<SystemProxyPanel />
<CapabilityMatrixPanel />
