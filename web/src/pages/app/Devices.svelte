<script>
  import Apple from '@lucide/svelte/icons/apple';
  import Globe from '@lucide/svelte/icons/globe';
  import RefreshCw from '@lucide/svelte/icons/refresh-cw';
  import Smartphone from '@lucide/svelte/icons/smartphone';
  import Trash2 from '@lucide/svelte/icons/trash-2';
  import ConfirmDialog from '../../components/ConfirmDialog.svelte';
  import EmptyState from '../../components/EmptyState.svelte';
  import FilterSelect from '../../components/FilterSelect.svelte';
  import { Badge } from '$lib/components/ui/badge/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import * as Table from '$lib/components/ui/table/index.js';
  import { api } from '$lib/api.js';
  import { ago, count, when } from '$lib/format.js';
  import { toast } from '$lib/toast.js';

  let { app, reload } = $props();

  const size = 50;

  let items = $state(null);
  let next = $state(null);
  let error = $state('');
  let busy = $state(false);
  let platform = $state('');

  // What the sender called the platform: android, ios, ios-fcm, web, or a name of its own.
  function describe(name) {
    const lower = name.toLowerCase();
    if (lower.startsWith('ios')) return { label: 'iOS', icon: Apple };
    if (lower === 'android') return { label: 'Android', icon: Smartphone };
    if (lower === 'web') return { label: 'Web', icon: Globe };
    return { label: name, icon: Smartphone };
  }

  async function load(more = false) {
    busy = true;
    try {
      const page = await api.devices(app.id, { limit: size, before: more ? next : undefined });
      items = more ? [...items, ...page.items] : page.items;
      next = page.next;
      error = '';
    } catch (e) {
      error = e.message;
      items ??= [];
    } finally {
      busy = false;
    }
  }
  load();

  const platforms = $derived([
    { value: '', label: 'All platforms' },
    ...[...new Set((items ?? []).map((d) => d.platform))].sort().map((p) => ({ value: p, label: describe(p).label })),
  ]);
  const shown = $derived((items ?? []).filter((d) => !platform || d.platform === platform));

  let removing = $state(null);
  let showRemove = $state(false);

  async function remove() {
    try {
      await api.removeDevice(app.id, removing.id);
      items = items.filter((d) => d.id !== removing.id);
      showRemove = false;
      toast('Device removed');
      reload();
    } catch (e) {
      toast(e.message, 'error');
    }
  }
</script>

<Card.Root>
  <Card.Header>
    <Card.Title>Devices</Card.Title>
    <Card.Description>
      Phones and browsers that can receive this app's notifications. They register through a site's access key; only the start of each token is shown.
    </Card.Description>
    <Card.Action>
      <Button variant="outline" size="sm" onclick={() => load()} disabled={busy}>
        <RefreshCw class={busy ? 'animate-spin' : ''} data-icon="inline-start" />Refresh
      </Button>
    </Card.Action>
  </Card.Header>
  <Card.Content class="grid gap-3">
    {#if error}
      <p class="text-sm text-destructive" role="alert">{error}</p>
    {/if}
    {#if items === null}
      {#each [0, 1, 2] as row (row)}<Skeleton class="h-10 w-full" />{/each}
    {:else if items.length === 0}
      <EmptyState
        icon={Smartphone}
        title="No devices yet"
        description="A device appears when the mobile app signs in to a site that uses one of this app's access keys."
      />
    {:else}
      <div class="flex flex-wrap items-center justify-between gap-2">
        <FilterSelect bind:value={platform} options={platforms} label="All platforms" />
        <span class="text-sm text-muted-foreground">{count(app.tokens)} in total</span>
      </div>
      <Table.Root>
        <Table.Header>
          <Table.Row>
            <Table.Head>Device</Table.Head>
            <Table.Head class="hidden sm:table-cell">Registered by</Table.Head>
            <Table.Head class="hidden md:table-cell">Last seen</Table.Head>
            <Table.Head class="w-12"></Table.Head>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {#each shown as device (device.id)}
            {@const kind = describe(device.platform)}
            <Table.Row>
              <Table.Cell>
                <div class="flex items-center gap-2.5">
                  <kind.icon class="size-4 shrink-0 text-muted-foreground" />
                  <div class="min-w-0">
                    <div class="text-sm font-medium">{kind.label}</div>
                    <div class="font-mono text-xs text-muted-foreground">{device.token}</div>
                  </div>
                </div>
              </Table.Cell>
              <Table.Cell class="hidden sm:table-cell">
                <div class="text-sm">{device.key.description || 'Untitled'}</div>
                <div class="flex items-center gap-1.5 font-mono text-xs text-muted-foreground">
                  {device.key.prefix}…
                  {#if device.key.revoked}<Badge variant="destructive">Revoked</Badge>{/if}
                </div>
              </Table.Cell>
              <Table.Cell class="hidden text-muted-foreground md:table-cell" title={when(device.updatedAt)}>{ago(device.updatedAt)}</Table.Cell>
              <Table.Cell class="text-end">
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label="Remove this device"
                  onclick={() => {
                    removing = device;
                    showRemove = true;
                  }}><Trash2 /></Button
                >
              </Table.Cell>
            </Table.Row>
          {/each}
        </Table.Body>
      </Table.Root>
      {#if shown.length === 0}
        <p class="py-2 text-center text-sm text-muted-foreground">No loaded device is on this platform.</p>
      {/if}
      {#if next}
        <div class="flex justify-center">
          <Button variant="outline" size="sm" onclick={() => load(true)} disabled={busy}>{busy ? 'Loading…' : 'Load more'}</Button>
        </div>
      {/if}
    {/if}
  </Card.Content>
</Card.Root>

<ConfirmDialog bind:open={showRemove} title="Remove this device?" confirmLabel="Remove device" onconfirm={remove}>
  {#if removing}
    The device <span class="font-mono">{removing.token}</span> will stop receiving notifications until the mobile app registers it again.
  {/if}
</ConfirmDialog>
