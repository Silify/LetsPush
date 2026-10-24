<script>
  let { open = $bindable(false), title, children } = $props();
  let dialog = $state();

  $effect(() => {
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  });
</script>

<dialog
  bind:this={dialog}
  onclose={() => (open = false)}
  onclick={(event) => {
    if (event.target === dialog) open = false;
  }}
>
  <div class="stack" style="gap: 18px">
    <div class="row between">
      <h2>{title}</h2>
      <button class="btn ghost small" type="button" aria-label="Close" onclick={() => (open = false)}>✕</button>
    </div>
    {@render children()}
  </div>
</dialog>
