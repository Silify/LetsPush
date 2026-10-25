<script>
  import ChevronsUpDown from '@lucide/svelte/icons/chevrons-up-down';
  import LogOut from '@lucide/svelte/icons/log-out';
  import Menu from '@lucide/svelte/icons/menu';
  import UserRound from '@lucide/svelte/icons/user-round';
  import Logo from './Logo.svelte';
  import Nav from './Nav.svelte';
  import ThemeToggle from './ThemeToggle.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
  import * as Sheet from '$lib/components/ui/sheet/index.js';
  import { api } from '$lib/api.js';
  import { loadApps, resetApps } from '$lib/apps.svelte.js';
  import { go } from '$lib/router.svelte.js';

  /** The frame of every signed-in page: a sidebar (a drawer on a phone), a top bar and the page. */
  let { email, onSignedOut, children } = $props();

  let menuOpen = $state(false);

  loadApps();

  async function signOut() {
    try {
      await api.logout();
    } finally {
      resetApps();
      onSignedOut();
    }
  }
</script>

{#snippet brand()}
  <a class="flex items-center gap-2.5 px-1 font-semibold tracking-tight" href="#/"><Logo /> Let's Push</a>
{/snippet}

<div class="flex min-h-screen">
  <aside class="sticky top-0 hidden h-screen w-60 shrink-0 flex-col gap-6 border-e bg-sidebar p-4 md:flex">
    {@render brand()}
    <Nav />
  </aside>

  <div class="flex min-w-0 flex-1 flex-col">
    <header class="sticky top-0 z-30 flex h-14 items-center gap-2 border-b bg-background/85 px-4 backdrop-blur sm:px-6">
      <Sheet.Root bind:open={menuOpen}>
        <Sheet.Trigger>
          {#snippet child({ props })}
            <Button {...props} variant="ghost" size="icon" class="md:hidden" aria-label="Open the menu"><Menu /></Button>
          {/snippet}
        </Sheet.Trigger>
        <Sheet.Content side="left" class="w-64 gap-6 bg-sidebar p-4" showCloseButton={false}>
          <Sheet.Header class="p-0">
            <Sheet.Title class="sr-only">Menu</Sheet.Title>
            {@render brand()}
          </Sheet.Header>
          <Nav onNavigate={() => (menuOpen = false)} />
        </Sheet.Content>
      </Sheet.Root>
      <div class="md:hidden">{@render brand()}</div>

      <div class="ms-auto flex items-center gap-1">
        <ThemeToggle />
        <DropdownMenu.Root>
          <DropdownMenu.Trigger>
            {#snippet child({ props })}
              <Button {...props} variant="ghost" size="sm" class="max-w-48 gap-1.5">
                <span class="grid size-5 shrink-0 place-items-center rounded-full bg-primary/15 text-xs font-semibold text-primary uppercase">{email[0]}</span>
                <span class="hidden truncate sm:inline">{email}</span>
                <ChevronsUpDown class="hidden sm:block" />
              </Button>
            {/snippet}
          </DropdownMenu.Trigger>
          <DropdownMenu.Content align="end" class="w-56">
            <DropdownMenu.Label class="truncate font-normal text-muted-foreground">{email}</DropdownMenu.Label>
            <DropdownMenu.Separator />
            <DropdownMenu.Item onclick={() => go('/account')}><UserRound class="size-4" />Account</DropdownMenu.Item>
            <DropdownMenu.Item onclick={signOut}><LogOut class="size-4" />Sign out</DropdownMenu.Item>
          </DropdownMenu.Content>
        </DropdownMenu.Root>
      </div>
    </header>

    <main class="mx-auto w-full max-w-5xl flex-1 px-4 py-6 sm:px-6 sm:py-8">
      {@render children()}
    </main>
  </div>
</div>
