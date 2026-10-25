<script>
  import * as Card from '$lib/components/ui/card/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import { count } from '$lib/format.js';

  /** One number with a label and an icon. value is null while it loads. tone: '' or 'danger'. */
  let { label, value = null, icon: Icon, tone = '' } = $props();

  const danger = $derived(tone === 'danger' && value > 0);
</script>

<Card.Root size="sm">
  <Card.Content class="flex items-center gap-3">
    <div class="grid size-9 shrink-0 place-items-center rounded-lg {danger ? 'bg-destructive/10 text-destructive' : 'bg-primary/10 text-primary'}">
      <Icon class="size-4" />
    </div>
    <div class="min-w-0">
      {#if value === null}
        <Skeleton class="mb-1 h-6 w-12" />
      {:else}
        <div class="text-2xl leading-none font-semibold tracking-tight">{count(value)}</div>
      {/if}
      <div class="mt-1 truncate text-xs text-muted-foreground">{label}</div>
    </div>
  </Card.Content>
</Card.Root>
