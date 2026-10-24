<script>
  import CopyBox from '../components/CopyBox.svelte';
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Checkbox } from '$lib/components/ui/checkbox/index.js';
  import * as Collapsible from '$lib/components/ui/collapsible/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { Switch } from '$lib/components/ui/switch/index.js';
  import * as Table from '$lib/components/ui/table/index.js';
  import { Textarea } from '$lib/components/ui/textarea/index.js';
  import { api } from '$lib/api.js';
  import { ago, permissionLabel, when } from '$lib/format.js';
  import { go } from '$lib/router.svelte.js';
  import { toast } from '$lib/toast.js';

  let { id } = $props();

  let app = $state(null);
  let logs = $state([]);
  let error = $state('');

  const serverUrl = `${location.protocol}//${location.hostname}`;
  const serverPort = location.port || (location.protocol === 'https:' ? '443' : '80');

  async function load() {
    try {
      app = await api.app(id);
      logs = await api.logs(id);
    } catch (e) {
      error = e.message;
    }
  }
  load();

  // Firebase key
  let replacing = $state(false);
  let serviceAccount = $state('');
  let credentialsError = $state('');
  let credentialsBusy = $state(false);

  async function readFile(event) {
    const file = event.target.files?.[0];
    if (file) serviceAccount = await file.text();
  }

  async function saveCredentials(event) {
    event.preventDefault();
    credentialsBusy = true;
    credentialsError = '';
    try {
      await api.setCredentials(id, serviceAccount.trim());
      replacing = false;
      serviceAccount = '';
      toast('Firebase key saved');
      await load();
    } catch (e) {
      credentialsError = e.message;
    } finally {
      credentialsBusy = false;
    }
  }

  // Access keys
  const allPermissions = ['createToken', 'deleteToken', 'sendNotification'];
  let creatingKey = $state(false);
  let description = $state('');
  let contact = $state('');
  let permissions = $state([...allPermissions]);
  let keyError = $state('');
  let keyBusy = $state(false);
  let issued = $state(null);
  let showIssued = $state(false);

  function openKeyForm() {
    description = contact = keyError = '';
    permissions = [...allPermissions];
    creatingKey = true;
  }

  function togglePermission(permission, on) {
    permissions = on ? [...permissions, permission] : permissions.filter((p) => p !== permission);
  }

  async function createKey(event) {
    event.preventDefault();
    keyBusy = true;
    keyError = '';
    try {
      issued = await api.createKey(id, { description, contact, permissions });
      creatingKey = false;
      showIssued = true;
      await load();
    } catch (e) {
      keyError = e.message;
    } finally {
      keyBusy = false;
    }
  }

  let revoking = $state(null);
  let showRevoke = $state(false);

  async function revoke() {
    try {
      await api.revokeKey(id, revoking.id);
      showRevoke = false;
      toast('Access key revoked');
      await load();
    } catch (e) {
      toast(e.message, 'error');
    }
  }

  // Settings and deleting
  async function setRequests(allow) {
    try {
      await api.setSettings(id, allow);
      app.allowKeyRequests = allow;
      toast(allow ? 'Sites may now request their own key' : 'Sites can no longer request a key');
    } catch (e) {
      toast(e.message, 'error');
    }
  }

  let deleting = $state(false);
  let confirmName = $state('');

  async function remove() {
    try {
      await api.deleteApp(id);
      deleting = false;
      toast('App deleted');
      go('/');
    } catch (e) {
      toast(e.message, 'error');
    }
  }
</script>

<main class="mx-auto grid max-w-3xl gap-6 px-4 py-8">
  <a class="text-sm text-muted-foreground hover:text-foreground" href="#/">← Apps</a>

  {#if error}
    <Alert.Root variant="destructive"><Alert.Description>{error}</Alert.Description></Alert.Root>
  {:else if app === null}
    <p class="text-sm text-muted-foreground">Loading…</p>
  {:else}
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="grid gap-2">
        <h1 class="text-2xl font-semibold tracking-tight">{app.name}</h1>
        <div class="flex gap-8">
          <div><span class="text-xl font-semibold tracking-tight">{app.keys}</span> <span class="text-sm text-muted-foreground">access keys</span></div>
          <div><span class="text-xl font-semibold tracking-tight">{app.tokens}</span> <span class="text-sm text-muted-foreground">devices</span></div>
        </div>
      </div>
      {#if app.hasCredentials}
        <Badge>Ready to send</Badge>
      {:else}
        <Badge variant="outline">Needs a Firebase key</Badge>
      {/if}
    </div>

    <Card.Root>
      <Card.Header>
        <Card.Title>Firebase key</Card.Title>
        <Card.Description>
          {#if app.hasCredentials}
            Sending through the Firebase project <code class="font-mono text-foreground">{app.projectId}</code>. The key is stored encrypted and never shown again.
          {:else}
            Add the project's service-account key to start sending notifications.
          {/if}
        </Card.Description>
        <Card.Action>
          <Button variant="outline" size="sm" onclick={() => (replacing = !replacing)}>
            {app.hasCredentials ? 'Replace' : 'Add key'}
          </Button>
        </Card.Action>
      </Card.Header>
      {#if replacing}
        <Card.Content>
          <form class="grid gap-3" onsubmit={saveCredentials}>
            <Label for="sa-file">Service-account key</Label>
            <Input id="sa-file" type="file" accept="application/json,.json" onchange={readFile} />
            <Textarea class="min-h-28 font-mono text-xs" bind:value={serviceAccount} placeholder="…or paste the JSON here" spellcheck={false} />
            {#if credentialsError}
              <Alert.Root variant="destructive" role="alert"><Alert.Description>{credentialsError}</Alert.Description></Alert.Root>
            {/if}
            <div class="flex gap-2">
              <Button type="submit" disabled={credentialsBusy || !serviceAccount.trim()}>
                {credentialsBusy ? 'Saving…' : 'Save key'}
              </Button>
              <Button type="button" variant="outline" onclick={() => (replacing = false)}>Cancel</Button>
            </div>
          </form>
        </Card.Content>
      {/if}
    </Card.Root>

    <Card.Root>
      <Card.Header>
        <Card.Title>Access keys</Card.Title>
        <Card.Description>A site needs one to register devices and send notifications.</Card.Description>
        <Card.Action><Button size="sm" onclick={openKeyForm}>New access key</Button></Card.Action>
      </Card.Header>
      <Card.Content>
        {#if app.accessKeys.length === 0}
          <p class="text-sm text-muted-foreground">No access keys yet.</p>
        {:else}
          <Table.Root>
            <Table.Header>
              <Table.Row>
                <Table.Head>Key</Table.Head>
                <Table.Head class="hidden sm:table-cell">Can</Table.Head>
                <Table.Head>Devices</Table.Head>
                <Table.Head class="hidden sm:table-cell">Last used</Table.Head>
                <Table.Head class="text-right"></Table.Head>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {#each app.accessKeys as key (key.id)}
                <Table.Row class={key.revokedAt ? 'opacity-50' : ''}>
                  <Table.Cell>
                    <div class="font-medium">{key.description || 'Untitled'}</div>
                    <div class="font-mono text-xs text-muted-foreground">{key.prefix}…</div>
                  </Table.Cell>
                  <Table.Cell class="hidden sm:table-cell">
                    <div class="flex flex-wrap gap-1">
                      {#each key.permissions as permission}<Badge variant="secondary">{permissionLabel(permission)}</Badge>{/each}
                    </div>
                  </Table.Cell>
                  <Table.Cell>{key.tokens}</Table.Cell>
                  <Table.Cell class="hidden text-muted-foreground sm:table-cell">{ago(key.lastUsedAt)}</Table.Cell>
                  <Table.Cell class="text-right">
                    {#if key.revokedAt}
                      <Badge variant="destructive">Revoked</Badge>
                    {:else}
                      <Button
                        variant="destructive"
                        size="sm"
                        onclick={() => {
                          revoking = key;
                          showRevoke = true;
                        }}>Revoke</Button
                      >
                    {/if}
                  </Table.Cell>
                </Table.Row>
              {/each}
            </Table.Body>
          </Table.Root>
        {/if}
      </Card.Content>
    </Card.Root>

    <Card.Root>
      <Collapsible.Root>
        <Card.Header>
          <Card.Title>Connect a Moodle site</Card.Title>
          <Card.Description>The values to enter in Moodle's mobile notification settings.</Card.Description>
          <Card.Action>
            <Collapsible.Trigger>
              {#snippet child({ props })}<Button variant="outline" size="sm" {...props}>Show</Button>{/snippet}
            </Collapsible.Trigger>
          </Card.Action>
        </Card.Header>
        <Collapsible.Content>
          <Card.Content class="grid gap-4 pt-4">
            <p class="text-sm text-muted-foreground">
              In Moodle, open <i>Site administration → Messaging → Mobile</i> and fill in the push server fields:
            </p>
            <dl class="grid grid-cols-[max-content_1fr] items-center gap-x-4 gap-y-3 text-sm">
              <dt class="text-muted-foreground">Server URL</dt>
              <dd><CopyBox value={serverUrl} /></dd>
              <dt class="text-muted-foreground">Port</dt>
              <dd><CopyBox value={serverPort} /></dd>
              <dt class="text-muted-foreground">App name</dt>
              <dd><CopyBox value={app.name} /></dd>
              <dt class="text-muted-foreground">Access key</dt>
              <dd class="text-muted-foreground">One of the keys above (shown only when it is created)</dd>
              <dt class="text-muted-foreground">Mobile app name</dt>
              <dd class="text-muted-foreground">Your app's package id, for example <code class="font-mono">org.silify.sayan</code></dd>
            </dl>
            <p class="text-xs text-muted-foreground">Then enable the mobile notifications output under <i>Plugins → Message outputs</i>.</p>
          </Card.Content>
        </Collapsible.Content>
      </Collapsible.Root>
    </Card.Root>

    <Card.Root>
      <Card.Header>
        <Card.Title>Recent activity</Card.Title>
        <Card.Action><Button variant="outline" size="sm" onclick={load}>Refresh</Button></Card.Action>
      </Card.Header>
      <Card.Content>
        {#if logs.length === 0}
          <p class="text-sm text-muted-foreground">Nothing yet. Sends, new devices and key changes appear here.</p>
        {:else}
          <ul class="divide-y text-sm">
            {#each logs as entry}
              <li class="grid gap-1 py-2 sm:grid-cols-[10rem_5.5rem_1fr] sm:gap-3">
                <span class="text-muted-foreground">{when(entry.at)}</span>
                <span><Badge variant={entry.action === 'error' ? 'destructive' : 'secondary'}>{entry.action}</Badge></span>
                <span class="break-words">{entry.info}</span>
              </li>
            {/each}
          </ul>
        {/if}
      </Card.Content>
    </Card.Root>

    <Card.Root>
      <Card.Header>
        <Card.Title>Settings</Card.Title>
      </Card.Header>
      <Card.Content>
        <div class="flex items-start justify-between gap-4">
          <div class="grid gap-1">
            <Label for="allow-requests">Let sites request their own access key</Label>
            <p class="max-w-prose text-sm text-muted-foreground">
              Turns on the "Request access key" button in Moodle. Anyone who can reach this server could then get a key, so leave
              it off unless you need it.
            </p>
          </div>
          <Switch id="allow-requests" checked={app.allowKeyRequests} onCheckedChange={setRequests} />
        </div>
      </Card.Content>
    </Card.Root>

    <Card.Root class="border-destructive/40">
      <Card.Header>
        <Card.Title>Delete this app</Card.Title>
        <Card.Description>Removes its access keys, devices and activity. Sites using it stop working.</Card.Description>
        <Card.Action>
          <Button
            variant="destructive"
            onclick={() => {
              confirmName = '';
              deleting = true;
            }}>Delete…</Button
          >
        </Card.Action>
      </Card.Header>
    </Card.Root>
  {/if}
</main>

<Dialog.Root bind:open={creatingKey}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>New access key</Dialog.Title>
      <Dialog.Description>Give each site its own key, so you can revoke one without affecting the others.</Dialog.Description>
    </Dialog.Header>
    <form class="grid gap-4" onsubmit={createKey}>
      <div class="grid gap-2">
        <Label for="key-description">Description</Label>
        <Input id="key-description" bind:value={description} maxlength={200} placeholder="for example: School A" />
      </div>
      <div class="grid gap-2">
        <Label for="key-contact">Contact <span class="font-normal text-muted-foreground">(optional)</span></Label>
        <Input id="key-contact" bind:value={contact} maxlength={200} placeholder="who runs the site" />
      </div>
      <fieldset class="grid gap-3">
        <legend class="mb-1 text-sm font-medium">What it can do</legend>
        {#each allPermissions as permission}
          <div class="flex items-center gap-2">
            <Checkbox
              id={`perm-${permission}`}
              checked={permissions.includes(permission)}
              onCheckedChange={(on) => togglePermission(permission, on)}
            />
            <Label for={`perm-${permission}`} class="font-normal">{permissionLabel(permission)}</Label>
          </div>
        {/each}
      </fieldset>
      {#if keyError}
        <Alert.Root variant="destructive" role="alert"><Alert.Description>{keyError}</Alert.Description></Alert.Root>
      {/if}
      <Dialog.Footer>
        <Button type="button" variant="outline" onclick={() => (creatingKey = false)}>Cancel</Button>
        <Button type="submit" disabled={keyBusy || permissions.length === 0}>{keyBusy ? 'Creating…' : 'Create key'}</Button>
      </Dialog.Footer>
    </form>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showIssued}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Copy your access key</Dialog.Title>
      <Dialog.Description>This is the only time the key is shown. Copy it now and keep it secret.</Dialog.Description>
    </Dialog.Header>
    {#if issued}
      <div class="grid gap-3">
        <CopyBox value={issued.key} />
        <p class="text-sm text-muted-foreground">
          Give it to the person who runs the site, together with the details under "Connect a Moodle site".
        </p>
      </div>
    {/if}
    <Dialog.Footer>
      <Button onclick={() => (showIssued = false)}>I have copied it</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={showRevoke}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Revoke this key?</Dialog.Title>
      {#if revoking}
        <Dialog.Description>
          <b>{revoking.description || 'This key'}</b> (<span class="font-mono">{revoking.prefix}…</span>) will stop working and its
          {revoking.tokens} registered devices will be removed. This cannot be undone.
        </Dialog.Description>
      {/if}
    </Dialog.Header>
    <Dialog.Footer>
      <Button variant="outline" onclick={() => (showRevoke = false)}>Cancel</Button>
      <Button variant="destructive" onclick={revoke}>Revoke key</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<Dialog.Root bind:open={deleting}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>Delete this app?</Dialog.Title>
      {#if app}
        <Dialog.Description>This removes <b>{app.name}</b> with everything under it. Type its name to confirm.</Dialog.Description>
      {/if}
    </Dialog.Header>
    {#if app}
      <form
        class="grid gap-4"
        onsubmit={(event) => {
          event.preventDefault();
          remove();
        }}
      >
        <Input bind:value={confirmName} placeholder={app.name} autocomplete="off" />
        <Dialog.Footer>
          <Button type="button" variant="outline" onclick={() => (deleting = false)}>Cancel</Button>
          <Button type="submit" variant="destructive" disabled={confirmName !== app.name}>Delete app</Button>
        </Dialog.Footer>
      </form>
    {/if}
  </Dialog.Content>
</Dialog.Root>
