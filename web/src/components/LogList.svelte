<script>
  import Activity from '@lucide/svelte/icons/activity';
  import KeyRound from '@lucide/svelte/icons/key-round';
  import RefreshCw from '@lucide/svelte/icons/refresh-cw';
  import Send from '@lucide/svelte/icons/send';
  import Settings from '@lucide/svelte/icons/settings';
  import Smartphone from '@lucide/svelte/icons/smartphone';
  import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
  import EmptyState from './EmptyState.svelte';
  import FilterSelect from './FilterSelect.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import { Switch } from '$lib/components/ui/switch/index.js';
  import { ago, clock, when } from '$lib/format.js';

  /**
   * The activity log. `load({ action, before, limit })` returns { items, next }.
   * Give `filters` to let the person narrow it, `live` to offer refreshing by itself, and `paged={false}` to
   * show only the first page, without "Load more".
   * `reloadKey` changes to make it start over (for example when another app is chosen).
   */
  let { load, showApp = false, filters = true, live = false, paged = true, size = 30, reloadKey = '', children } = $props();

  const actions = [
    { value: '', label: 'All activity' },
    { value: 'push', label: 'Sent' },
    { value: 'error', label: 'Errors' },
    { value: 'token', label: 'Devices' },
    { value: 'accesskey', label: 'Access keys' },
    { value: 'app', label: 'App changes' },
  ];

  const icons = { push: Send, error: TriangleAlert, token: Smartphone, accesskey: KeyRound, app: Settings };

  let action = $state('');
  let items = $state(null);
  let next = $state(null);
  let error = $state('');
  let loadingMore = $state(false);
  let refreshing = $state(false);
  let auto = $state(false);

  async function refresh() {
    refreshing = true;
    try {
      const page = await load({ action, limit: size });
      items = page.items;
      next = page.next;
      error = '';
    } catch (e) {
      error = e.message;
      items ??= [];
    } finally {
      refreshing = false;
    }
  }

  async function more() {
    loadingMore = true;
    try {
      const page = await load({ action, limit: size, before: next });
      items = [...items, ...page.items];
      next = page.next;
    } catch (e) {
      error = e.message;
    } finally {
      loadingMore = false;
    }
  }

  // Start over when the filter or the key changes.
  $effect(() => {
    action;
    reloadKey;
    items = null;
    refresh();
  });

  // Refresh by itself while the page is open and looked at.
  $effect(() => {
    if (!auto) return;
    const timer = setInterval(() => {
      if (!document.hidden) refresh();
    }, 10_000);
    return () => clearInterval(timer);
  });
</script>

<div class="grid gap-3">
  <div class="flex flex-wrap items-center gap-2">
    {#if filters}<FilterSelect bind:value={action} options={actions} label="All activity" />{/if}
    {@render children?.()}
    <div class="ms-auto flex items-center gap-3" class:hidden={!filters && !live}>
      {#if live}
        <div class="flex items-center gap-2">
          <Switch id="auto-refresh" bind:checked={auto} />
          <Label for="auto-refresh" class="font-normal text-muted-foreground">Live</Label>
        </div>
      {/if}
      <Button variant="outline" size="sm" onclick={refresh} disabled={refreshing}>
        <RefreshCw class={refreshing ? 'animate-spin' : ''} data-icon="inline-start" />Refresh
      </Button>
    </div>
  </div>

  {#if error}
    <p class="text-sm text-destructive" role="alert">{error}</p>
  {/if}

  {#if items === null}
    <div class="grid gap-2">
      {#each [0, 1, 2, 3] as row (row)}<Skeleton class="h-11 w-full" />{/each}
    </div>
  {:else if items.length === 0}
    <EmptyState
      icon={Activity}
      title={action ? 'Nothing of this kind yet' : 'No activity yet'}
      description="Sends, new devices, errors and key changes appear here as they happen."
    />
  {:else}
    <ul class="divide-y rounded-lg border">
      {#each items as entry (entry.id)}
        {@const Icon = icons[entry.action] ?? Activity}
        <li class="flex items-start gap-3 px-3 py-2.5 text-sm">
          <div
            class="mt-0.5 grid size-6 shrink-0 place-items-center rounded-md {entry.action === 'error'
              ? 'bg-destructive/10 text-destructive'
              : 'bg-muted text-muted-foreground'}"
          >
            <Icon class="size-3.5" />
          </div>
          <div class="min-w-0 flex-1">
            <p class="break-words">{entry.info}</p>
            {#if showApp && entry.appName}
              <a class="text-xs text-muted-foreground hover:text-foreground" href="#/apps/{entry.appId}">{entry.appName}</a>
            {/if}
          </div>
          <time class="shrink-0 text-xs text-muted-foreground" datetime={entry.at} title={when(entry.at)}>
            <span class="hidden sm:inline">{clock(entry.at)} · </span>{ago(entry.at)}
          </time>
        </li>
      {/each}
    </ul>
    {#if next && paged}
      <div class="flex justify-center">
        <Button variant="outline" size="sm" onclick={more} disabled={loadingMore}>{loadingMore ? 'Loading…' : 'Load more'}</Button>
      </div>
    {/if}
  {/if}
</div>
