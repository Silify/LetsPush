<script>
  import KeyRound from '@lucide/svelte/icons/key-round';
  import Plus from '@lucide/svelte/icons/plus';
  import CopyBox from '../../components/CopyBox.svelte';
  import EmptyState from '../../components/EmptyState.svelte';
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Checkbox } from '$lib/components/ui/checkbox/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { Switch } from '$lib/components/ui/switch/index.js';
  import * as Table from '$lib/components/ui/table/index.js';
  import ConfirmDialog from '../../components/ConfirmDialog.svelte';
  import { api } from '$lib/api.js';
  import { ago, allPermissions, count, permissionHint, permissionLabel, when } from '$lib/format.js';
  import { toast } from '$lib/toast.js';
  import { href as link } from '$lib/router.svelte.js';

  let { app, reload } = $props();

  let showRevoked = $state(false);
  const hasRevoked = $derived(app.accessKeys.some((k) => k.revokedAt));
  const keys = $derived(app.accessKeys.filter((k) => showRevoked || !k.revokedAt));

  // New key
  let creating = $state(false);
  let description = $state('');
  let contact = $state('');
  let permissions = $state([...allPermissions]);
  let keyError = $state('');
  let keyBusy = $state(false);
  let issued = $state(null);
  let showIssued = $state(false);

  function openForm() {
    description = contact = keyError = '';
    permissions = [...allPermissions];
    creating = true;
  }

  function toggle(permission, on) {
    permissions = on ? [...permissions, permission] : permissions.filter((p) => p !== permission);
  }

  async function create(event) {
    event.preventDefault();
    keyBusy = true;
    keyError = '';
    try {
      issued = await api.createKey(app.id, { description: description.trim(), contact: contact.trim(), permissions });
      creating = false;
      showIssued = true;
      await reload();
    } catch (e) {
      keyError = e.message;
    } finally {
      keyBusy = false;
    }
  }

  // Revoke
  let revoking = $state(null);
  let showRevoke = $state(false);

  async function revoke() {
    try {
      await api.revokeKey(app.id, revoking.id);
      showRevoke = false;
      toast('Access key revoked');
      await reload();
    } catch (e) {
      toast(e.message, 'error');
    }
  }
</script>

<Card.Root>
  <Card.Header>
    <Card.Title>Access keys</Card.Title>
    <Card.Description>
      A site needs a key to register devices and send notifications. Give each site its own, so you can revoke one without touching the others.
    </Card.Description>
    <Card.Action><Button onclick={openForm}><Plus data-icon="inline-start" />New access key</Button></Card.Action>
  </Card.Header>
  <Card.Content>
    {#if app.accessKeys.length === 0}
      <EmptyState icon={KeyRound} title="No access keys yet" description="Create one, then give it to the site that will send notifications.">
        <Button onclick={openForm}><Plus data-icon="inline-start" />New access key</Button>
      </EmptyState>
    {:else}
      {#if hasRevoked}
        <div class="mb-3 flex items-center justify-end gap-2">
          <Switch id="show-revoked" bind:checked={showRevoked} />
          <Label for="show-revoked" class="font-normal text-muted-foreground">Show revoked keys</Label>
        </div>
      {/if}
      <Table.Root>
        <Table.Header>
          <Table.Row>
            <Table.Head>Key</Table.Head>
            <Table.Head class="hidden md:table-cell">Can</Table.Head>
            <Table.Head class="text-end">Devices</Table.Head>
            <Table.Head class="hidden sm:table-cell">Last used</Table.Head>
            <Table.Head class="w-24"></Table.Head>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {#each keys as key (key.id)}
            <Table.Row class={key.revokedAt ? 'opacity-50' : ''}>
              <Table.Cell>
                <div class="font-medium">{key.description || 'Untitled'}</div>
                <div class="font-mono text-xs text-muted-foreground">{key.prefix}…</div>
                {#if key.contact}<div class="text-xs text-muted-foreground">{key.contact}</div>{/if}
              </Table.Cell>
              <Table.Cell class="hidden md:table-cell">
                <div class="flex flex-wrap gap-1">
                  {#each key.permissions as permission (permission)}<Badge variant="secondary">{permissionLabel(permission)}</Badge>{/each}
                </div>
              </Table.Cell>
              <Table.Cell class="text-end tabular-nums">
                <a class="hover:underline" href={link(`/apps/${app.id}/devices`)}>{count(key.tokens)}</a>
              </Table.Cell>
              <Table.Cell class="hidden text-muted-foreground sm:table-cell" title={when(key.lastUsedAt)}>{ago(key.lastUsedAt)}</Table.Cell>
              <Table.Cell class="text-end">
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

<Dialog.Root bind:open={creating}>
  <Dialog.Content class="sm:max-w-md">
    <Dialog.Header>
      <Dialog.Title>New access key</Dialog.Title>
      <Dialog.Description>You will see the key once, right after you create it.</Dialog.Description>
    </Dialog.Header>
    <form class="grid gap-4" onsubmit={create}>
      <div class="grid gap-2">
        <Label for="key-description">Description</Label>
        <Input id="key-description" bind:value={description} maxlength={200} placeholder="for example: School A" autocomplete="off" />
      </div>
      <div class="grid gap-2">
        <Label for="key-contact">Contact <span class="font-normal text-muted-foreground">(optional)</span></Label>
        <Input id="key-contact" bind:value={contact} maxlength={200} placeholder="who runs the site" autocomplete="off" />
      </div>
      <fieldset class="grid gap-3">
        <legend class="mb-1 text-sm font-medium">What it can do</legend>
        {#each allPermissions as permission (permission)}
          <div class="flex items-start gap-2.5">
            <Checkbox id={`perm-${permission}`} class="mt-0.5" checked={permissions.includes(permission)} onCheckedChange={(on) => toggle(permission, on)} />
            <Label for={`perm-${permission}`} class="grid gap-0.5 font-normal">
              <span class="font-medium">{permissionLabel(permission)}</span>
              <span class="text-xs text-muted-foreground">{permissionHint(permission)}</span>
            </Label>
          </div>
        {/each}
      </fieldset>
      {#if keyError}
        <Alert.Root variant="destructive" role="alert"><Alert.Description>{keyError}</Alert.Description></Alert.Root>
      {/if}
      <Dialog.Footer>
        <Button type="button" variant="outline" onclick={() => (creating = false)}>Cancel</Button>
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
        <CopyBox value={issued.key} label="Copy the access key" />
        <p class="text-sm text-muted-foreground">
          Give it to the person who runs the site, with the details under <a class="underline" href={link(`/apps/${app.id}/connect`)} onclick={() => (showIssued = false)}>Connect</a>.
        </p>
      </div>
    {/if}
    <Dialog.Footer><Button onclick={() => (showIssued = false)}>I have copied it</Button></Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

<ConfirmDialog bind:open={showRevoke} title="Revoke this key?" confirmLabel="Revoke key" onconfirm={revoke}>
  {#if revoking}
    <b>{revoking.description || 'This key'}</b> (<span class="font-mono">{revoking.prefix}…</span>) will stop working and its
    {revoking.tokens} registered devices will be removed. This cannot be undone.
  {/if}
</ConfirmDialog>
