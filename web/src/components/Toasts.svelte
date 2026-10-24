<script>
  import { toasts } from '../lib/toast.svelte.js';
</script>

<div class="toasts" role="status" aria-live="polite">
  {#each toasts as item (item.id)}
    <div class="toast" class:error={item.kind === 'error'}>{item.message}</div>
  {/each}
</div>
