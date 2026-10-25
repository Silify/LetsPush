<script>
  import Upload from '@lucide/svelte/icons/upload';
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { Textarea } from '$lib/components/ui/textarea/index.js';
  import { api } from '$lib/api.js';
  import { loadApps } from '$lib/apps.svelte.js';
  import { readFileInput } from '$lib/format.js';
  import { go } from '$lib/router.svelte.js';
  import { toast } from '$lib/toast.js';

  /** The form for a new app. It opens the new app when it is made. */
  let { open = $bindable(false) } = $props();

  let name = $state('');
  let serviceAccount = $state('');
  let busy = $state(false);
  let error = $state('');

  const nameOk = $derived(/^[A-Za-z0-9._-]{1,64}$/.test(name.trim()));

  $effect(() => {
    if (open) {
      name = serviceAccount = error = '';
    }
  });

  async function chooseFile(event) {
    serviceAccount = (await readFileInput(event)) || serviceAccount;
  }

  async function create(event) {
    event.preventDefault();
    busy = true;
    error = '';
    try {
      const app = await api.createApp(name.trim(), serviceAccount.trim());
      open = false;
      toast('App created');
      await loadApps();
      go(`/apps/${app.id}`);
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <Dialog.Title>New app</Dialog.Title>
      <Dialog.Description>An app is one mobile app and its Firebase project. Sites send notifications to it.</Dialog.Description>
    </Dialog.Header>
    <form class="grid gap-4" onsubmit={create}>
      <div class="grid gap-2">
        <Label for="app-name">Name</Label>
        <Input id="app-name" bind:value={name} maxlength={64} placeholder="for example: sayan" autocomplete="off" required />
        <p class="text-xs text-muted-foreground">
          Sites send this name with every request, so keep it short. Letters, digits, dots, dashes and underscores.
        </p>
      </div>
      <div class="grid gap-2">
        <Label for="app-key">
          Firebase service-account key <span class="font-normal text-muted-foreground">(optional now, needed to send)</span>
        </Label>
        <label class="flex cursor-pointer items-center gap-2 rounded-lg border border-dashed px-3 py-2.5 text-sm text-muted-foreground hover:bg-muted/50">
          <Upload class="size-4" />
          <span class="truncate">{serviceAccount ? 'Key loaded. Choose another file to replace it' : 'Choose the downloaded .json file'}</span>
          <input class="sr-only" type="file" accept="application/json,.json" onchange={chooseFile} />
        </label>
        <Textarea id="app-key" class="max-h-32 min-h-20 font-mono text-xs" bind:value={serviceAccount} placeholder="…or paste the JSON here" spellcheck={false} />
        <p class="text-xs text-muted-foreground">
          Firebase console → Project settings → Service accounts → Generate new private key. It is stored encrypted and never shown again.
        </p>
      </div>
      {#if error}
        <Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>
      {/if}
      <Dialog.Footer>
        <Button type="button" variant="outline" onclick={() => (open = false)}>Cancel</Button>
        <Button type="submit" disabled={busy || !nameOk}>{busy ? 'Creating…' : 'Create app'}</Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
