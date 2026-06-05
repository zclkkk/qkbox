<script lang="ts">
  let { state, label = state }: { state: string | boolean | null | undefined; label?: string | boolean | null } = $props();

  const normalized = $derived(String(state ?? "unknown"));
</script>

<span class="state" data-state={normalized}>{label ?? normalized}</span>
