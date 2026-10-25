<script>
  import Activity from '@lucide/svelte/icons/activity';
  import AppWindow from '@lucide/svelte/icons/app-window';
  import LayoutDashboard from '@lucide/svelte/icons/layout-dashboard';
  import UserRound from '@lucide/svelte/icons/user-round';
  import { apps } from '$lib/apps.svelte.js';
  import { route } from '$lib/router.svelte.js';

  /** The sidebar's links. onNavigate runs after a click, so a mobile menu can close itself. */
  let { onNavigate = () => {} } = $props();

  const items = [
    { name: 'overview', label: 'Overview', href: '#/', icon: LayoutDashboard },
    { name: 'apps', label: 'Apps', href: '#/apps', icon: AppWindow },
    { name: 'activity', label: 'Activity', href: '#/activity', icon: Activity },
    { name: 'account', label: 'Account', href: '#/account', icon: UserRound },
  ];

  // The app the person is in counts as being in "Apps" too.
  const current = $derived(route.name === 'app' ? 'apps' : route.name);
  const shown = $derived((apps.list ?? []).slice(0, 8));
</script>

<nav class="grid gap-0.5" aria-label="Main">
  {#each items as item (item.name)}
    <a
      href={item.href}
      onclick={onNavigate}
      aria-current={current === item.name ? 'page' : undefined}
      class="flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground aria-[current=page]:bg-primary/10 aria-[current=page]:text-primary"
    >
      <item.icon class="size-4" />
      {item.label}
    </a>
    {#if item.name === 'apps' && shown.length}
      <ul class="mb-1 ms-4 grid gap-0.5 border-s ps-2">
        {#each shown as app (app.id)}
          <li>
            <a
              href="#/apps/{app.id}"
              onclick={onNavigate}
              aria-current={route.name === 'app' && route.id === app.id ? 'page' : undefined}
              class="flex items-center gap-2 rounded-md px-2 py-1 text-[0.8rem] text-muted-foreground transition-colors hover:text-foreground aria-[current=page]:font-medium aria-[current=page]:text-foreground"
            >
              <span class="size-1.5 shrink-0 rounded-full {app.hasCredentials ? 'bg-primary' : 'bg-amber-500'}" title={app.hasCredentials ? 'Ready to send' : 'Needs a Firebase key'}></span>
              <span class="truncate">{app.name}</span>
            </a>
          </li>
        {/each}
      </ul>
    {/if}
  {/each}
</nav>
