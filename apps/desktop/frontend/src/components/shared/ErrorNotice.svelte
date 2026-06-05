<script lang="ts">
  import { CircleAlert } from "@lucide/svelte";

  let { message, tone = "error" }: { message: string | null | undefined; tone?: "error" | "default" } = $props();
</script>

{#if message}
  <section class="notice" class:error={tone === "error"}>
    {#if tone === "error"}
      <CircleAlert size={18} />
    {/if}
    <span>{message}</span>
  </section>
{/if}
