<script>
  import Modal from '../components/Modal.svelte';
  import CopyBox from '../components/CopyBox.svelte';
  import { api } from '../lib/api.js';
  import { ago, permissionLabel, when } from '../lib/format.js';
  import { go } from '../lib/router.svelte.js';
  import { toast } from '../lib/toast.svelte.js';

  let { id } = $props();

  let app = $state(null);
  let logs = $state([]);
  let error = $state('');

  const serverUrl = `${location.protocol}//${location.hostname}`;
  const serverPort = location.port || (location.protocol === 'https:' ? '443' : '80');

  async function load() {
    try {
      app = await api.app(id);
      logs = await api.logs(id);
    } catch (e) {
      error = e.message;
    }
  }
  load();

  // Firebase key
  let replacing = $state(false);
  let serviceAccount = $state('');
  let credentialsError = $state('');
  let credentialsBusy = $state(false);

  async function readFile(event) {
    const file = event.target.files?.[0];
    if (file) serviceAccount = await file.text();
  }

  async function saveCredentials(event) {
    event.preventDefault();
    credentialsBusy = true;
    credentialsError = '';
    try {
      await api.setCredentials(id, serviceAccount.trim());
      replacing = false;
      serviceAccount = '';
      toast('Firebase key saved');
      await load();
    } catch (e) {
      credentialsError = e.message;
    } finally {
      credentialsBusy = false;
    }
  }

  // Access keys
  const allPermissions = ['createToken', 'deleteToken', 'sendNotification'];
  let creatingKey = $state(false);
  let description = $state('');
  let contact = $state('');
  let permissions = $state([...allPermissions]);
  let keyError = $state('');
  let keyBusy = $state(false);
  let issued = $state(null);
  let showIssued = $state(false);

  function openKeyForm() {
    description = contact = keyError = '';
    permissions = [...allPermissions];
    creatingKey = true;
  }

  async function createKey(event) {
    event.preventDefault();
    keyBusy = true;
    keyError = '';
    try {
      issued = await api.createKey(id, { description, contact, permissions });
      creatingKey = false;
      showIssued = true;
      await load();
    } catch (e) {
      keyError = e.message;
    } finally {
      keyBusy = false;
    }
  }

  let revoking = $state(null);
  let showRevoke = $state(false);

  async function revoke() {
    try {
      await api.revokeKey(id, revoking.id);
      showRevoke = false;
      toast('Access key revoked');
      await load();
    } catch (e) {
      toast(e.message, 'error');
    }
  }

  // Settings and deleting
  async function toggleRequests(event) {
    const allow = event.target.checked;
    try {
      await api.setSettings(id, allow);
      app.allowKeyRequests = allow;
      toast(allow ? 'Sites may now request their own key' : 'Sites can no longer request a key');
    } catch (e) {
      event.target.checked = !allow;
      toast(e.message, 'error');
    }
  }

  let deleting = $state(false);
  let confirmName = $state('');

  async function remove() {
    try {
      await api.deleteApp(id);
      deleting = false;
      toast('App deleted');
      go('/');
    } catch (e) {
      toast(e.message, 'error');
    }
  }
</script>

<div class="page">
  <a class="small" href="#/">← Apps</a>

  {#if error}
    <p class="notice error">{error}</p>
  {:else if app === null}
    <p class="muted">Loading…</p>
  {:else}
    <div class="row between">
      <div class="stack" style="gap: 4px">
        <h1>{app.name}</h1>
        <div class="stats">
          <div class="stat"><b>{app.keys}</b><span class="muted small">access keys</span></div>
          <div class="stat"><b>{app.tokens}</b><span class="muted small">devices</span></div>
        </div>
      </div>
      {#if app.hasCredentials}
        <span class="badge ok">Ready to send</span>
      {:else}
        <span class="badge warn">Needs a Firebase key</span>
      {/if}
    </div>

    <section class="card">
      <header>
        <h2>Firebase key</h2>
        <button class="btn small" type="button" onclick={() => (replacing = !replacing)}>
          {app.hasCredentials ? 'Replace' : 'Add key'}
        </button>
      </header>
      {#if app.hasCredentials}
        <p>Sending through the Firebase project <code>{app.projectId}</code>. The key is stored encrypted and never shown again.</p>
      {:else}
        <p class="notice">Add the project's service-account key to start sending notifications.</p>
      {/if}
      {#if replacing}
        <form class="stack" onsubmit={saveCredentials}>
          <label>
            Service-account key
            <input class="input" type="file" accept="application/json,.json" onchange={readFile} />
            <textarea bind:value={serviceAccount} placeholder="…or paste the JSON here" spellcheck="false"></textarea>
          </label>
          {#if credentialsError}<p class="notice error" role="alert">{credentialsError}</p>{/if}
          <div class="row">
            <button class="btn primary" disabled={credentialsBusy || !serviceAccount.trim()}>
              {credentialsBusy ? 'Saving…' : 'Save key'}
            </button>
            <button class="btn" type="button" onclick={() => (replacing = false)}>Cancel</button>
          </div>
        </form>
      {/if}
    </section>

    <section class="card">
      <header>
        <h2>Access keys</h2>
        <button class="btn primary small" type="button" onclick={openKeyForm}>New access key</button>
      </header>
      {#if app.accessKeys.length === 0}
        <p class="muted">No access keys yet. A site needs one to register devices and send notifications.</p>
      {:else}
        <table class="table">
          <thead>
            <tr>
              <th>Key</th>
              <th class="hide-small">Can</th>
              <th>Devices</th>
              <th class="hide-small">Last used</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each app.accessKeys as key (key.id)}
              <tr class:revoked={key.revokedAt}>
                <td>
                  <div>{key.description || 'Untitled'}</div>
                  <div class="muted small mono">{key.prefix}…</div>
                </td>
                <td class="hide-small">
                  <div class="row" style="gap: 4px">
                    {#each key.permissions as permission}<span class="badge">{permissionLabel(permission)}</span>{/each}
                  </div>
                </td>
                <td>{key.tokens}</td>
                <td class="hide-small muted small">{ago(key.lastUsedAt)}</td>
                <td>
                  {#if key.revokedAt}
                    <span class="badge danger">Revoked</span>
                  {:else}
                    <button
                      class="btn danger small"
                      type="button"
                      onclick={() => {
                        revoking = key;
                        showRevoke = true;
                      }}>Revoke</button
                    >
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </section>

    <section class="card">
      <details>
        <summary><b>Connect a Moodle site</b></summary>
        <div class="stack" style="margin-top: 14px">
          <p class="muted">
            In Moodle, open <i>Site administration → Messaging → Mobile</i> and fill in the push server fields:
          </p>
          <dl class="kv">
            <dt>Server URL</dt>
            <dd><CopyBox value={serverUrl} /></dd>
            <dt>Port</dt>
            <dd><CopyBox value={serverPort} /></dd>
            <dt>App name</dt>
            <dd><CopyBox value={app.name} /></dd>
            <dt>Access key</dt>
            <dd class="muted small">One of the keys above (shown only when it is created)</dd>
            <dt>Mobile app name</dt>
            <dd class="muted small">Your app's package id, for example <code>org.silify.sayan</code></dd>
          </dl>
          <p class="muted small">Then enable the mobile notifications output under <i>Plugins → Message outputs</i>.</p>
        </div>
      </details>
    </section>

    <section class="card">
      <header>
        <h2>Recent activity</h2>
        <button class="btn small" type="button" onclick={load}>Refresh</button>
      </header>
      {#if logs.length === 0}
        <p class="muted">Nothing yet. Sends, new devices and key changes appear here.</p>
      {:else}
        <ul class="log">
          {#each logs as entry}
            <li>
              <span class="muted">{when(entry.at)}</span>
              <span><span class="badge" class:danger={entry.action === 'error'}>{entry.action}</span></span>
              <span>{entry.info}</span>
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <section class="card">
      <h2>Settings</h2>
      <label class="check">
        <input type="checkbox" checked={app.allowKeyRequests} onchange={toggleRequests} />
        <span>
          Let sites request their own access key
          <span class="hint" style="display: block">
            Turns on the "Request access key" button in Moodle. Anyone who can reach this server could then get a key, so
            leave it off unless you need it.
          </span>
        </span>
      </label>
    </section>

    <section class="card danger">
      <header>
        <div class="stack" style="gap: 2px">
          <h2>Delete this app</h2>
          <span class="muted small">Removes its access keys, devices and activity. Sites using it stop working.</span>
        </div>
        <button
          class="btn danger"
          type="button"
          onclick={() => {
            confirmName = '';
            deleting = true;
          }}>Delete…</button
        >
      </header>
    </section>
  {/if}
</div>

<Modal bind:open={creatingKey} title="New access key">
  <form class="stack" onsubmit={createKey}>
    <label>
      Description
      <input class="input" bind:value={description} maxlength="200" placeholder="for example: School A" />
    </label>
    <label>
      Contact <span class="hint">(optional)</span>
      <input class="input" bind:value={contact} maxlength="200" placeholder="who runs the site" />
    </label>
    <fieldset style="border: 0; padding: 0; margin: 0; display: grid; gap: 8px">
      <legend style="font-weight: 550; font-size: 0.92rem; margin-bottom: 6px">What it can do</legend>
      {#each allPermissions as permission}
        <label class="check">
          <input type="checkbox" value={permission} bind:group={permissions} />
          {permissionLabel(permission)}
        </label>
      {/each}
    </fieldset>
    {#if keyError}<p class="notice error" role="alert">{keyError}</p>{/if}
    <div class="row" style="justify-content: flex-end">
      <button class="btn" type="button" onclick={() => (creatingKey = false)}>Cancel</button>
      <button class="btn primary" disabled={keyBusy || permissions.length === 0}>{keyBusy ? 'Creating…' : 'Create key'}</button>
    </div>
  </form>
</Modal>

<Modal bind:open={showIssued} title="Copy your access key">
  {#if issued}
    <div class="stack">
      <p class="notice">This is the only time the key is shown. Copy it now and keep it secret.</p>
      <CopyBox value={issued.key} />
      <p class="muted small">Give it to the person who runs the site, together with the connection details under "Connect a Moodle site".</p>
      <div class="row" style="justify-content: flex-end">
        <button class="btn primary" type="button" onclick={() => (showIssued = false)}>I have copied it</button>
      </div>
    </div>
  {/if}
</Modal>

<Modal bind:open={showRevoke} title="Revoke this key?">
  {#if revoking}
    <div class="stack">
      <p>
        <b>{revoking.description || 'This key'}</b> (<span class="mono">{revoking.prefix}…</span>) will stop working and its
        {revoking.tokens} registered devices will be removed. This cannot be undone.
      </p>
      <div class="row" style="justify-content: flex-end">
        <button class="btn" type="button" onclick={() => (showRevoke = false)}>Cancel</button>
        <button class="btn danger" type="button" onclick={revoke}>Revoke key</button>
      </div>
    </div>
  {/if}
</Modal>

<Modal bind:open={deleting} title="Delete this app?">
  {#if app}
    <form
      class="stack"
      onsubmit={(event) => {
        event.preventDefault();
        remove();
      }}
    >
      <p>This removes <b>{app.name}</b> with everything under it. Type its name to confirm.</p>
      <input class="input" bind:value={confirmName} placeholder={app.name} autocomplete="off" />
      <div class="row" style="justify-content: flex-end">
        <button class="btn" type="button" onclick={() => (deleting = false)}>Cancel</button>
        <button class="btn danger" disabled={confirmName !== app.name}>Delete app</button>
      </div>
    </form>
  {/if}
</Modal>
