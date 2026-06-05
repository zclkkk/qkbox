<script lang="ts">
  let { label, value }: { label: string; value: string | number | boolean | null | undefined } = $props();
</script>

<div>
  <span class="label">{label}</span>
  <strong>{value ?? "unknown"}</strong>
</div>
