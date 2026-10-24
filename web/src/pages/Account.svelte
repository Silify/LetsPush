<script>
  import * as Alert from '$lib/components/ui/alert/index.js';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { api } from '$lib/api.js';
  import { toast } from '$lib/toast.js';

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

<main class="mx-auto grid max-w-3xl gap-6 px-4 py-8">
  <div class="grid gap-1">
    <a class="text-sm text-muted-foreground hover:text-foreground" href="#/">← Apps</a>
    <h1 class="text-2xl font-semibold tracking-tight">Account</h1>
    <p class="text-sm text-muted-foreground">Signed in as {email}</p>
  </div>

  <Card.Root class="max-w-md">
    <form onsubmit={submit}>
      <Card.Header>
        <Card.Title>Change password</Card.Title>
      </Card.Header>
      <Card.Content class="grid gap-4">
        <div class="grid gap-2">
          <Label for="current">Current password</Label>
          <Input id="current" type="password" autocomplete="current-password" required bind:value={current} />
        </div>
        <div class="grid gap-2">
          <Label for="new">New password</Label>
          <Input id="new" type="password" autocomplete="new-password" required minlength={10} bind:value={next} />
          <p class="text-xs text-muted-foreground">At least 10 characters.</p>
        </div>
        {#if error}
          <Alert.Root variant="destructive" role="alert"><Alert.Description>{error}</Alert.Description></Alert.Root>
        {/if}
      </Card.Content>
      <Card.Footer>
        <Button type="submit" disabled={busy}>{busy ? 'Saving…' : 'Change password'}</Button>
      </Card.Footer>
    </form>
  </Card.Root>
</main>
