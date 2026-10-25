<script>
  import Code from '@lucide/svelte/icons/code';
  import GraduationCap from '@lucide/svelte/icons/graduation-cap';
  import CopyBox from '../../components/CopyBox.svelte';
  import * as Card from '$lib/components/ui/card/index.js';
  import { href as link } from '$lib/router.svelte.js';

  let { app } = $props();

  // How a site reaches this server: Moodle wants the address and the port apart.
  const origin = location.origin;
  const serverUrl = `${location.protocol}//${location.hostname}`;
  const serverPort = location.port || (location.protocol === 'https:' ? '443' : '80');

  const register = `curl -X POST ${origin}/v1/devices \\
  -H "Authorization: Bearer <access key>" \\
  -d '{"token": "<device token>", "platform": "android"}'`;

  const send = `curl -X POST ${origin}/v1/notifications \\
  -H "Authorization: Bearer <access key>" \\
  -d '{
    "to": {"token": "<device token>"},
    "title": "Assignment graded",
    "body": "Your essay was graded",
    "image": "https://example.com/picture.png",
    "data": {"course": "12"}
  }'`;
</script>

<div class="grid gap-6">
  <Card.Root>
    <Card.Header>
      <Card.Title class="flex items-center gap-2"><GraduationCap class="size-4 text-primary" />Connect a Moodle site</Card.Title>
      <Card.Description>
        In Moodle, open <i>Site administration → Messaging → Mobile</i> and fill in the push server fields. No plugin is needed.
      </Card.Description>
    </Card.Header>
    <Card.Content class="grid gap-4">
      <dl class="grid items-center gap-x-4 gap-y-3 text-sm sm:grid-cols-[max-content_1fr]">
        <dt class="text-muted-foreground">Server URL</dt>
        <dd><CopyBox value={serverUrl} label="Copy the server URL" /></dd>
        <dt class="text-muted-foreground">Port</dt>
        <dd><CopyBox value={serverPort} label="Copy the port" /></dd>
        <dt class="text-muted-foreground">App name</dt>
        <dd><CopyBox value={app.name} label="Copy the app name" /></dd>
        <dt class="text-muted-foreground">Access key</dt>
        <dd class="text-muted-foreground">
          One of this app's <a class="underline" href={link(`/apps/${app.id}/keys`)}>access keys</a>. It is shown only when it is created.
        </dd>
        <dt class="text-muted-foreground">Mobile app name</dt>
        <dd class="text-muted-foreground">Your app's package id, for example <code class="font-mono">org.example.app</code></dd>
      </dl>
      <p class="text-xs text-muted-foreground">
        Then enable the mobile notifications output under <i>Plugins → Message outputs</i>. Moodle's own check on that page flags a wrong address or key.
      </p>
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header>
      <Card.Title class="flex items-center gap-2"><Code class="size-4 text-primary" />Use the API from your own code</Card.Title>
      <Card.Description>
        For your own plugin, site or script. Send a title, a body, an image and any data, and ask later how the delivery ended. Use an access key as the Bearer token.
      </Card.Description>
    </Card.Header>
    <Card.Content class="grid gap-4">
      <div class="grid gap-2">
        <p class="text-sm font-medium">1. Register a device</p>
        <CopyBox value={register} multiline label="Copy the command" />
      </div>
      <div class="grid gap-2">
        <p class="text-sm font-medium">2. Send a notification</p>
        <CopyBox value={send} multiline label="Copy the command" />
        <p class="text-xs text-muted-foreground">
          The answer has an <code class="font-mono">id</code>; <code class="font-mono">GET {origin}/v1/notifications/&lt;id&gt;</code> tells you how it ended. The full reference is <code class="font-mono">docs/11-native-api.md</code> in the repository.
        </p>
      </div>
    </Card.Content>
  </Card.Root>
</div>
