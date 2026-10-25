<script>
  import LogList from '../../components/LogList.svelte';
  import * as Card from '$lib/components/ui/card/index.js';
  import { api } from '$lib/api.js';

  let { app } = $props();
</script>

<Card.Root>
  <Card.Header>
    <Card.Title>Activity</Card.Title>
    <Card.Description>
      What happened to this app. It holds no message text and no whole device tokens. When a send fails, the reason Firebase gave is here.
    </Card.Description>
  </Card.Header>
  <Card.Content>
    <LogList load={(o) => api.appLogs(app.id, o)} live />
  </Card.Content>
</Card.Root>
