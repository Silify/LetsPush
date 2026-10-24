import { toast as sonner } from 'svelte-sonner';

/** A short message at the bottom of the page. */
export function toast(message, kind = 'ok') {
  if (kind === 'error') sonner.error(message);
  else sonner.success(message);
}
