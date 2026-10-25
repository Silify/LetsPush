<script>
  import LogOut from '@lucide/svelte/icons/log-out';
  import Monitor from '@lucide/svelte/icons/monitor';
  import Moon from '@lucide/svelte/icons/moon';
  import Sun from '@lucide/svelte/icons/sun';
  import { setMode, userPrefersMode } from 'mode-watcher';
  import PageHeader from '../components/PageHeader.svelte';
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { api } from '$lib/api.js';
  import { toast } from '$lib/toast.js';

  let { email, onSignedOut } = $props();

  let current = $state('');
  let next = $state('');
  let confirm = $state('');
  let error = $state('');
  let busy = $state(false);

  const mismatch = $derived(confirm !== '' && confirm !== next);
  const tooShort = $derived(next !== '' && next.length < 10);

  async function change(event) {
    event.preventDefault();
    busy = true;
    error = '';
    try {
      await api.changePassword(current, next);
      current = next = confirm = '';
      toast('Password changed');
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }

  async function signOut() {
    try {
      await api.logout();
    } finally {
      onSignedOut();
    }
  }

  const themes = [
    { value: 'light', label: 'Light', icon: Sun },
    { value: 'dark', label: 'Dark', icon: Moon },
    { value: 'system', label: 'System', icon: Monitor },
  ];
</script>

<PageHeader title="Account" description={email} />

<div class="grid max-w-2xl gap-6">
  <Card.Root>
    <Card.Header>
      <Card.Title>Appearance</Card.Title>
      <Card.Description>The dashboard remembers your choice on this device. “System” follows your device.</Card.Description>
    </Card.Header>
    <Card.Content>
      <div class="grid grid-cols-3 gap-2" role="radiogroup" aria-label="Theme">
        {#each themes as theme (theme.value)}
          <button
            type="button"
            role="radio"
            aria-checked={userPrefersMode.current === theme.value}
            onclick={() => setMode(theme.value)}
            class="flex flex-col items-center gap-2 rounded-lg border px-3 py-3 text-sm transition-colors hover:bg-muted aria-checked:border-primary aria-checked:bg-primary/5 aria-checked:text-primary"
          >
            <theme.icon class="size-5" />
            {theme.label}
          </button>
        {/each}
      </div>
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header>
      <Card.Title>Change password</Card.Title>
      <Card.Description>Use at least 10 characters.</Card.Description>
    </Card.Header>
    <Card.Content>
      <form class="grid gap-4" onsubmit={change}>
        <input type="email" value={email} autocomplete="username" hidden readonly />
        <div class="grid gap-2">
          <Label for="current">Current password</Label>
          <Input id="current" type="password" bind:value={current} autocomplete="current-password" required />
        </div>
        <div class="grid gap-2">
          <Label for="new">New password</Label>
          <Input id="new" type="password" bind:value={next} autocomplete="new-password" required aria-invalid={tooShort} />
          {#if tooShort}<p class="text-xs text-destructive">At least 10 characters.</p>{/if}
        </div>
        <div class="grid gap-2">
          <Label for="confirm">Repeat the new password</Label>
          <Input id="confirm" type="password" bind:value={confirm} autocomplete="new-password" required aria-invalid={mismatch} />
          {#if mismatch}<p class="text-xs text-destructive">The passwords do not match.</p>{/if}
        </div>
        {#if error}
          <Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>
        {/if}
        <div><Button type="submit" disabled={busy || !current || next.length < 10 || next !== confirm}>{busy ? 'Saving…' : 'Change password'}</Button></div>
      </form>
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header>
      <Card.Title>Sign out</Card.Title>
      <Card.Description>Ends your session on this device.</Card.Description>
      <Card.Action><Button variant="outline" onclick={signOut}><LogOut data-icon="inline-start" />Sign out</Button></Card.Action>
    </Card.Header>
  </Card.Root>
</div>
