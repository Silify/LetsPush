<p align="center"><img src="assets/logo.svg" alt="LetsPush" width="192" height="192"></p>

# LetsPush

A small push notification gateway. Your sites and apps send it notifications over a simple HTTP API, and
it delivers them to phones through Firebase Cloud Messaging.

It speaks the push API that **Moodle's mobile notifications** use, so a Moodle site can send through it
with no plugin and no change to Moodle: its admin only fills in a few settings. A native API for other
platforms is planned.

**Status:** early. The Moodle-compatible API and the admin interface work and are tested; see
[what is not here yet](#not-here-yet).

- One Go binary with the web interface inside. SQLite for storage, no other service to run.
- Firebase keys are stored encrypted; access keys are stored only as hashes.
- Each site can only reach the devices it registered itself.

## Run it

You need Docker and a Firebase project (the free plan is enough) with a service-account key.

```bash
cp .env.example .env
```

Edit `.env`: set `LETSPUSH_SECRET` (`openssl rand -hex 32`), and the first admin's email and password.
Then:

```bash
docker compose up -d --build
```

Open <http://127.0.0.1:8080>, sign in, and:

1. **New app.** Give it a name (sites send it as the app name) and upload the Firebase service-account key
   (*Firebase console → Project settings → Service accounts → Generate new private key*).
2. **New access key.** Copy it when it is shown; it cannot be shown again.
3. **Connect a site.** The app page lists exactly what to enter in Moodle.

Put a reverse proxy with HTTPS in front before real use; the container only listens on `127.0.0.1`. With
Caddy, `push.example.com { reverse_proxy 127.0.0.1:8080 }` is enough.

Remove `LETSPUSH_ADMIN_PASSWORD` from `.env` after the first start. It is only used while there is no admin.

### Without Docker

```bash
make build
LETSPUSH_SECRET=… LETSPUSH_ADMIN_EMAIL=… LETSPUSH_ADMIN_PASSWORD=… ./bin/letspush
```

Needs Go 1.25 and Node 22 or newer.

## Settings

All through the environment.

| Variable | Default | |
|---|---|---|
| `LETSPUSH_SECRET` | — | **Required**, at least 32 characters. Encrypts stored Firebase keys and signs sessions. Losing it makes stored keys unreadable. |
| `LETSPUSH_ADDR` | `127.0.0.1:8080` | Where to listen. |
| `LETSPUSH_DB` | `./data/letspush.db` | The SQLite file. Back it up. |
| `LETSPUSH_ADMIN_EMAIL`, `LETSPUSH_ADMIN_PASSWORD` | — | Create the first admin if there is none. Password: 10 characters or more. |
| `LETSPUSH_LOG_RETENTION_DAYS` | `30` | How long the activity log is kept. |
| `LETSPUSH_LOG_FORMAT` | `text` | How the server writes its own log to standard error: `text` or `json`. |

## Connecting Moodle

In Moodle, *Site administration → Messaging → Mobile*, set the push server fields:

| Field | Value |
|---|---|
| Server URL | your server's address, for example `https://push.example.com` |
| Port | `443` (or your port) |
| App name | the app's name in LetsPush |
| Access key | a key from the app's page |
| Mobile app name | the mobile app's package id |

Then enable the mobile notifications output under *Plugins → Message outputs*. Moodle's own health check
on that page flags a wrong address or a wrong key.

Moodle blocks requests to private addresses and to ports other than 80 and 443 by default; if the server
is on a private network, allow its host and port under *Site administration → General → Security → HTTP
security*.

## The compatible API

All requests carry `X-AN-APP-NAME` (the app's name) and `X-AN-APP-KEY` (an access key).

| Request | Purpose | Success | Failures |
|---|---|---|---|
| `GET /` | connectivity check | `200` | |
| `POST /accesskeys/` | "Request access key" in Moodle. Off unless the app allows it. | `200 {"accesskey": …}` | `403` when not allowed |
| `POST /tokens/{token}` (form: `device`) | register a device | `200 {"status":"ok"}` or `{"status":"token exists"}` | `400 Invalid token`, `403` |
| `DELETE /tokens/{token}` | remove a device | `200 {"status":"deleted"}` | `404` |
| `POST /api/v2/tokens` (JSON: `token`, `device`, `channel`) | register a device | as above | |
| `POST /api/v2/push/` (JSON: `token`, `device`, `alert`, `extra`, `fcm`) | send | `202` | `404 {"error":"Unregistered token"}` for a dead device |

Common errors: `400` with `app name is required`, `app key is required`, `Invalid application name` or
`Invalid access key`; `403` when the key lacks the permission.

How a push becomes a notification:

- A Moodle push (its `extra` has `processor: "moodle"` and no `alert`): title is the site's full name, body
  is the short message.
- `alert` as text: used as title and body. As an object: its `title` and `body`.
- `alert: null`, or no text at all: a data-only message, which Moodle uses for encrypted notifications.
- All of `extra` (and `fcm.data` on top) goes to the app as the message's data, as strings.

Delivery: LetsPush waits up to 1.5 seconds for Firebase and answers with the real result (a dead device is
`404 Unregistered token`). If Firebase is slower, it answers `202` and finishes the delivery on its own, so a
caller that gives up does not lose the notification. A busy or unreachable Firebase (429, 5xx, dropped
connection) is retried up to three times with a growing pause. Firebase's own explanation of a refusal, for
example `PERMISSION_DENIED`, goes to the app's activity log, never back to the site.

Every answer carries an `X-Request-Id`. Send your own `X-Request-Id` to follow a request, or take the one in
the answer and search the server's log (and the activity log, for failures) for it.

A device token belongs to the access key that registered it. A key can only send to its own devices, and a
revoked key's devices go with it. An unknown token is registered on first use if the key may.

## Security notes

- Access keys are shown once and stored as SHA-256 hashes. Firebase service-account keys are sealed with
  AES-256-GCM using a key derived from `LETSPUSH_SECRET`. Passwords use PBKDF2-SHA-256.
- Admin sign-in uses a signed, `HttpOnly`, `SameSite=Strict` cookie. Changes also need a header that other
  sites cannot add. Repeated wrong passwords from one address and email are refused for 15 minutes.
- The activity log never holds message text, whole device tokens or access keys.
- Anyone with an access key can send to the devices that key registered; treat keys as secrets and give each
  site its own.

## Not here yet

APNs and other platforms (all sends go through Firebase), topics and broadcast, scheduling, a native API for
other platforms, and more than one admin or role. Several instances at once are not supported: storage is a
single SQLite file.

## Development

```bash
make test        # go vet and go test
cd web && npm run dev   # the interface, proxied to a running server on :8080
```

The interface (`web/`, Svelte and Vite) builds into `internal/webui/dist`, which the binary embeds.

## License

Apache License 2.0. See [LICENSE](LICENSE).
