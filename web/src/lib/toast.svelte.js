/** Short messages at the bottom of the page. */

export const toasts = $state([]);

let nextId = 1;

export function toast(message, kind = 'ok') {
  const id = nextId++;
  toasts.push({ id, message, kind });
  setTimeout(() => {
    const index = toasts.findIndex((t) => t.id === id);
    if (index >= 0) toasts.splice(index, 1);
  }, kind === 'error' ? 6000 : 3000);
}
