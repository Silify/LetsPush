<script>
  import Check from '@lucide/svelte/icons/check';
  import Copy from '@lucide/svelte/icons/copy';
  import { Button } from '$lib/components/ui/button/index.js';
  import { copyText } from '$lib/format.js';
  import { toast } from '$lib/toast.js';

  /** Text in a box with a copy button. multiline keeps line breaks, for commands. */
  let { value, multiline = false, label = 'Copy' } = $props();

  let copied = $state(false);

  async function copy() {
    if (await copyText(value)) {
      copied = true;
      setTimeout(() => (copied = false), 1500);
    } else {
      toast('Select the text and copy it', 'error');
    }
  }
</script>

<div class="flex items-stretch gap-2">
  <code
    class="min-w-0 flex-1 rounded-md border bg-muted px-3 py-2 font-mono text-sm {multiline
      ? 'overflow-x-auto text-xs leading-relaxed whitespace-pre'
      : 'break-all'}">{value}</code
  >
  <Button variant="outline" size="icon" class="h-auto w-9 shrink-0" onclick={copy} aria-label={label}>
    {#if copied}<Check class="text-primary" />{:else}<Copy />{/if}
  </Button>
</div>
