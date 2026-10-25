<script>
  import { ModeWatcher } from 'mode-watcher';
  import Shell from './components/Shell.svelte';
  import Login from './pages/Login.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { Skeleton } from '$lib/components/ui/skeleton/index.js';
  import { Toaster } from '$lib/components/ui/sonner/index.js';
  import { api, whenSignedOut } from '$lib/api.js';
  import { resetApps } from '$lib/apps.svelte.js';
  import { go, route } from '$lib/router.svelte.js';

  // Each page is its own file that loads when it is first opened, so the sign-in page stays small.
  const pages = {
    overview: () => import('./pages/Overview.svelte'),
    apps: () => import('./pages/Apps.svelte'),
    app: () => import('./pages/AppPage.svelte'),
    activity: () => import('./pages/Activity.svelte'),
    account: () => import('./pages/Account.svelte'),
  };

  // null while checking, '' when signed out, otherwise the admin's email.
  let email = $state(null);

  function signedOut() {
    resetApps();
    email = '';
  }

  whenSignedOut(signedOut);

  api
    .me()
    .then((me) => (email = me.email))
    .catch(() => (email = ''));

  function signedIn(address) {
    email = address;
    go('/');
  }

  // Moving between the tabs of one app keeps the page; moving to another app or page starts a new one.
  const pageKey = $derived(route.name === 'app' ? `app-${route.id}` : route.name);
</script>

<!-- Follows the system light or dark setting until the person chooses. -->
<ModeWatcher defaultMode="system" />

{#if email === null}
  <div class="grid min-h-screen place-items-center">
    <Skeleton class="h-8 w-40" />
  </div>
{:else if email === ''}
  <Login onSignedIn={signedIn} />
{:else}
  <Shell {email} onSignedOut={signedOut}>
    {#key pageKey}
      {#await pages[route.name]()}
        <div class="grid gap-4">
          <Skeleton class="h-8 w-56" />
          <Skeleton class="h-32 w-full" />
        </div>
      {:then page}
        {@const Page = page.default}
        <Page {email} onSignedOut={signedOut} />
      {:catch}
        <div class="flex items-center gap-3 text-sm" role="alert">
          <span class="text-destructive">This page could not be loaded. If Let's Push was just updated, reload.</span>
          <Button variant="outline" size="sm" onclick={() => location.reload()}>Reload</Button>
        </div>
      {/await}
    {/key}
  </Shell>
{/if}

<Toaster richColors={false} position="bottom-center" />
