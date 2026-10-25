# Getting Started

How to set up, run, test and connect LetsPush. For running a server in production, see
[Operations](06-operations.md).

LetsPush needs a Firebase project (the free plan is enough) with a service-account key, and a site to
send from. Moodle is the one this project is built for.

## Prerequisites

| Tool            | Version              | Notes                                                        |
|-----------------|----------------------|--------------------------------------------------------------|
| Docker          | recent, with Compose | The usual way to run it                                      |
| Go              | **1.25**             | `go` in `go.mod` is the source of truth. Only to build or test without Docker |
| Node            | **22 or newer**      | Builds the web interface. The Docker image uses Node 24      |
| Git             | any recent           |                                                              |

Without Docker you need Go and Node. With Docker you need neither.

## Run with Docker Compose

```bash
cp .env.example .env
```

Edit `.env`:

- `LETSPUSH_SECRET`: at least 32 characters. Make one with `openssl rand -hex 32`.
- `LETSPUSH_ADMIN_EMAIL` and `LETSPUSH_ADMIN_PASSWORD` (10 characters or more): the first admin.

Then:

```bash
docker compose up -d --build
```

Open <http://127.0.0.1:8080> and sign in. Remove `LETSPUSH_ADMIN_PASSWORD` from `.env` after the first
start: it is only used while there is no admin.

The container listens on `127.0.0.1:8080` only. Put a reverse proxy with HTTPS in front of it before
real use ([Operations](06-operations.md#1-deploying)).

`docker-compose.override.yml` is git-ignored. It is for local needs, such as joining the network of a
local Moodle so that Moodle can reach LetsPush as `http://letspush:8080`.

## Run without Docker

```bash
make build
LETSPUSH_SECRET=… LETSPUSH_ADMIN_EMAIL=… LETSPUSH_ADMIN_PASSWORD=… ./bin/letspush
```

`make build` builds the web interface into `internal/webui/dist` and then one static binary with the
interface embedded. Other targets:

| Target       | Does                                                            |
|--------------|-----------------------------------------------------------------|
| `make web`   | Builds only the interface (`npm ci`, then `npm run build`)      |
| `make test`  | `go vet ./...` and `go test ./...`                              |
| `make run`   | Builds, then runs `./bin/letspush`                              |
| `make clean` | Removes `bin/`                                                  |

`letspush -version` prints the version (set at build time, see
[Versioning conventions](09-versioning-conventions.md#6-source-of-truth-and-git-tags)).

## Settings

All settings come from the environment. There is no config file.

| Variable                       | Default                | Meaning                                                         |
|--------------------------------|------------------------|-----------------------------------------------------------------|
| `LETSPUSH_SECRET`              | none, **required**     | At least 32 characters. Seals stored Firebase keys and signs sessions. Losing it makes stored keys unreadable |
| `LETSPUSH_ADDR`                | `127.0.0.1:8080`       | Where to listen                                                 |
| `LETSPUSH_DB`                  | `./data/letspush.db`   | The SQLite file                                                 |
| `LETSPUSH_ADMIN_EMAIL`, `LETSPUSH_ADMIN_PASSWORD` | none | Create the first admin if there is none. Set both or neither |
| `LETSPUSH_LOG_RETENTION_DAYS`  | `30`                   | How long the activity log is kept                               |
| `LETSPUSH_LOG_FORMAT`          | `text`                 | The server's own log: `text` or `json`                          |

The Docker image sets `LETSPUSH_ADDR=0.0.0.0:8080` and `LETSPUSH_DB=/data/letspush.db`, and keeps the
data in the `letspush-data` volume. The build arguments `NODE_IMAGE`, `GO_IMAGE`, `RUN_IMAGE` and
`GOPROXY` (in `.env`) select base images and a Go proxy, for example a registry mirror.

`internal/config/config.go` reads and checks every setting at startup and stops with a clear message
when one is wrong.

## First setup

1. **New app.** Give it a name. Sites send the name as `X-AN-APP-NAME`. Upload the Firebase
   service-account key: *Firebase console → Project settings → Service accounts → Generate new private
   key*. The key is optional now and needed before sending.
2. **New access key.** Copy it when it is shown. It is stored only as a hash and cannot be shown again.
3. **Connect a site.** The app's page lists exactly what to enter in Moodle.

An app can also let sites ask for their own access key (the *Request access key* button in Moodle). That
is off until you turn it on in the app's settings.

### Firebase

The service account needs two things in the Firebase project's Google Cloud settings:

- The **Firebase Cloud Messaging API (V1)** is enabled.
- The account has a role that includes the `cloudmessaging.messages.create` permission, for example
  *Firebase Cloud Messaging API Admin*.

The device token must come from an app of the **same** Firebase project as the key. A key from another
project is refused. When Firebase refuses a push, the app's activity log says why; see
[Troubleshooting](#troubleshooting).

### Connect Moodle

In Moodle, *Site administration → Messaging → Mobile*, set:

| Field             | Value                                                              |
|-------------------|--------------------------------------------------------------------|
| Server URL        | your server's address, for example `https://push.example.com`     |
| Port              | `443` (or your port)                                               |
| App name          | the app's name in LetsPush                                         |
| Access key        | a key from the app's page                                          |
| Mobile app name   | the mobile app's package id                                        |

Then enable the mobile notifications output under *Plugins → Message outputs*. Moodle's own health
check on that page flags a wrong address or a wrong key.

Moodle blocks requests to private addresses and to ports other than 80 and 443 by default. If LetsPush
is on a private network, allow its host and port under *Site administration → General → Security →
HTTP security*.

## Develop

Run the server (the interface is whatever was last built into it), then the interface with live reload:

```bash
# terminal 1: a server on :8080
LETSPUSH_SECRET=dev-secret-dev-secret-dev-secret-123 LETSPUSH_ADMIN_EMAIL=a@a.test \
  LETSPUSH_ADMIN_PASSWORD=devpassword123 go run ./cmd/letspush

# terminal 2: the interface on :5173, proxying /admin/api to :8080
cd web && npm ci && npm run dev
```

The first command uses values for local development only. Never reuse them anywhere real.

Test by hand with `curl` against the compatible API, with no Moodle and no phone:

```bash
H="-H X-AN-APP-NAME:myapp -H X-AN-APP-KEY:<key>"
curl $H -X POST localhost:8080/api/v2/tokens -H 'Content-Type: application/json' \
  -d '{"token":"<fcm-token>","device":"android"}'
curl $H -X POST localhost:8080/api/v2/push/ -H 'Content-Type: application/json' \
  -d '{"token":"<fcm-token>","alert":{"title":"Hi","body":"test"}}'
```

A real FCM token is needed for a notification to arrive. A made-up token still exercises sign-in and
the permission check against Firebase, which is where setup mistakes show up.

### Tests

```bash
make test
```

Tests use an in-memory SQLite database and fake push services, so they need no network and no
Firebase. They are the contract for the compatible API: do not loosen a test in
`internal/compat/moodle_test.go` to make a change pass. See
[Architecture](03-architecture.md#9-testing).

### The web interface

`web/` is a Svelte 5 app (plain JavaScript) built by Vite into `internal/webui/dist`, which the Go
binary embeds. `.gitkeep` in that folder keeps the embed working before the first build; the rest of
the folder is git-ignored. Components in `web/src/lib/components/ui/` come from
[shadcn-svelte](https://shadcn-svelte.com); `web/components.json` tells its CLI how to generate them.
Regenerating a component overwrites local edits to it.

## Troubleshooting

| Symptom                                                        | Likely cause / fix                                                         |
|----------------------------------------------------------------|----------------------------------------------------------------------------|
| Startup stops with *LETSPUSH_SECRET must be at least 32 characters* | Set a longer secret                                                   |
| Stored Firebase keys stop working after a restart              | `LETSPUSH_SECRET` changed; put the old value back                           |
| *there is no admin yet* in the log                             | Set the admin email and password and restart                                |
| The page says the interface *has not been built*               | Run `make build`, or build the Docker image                                 |
| Activity log: `firebase send failed with status 403: …`        | Read Google's reason after the status. Usually the FCM API is disabled for the project, the service account lacks a messaging role, or the token belongs to another project |
| Activity log: `sign-in failed with status 400: invalid_grant …` | The service-account key is wrong or revoked; upload a new one              |
| Moodle: *Invalid access key*                                   | The key is revoked, belongs to another app, or was mistyped                 |
| Moodle: the push is `404 Unregistered token`                   | The device is gone from Firebase; LetsPush removed it and Moodle forgets it |
| Moodle cannot reach the server                                 | Moodle's HTTP security blocks the host or port; see *Connect Moodle*        |

More in [Operations](06-operations.md#6-troubleshooting).

---

© 2026 Silify
