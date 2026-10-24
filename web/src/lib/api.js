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

export const api = {
  me: () => request('GET', '/me'),
  login: (email, password) => request('POST', '/login', { email, password }),
  logout: () => request('POST', '/logout'),
  changePassword: (current, next) => request('POST', '/password', { current, new: next }),

  apps: () => request('GET', '/apps'),
  app: (id) => request('GET', `/apps/${id}`),
  createApp: (name, serviceAccount) => request('POST', '/apps', { name, serviceAccount }),
  deleteApp: (id) => request('DELETE', `/apps/${id}`),
  setCredentials: (id, serviceAccount) => request('PUT', `/apps/${id}/credentials`, { serviceAccount }),
  setSettings: (id, allowKeyRequests) => request('PUT', `/apps/${id}/settings`, { allowKeyRequests }),
  logs: (id) => request('GET', `/apps/${id}/logs?limit=50`),

  createKey: (id, key) => request('POST', `/apps/${id}/keys`, key),
  revokeKey: (id, keyId) => request('DELETE', `/apps/${id}/keys/${keyId}`),
};
