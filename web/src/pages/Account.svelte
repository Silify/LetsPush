<script>
  import { api } from '../lib/api.js';
  import { toast } from '../lib/toast.svelte.js';

  let { email } = $props();

  let current = $state('');
  let next = $state('');
  let error = $state('');
  let busy = $state(false);

  async function submit(event) {
    event.preventDefault();
    error = '';
    busy = true;
    try {
      await api.changePassword(current, next);
      current = next = '';
      toast('Password changed');
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }
</script>

<div class="page">
  <div class="stack" style="gap: 4px">
    <a class="small" href="#/">← Apps</a>
    <h1>Account</h1>
    <p class="muted">Signed in as {email}</p>
  </div>

  <form class="card" onsubmit={submit} style="max-width: 460px">
    <h2>Change password</h2>
    <label>
      Current password
      <input class="input" type="password" autocomplete="current-password" required bind:value={current} />
    </label>
    <label>
      New password
      <input class="input" type="password" autocomplete="new-password" required minlength="10" bind:value={next} />
      <span class="hint">At least 10 characters.</span>
    </label>
    {#if error}<p class="notice error" role="alert">{error}</p>{/if}
    <div><button class="btn primary" disabled={busy}>{busy ? 'Saving…' : 'Change password'}</button></div>
  </form>
</div>
