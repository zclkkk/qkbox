<script lang="ts">
  let {
    label,
    disabled = false,
    spinning = false,
    onclick,
    children
  }: {
    label: string;
    disabled?: boolean;
    spinning?: boolean;
    onclick?: () => void | Promise<void>;
    children: import("svelte").Snippet;
  } = $props();
</script>

<button class="icon-button" type="button" aria-label={label} {disabled} {onclick}>
  <span class:spin={spinning}>
    {@render children()}
  </span>
</button>
