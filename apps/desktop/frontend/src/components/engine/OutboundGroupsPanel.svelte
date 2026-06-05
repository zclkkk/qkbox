<script lang="ts">
  import EmptyState from "../shared/EmptyState.svelte";
  import StateBadge from "../shared/StateBadge.svelte";
  import { engineState } from "../../lib/state/engine.svelte";
</script>

<section class="panel">
  <div class="panel-title">
    <h2>Outbound Groups</h2>
    <button type="button" onclick={() => engineState.refreshGroups()} disabled={!engineState.started || engineState.loading}>Refresh</button>
  </div>

  {#if engineState.started}
    <div class="group-list">
      {#each engineState.groups as group}
        <div class="group-row">
          <div class="panel-title">
            <div>
              <strong>{group.tag}</strong>
              <span class="label">{group.type} · selected {group.selected || "none"}</span>
            </div>
            {#if group.type === "urltest"}
              <button type="button" onclick={() => engineState.urlTest(group.tag)} disabled={engineState.urlTestingGroup === group.tag}>
                {engineState.urlTestingGroup === group.tag ? "Testing..." : "URLTest"}
              </button>
            {/if}
          </div>
          <div class="outbound-options">
            {#each group.outbounds as outbound}
              {#if group.type === "selector"}
                <button type="button" class:active={outbound.tag === group.selected} onclick={() => engineState.selectOutbound(group.tag, outbound.tag)}>
                  {outbound.tag}
                  {#if outbound.delay_ms}
                    <span>{outbound.delay_ms} ms</span>
                  {/if}
                </button>
              {:else}
                <span class:active={outbound.tag === group.selected}>
                  {outbound.tag}
                  {#if outbound.delay_ms}
                    · {outbound.delay_ms} ms
                  {/if}
                </span>
              {/if}
            {/each}
          </div>
          {#if engineState.urlTestResults[group.tag]?.length}
            <div class="urltest-results">
              {#each engineState.urlTestResults[group.tag] as result}
                <div>
                  <strong>{result.outbound}</strong>
                  {#if result.error_code}
                    <StateBadge state="error" label={result.error_code} />
                    <span>{result.error_message}</span>
                  {:else}
                    <StateBadge state="available" label={`${result.delay_ms} ms`} />
                  {/if}
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {:else}
        <EmptyState message="No runtime groups" />
      {/each}
    </div>
  {:else}
    <EmptyState message="Runtime groups unavailable" />
  {/if}
</section>
