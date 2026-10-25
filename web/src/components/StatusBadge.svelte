<script>
  import CircleCheck from '@lucide/svelte/icons/circle-check';
  import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
  import { Badge } from '$lib/components/ui/badge/index.js';

  /** Whether an app can send: it has a Firebase key or it does not. */
  let { ready } = $props();
</script>

{#if ready}
  <Badge variant="secondary" class="text-primary"><CircleCheck data-icon="inline-start" />Ready to send</Badge>
{:else}
  <Badge variant="outline" class="text-amber-600 dark:text-amber-400"><TriangleAlert data-icon="inline-start" />Needs a Firebase key</Badge>
{/if}
