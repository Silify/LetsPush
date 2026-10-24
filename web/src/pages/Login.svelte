<script>
  import Logo from '../components/Logo.svelte';
  import { api } from '../lib/api.js';

  let { onSignedIn } = $props();

  let email = $state('');
  let password = $state('');
  let error = $state('');
  let busy = $state(false);

  async function submit(event) {
    event.preventDefault();
    error = '';
    busy = true;
    try {
      const me = await api.login(email, password);
      onSignedIn(me.email);
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }
</script>

<div class="center">
  <form class="login" onsubmit={submit}>
    <div class="stack" style="justify-items: center; text-align: center">
      <Logo />
      <h1>LetsPush</h1>
      <p class="muted">Sign in to manage your apps and access keys.</p>
    </div>

    <div class="card">
      <label>
        Email
        <input class="input" type="email" autocomplete="username" required bind:value={email} />
      </label>
      <label>
        Password
        <input class="input" type="password" autocomplete="current-password" required bind:value={password} />
      </label>
      {#if error}<p class="notice error" role="alert">{error}</p>{/if}
      <button class="btn primary" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
    </div>
  </form>
</div>
