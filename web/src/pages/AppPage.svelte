<script>
  import Activity from '@lucide/svelte/icons/activity';
  import Flame from '@lucide/svelte/icons/flame';
  import KeyRound from '@lucide/svelte/icons/key-round';
  import LayoutDashboard from '@lucide/svelte/icons/layout-dashboard';
  import Plug from '@lucide/svelte/icons/plug';
  import Settings from '@lucide/svelte/icons/settings';
  import Smartphone from '@lucide/svelte/icons/smartphone';
  import PageHeader from '../components/PageHeader.svelte';
  import StatusBadge from '../components/StatusBadge.svelte';
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import * as Tabs from '$lib/components/ui/tabs/index.js';
  import { api } from '$lib/api.js';
  import { loadApps } from '$lib/apps.svelte.js';
  import { go, route } from '$lib/router.svelte.js';

  // One app, with its tabs. The app is loaded once here and shared with the tab that is open; a tab that
  // changes something calls reload() so the counts and the status stay right.
  const id = route.id;

  const tabs = [
    { name: 'overview', label: 'Overview', icon: LayoutDashboard, load: () => import('./app/Overview.svelte') },
    { name: 'keys', label: 'Access keys', icon: KeyRound, load: () => import('./app/Keys.svelte') },
    { name: 'devices', label: 'Devices', icon: Smartphone, load: () => import('./app/Devices.svelte') },
    { name: 'activity', label: 'Activity', icon: Activity, load: () => import('./app/Activity.svelte') },
    { name: 'firebase', label: 'Firebase', icon: Flame, load: () => import('./app/Firebase.svelte') },
    { name: 'connect', label: 'Connect', icon: Plug, load: () => import('./app/Connect.svelte') },
    { name: 'settings', label: 'Settings', icon: Settings, load: () => import('./app/Settings.svelte') },
  ];

  let app = $state(null);
  let error = $state('');

  async function reload() {
    try {
      app = await api.app(id);
      loadApps();
    } catch (e) {
      error = e.message;
    }
  }
  reload();

  const open = $derived(tabs.find((t) => t.name === route.tab) ?? tabs[0]);

  function choose(name) {
    go(name === 'overview' ? `/apps/${id}` : `/apps/${id}/${name}`);
  }
</script>

{#if error}
  <PageHeader title="App" crumbs={[{ label: 'Apps', href: '#/apps' }]} />
  <Alert.Root variant="destructive"><Alert.Description>{error}</Alert.Description></Alert.Root>
{:else if app === null}
  <div class="grid gap-4">
    <Skeleton class="h-4 w-24" />
    <Skeleton class="h-8 w-56" />
    <Skeleton class="h-9 w-full max-w-xl" />
    <Skeleton class="h-40 w-full" />
  </div>
{:else}
  <PageHeader
    title={app.name}
    description={app.hasCredentials ? `Firebase project ${app.projectId}` : 'Add a Firebase key to start sending.'}
    crumbs={[{ label: 'Apps', href: '#/apps' }, { label: app.name }]}
  >
    {#snippet actions()}<StatusBadge ready={app.hasCredentials} />{/snippet}
  </PageHeader>

  <Tabs.Root value={open.name} onValueChange={choose} class="mb-6">
    <div class="-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0">
      <Tabs.List variant="line" class="w-max border-b">
        {#each tabs as tab (tab.name)}
          <Tabs.Trigger value={tab.name} class="gap-1.5"><tab.icon class="size-4" />{tab.label}</Tabs.Trigger>
        {/each}
      </Tabs.List>
    </div>
  </Tabs.Root>

  {#key open.name}
    {#await open.load()}
      <Skeleton class="h-40 w-full" />
    {:then module}
      {@const Tab = module.default}
      <Tab {app} {reload} />
    {:catch}
      <div class="flex items-center gap-3 text-sm" role="alert">
        <span class="text-destructive">This tab could not be loaded. If LetsPush was just updated, reload.</span>
        <Button variant="outline" size="sm" onclick={() => location.reload()}>Reload</Button>
      </div>
    {/await}
  {/key}
{/if}
