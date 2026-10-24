const dateTime = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' });
const relative = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });

export function when(value) {
  return value ? dateTime.format(new Date(value)) : '';
}

/** "3 minutes ago", or "never" for no date. */
export function ago(value) {
  if (!value) return 'never';
  const seconds = Math.round((new Date(value).getTime() - Date.now()) / 1000);
  for (const [unit, size] of [['day', 86400], ['hour', 3600], ['minute', 60]]) {
    if (Math.abs(seconds) >= size) return relative.format(Math.round(seconds / size), unit);
  }
  return 'just now';
}

const names = {
  createToken: 'Register devices',
  deleteToken: 'Remove devices',
  sendNotification: 'Send',
};

export function permissionLabel(permission) {
  return names[permission] ?? permission;
}

export async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}
