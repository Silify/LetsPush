<script>
  import { Button } from '$lib/components/ui/button/index.js';
  import { copyText } from '$lib/format.js';
  import { toast } from '$lib/toast.js';

  let { value } = $props();

  async function copy() {
    if (await copyText(value)) toast('Copied');
    else toast('Select the text and copy it', 'error');
  }
</script>

<div class="flex items-stretch gap-2">
  <code class="min-w-0 flex-1 rounded-md border bg-muted px-3 py-2 font-mono text-sm break-all">{value}</code>
  <Button variant="outline" size="sm" class="h-auto" onclick={copy}>Copy</Button>
</div>
