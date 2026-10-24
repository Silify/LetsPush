/** A tiny hash router: #/ is the apps list, #/apps/3 an app, #/account the account page. */

export const route = $state({ name: 'apps', id: null });

function read() {
  const path = location.hash.replace(/^#/, '') || '/';
  const app = path.match(/^\/apps\/(\d+)$/);
  if (app) {
    route.name = 'app';
    route.id = Number(app[1]);
  } else if (path === '/account') {
    route.name = 'account';
    route.id = null;
  } else {
    route.name = 'apps';
    route.id = null;
  }
}

window.addEventListener('hashchange', read);
read();

export function go(path) {
  location.hash = path;
}
