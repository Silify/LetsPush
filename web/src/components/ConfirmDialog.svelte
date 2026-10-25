<script>
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { Input } from '$lib/components/ui/input/index.js';

  /**
   * Asks before something that cannot be undone. Give it `type` to make the person type a word first.
   * onconfirm may be async; the button shows it is busy until it finishes. It closes the dialog itself
   * when it succeeds, by setting open to false.
   */
  let { open = $bindable(false), title, confirmLabel = 'Confirm', type = '', onconfirm, children } = $props();

  let typed = $state('');
  let busy = $state(false);

  $effect(() => {
    if (open) typed = '';
  });

  async function confirm(event) {
    event.preventDefault();
    busy = true;
    try {
      await onconfirm();
    } finally {
      busy = false;
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>{title}</Dialog.Title>
      <Dialog.Description>{@render children?.()}</Dialog.Description>
    </Dialog.Header>
    <form class="grid gap-4" onsubmit={confirm}>
      {#if type}
        <Input bind:value={typed} placeholder={type} autocomplete="off" aria-label="Type {type} to confirm" />
      {/if}
      <Dialog.Footer>
        <Button type="button" variant="outline" onclick={() => (open = false)}>Cancel</Button>
        <Button type="submit" variant="destructive" disabled={busy || (type !== '' && typed !== type)}>
          {busy ? 'Working…' : confirmLabel}
        </Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
