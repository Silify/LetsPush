<script>
  /** What to show instead of an empty list: an icon, a title, a line of help and maybe a button. */
  let { icon: Icon, title, description = '', children } = $props();
</script>

<div class="grid place-items-center gap-3 px-4 py-10 text-center">
  <div class="grid size-11 place-items-center rounded-full bg-muted text-muted-foreground"><Icon class="size-5" /></div>
  <div class="grid gap-1">
    <p class="font-medium">{title}</p>
    {#if description}<p class="max-w-sm text-sm text-muted-foreground">{description}</p>{/if}
  </div>
  {@render children?.()}
</div>
