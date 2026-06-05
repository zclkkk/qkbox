<script lang="ts">
  let {
    open = false,
    title,
    message,
    confirmLabel = "Confirm",
    cancelLabel = "Cancel",
    onconfirm,
    oncancel
  }: {
    open?: boolean;
    title: string;
    message: string;
    confirmLabel?: string;
    cancelLabel?: string;
    onconfirm?: () => void | Promise<void>;
    oncancel?: () => void;
  } = $props();
</script>

{#if open}
  <div class="dialog-backdrop">
    <section class="dialog">
      <h2>{title}</h2>
      <p>{message}</p>
      <div class="button-row">
        <button type="button" onclick={oncancel}>{cancelLabel}</button>
        <button type="button" onclick={onconfirm}>{confirmLabel}</button>
      </div>
    </section>
  </div>
{/if}
