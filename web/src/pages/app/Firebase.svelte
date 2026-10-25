<script>
  import CircleCheck from '@lucide/svelte/icons/circle-check';
  import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
  import Upload from '@lucide/svelte/icons/upload';
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Textarea } from '$lib/components/ui/textarea/index.js';
  import { api } from '$lib/api.js';
  import { readFileInput } from '$lib/format.js';
  import { toast } from '$lib/toast.js';

  let { app, reload } = $props();

  let replacing = $state(!app.hasCredentials);
  let serviceAccount = $state('');
  let error = $state('');
  let busy = $state(false);

  async function chooseFile(event) {
    serviceAccount = (await readFileInput(event)) || serviceAccount;
  }

  async function save(event) {
    event.preventDefault();
    busy = true;
    error = '';
    try {
      await api.setCredentials(app.id, serviceAccount.trim());
      serviceAccount = '';
      replacing = false;
      toast('Firebase key saved');
      await reload();
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }
</script>

<div class="grid gap-6">
  <Card.Root>
    <Card.Header>
      <Card.Title>Firebase key</Card.Title>
      <Card.Description>LetsPush signs in to Firebase with this key to deliver notifications. It is stored encrypted and never shown again.</Card.Description>
      {#if app.hasCredentials && !replacing}
        <Card.Action><Button variant="outline" size="sm" onclick={() => (replacing = true)}>Replace</Button></Card.Action>
      {/if}
    </Card.Header>
    <Card.Content class="grid gap-4">
      {#if app.hasCredentials}
        <div class="flex items-center gap-2 text-sm">
          <CircleCheck class="size-4 text-primary" />
          Sending through the Firebase project <code class="rounded bg-muted px-1.5 py-0.5 font-mono text-foreground">{app.projectId}</code>
        </div>
      {:else}
        <div class="flex items-center gap-2 text-sm">
          <TriangleAlert class="size-4 text-amber-500" />
          No key yet, so this app cannot send.
        </div>
      {/if}

      {#if replacing}
        <form class="grid gap-3" onsubmit={save}>
          <label class="flex cursor-pointer items-center gap-2 rounded-lg border border-dashed px-3 py-2.5 text-sm text-muted-foreground hover:bg-muted/50">
            <Upload class="size-4" />
            <span class="truncate">{serviceAccount ? 'Key loaded. Choose another file to replace it' : 'Choose the downloaded .json file'}</span>
            <input class="sr-only" type="file" accept="application/json,.json" onchange={chooseFile} />
          </label>
          <Textarea class="max-h-40 min-h-24 font-mono text-xs" bind:value={serviceAccount} placeholder="…or paste the JSON here" spellcheck={false} aria-label="Service-account key" />
          {#if error}
            <Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>
          {/if}
          <div class="flex gap-2">
            <Button type="submit" disabled={busy || !serviceAccount.trim()}>{busy ? 'Saving…' : 'Save key'}</Button>
            {#if app.hasCredentials}<Button type="button" variant="outline" onclick={() => (replacing = false)}>Cancel</Button>{/if}
          </div>
        </form>
      {/if}
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header>
      <Card.Title>Where to get it</Card.Title>
    </Card.Header>
    <Card.Content class="grid gap-4 text-sm">
      <p>In the Firebase console, open <i>Project settings → Service accounts → Generate new private key</i> and upload the file here.</p>
      <div class="grid gap-1.5">
        <p class="font-medium">If a send fails with 403</p>
        <p class="text-muted-foreground">The Activity tab shows the reason Google gave. These are the usual causes:</p>
        <ul class="list-disc space-y-1 ps-5 text-muted-foreground">
          <li>The <b>Firebase Cloud Messaging API (V1)</b> is not enabled for the project in Google Cloud.</li>
          <li>The service account has no role that can send, such as <i>Firebase Cloud Messaging API Admin</i>.</li>
          <li>The device token comes from a different Firebase project than this key.</li>
        </ul>
      </div>
    </Card.Content>
  </Card.Root>
</div>
