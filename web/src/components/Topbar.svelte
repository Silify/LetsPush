<script>
  import Logo from './Logo.svelte';
  import { api } from '../lib/api.js';
  import { go } from '../lib/router.svelte.js';

  let { email, onSignedOut } = $props();

  async function signOut() {
    try {
      await api.logout();
    } finally {
      onSignedOut();
    }
  }
</script>

<header class="topbar">
  <div class="inner">
    <a class="brand" href="#/"><Logo /> LetsPush</a>
    <div class="row">
      <button class="btn ghost small" type="button" onclick={() => go('/account')}>{email}</button>
      <button class="btn small" type="button" onclick={signOut}>Sign out</button>
    </div>
  </div>
</header>
