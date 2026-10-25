<script>
  import Eye from '@lucide/svelte/icons/eye';
  import EyeOff from '@lucide/svelte/icons/eye-off';
  import Logo from '../components/Logo.svelte';
  import ThemeToggle from '../components/ThemeToggle.svelte';
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { api } from '$lib/api.js';

  let { onSignedIn } = $props();

  let email = $state('');
  let password = $state('');
  let showPassword = $state(false);
  let error = $state('');
  let busy = $state(false);

  async function submit(event) {
    event.preventDefault();
    busy = true;
    error = '';
    try {
      const me = await api.login(email.trim(), password);
      onSignedIn(me.email);
    } catch (e) {
      error = e.message;
    } finally {
      busy = false;
    }
  }
</script>

<div class="absolute end-4 top-4"><ThemeToggle /></div>

<main class="grid min-h-screen place-items-center px-4 py-10">
  <div class="grid w-full max-w-sm gap-6">
    <div class="grid justify-items-center gap-3 text-center">
      <div class="scale-150"><Logo /></div>
      <div>
        <h1 class="text-2xl font-semibold tracking-tight">LetsPush</h1>
        <p class="text-sm text-muted-foreground">Sign in to manage your push gateway.</p>
      </div>
    </div>

    <Card.Root>
      <Card.Content>
        <form class="grid gap-4" onsubmit={submit}>
          <div class="grid gap-2">
            <Label for="email">Email</Label>
            <Input id="email" type="email" bind:value={email} autocomplete="username" required autofocus />
          </div>
          <div class="grid gap-2">
            <Label for="password">Password</Label>
            <div class="relative">
              <Input
                id="password"
                type={showPassword ? 'text' : 'password'}
                bind:value={password}
                autocomplete="current-password"
                required
                class="pe-9"
              />
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                class="absolute end-0.5 top-0.5"
                onclick={() => (showPassword = !showPassword)}
                aria-label={showPassword ? 'Hide the password' : 'Show the password'}
              >
                {#if showPassword}<EyeOff />{:else}<Eye />{/if}
              </Button>
            </div>
          </div>
          {#if error}
            <Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>
          {/if}
          <Button type="submit" size="lg" disabled={busy || !email || !password}>{busy ? 'Signing in…' : 'Sign in'}</Button>
        </form>
      </Card.Content>
    </Card.Root>
  </div>
</main>
