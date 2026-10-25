# Architecture

How LetsPush is built and why. Read this before changing the structure; for where a piece of code goes,
see [Project structure](04-project-structure.md).

## 1. What drives the design

- **Moodle compatibility is a contract.** Real Moodle sites send real requests and look for exact
  answers (for example `404` with `Unregistered token`). The API in front of the gateway is shaped by
  that, and tests pin it.
- **More than Moodle later.** More kinds of sites, more push services (APNs, Web Push), a native API
  and other databases are expected. None of them should need edits across the code.
- **Small to run.** One binary, one data file, no other service. An operator can understand it in an
  afternoon.
- **Secrets are real.** A server holds Firebase keys that can send as the customer's app. Keys are
  sealed, access keys are hashed, and logs never hold message text.
- **Moodle is slow to forgive.** Moodle waits **two seconds** for a push and gives up silently. The
  gateway must behave well when Firebase is slower than that.

## 2. Principles

- **A core that knows nothing about the outside.** The `gateway` package registers devices and delivers
  notifications. It does not know HTTP, Moodle or Firebase.
- **APIs and push services plug in.** An API turns its requests into gateway calls. A push service
  implements one interface. Neither knows the other.
- **Interfaces belong to the consumer.** A package declares the small interface it needs
  (`gateway.Store`, `gateway.Provider`, `compat.Store`); the concrete type satisfies it without knowing.
  There is no shared "interfaces" package.
- **Errors are values, and the edge translates them.** The gateway returns sentinel errors
  (`ErrUnregistered`, `ErrNoPermission`…). Each API maps them to its own status codes and texts.
- **Time and randomness are injected** where tests need them (`now`, a client's `Sleep`).
- **Simple and boring.** The standard library first. One dependency (the SQLite driver).

## 3. Layers

```text
 sites (Moodle…)                 admin (browser)
        │                              │
        ▼                              ▼
 ┌─────────────┐                ┌─────────────┐
 │ api/compat  │                │   admin     │   adapters: HTTP in, gateway/store calls out
 └──────┬──────┘                └──────┬──────┘
        │                              │
        ▼                              │
 ┌─────────────┐   ┌───────────────┐   │
 │   gateway   │──►│ Provider (fcm)│──►│ Firebase     (push services, behind one interface)
 └──────┬──────┘   └───────────────┘   │
        ▼                              ▼
 ┌────────────────────────────────────────┐
 │ store (SQLite)        secret (sealing) │          foundations
 └────────────────────────────────────────┘
```

| Layer            | Package                | Job                                                                  |
|------------------|------------------------|----------------------------------------------------------------------|
| Composition      | `cmd/letspush`, `internal/server` | Build the pieces, wire them, serve HTTP, shut down cleanly |
| Adapters         | `internal/compat`, `internal/admin`, `internal/webui` | Translate requests; no business rules |
| Core             | `internal/gateway`     | Permissions, device registration, picking a provider, delivery, logging |
| Push services    | `internal/fcm`         | Implement `gateway.Provider` for Firebase                            |
| Foundations      | `internal/store`, `internal/secret`, `internal/config`, `internal/reqid` | Data, sealing and hashing, settings, request IDs |

`internal/server` is the only place that knows every package. Dependencies point down: adapters call the
core; the core never calls an adapter. See the import rules in
[Project structure](04-project-structure.md#2-the-big-picture).

## 4. How a push flows

A Moodle site sends `POST /api/v2/push/`:

1. **Request ID.** `server` gives the request an ID (the caller's `X-Request-Id` if it is fit to be
   logged) and puts it in the context.
2. **Authenticate.** `compat` reads `X-AN-APP-NAME` and `X-AN-APP-KEY`, finds the app, and finds the key
   by its hash. A key from another app, a revoked key, or an unknown name is `400`.
3. **Translate.** `compat` turns the AirNotifier body (`token`, `device`, `alert`, `extra`, `fcm`) into
   a `gateway.Message`: a title, a body and string data. The rules are in
   [Compatible API](05-compatible-api.md#how-a-push-becomes-a-notification).
4. **Send.** `gateway.Send`:
   1. checks the key may send;
   2. finds the device among the key's own, or registers it if the key may;
   3. picks the provider for the device and opens the app's sealed key;
   4. starts the delivery on its own goroutine and **waits up to 1.5 seconds** for it.
5. **Deliver.** The `fcm` provider signs in to Google (the access token is cached and renewed), sends,
   and retries when Google is busy. It reports `ErrUnregistered` for a dead device.
6. **Record.** The delivery logs the outcome to the app's activity log and the server log. A dead device
   is deleted.
7. **Answer.** If the provider answered in time, `Send` returns its result. If not, it returns
   **Pending**, and `compat` answers `202`. The delivery finishes by itself.

Registering a device (`POST /tokens/{token}`) and removing one go through `gateway.Register` and
`gateway.Remove`, so a native API will share their rules.

## 5. Keys, devices and permissions

- An **app** is one customer app: a name, its provider keys, and whether sites may request their own
  access key.
- An **access key** belongs to an app and carries permissions: *create token*, *delete token*, *send
  notification*. Only its SHA-256 hash is stored, plus its first six characters to recognise it in a
  list. It is shown once.
- A **device token belongs to the key that registered it.** A key can send only to its own devices,
  and revoking or deleting a key removes its devices. This is stricter than AirNotifier, where a
  device belongs to the app, and it is deliberate: one leaked key reaches only one site's devices.
- A key that may create tokens registers an unknown device on first use, as AirNotifier does.

## 6. Delivery: waiting, retrying and errors

### The bounded wait

Moodle waits two seconds. Cancelling a Firebase call because the caller hung up would lose the
notification, and answering only after Firebase finished would make Moodle give up on slow days. So:

- The delivery runs on its own context, derived from the request's values but **not** its
  cancellation, with a 60-second limit.
- `Send` waits up to 1.5 seconds. A quick result is returned as it is, so Moodle still gets
  `404 Unregistered token` and can clean up.
- A slow delivery returns **Pending** (`202`). Its result still reaches the log, and a dead device
  found late is still removed.
- On shutdown `main` waits for deliveries still running (`Service.Wait`).

### Retrying

The Firebase client (`internal/fcm`) tries a send up to three times when Google answers `429` or `5xx`,
the network drops the request, or the access token is refused (it fetches a new one). The pause
doubles from 300 ms with jitter, honours `Retry-After`, and never exceeds 10 seconds. Other refusals
(`400`, `403`, `404`) are not retried. Rarely a retry after a lost connection can deliver the same
notification twice.

### Errors

The gateway returns sentinel errors; the compatible API maps them:

| Gateway error      | Meaning                                              | Compatible API answer                              |
|--------------------|------------------------------------------------------|----------------------------------------------------|
| `ErrNoPermission`  | The key may not do this                              | `403` `No permission to …`                         |
| `ErrInvalidToken`  | The token is empty or too long                       | `400` `Invalid token`                              |
| `ErrUnknownToken`  | Unknown device and the key may not register it       | `400` `Unknown token and you have no permission to create` |
| `ErrNoProvider`    | The app has no working key for this device           | `503` `This app has no working Firebase key`       |
| `ErrUnregistered`  | The push service no longer knows the device          | `404` `Unregistered token`                         |
| `ErrDelivery`      | The push service refused or was unreachable          | `502` `error response from fcm`                    |
| anything else      | A bug or a storage failure                           | `500` `something went wrong`                       |

The compatible API's bodies are `{"error": "…"}`. Google's own explanation of a refusal goes to the
logs and never to the calling site.

## 7. Security and privacy

- **Firebase keys** are sealed with AES-256-GCM using a key derived from `LETSPUSH_SECRET` (PBKDF2). The
  database file is created with mode `0600`.
- **Access keys** are random 128-bit values, shown once, stored as SHA-256 hashes.
- **Admin passwords** use PBKDF2-SHA-256. Sign-in sets a signed `HttpOnly`, `SameSite=Strict` cookie
  valid for 12 hours. Requests that change something also need an `X-Requested-With: letspush` header
  that other sites cannot add. After five wrong passwords for one address and email, further tries are
  refused for 15 minutes. Unknown emails take as long as known ones to answer.
- **Logs hold no message text, no access keys and no whole device tokens.** A token appears as its first
  characters. The access log hides tokens in paths.
- **Security headers** (CSP, `X-Frame-Options`, `nosniff`, no referrer) are sent on every answer.
- Request bodies and the number of tracked sign-in failures are capped.

## 8. Data and storage

SQLite, one file, opened in WAL mode with foreign keys on and one connection (SQLite writes are serial
anyway). Timestamps are integers (Unix seconds).

| Table           | Holds                                                                         |
|-----------------|-------------------------------------------------------------------------------|
| `admins`        | Email and password hash                                                       |
| `apps`          | Name, whether sites may request keys                                          |
| `app_providers` | An app's sealed key per push service: `(app_id, kind)`, label, credentials    |
| `access_keys`   | Hash, prefix, description, contact, permissions, last used, revoked           |
| `tokens`        | Device token, platform, channel, owning key; unique per `(key, token)`        |
| `logs`          | The activity log: app, action, text, time                                     |

**Migrations** are SQL files in `internal/store/migrations`, applied in name order, each in a
transaction and recorded in `schema_migrations`. A migration is never edited after release; add a new
one. A key's `last_used_at` is written at most once a minute. Old log entries are purged hourly after
`LETSPUSH_LOG_RETENTION_DAYS`.

The rest of the code sees the database only through the `Store` interfaces each package declares. SQL
stays inside `internal/store`, which is what makes another database possible later (see
[Extending](#11-extending)).

## 9. Testing

- **Go tests next to the code**, run with `make test`. They use `store.Open(":memory:")` and fake
  providers; nothing touches the network.
- **The compatible API is pinned** by `internal/compat/moodle_test.go`, which sends requests built
  as the current `message_airnotifier` plugin builds them. Treat a failure there as a broken promise to
  real sites, not as a test to update.
- **Providers are tested against a fake Google** (`httptest`), including sign-in, caching, retries and
  error reasons.
- **The gateway is tested for its timing promises**: the bounded wait, a caller hanging up, a dead
  device found late.
- There are no tests for the web interface yet.

## 10. Deliberate non-goals

Not built, on purpose, for now:

- APNs directly, Web Push directly, and other providers (all delivery goes through Firebase).
- Topics, broadcast and scheduling.
- A native API (planned as `/v1`: Bearer authentication, one JSON error shape, asynchronous delivery
  with a status to look up).
- More than one instance: storage is a single SQLite file.
- More than one admin, or roles.
- Other databases (PostgreSQL, MySQL), though the structure leaves room for them.
- A delivery queue that survives a restart. A delivery in flight when the server stops is finished
  during shutdown if it can be, and lost otherwise.

## 11. Extending

**A push service.** Add a package that implements `gateway.Provider`
(`Send(ctx, credentials, device, notification) error`, returning `gateway.ErrUnregistered` for a dead
device), register it under a kind in `cmd/letspush/main.go`, store its key in `app_providers` under
that kind, and teach `gateway.kindFor` which devices use it. The admin interface only knows how to take
a Firebase key today, so it needs a way to take the new one too.

**An API.** Add a package beside `compat`. Authenticate the request, build a `gateway.Message`, call
`Send`, `Register` or `Remove`, and map the sentinel errors. Mount its routes in `internal/server`.
Business rules belong in the gateway, not in the new package.

**A database.** Implement the `Store` interfaces of the packages that use one, with the same behavior,
in its own package with its own migrations; run the existing store tests against it. Keep SQL out of
every other package.

---

© 2026 Silify
