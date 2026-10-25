/**
 * A tiny hash router.
 *
 *   #/                      overview
 *   #/apps                  the apps
 *   #/apps/3  #/apps/3/keys one app, and the tab of it (overview, keys, devices, activity, firebase, connect, settings)
 *   #/activity              the activity of every app
 *   #/account               the signed-in admin
 */

export const appTabs = ['overview', 'keys', 'devices', 'activity', 'firebase', 'connect', 'settings'];

export const route = $state({ name: 'overview', id: null, tab: 'overview' });

function read() {
  const path = location.hash.replace(/^#/, '') || '/';
  const app = path.match(/^\/apps\/(\d+)(?:\/([a-z]+))?\/?$/);

  route.id = null;
  route.tab = 'overview';
  if (app) {
    route.name = 'app';
    route.id = Number(app[1]);
    route.tab = appTabs.includes(app[2]) ? app[2] : 'overview';
  } else if (path.startsWith('/apps')) {
    route.name = 'apps';
  } else if (path === '/activity') {
    route.name = 'activity';
  } else if (path === '/account') {
    route.name = 'account';
  } else {
    route.name = 'overview';
  }
}

window.addEventListener('hashchange', read);
read();

export function go(path) {
  location.hash = path;
}

/** The address of a page, for an <a href>. */
export function href(path) {
  return `#${path}`;
}
