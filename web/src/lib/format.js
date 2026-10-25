const dateTime = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' });
const time = new Intl.DateTimeFormat(undefined, { timeStyle: 'medium' });
const relative = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
const number = new Intl.NumberFormat();

/** "Oct 25, 2026, 12:34 PM" */
export function when(value) {
  return value ? dateTime.format(new Date(value)) : '';
}

/** "12:34:56 PM" */
export function clock(value) {
  return value ? time.format(new Date(value)) : '';
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

/** 1234 -> "1,234" */
export function count(value) {
  return number.format(value ?? 0);
}

const permissions = {
  createToken: { label: 'Register devices', hint: 'Add a device so it can receive notifications' },
  deleteToken: { label: 'Remove devices', hint: 'Remove a device it registered' },
  sendNotification: { label: 'Send', hint: 'Send notifications to the devices it registered' },
};

export const allPermissions = Object.keys(permissions);

export function permissionLabel(permission) {
  return permissions[permission]?.label ?? permission;
}

export function permissionHint(permission) {
  return permissions[permission]?.hint ?? '';
}

export async function copyText(text) {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}

/** Reads a chosen file as text, for the Firebase key. */
export async function readFileInput(event) {
  const file = event.target.files?.[0];
  return file ? file.text() : '';
}
