<script>
  import ConfirmDialog from '../../components/ConfirmDialog.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { Label } from '$lib/components/ui/label/index.js';
  import { Switch } from '$lib/components/ui/switch/index.js';
  import { api } from '$lib/api.js';
  import { loadApps } from '$lib/apps.svelte.js';
  import { go } from '$lib/router.svelte.js';
  import { toast } from '$lib/toast.js';

  let { app, reload } = $props();

  let allow = $state(app.allowKeyRequests);

  async function setRequests(on) {
    const before = allow;
    allow = on;
    try {
      await api.setSettings(app.id, on);
      toast(on ? 'Sites may now request their own key' : 'Sites can no longer request a key');
      await reload();
    } catch (e) {
      allow = before;
      toast(e.message, 'error');
    }
  }

  let deleting = $state(false);

  async function remove() {
    try {
      await api.deleteApp(app.id);
      deleting = false;
      toast('App deleted');
      await loadApps();
      go('/apps');
    } catch (e) {
      toast(e.message, 'error');
    }
  }
</script>

<div class="grid gap-6">
  <Card.Root>
    <Card.Header>
      <Card.Title>Access key requests</Card.Title>
    </Card.Header>
    <Card.Content>
      <div class="flex items-start justify-between gap-4">
        <div class="grid gap-1">
          <Label for="allow-requests">Let sites request their own access key</Label>
          <p class="max-w-prose text-sm text-muted-foreground">
            Turns on the “Request access key” button in Moodle. Anyone who can reach this server could then get a key, so leave it off unless you need it.
          </p>
        </div>
        <Switch id="allow-requests" checked={allow} onCheckedChange={setRequests} />
      </div>
    </Card.Content>
  </Card.Root>

  <Card.Root class="border-destructive/40">
    <Card.Header>
      <Card.Title>Delete this app</Card.Title>
      <Card.Description>Removes its access keys, devices and activity. Sites using it stop working, and this cannot be undone.</Card.Description>
      <Card.Action><Button variant="destructive" onclick={() => (deleting = true)}>Delete…</Button></Card.Action>
    </Card.Header>
  </Card.Root>
</div>

<ConfirmDialog bind:open={deleting} title="Delete this app?" confirmLabel="Delete app" type={app.name} onconfirm={remove}>
  This removes <b>{app.name}</b> with everything under it. Type its name to confirm.
</ConfirmDialog>
