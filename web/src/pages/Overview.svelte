<script>
  import AppWindow from '@lucide/svelte/icons/app-window';
  import ArrowRight from '@lucide/svelte/icons/arrow-right';
  import Rocket from '@lucide/svelte/icons/rocket';
  import Send from '@lucide/svelte/icons/send';
  import Smartphone from '@lucide/svelte/icons/smartphone';
  import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
  import KeyRound from '@lucide/svelte/icons/key-round';
  import Plus from '@lucide/svelte/icons/plus';
  import LogList from '../components/LogList.svelte';
  import NewAppDialog from '../components/NewAppDialog.svelte';
  import PageHeader from '../components/PageHeader.svelte';
  import StatCard from '../components/StatCard.svelte';
  import StatusBadge from '../components/StatusBadge.svelte';
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { api } from '$lib/api.js';
  import { apps } from '$lib/apps.svelte.js';
  import { count } from '$lib/format.js';

  let stats = $state(null);
  let error = $state('');
  let creating = $state(false);

  api
    .overview()
    .then((s) => (stats = s))
    .catch((e) => (error = e.message));

  const steps = [
    { title: 'Create an app', text: 'One per mobile app. Give it its Firebase service-account key.' },
    { title: 'Create an access key', text: 'One per site that sends notifications, so you can revoke each separately.' },
    { title: 'Connect your site', text: 'Enter the values in Moodle, or call the API from your own code.' },
  ];
</script>

<PageHeader title="Overview" description="How your push gateway is doing right now.">
  {#snippet actions()}
    <Button onclick={() => (creating = true)}><Plus data-icon="inline-start" />New app</Button>
  {/snippet}
</PageHeader>

{#if error}
  <Alert.Root variant="destructive" class="mb-6"><Alert.Description>{error}</Alert.Description></Alert.Root>
{/if}

{#if apps.list && apps.list.length === 0}
  <Card.Root class="mb-6">
    <Card.Header>
      <Card.Title class="flex items-center gap-2 text-base"><Rocket class="size-4 text-primary" />Welcome to LetsPush</Card.Title>
      <Card.Description>Three steps and your first notification is on its way.</Card.Description>
    </Card.Header>
    <Card.Content class="grid gap-4">
      <ol class="grid gap-3 sm:grid-cols-3">
        {#each steps as step, i (i)}
          <li class="flex gap-3">
            <span class="grid size-6 shrink-0 place-items-center rounded-full bg-primary/10 text-xs font-semibold text-primary">{i + 1}</span>
            <div>
              <p class="text-sm font-medium">{step.title}</p>
              <p class="text-sm text-muted-foreground">{step.text}</p>
            </div>
          </li>
        {/each}
      </ol>
      <div><Button onclick={() => (creating = true)}><Plus data-icon="inline-start" />Create your first app</Button></div>
    </Card.Content>
  </Card.Root>
{:else if stats && stats.appsNeedingKey > 0}
  <Alert.Root class="mb-6">
    <TriangleAlert />
    <Alert.Title>{stats.appsNeedingKey === 1 ? 'One app cannot send yet' : `${stats.appsNeedingKey} apps cannot send yet`}</Alert.Title>
    <Alert.Description>
      They need a Firebase service-account key. Open the app and add it under <a class="underline" href="#/apps">Firebase</a>.
    </Alert.Description>
  </Alert.Root>
{/if}

<div class="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-5">
  <StatCard label="Apps" value={stats?.apps ?? null} icon={AppWindow} />
  <StatCard label="Access keys" value={stats?.accessKeys ?? null} icon={KeyRound} />
  <StatCard label="Devices" value={stats?.devices ?? null} icon={Smartphone} />
  <StatCard label="Sent (24 h)" value={stats?.sent24h ?? null} icon={Send} />
  <StatCard label="Errors (24 h)" value={stats?.errors24h ?? null} icon={TriangleAlert} tone="danger" />
</div>

<div class="grid gap-6 xl:grid-cols-5">
  <Card.Root class="xl:col-span-2">
    <Card.Header>
      <Card.Title>Your apps</Card.Title>
      <Card.Action><Button variant="ghost" size="sm" href="#/apps">All<ArrowRight data-icon="inline-end" /></Button></Card.Action>
    </Card.Header>
    <Card.Content>
      {#if apps.list === null}
        <p class="text-sm text-muted-foreground">Loading…</p>
      {:else if apps.list.length === 0}
        <p class="text-sm text-muted-foreground">No apps yet.</p>
      {:else}
        <ul class="-my-1 divide-y">
          {#each apps.list.slice(0, 6) as app (app.id)}
            <li>
              <a class="flex items-center justify-between gap-3 py-2.5 hover:text-primary" href="#/apps/{app.id}">
                <div class="min-w-0">
                  <p class="truncate text-sm font-medium">{app.name}</p>
                  <p class="text-xs text-muted-foreground">{count(app.tokens)} devices · {count(app.keys)} keys</p>
                </div>
                <StatusBadge ready={app.hasCredentials} />
              </a>
            </li>
          {/each}
        </ul>
      {/if}
    </Card.Content>
  </Card.Root>

  <Card.Root class="xl:col-span-3">
    <Card.Header>
      <Card.Title>Recent activity</Card.Title>
      <Card.Action><Button variant="ghost" size="sm" href="#/activity">All<ArrowRight data-icon="inline-end" /></Button></Card.Action>
    </Card.Header>
    <Card.Content>
      <LogList load={(o) => api.logs(o)} showApp filters={false} paged={false} size={8} />
    </Card.Content>
  </Card.Root>
</div>

<NewAppDialog bind:open={creating} />
