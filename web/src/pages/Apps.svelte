<script>
  import Modal from '../components/Modal.svelte';
  import { api } from '../lib/api.js';
  import { go } from '../lib/router.svelte.js';
  import { toast } from '../lib/toast.svelte.js';

  let apps = $state(null);
  let error = $state('');

  let creating = $state(false);
  let name = $state('');
  let serviceAccount = $state('');
  let busy = $state(false);
  let formError = $state('');

  async function load() {
    try {
      apps = await api.apps();
    } catch (e) {
      error = e.message;
    }
  }
  load();

  function open() {
    name = '';
    serviceAccount = '';
    formError = '';
    creating = true;
  }

  async function readFile(event) {
    const file = event.target.files?.[0];
    if (file) serviceAccount = await file.text();
  }

  async function create(event) {
    event.preventDefault();
    busy = true;
    formError = '';
    try {
      const app = await api.createApp(name.trim(), serviceAccount.trim());
      creating = false;
      toast('App created');
      go(`/apps/${app.id}`);
    } catch (e) {
      formError = e.message;
    } finally {
      busy = false;
    }
  }
</script>

<div class="page">
  <div class="row between">
    <div class="stack" style="gap: 4px">
      <h1>Apps</h1>
      <p class="muted">Each app is one mobile app, with its own Firebase key and access keys.</p>
    </div>
    <button class="btn primary" type="button" onclick={open}>New app</button>
  </div>

  {#if error}
    <p class="notice error">{error}</p>
  {:else if apps === null}
    <p class="muted">Loading…</p>
  {:else if apps.length === 0}
    <div class="card">
      <div class="empty">
        <h2>No apps yet</h2>
        <p>Add the app you send notifications to. You will need its Firebase service-account key.</p>
        <button class="btn primary" type="button" onclick={open}>Create your first app</button>
      </div>
    </div>
  {:else}
    {#each apps as app (app.id)}
      <a class="card" href={`#/apps/${app.id}`} style="text-decoration: none; color: inherit">
        <header>
          <div class="stack" style="gap: 2px">
            <h2>{app.name}</h2>
            <span class="muted small">{app.projectId ? `Firebase project ${app.projectId}` : 'No Firebase key yet'}</span>
          </div>
          {#if app.hasCredentials}
            <span class="badge ok">Ready to send</span>
          {:else}
            <span class="badge warn">Needs a Firebase key</span>
          {/if}
        </header>
        <div class="stats">
          <div class="stat"><b>{app.keys}</b><span class="muted small">access keys</span></div>
          <div class="stat"><b>{app.tokens}</b><span class="muted small">devices</span></div>
        </div>
      </a>
    {/each}
  {/if}
</div>

<Modal bind:open={creating} title="New app">
  <form class="stack" onsubmit={create}>
    <label>
      Name
      <input class="input" required maxlength="64" bind:value={name} placeholder="for example: sayan" />
      <span class="hint">Sites send this as the app name. Letters, digits, dots, dashes and underscores.</span>
    </label>

    <label>
      Firebase service-account key <span class="hint">(optional now, needed to send)</span>
      <input class="input" type="file" accept="application/json,.json" onchange={readFile} />
      <textarea bind:value={serviceAccount} placeholder="…or paste the JSON here" spellcheck="false"></textarea>
      <span class="hint">
        Firebase console → Project settings → Service accounts → Generate new private key. It is stored encrypted.
      </span>
    </label>

    {#if formError}<p class="notice error" role="alert">{formError}</p>{/if}
    <div class="row" style="justify-content: flex-end">
      <button class="btn" type="button" onclick={() => (creating = false)}>Cancel</button>
      <button class="btn primary" disabled={busy}>{busy ? 'Creating…' : 'Create app'}</button>
    </div>
  </form>
</Modal>
