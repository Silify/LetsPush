<script>
  import FilterSelect from '../components/FilterSelect.svelte';
  import LogList from '../components/LogList.svelte';
  import PageHeader from '../components/PageHeader.svelte';
  import { api } from '$lib/api.js';
  import { apps, loadApps } from '$lib/apps.svelte.js';

  loadApps();

  let app = $state('');

  const options = $derived([
    { value: '', label: 'All apps' },
    ...(apps.list ?? []).map((a) => ({ value: String(a.id), label: a.name })),
  ]);
</script>

<PageHeader title="Activity" description="Everything that happened across your apps: sends, errors, new devices and key changes." />

<LogList load={(o) => api.logs({ ...o, app })} showApp live reloadKey={app}>
  <FilterSelect bind:value={app} {options} label="All apps" />
</LogList>
