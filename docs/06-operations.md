# Operations

Running a LetsPush server: deploying, backing up, upgrading, reading the logs and fixing what goes
wrong. To set it up the first time, see [Getting started](02-getting-started.md).

## 1. Deploying

**What runs.** One container, one process, one data file. The image runs as an unprivileged user
(`letspush`, uid 10001), keeps its data in the `/data` volume, and has a health check that fetches
`GET /`.

**HTTPS.** The container listens on `127.0.0.1:8080` only, because sites send access keys and device
tokens. Put a reverse proxy with HTTPS in front of it. With Caddy:

```text
push.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

The proxy must pass `X-Forwarded-Proto` (Caddy, nginx's usual setup and Traefik do), because the
admin cookie is marked `Secure` only when LetsPush is told the request came over HTTPS.

**Behind a proxy**, the address LetsPush sees is the proxy's. The sign-in throttle (five wrong passwords
per address and email in 15 minutes) therefore works per email only. Choose a strong admin password.

**One instance.** Storage is one SQLite file; do not run two servers on the same file, and do not put
several behind a load balancer.

**Settings.** Set `LETSPUSH_SECRET` once and keep it. Remove `LETSPUSH_ADMIN_PASSWORD` after the first
start. All settings are in [Getting started](02-getting-started.md#settings).

## 2. Secrets

| Secret                    | Where it lives                       | If it is lost or leaks                                   |
|---------------------------|--------------------------------------|----------------------------------------------------------|
| `LETSPUSH_SECRET`         | The environment (`.env`)             | Lost: stored Firebase keys cannot be opened, sessions end. Upload the keys again |
| Firebase service-account key | The database, sealed              | Revoke it in Google Cloud, create a new one, upload it in the app's page |
| Access keys               | A hash in the database; the site holds the key | Revoke it in the app's page and give the site a new one |
| Admin password            | A hash in the database               | Change it under *Account*                                |

Changing `LETSPUSH_SECRET` is not a rotation: it makes every stored Firebase key unreadable. Pushes to
those apps then fail with `503` and the activity log says the key could not be opened. To change it
deliberately, change it, then upload each app's Firebase key again.

## 3. Backups

Everything LetsPush knows is in `LETSPUSH_DB` (in Docker, the `letspush-data` volume). The database runs
in WAL mode, so copying only the `.db` file of a running server can give an inconsistent copy.

- **Safest:** stop the container, copy the whole data folder, start it again.
- **While running:** copy the `.db`, `-wal` and `-shm` files together, or use SQLite's backup tools on
  the file.

Back up `LETSPUSH_SECRET` **separately** from the database: the backup is useless without it, and
together they hold your Firebase keys.

Restore by putting the files back before starting the server.

## 4. Upgrading

1. Back up the data (section 3).
2. Pull the new source and rebuild: `docker compose up -d --build`.
3. Watch the log for the first start.

Schema changes are SQL migrations that run automatically at startup, each in a transaction, and are
recorded so they run once. **There is no downgrade.** To go back to an older version, restore the backup
taken before the upgrade. Migration `002` moved each app's Firebase key to the `app_providers` table;
the old columns are gone afterwards.

## 5. Logs

There are two logs with different readers.

**The server's log** goes to standard error (`docker compose logs letspush`). It is plain text, or JSON
lines with `LETSPUSH_LOG_FORMAT=json`. Each request has a line with `method`, `path`, `status`,
`duration_ms` and `request_id`; server errors are logged as errors. Device tokens are hidden in paths.

**The activity log** is per app, in the web interface: devices added and removed, pushes sent, access
keys created and revoked, and failures with the reason Firebase gave. It holds no message text and no
whole device tokens. It is purged after `LETSPUSH_LOG_RETENTION_DAYS` (default 30).

**Request IDs.** Every answer carries `X-Request-Id`. A failed delivery in the activity log ends with
`(request <id>)`; search the server's log for the same ID to see the request. A site can send its own
`X-Request-Id` (letters, digits, `.`, `_`, `-`, up to 64 characters) to follow a request across systems.

## 6. Troubleshooting

Start with the app's activity log, then the server's log.

| Symptom                                                           | Look at / do                                                                  |
|-------------------------------------------------------------------|-------------------------------------------------------------------------------|
| `firebase send failed with status 403: PERMISSION_DENIED …`       | Google's reason follows. Enable the Firebase Cloud Messaging API (V1) for the project, and give the service account a messaging role |
| `…SENDER_ID_MISMATCH`                                             | The device token comes from a different Firebase project than the key         |
| `firebase send failed with status 400 … INVALID_ARGUMENT`         | The token is malformed, or the payload is rejected; read the reason after the status |
| `firebase send failed with status 429` or `5xx`, repeatedly       | Google is limiting or failing. LetsPush already retried three times           |
| `firebase sign-in failed … invalid_grant`                         | The service-account key is wrong or was deleted. Upload a new one             |
| `the app's key could not be opened`                               | `LETSPUSH_SECRET` is not the one the key was stored with                       |
| Moodle shows pushes arriving late or not at all, no errors here   | Check Moodle's HTTP security settings and that it can reach the server        |
| Every push is `202` but nothing arrives                            | The activity log shows what Firebase said later; also check the device's notification permission and that the app uses the same Firebase project |
| *Invalid access key* from Moodle                                   | The key was revoked or belongs to another app                                  |
| The admin page loads but sign-in does not stay                     | The proxy does not send `X-Forwarded-Proto`, or the browser blocks the cookie |
| Locked out after wrong passwords                                   | Wait 15 minutes                                                                |
| Lost the admin password                                            | Change it under *Account* while signed in. If nobody can sign in: stop the server, delete the row in the `admins` table of the database file (with the `sqlite3` tool on the host), and start it with `LETSPUSH_ADMIN_EMAIL` and `LETSPUSH_ADMIN_PASSWORD` set. An admin is created only when there is none |

A `202` from LetsPush means "taken", not "delivered": when Firebase is slow it answers before the result
is known. The activity log always gets the outcome.

---

© 2026 Silify
