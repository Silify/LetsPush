/** The admin API. Every change carries a header a page on another site cannot add. */

export class ApiError extends Error {
  constructor(status, message) {
    super(message);
    this.status = status;
  }
}

let onSignedOut = () => {};

/** Called when the server says the session is over. */
export function whenSignedOut(callback) {
  onSignedOut = callback;
}

async function request(method, path, body) {
  const headers = { 'X-Requested-With': 'letspush' };
  if (body !== undefined) headers['Content-Type'] = 'application/json';

  let response;
  try {
    response = await fetch('/admin/api' + path, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      credentials: 'same-origin',
    });
  } catch {
    throw new ApiError(0, 'Cannot reach the server.');
  }

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    if (response.status === 401 && path !== '/login') onSignedOut();
    throw new ApiError(response.status, data.error || 'Something went wrong.');
  }
  return data;
}

/** "?a=1&b=2" from the values that are set. */
function query(params) {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') search.set(key, value);
  }
  const text = search.toString();
  return text ? `?${text}` : '';
}

export const api = {
  me: () => request('GET', '/me'),
  login: (email, password) => request('POST', '/login', { email, password }),
  logout: () => request('POST', '/logout'),
  changePassword: (current, next) => request('POST', '/password', { current, new: next }),

  overview: () => request('GET', '/overview'),
  /** Every app's activity. Options: app, action, before, limit. Resolves to { items, next }. */
  logs: (options = {}) => request('GET', `/logs${query(options)}`),

  apps: () => request('GET', '/apps'),
  app: (id) => request('GET', `/apps/${id}`),
  createApp: (name, serviceAccount) => request('POST', '/apps', { name, serviceAccount }),
  deleteApp: (id) => request('DELETE', `/apps/${id}`),
  setCredentials: (id, serviceAccount) => request('PUT', `/apps/${id}/credentials`, { serviceAccount }),
  setSettings: (id, allowKeyRequests) => request('PUT', `/apps/${id}/settings`, { allowKeyRequests }),
  /** One app's activity. Options: action, before, limit. Resolves to { items, next }. */
  appLogs: (id, options = {}) => request('GET', `/apps/${id}/logs${query(options)}`),
  /** One app's devices, newest first. Options: before, limit. Resolves to { items, next }. */
  devices: (id, options = {}) => request('GET', `/apps/${id}/devices${query(options)}`),
  removeDevice: (id, deviceId) => request('DELETE', `/apps/${id}/devices/${deviceId}`),

  createKey: (id, key) => request('POST', `/apps/${id}/keys`, key),
  revokeKey: (id, keyId) => request('DELETE', `/apps/${id}/keys/${keyId}`),
};
