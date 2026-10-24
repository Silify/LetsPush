<script>
  import Logo from './Logo.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import { api } from '$lib/api.js';
  import { go } from '$lib/router.svelte.js';

  let { email, onSignedOut } = $props();

  async function signOut() {
    try {
      await api.logout();
    } finally {
      onSignedOut();
    }
  }
</script>

<header class="border-b bg-card">
  <div class="mx-auto flex max-w-3xl items-center justify-between gap-3 px-4 py-3">
    <a class="flex items-center gap-2.5 font-semibold tracking-tight" href="#/"><Logo /> LetsPush</a>
    <div class="flex items-center gap-1">
      <Button variant="ghost" size="sm" class="max-w-40 truncate sm:max-w-none" onclick={() => go('/account')}>{email}</Button>
      <Button variant="outline" size="sm" onclick={signOut}>Sign out</Button>
    </div>
  </div>
</header>
