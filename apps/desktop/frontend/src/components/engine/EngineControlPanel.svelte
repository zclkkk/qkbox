<script lang="ts">
  import { RefreshCw } from "@lucide/svelte";
  import ErrorNotice from "../shared/ErrorNotice.svelte";
  import IconButton from "../shared/IconButton.svelte";
  import StateBadge from "../shared/StateBadge.svelte";
  import { formatDurationSince, formatTimestamp } from "../../lib/format";
  import { engineState } from "../../lib/state/engine.svelte";
  import { runtimeEvents } from "../../lib/state/runtime-events.svelte";

  const busyStates = new Set(["VALIDATING", "STARTING", "STOPPING"]);

  let canStart = $derived(
    !engineState.loading &&
      !busyStates.has(engineState.status?.state ?? "") &&
      engineState.status?.state !== "STARTED" &&
      engineState.status?.state !== "FATAL"
  );
  let canStop = $derived(
    !engineState.loading &&
      !busyStates.has(engineState.status?.state ?? "") &&
      engineState.status?.state !== "IDLE" &&
      engineState.status?.state !== "UNINITIALIZED"
  );
</script>

<section class="panel">
  <div class="panel-title">
    <h2>Engine</h2>
    <IconButton label="Refresh engine" onclick={() => engineState.refresh()} disabled={engineState.loading} spinning={engineState.loading}>
      <RefreshCw size={16} />
    </IconButton>
  </div>

  <div class="engine-toolbar">
    <StateBadge state={engineState.status?.state} label={engineState.status?.state ?? "UNKNOWN"} />
    <button type="button" onclick={() => engineState.start()} disabled={!canStart}>Start</button>
    <button type="button" onclick={() => engineState.stop()} disabled={!canStop}>Stop</button>
  </div>

  <div class="metrics">
    <div>
      <span class="label">Active profile</span>
      <strong>{engineState.status?.active_profile_id || "none"}</strong>
    </div>
    <div>
      <span class="label">Started</span>
      <strong>{formatTimestamp(engineState.status?.started_at)}</strong>
    </div>
    <div>
      <span class="label">Uptime</span>
      <strong>{engineState.started ? formatDurationSince(engineState.status?.started_at) : "offline"}</strong>
    </div>
    <div>
      <span class="label">Event bridge</span>
      <strong>{runtimeEvents.running ? "connected" : "stopped"}</strong>
    </div>
  </div>

  <ErrorNotice message={engineState.error} />
  <ErrorNotice message={runtimeEvents.bridgeError} />
  {#if engineState.status?.last_error_message}
    <ErrorNotice message={`${engineState.status.last_error_code}: ${engineState.status.last_error_message}`} />
  {/if}
</section>
