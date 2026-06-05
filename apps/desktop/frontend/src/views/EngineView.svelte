<script lang="ts">
  import CapabilityList from "../components/shared/CapabilityList.svelte";
  import ConnectionsPanel from "../components/engine/ConnectionsPanel.svelte";
  import EngineControlPanel from "../components/engine/EngineControlPanel.svelte";
  import LogsPanel from "../components/engine/LogsPanel.svelte";
  import OutboundGroupsPanel from "../components/engine/OutboundGroupsPanel.svelte";
  import TrafficPanel from "../components/engine/TrafficPanel.svelte";
  import { engineState } from "../lib/state/engine.svelte";
</script>

<EngineControlPanel />
<TrafficPanel />
<ConnectionsPanel />
<OutboundGroupsPanel />
<LogsPanel />
<CapabilityList title="Runtime" items={engineState.capabilities} />
