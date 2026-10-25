<script>
  import AppWindow from '@lucide/svelte/icons/app-window';
  import KeyRound from '@lucide/svelte/icons/key-round';
  import Plus from '@lucide/svelte/icons/plus';
  import Search from '@lucide/svelte/icons/search';
  import Smartphone from '@lucide/svelte/icons/smartphone';
  import EmptyState from '../components/EmptyState.svelte';
  import NewAppDialog from '../components/NewAppDialog.svelte';
  import PageHeader from '../components/PageHeader.svelte';
  import StatusBadge from '../components/StatusBadge.svelte';
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import { apps, loadApps } from '$lib/apps.svelte.js';
  import { ago, count } from '$lib/format.js';

  loadApps();

  let creating = $state(false);
  let search = $state('');

  const shown = $derived(
    (apps.list ?? []).filter((app) => app.name.toLowerCase().includes(search.trim().toLowerCase())),
  );
</script>

<PageHeader title="Apps" description="Each app is a mobile app with its own Firebase key, access keys and devices.">
  {#snippet actions()}
    <Button onclick={() => (creating = true)}><Plus data-icon="inline-start" />New app</Button>
  {/snippet}
</PageHeader>

{#if apps.error}
  <Alert.Root variant="destructive"><Alert.Description>{apps.error}</Alert.Description></Alert.Root>
{:else if apps.list === null}
  <div class="grid gap-3 sm:grid-cols-2">
    {#each [0, 1] as card (card)}<Skeleton class="h-32 w-full" />{/each}
  </div>
{:else if apps.list.length === 0}
  <Card.Root>
    <EmptyState icon={AppWindow} title="No apps yet" description="Create an app, give it its Firebase key, and sites can start sending notifications.">
      <Button onclick={() => (creating = true)}><Plus data-icon="inline-start" />New app</Button>
    </EmptyState>
  </Card.Root>
{:else}
  {#if apps.list.length > 4}
    <div class="relative mb-4 max-w-xs">
      <Search class="pointer-events-none absolute start-2.5 top-2.5 size-4 text-muted-foreground" />
      <Input class="ps-8" bind:value={search} placeholder="Search apps" aria-label="Search apps" />
    </div>
  {/if}
  {#if shown.length === 0}
    <p class="text-sm text-muted-foreground">No app matches “{search}”.</p>
  {:else}
    <div class="grid gap-3 sm:grid-cols-2">
      {#each shown as app (app.id)}
        <a href="#/apps/{app.id}" class="group block rounded-xl focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none">
          <Card.Root class="h-full transition-colors group-hover:bg-muted/40">
            <Card.Header>
              <Card.Title class="truncate">{app.name}</Card.Title>
              <Card.Description class="truncate">
                {app.hasCredentials ? `Firebase project ${app.projectId}` : 'No Firebase key yet'}
              </Card.Description>
              <Card.Action><StatusBadge ready={app.hasCredentials} /></Card.Action>
            </Card.Header>
            <Card.Content class="flex flex-wrap items-center gap-x-5 gap-y-1 text-sm text-muted-foreground">
              <span class="flex items-center gap-1.5"><KeyRound class="size-3.5" />{count(app.keys)} access keys</span>
              <span class="flex items-center gap-1.5"><Smartphone class="size-3.5" />{count(app.tokens)} devices</span>
              <span class="ms-auto text-xs">Created {ago(app.createdAt)}</span>
            </Card.Content>
          </Card.Root>
        </a>
      {/each}
    </div>
  {/if}
{/if}

<NewAppDialog bind:open={creating} />
