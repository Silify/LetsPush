<script>
  import ArrowRight from '@lucide/svelte/icons/arrow-right';
  import CircleCheck from '@lucide/svelte/icons/circle-check';
  import Circle from '@lucide/svelte/icons/circle';
  import KeyRound from '@lucide/svelte/icons/key-round';
  import Send from '@lucide/svelte/icons/send';
  import Smartphone from '@lucide/svelte/icons/smartphone';
  import TriangleAlert from '@lucide/svelte/icons/triangle-alert';
  import LogList from '../../components/LogList.svelte';
  import StatCard from '../../components/StatCard.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  import * as Card from '$lib/components/ui/card/index.js';
  import { api } from '$lib/api.js';

  let { app } = $props();

  const activeKeys = $derived(app.accessKeys.filter((k) => !k.revokedAt).length);

  // What it takes for an app to work, in order, each step linking to where it is done.
  const steps = $derived([
    { done: app.hasCredentials, text: 'Add the Firebase service-account key', href: `#/apps/${app.id}/firebase` },
    { done: activeKeys > 0, text: 'Create an access key for a site', href: `#/apps/${app.id}/keys` },
    { done: app.tokens > 0, text: 'Register a device (the mobile app does this)', href: `#/apps/${app.id}/connect` },
    { done: app.sent24h > 0, text: 'Send a notification', href: `#/apps/${app.id}/connect` },
  ]);
  const complete = $derived(steps.every((s) => s.done));
</script>

<div class="grid gap-6">
  <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
    <StatCard label="Access keys" value={activeKeys} icon={KeyRound} />
    <StatCard label="Devices" value={app.tokens} icon={Smartphone} />
    <StatCard label="Sent (24 h)" value={app.sent24h} icon={Send} />
    <StatCard label="Errors (24 h)" value={app.errors24h} icon={TriangleAlert} tone="danger" />
  </div>

  <div class="grid gap-6 xl:grid-cols-5">
    {#if !complete}
      <Card.Root class="xl:col-span-2">
        <Card.Header>
          <Card.Title>Getting set up</Card.Title>
          <Card.Description>{steps.filter((s) => s.done).length} of {steps.length} done</Card.Description>
        </Card.Header>
        <Card.Content>
          <ol class="grid gap-1">
            {#each steps as step (step.text)}
              <li>
                <a class="flex items-center gap-2.5 rounded-md px-1 py-1.5 text-sm hover:bg-muted" href={step.href}>
                  {#if step.done}<CircleCheck class="size-4 shrink-0 text-primary" />{:else}<Circle class="size-4 shrink-0 text-muted-foreground" />{/if}
                  <span class={step.done ? 'text-muted-foreground line-through' : ''}>{step.text}</span>
                </a>
              </li>
            {/each}
          </ol>
        </Card.Content>
      </Card.Root>
    {/if}

    <Card.Root class={complete ? 'xl:col-span-5' : 'xl:col-span-3'}>
      <Card.Header>
        <Card.Title>Recent activity</Card.Title>
        <Card.Action>
          <Button variant="ghost" size="sm" href="#/apps/{app.id}/activity">All<ArrowRight data-icon="inline-end" /></Button>
        </Card.Action>
      </Card.Header>
      <Card.Content>
        <LogList load={(o) => api.appLogs(app.id, o)} filters={false} paged={false} size={6} />
      </Card.Content>
    </Card.Root>
  </div>
</div>
