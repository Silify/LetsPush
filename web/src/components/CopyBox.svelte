<script>
  import { copyText } from '../lib/format.js';
  import { toast } from '../lib/toast.svelte.js';

  let { value, label = 'Copy' } = $props();

  async function copy() {
    toast((await copyText(value)) ? 'Copied' : 'Select the text and copy it', 'ok');
  }
</script>

<div class="copybox">
  <code>{value}</code>
  <button class="btn small" type="button" onclick={copy}>{label}</button>
</div>
