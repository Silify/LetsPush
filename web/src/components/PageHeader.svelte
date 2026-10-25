<script>
  import ChevronRight from '@lucide/svelte/icons/chevron-right';

  /**
   * The top of a page: an optional trail of links, the title, a line under it, and room for actions.
   * crumbs: [{ label, href }]; the last one, without an href, is the current page.
   */
  let { title, description = '', crumbs = [], children, actions } = $props();
</script>

<div class="mb-6 grid gap-3">
  {#if crumbs.length}
    <nav class="flex flex-wrap items-center gap-1 text-sm text-muted-foreground" aria-label="Breadcrumb">
      {#each crumbs as crumb, i (i)}
        {#if i > 0}<ChevronRight class="size-3.5" />{/if}
        {#if crumb.href}
          <a class="hover:text-foreground" href={crumb.href}>{crumb.label}</a>
        {:else}
          <span class="text-foreground">{crumb.label}</span>
        {/if}
      {/each}
    </nav>
  {/if}
  <div class="flex flex-wrap items-start justify-between gap-3">
    <div class="grid min-w-0 gap-1">
      <h1 class="truncate text-2xl font-semibold tracking-tight">{title}</h1>
      {#if description}<p class="max-w-prose text-sm text-muted-foreground">{description}</p>{/if}
      {@render children?.()}
    </div>
    {#if actions}<div class="flex shrink-0 items-center gap-2">{@render actions()}</div>{/if}
  </div>
</div>
