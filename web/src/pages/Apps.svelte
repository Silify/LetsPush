<script>
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { Textarea } from '$lib/components/ui/textarea/index.js';
  import { api } from '$lib/api.js';
  import { go } from '$lib/router.svelte.js';
  import { toast } from '$lib/toast.js';

  let apps = $state(null);
  let error = $state('');

  let creating = $state(false);
  let name = $state('');
  let serviceAccount = $state('');
  let busy = $state(false);
  let formError = $state('');

  async function load() {
    try {
      apps = await api.apps();
    } catch (e) {
      error = e.message;
    }
  }
  load();

  function open() {
    name = '';
    serviceAccount = '';
    formError = '';
    creating = true;
  }

  async function readFile(event) {
    const file = event.target.files?.[0];
    if (file) serviceAccount = await file.text();
  }

  async function create(event) {
    event.preventDefault();
    busy = true;
    formError = '';
    try {
      const app = await api.createApp(name.trim(), serviceAccount.trim());
      creating = false;
      toast('App created');
      go(`/apps/${app.id}`);
    } catch (e) {
      formError = e.message;
    } finally {
      busy = false;
    }
  }
</script>

<main class="mx-auto grid max-w-3xl gap-6 px-4 py-8">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div class="grid gap-1">
      <h1 class="text-2xl font-semibold tracking-tight">Apps</h1>
      <p class="text-sm text-muted-foreground">Each app is one mobile app, with its own Firebase key and access keys.</p>
    </div>
    <Button onclick={open}>New app</Button>
  </div>

  {#if error}
    <Alert.Root variant="destructive"><Alert.Description>{error}</Alert.Description></Alert.Root>
  {:else if apps === null}
    <p class="text-sm text-muted-foreground">Loading…</p>
  {:else if apps.length === 0}
    <Card.Root>
      <Card.Content class="grid justify-items-center gap-3 py-10 text-center">
        <h2 class="text-lg font-medium">No apps yet</h2>
        <p class="max-w-sm text-sm text-muted-foreground">
          Add the app you send notifications to. You will need its Firebase service-account key.
        </p>
        <Button onclick={open}>Create your first app</Button>
      </Card.Content>
    </Card.Root>
  {:else}
    {#each apps as app (app.id)}
      <a href={`#/apps/${app.id}`} class="block rounded-xl transition-shadow hover:shadow-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring">
        <Card.Root>
          <Card.Header>
            <Card.Title>{app.name}</Card.Title>
            <Card.Description>{app.projectId ? `Firebase project ${app.projectId}` : 'No Firebase key yet'}</Card.Description>
            <Card.Action>
              {#if app.hasCredentials}
                <Badge>Ready to send</Badge>
              {:else}
                <Badge variant="outline">Needs a Firebase key</Badge>
              {/if}
            </Card.Action>
          </Card.Header>
          <Card.Content class="flex gap-8">
            <div><div class="text-xl font-semibold tracking-tight">{app.keys}</div><div class="text-xs text-muted-foreground">access keys</div></div>
            <div><div class="text-xl font-semibold tracking-tight">{app.tokens}</div><div class="text-xs text-muted-foreground">devices</div></div>
          </Card.Content>
        </Card.Root>
      </a>
    {/each}
  {/if}
</main>

<Dialog.Root bind:open={creating}>
  <Dialog.Content class="sm:max-w-lg">
    <Dialog.Header>
      <Dialog.Title>New app</Dialog.Title>
      <Dialog.Description>An app is one mobile app that notifications are sent to.</Dialog.Description>
    </Dialog.Header>
    <form class="grid gap-4" onsubmit={create}>
      <div class="grid gap-2">
        <Label for="app-name">Name</Label>
        <Input id="app-name" required maxlength={64} bind:value={name} placeholder="for example: sayan" />
        <p class="text-xs text-muted-foreground">
          Sites send this as the app name. Letters, digits, dots, dashes and underscores.
        </p>
      </div>

      <div class="grid gap-2">
        <Label for="app-key">Firebase service-account key <span class="font-normal text-muted-foreground">(optional now, needed to send)</span></Label>
        <Input id="app-key" type="file" accept="application/json,.json" onchange={readFile} />
        <Textarea class="min-h-28 font-mono text-xs" bind:value={serviceAccount} placeholder="…or paste the JSON here" spellcheck={false} />
        <p class="text-xs text-muted-foreground">
          Firebase console → Project settings → Service accounts → Generate new private key. It is stored encrypted.
        </p>
      </div>

      {#if formError}
        <Alert.Root variant="destructive" role="alert"><Alert.Description>{formError}</Alert.Description></Alert.Root>
      {/if}
      <Dialog.Footer>
        <Button type="button" variant="outline" onclick={() => (creating = false)}>Cancel</Button>
        <Button type="submit" disabled={busy}>{busy ? 'Creating…' : 'Create app'}</Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>
