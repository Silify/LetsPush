<script>
  import { ModeWatcher } from 'mode-watcher';
  import Topbar from './components/Topbar.svelte';
  import Account from './pages/Account.svelte';
  import AppDetail from './pages/AppDetail.svelte';
  import Apps from './pages/Apps.svelte';
  import Login from './pages/Login.svelte';
  import { Toaster } from '$lib/components/ui/sonner/index.js';
  import { api, whenSignedOut } from '$lib/api.js';
  import { go, route } from '$lib/router.svelte.js';

  // null while checking, '' when signed out, otherwise the admin's email.
  let email = $state(null);

  whenSignedOut(() => (email = ''));

  api
    .me()
    .then((me) => (email = me.email))
    .catch(() => (email = ''));

  function signedIn(address) {
    email = address;
    go('/');
  }
</script>

<!-- Follows the system light or dark setting. -->
<ModeWatcher defaultMode="system" />

{#if email === null}
  <div class="grid min-h-screen place-items-center"><p class="text-sm text-muted-foreground">Loading…</p></div>
{:else if email === ''}
  <Login onSignedIn={signedIn} />
{:else}
  <Topbar {email} onSignedOut={() => (email = '')} />
  {#if route.name === 'app'}
    {#key route.id}
      <AppDetail id={route.id} />
    {/key}
  {:else if route.name === 'account'}
    <Account {email} />
  {:else}
    <Apps />
  {/if}
{/if}

<Toaster richColors={false} position="bottom-center" />
