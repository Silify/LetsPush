/**
 * The list of apps, kept once for the whole dashboard: the sidebar, the overview and the apps page all read
 * it, and a change made in one place shows in the others without another request.
 */

import { api } from './api.js';

export const apps = $state({ list: null, error: '' });

let loading = null;

/** Loads the apps. Concurrent calls share one request. */
export function loadApps() {
  loading ??= api
    .apps()
    .then((list) => {
      apps.list = list;
      apps.error = '';
    })
    .catch((e) => {
      apps.error = e.message;
    })
    .finally(() => {
      loading = null;
    });
  return loading;
}

/** Forget what is loaded, for after signing out. */
export function resetApps() {
  apps.list = null;
  apps.error = '';
}
