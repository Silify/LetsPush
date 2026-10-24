<script>
  import Logo from '../components/Logo.svelte';
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { api } from '$lib/api.js';

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

<div class="grid min-h-screen place-items-center p-4">
  <div class="grid w-full max-w-sm gap-6">
    <div class="grid justify-items-center gap-2 text-center">
      <Logo />
      <h1 class="text-2xl font-semibold tracking-tight">LetsPush</h1>
      <p class="text-sm text-muted-foreground">Sign in to manage your apps and access keys.</p>
    </div>

    <Card.Root>
      <form onsubmit={submit}>
        <Card.Content class="grid gap-4">
          <div class="grid gap-2">
            <Label for="email">Email</Label>
            <Input id="email" type="email" autocomplete="username" required bind:value={email} />
          </div>
          <div class="grid gap-2">
            <Label for="password">Password</Label>
            <Input id="password" type="password" autocomplete="current-password" required bind:value={password} />
          </div>
          {#if error}
            <Alert.Root variant="destructive" role="alert">
              <Alert.Description>{error}</Alert.Description>
            </Alert.Root>
          {/if}
          <Button type="submit" disabled={busy}>{busy ? 'Signing in…' : 'Sign in'}</Button>
        </Card.Content>
      </form>
    </Card.Root>
  </div>
</div>
