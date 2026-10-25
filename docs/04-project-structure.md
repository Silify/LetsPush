# Project Structure

Where code lives in LetsPush, which packages may import which, and how to decide where something new
goes. The design behind it is in [Architecture](03-architecture.md).

## 1. Mindset

- **Structure serves the dependencies.** A folder exists because it has one job and a clear place in the
  import rules, not because the pattern calls for it.
- **A package is a small idea.** If you cannot say in one sentence what a package is for, it is two
  packages or the wrong one.
- **Add by adding.** A new API, push service or database is a new package plus one line in the wiring.
- **No premature layers.** Do not add a folder, interface or type for a second implementation that does
  not exist yet. Add it when the second one arrives.
- **Everything is `internal/`.** LetsPush is a program, not a library. Nothing outside the module may
  import its packages.

## 2. The big picture

```text
cmd/letspush/        the program: reads settings, wires the pieces, runs the server
internal/
  server/            composition: routes, middleware, startup helpers
  compat/            adapter: the AirNotifier / Moodle-compatible API
  native/            adapter: LetsPush's own API, /v1
  admin/             adapter: the JSON API behind the web interface
  webui/             serves the built web interface (embedded)
  gateway/           the core: devices, permissions, delivery
  fcm/               push service: Firebase Cloud Messaging
  store/             data: SQLite, migrations
  secret/            sealing, hashing, random keys
  config/            settings from the environment
  reqid/             request IDs and the log handler that adds them
web/                 the web interface source (Svelte), built into internal/webui/dist
docs/                these documents
assets/              the logo
scripts/             helper scripts
```

### Allowed dependencies

An arrow means *may import*. Anything not listed is not allowed.

```text
cmd/letspush ──► everything in internal/ it wires
server       ──► compat, native, admin, webui, gateway, store, secret, reqid
compat       ──► gateway, store, secret
native       ──► gateway, store, secret, reqid
admin        ──► store, secret, fcm
fcm          ──► gateway
gateway      ──► store, reqid
store, secret, config, reqid, webui   ──► standard library only
```

The rules that matter:

- **`gateway` never imports an adapter or a push service.** It declares the `Provider` interface; `fcm`
  implements it and so imports `gateway`, not the other way round.
- **Adapters never import each other.** `compat`, `native` and `admin` share nothing but the
  foundations. If two APIs need the same rule, it belongs in `gateway`.
- **`store` imports nothing of ours.** Other packages describe what they need from the database as small
  interfaces of their own, and `*store.Store` satisfies them.
- **Only `cmd/letspush` and `server` know about every package.**
- **`admin` imports `fcm` only to check an uploaded Firebase key.** If more push services come, give
  `gateway` a way to validate a key and drop this import.

## 3. What each folder is for

### `cmd/letspush/`

`main` reads settings, opens the database, builds the gateway with its providers, creates the first
admin, starts the server and shuts down on a signal (waiting for running deliveries). No logic that a
test would want lives here. `version` is set at build time with `-ldflags "-X main.version=…"`.

### `internal/server/`

Builds the one `http.Handler`: mounts the compatible API, the admin API and the web interface, and wraps
them with request IDs, panic recovery, security headers and the access log. Also holds
`EnsureAdmin` (the first admin from the settings) and `PurgeLogs` (the hourly cleanup).

### `internal/compat/`

The AirNotifier-compatible API: `handler.go` (routes, authentication by app name and key), `tokens.go`,
`push.go`, `accesskeys.go`, and `payload.go` (turning Moodle's body into a notification). It makes
decisions only about the **wire format**; rules belong in the gateway. Tests include
`moodle_test.go`, the pinned requests of the real plugin.

### `internal/native/`

LetsPush's own API under `/v1`: `handler.go` (routes, Bearer authentication, the one error shape),
`devices.go`, and `notifications.go` (validating a request and turning it into a `gateway.Notification`).
Its rules about what a request may contain live here, as they are about the wire format; who may send and
what happens to a delivery live in the gateway. See [Native API](11-native-api.md).

### `internal/admin/`

The JSON API for the web interface under `/admin/api/`: sign-in and sessions (`session.go`,
`throttle.go`), apps, access keys and the log. Handlers are thin; `protect` wraps every route that needs
a signed-in admin.

### `internal/webui/`

Serves `internal/webui/dist`, embedded into the binary. A path with no file behind it gets
`index.html`, so the interface's own routes survive a reload. `dist/` is git-ignored except `.gitkeep`.

### `internal/gateway/`

The core. `Service` registers and removes devices, sends notifications (with the bounded wait) and
records the outcome. It declares `Provider`, `Store` and `Opener`, and the sentinel errors every API
maps. See [Architecture](03-architecture.md#4-how-a-push-flows).

### `internal/fcm/`

`client.go` talks to Google (sign-in with a signed assertion, a cached access token, sending, retries,
Google's error reasons). `credentials.go` reads a service-account key. `provider.go` adapts the client
to `gateway.Provider` and caches parsed keys.

### `internal/store/`

All SQL. One file per area (`apps.go`, `keys.go`, `tokens.go`, `logs.go`, `admins.go`), `store.go` for
opening and migrating, and `migrations/*.sql`. It turns driver errors into `ErrNotFound` and
`ErrConflict`.

### `internal/secret/`

`Box` (seal and open with AES-256-GCM), password hashing, access-key generation and hashing, constant
time comparison.

### `internal/config/` and `internal/reqid/`

`config` reads and validates the environment. `reqid` creates and carries request IDs and provides the
log handler that adds `request_id` to records.

### `web/`

The web interface, plain JavaScript with Svelte 5.

```text
web/src/
  main.js, App.svelte, app.css   entry, sign-in gate and page loader, theme (design tokens as CSS variables)
  pages/                         one file per screen: Login, Overview, Apps, Activity, Account, AppPage
  pages/app/                     the tabs of one app: Overview, Keys, Devices, Activity, Firebase, Connect, Settings
  components/                    app-specific pieces: Shell, Nav, ThemeToggle, PageHeader, LogList, StatCard…
  lib/api.js                     the only place that calls /admin/api
  lib/router.svelte.js           the small router (#/apps/3/devices)
  lib/apps.svelte.js             the list of apps, loaded once and shared by the sidebar and the pages
  lib/components/ui/             shadcn-svelte components (generated; edit with care)
web/components.json              settings of the shadcn-svelte CLI
web/scripts/clean.mjs            empties the build folder, keeping .gitkeep
```

## 4. Where does this code go?

| I am adding…                                              | It goes in…                                              |
|-----------------------------------------------------------|----------------------------------------------------------|
| A rule about who may send or register (any API)           | `gateway`                                                |
| A new field or quirk of the Moodle request body           | `compat/payload.go`                                      |
| A new status code or message for the Moodle API           | `compat`, mapped from a gateway error                    |
| A new field or endpoint for new integrations              | `native` (and the `gateway.Notification` if it is sent), then [Native API](11-native-api.md) |
| A new gateway error                                       | `gateway` (a sentinel), then map it in every API         |
| A new push service                                        | A new package implementing `gateway.Provider`            |
| A new table or column                                     | A **new** migration in `store/migrations`, plus `store`  |
| A new setting                                             | `config`, then document it in [Getting started](02-getting-started.md) |
| A new admin screen                                        | `web/src/pages` (a tab of an app: `pages/app`), a call in `lib/api.js`, a handler in `admin` |
| A new setting an app owns                                 | `apps` column (migration), `store`, `admin`, the app page |
| A new log line about a request                            | `slog.*Context(ctx, …)` so it carries the request ID     |
| Something used by two adapters                            | Probably `gateway`; if it is not a rule, `secret` or `reqid` |

## 5. Design rules behind the structure

- **Rules live in one place.** If two APIs would need the same check, it belongs in the gateway.
- **Adapters translate, nothing else.** An adapter parses and authenticates a request, calls the
  gateway, and maps the result. If a handler grows a decision about devices or delivery, move it.
- **The gateway is testable without HTTP or Google.** Keep it that way: no `net/http`, no Firebase types.
- **SQL stays in `store`.** No other package may build a query.
- **Secrets cross packages as bytes or sealed strings, never as logged values.**
- **Context carries the request.** Pass `context.Context` first; the request ID rides in it.

## 6. Smells to avoid

- A handler that talks to the database directly for something other than its own authentication.
- An `if platform == "ios"` in an adapter. Device-specific delivery belongs to the provider choice in
  the gateway.
- A package named `utils`, `common` or `helpers`.
- Importing `compat` from `admin`, or the reverse.
- A migration that was edited after release.
- A log line that includes a message body, an access key or a whole device token.
- A new interface with one method and one implementation that nobody swaps or fakes.

## 7. Changing the structure

If a rule here stops fitting, change this document in the same pull request as the code, so the two stay
true together. Move code before adding rules; add rules before adding exceptions.

---

© 2026 Silify
