<script>
  import * as Select from '$lib/components/ui/select/index.js';

  /** A small dropdown for a filter. options: [{ value, label }]. */
  let { value = $bindable(''), options, label, class: className = '' } = $props();

  const selected = $derived(options.find((o) => o.value === value)?.label ?? label);
</script>

<Select.Root type="single" bind:value>
  <Select.Trigger class="min-w-36 {className}" aria-label={label}>{selected}</Select.Trigger>
  <Select.Content>
    {#each options as option (option.value)}
      <Select.Item value={option.value} label={option.label}>{option.label}</Select.Item>
    {/each}
  </Select.Content>
</Select.Root>
