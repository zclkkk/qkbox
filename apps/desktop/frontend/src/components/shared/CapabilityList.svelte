<script lang="ts">
  import type { Capability } from "../../lib/api/client";
  import EmptyState from "./EmptyState.svelte";
  import StateBadge from "./StateBadge.svelte";

  let { title, items }: { title: string; items: Capability[] } = $props();
</script>

<section class="panel">
  <h2>{title}</h2>
  <div class="capabilities">
    {#each items as item}
      <div class="capability">
        <div>
          <strong>{item.name}</strong>
          {#if item.reason}
            <span>{item.reason}</span>
          {/if}
        </div>
        <StateBadge state={item.state} label={item.state} />
      </div>
    {:else}
      <EmptyState message="No capabilities reported" />
    {/each}
  </div>
</section>
