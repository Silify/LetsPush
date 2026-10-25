# Compatible API

The HTTP API that LetsPush shares with [AirNotifier](https://github.com/dcai/airnotifier), as Moodle's
`message_airnotifier` plugin uses it. This is the reference for what it accepts and answers. The code is
in `internal/compat`; the tests that pin it are in `internal/compat/moodle_test.go`.

A site that works with AirNotifier works with LetsPush by changing its server address, app name and
access key. Changes here must keep that true; see [Changing this API](#changing-this-api).

## Authentication

Every request carries two headers:

| Header           | Value                                                    |
|------------------|----------------------------------------------------------|
| `X-AN-APP-NAME`  | The app's name in LetsPush                               |
| `X-AN-APP-KEY`   | An access key of that app                                |

`X-AN-APP-KEY` may be omitted only for `POST /accesskeys/` (Moodle sends the text `none` there; it is
ignored).

Failures are `400` with `{"error": "…"}`:

| Text                       | When                                                         |
|----------------------------|--------------------------------------------------------------|
| `app name is required`     | No `X-AN-APP-NAME`                                           |
| `app key is required`      | No `X-AN-APP-KEY` where one is needed                        |
| `Invalid application name` | No app has that name                                         |
| `Invalid access key`       | Unknown, revoked, or belonging to another app                |

A key carries permissions: **create token**, **delete token**, **send notification**. A missing
permission is `403`.

Request bodies are limited to 64 KB. Answers are JSON.

## Endpoints

| Request                                  | Purpose                              | Success                                      | Failures                                |
|------------------------------------------|--------------------------------------|----------------------------------------------|-----------------------------------------|
| `GET /`                                  | Connectivity check                   | `200`                                        |                                         |
| `POST /accesskeys/`                      | "Request access key" in Moodle       | `200 {"accesskey": "…"}`                     | `403` unless the app allows it          |
| `POST /tokens/{token}` (form: `device`, `channel`) | Register a device          | `200 {"status":"ok"}` or `{"status":"token exists"}` | `400 Invalid token`, `403`      |
| `DELETE /tokens/{token}`                 | Remove a device                      | `200 {"status":"deleted"}`                   | `404 {"status":"Token doesn't exist"}`, `403` |
| `POST /api/v2/tokens` (JSON: `token`, `device`, `channel`) | Register a device  | as above                                     | `400 invalid JSON`, `400 Invalid token`, `403` |
| `DELETE /api/v2/tokens/{token}`          | Remove a device                      | as above                                     | as above                                |
| `POST /api/v2/push/` (JSON)              | Send a notification                  | `202 {"status":"accepted"}`                  | see below                               |

`GET /` is answered by the web interface's page. Moodle's health check expects `200` or `302`, and the
Docker image's health check uses it too. A trailing slash is optional on the `/api/v2/` routes.

### Registering a device

`device` is the platform the app reports: `Android`, `iOS`, `iOS-fcm`, `web`… It is stored in lowercase.
Every platform is delivered through Firebase today. If `device` is missing, it is `fcm`. `channel`
defaults to `default`.

A token is trimmed, must not be empty and must not exceed 4096 characters. Registering a token the key
already has changes nothing and answers `token exists`.

The token `testtoken` is Moodle's probe for a working access key. It answers `{"status":"ok"}` for a key
that may create tokens and **is not stored**. An invalid key answers `400 Invalid access key`.

### Requesting an access key

`POST /accesskeys/` with a form: `url` (the site's address), `siteid` (ignored) and `description`. It
only works for an app whose administrator has turned on *allow key requests*; otherwise keys are made in
the web interface. The key it creates has all three permissions; the site's address is stored with it.

### Sending

`POST /api/v2/push/` with a JSON body. Moodle sends it with a form content type; the body is parsed as
JSON whatever the content type says.

```json
{
  "device": "Android",
  "token": "<fcm token>",
  "alert": "text, or {\"title\": …, \"body\": …}, or null",
  "extra": { "processor": "moodle", "sitefullname": "…", "smallmessage": "…" },
  "fcm":   { "data": { "…": "…" } }
}
```

Answers:

| Status | Body                                                            | When                                                                  |
|--------|-----------------------------------------------------------------|-----------------------------------------------------------------------|
| `202`  | `{"status":"accepted"}`                                         | Firebase accepted it, or it is still on its way (see below)           |
| `400`  | `invalid JSON` / `Invalid token`                                | Bad body, empty or oversized token                                    |
| `400`  | `Unknown token and you have no permission to create`            | The key has not registered this device and may not now                |
| `403`  | `No permission to send notification`                            |                                                                       |
| `404`  | `Unregistered token`                                            | Firebase no longer knows the device. It is removed. **Moodle deletes its copy only on exactly this answer** |
| `502`  | `error response from fcm`                                       | Firebase refused or could not be reached                              |
| `503`  | `This app has no working Firebase key`                          | The app has no key, or it cannot be opened                            |
| `500`  | `something went wrong`                                          | A bug or a storage failure                                            |

**Timing.** Moodle waits two seconds. LetsPush waits up to 1.5 seconds for Firebase and answers with
its real result; if Firebase is slower it answers `202` and finishes the delivery on its own. A device
found dead after that is still removed, but Moodle is not told. See
[Architecture](03-architecture.md#6-delivery-waiting-retrying-and-errors).

A device that the key has not registered yet is registered by the push if the key may create tokens,
using `device` and `channel` from the body.

### How a push becomes a notification

- **A Moodle push** (`extra.processor` is `moodle`, no `alert`): the title is `extra.sitefullname`
  (or `Notification`), the body is `extra.smallmessage` (or `extra.fullmessage`).
- **`alert` as text**: used as title and body. **As an object**: its `title` and `body`.
- **`alert: null`**, or no text at all: a **data-only** message. Moodle sends this for encrypted
  notifications, where the readable fields are sealed with the device's key.
- **Data**: all of `extra`, then `fcm.data` on top, go to the app as the message's data. Values are
  strings: text as it is, booleans `1` or `0`, numbers as written, nested values as JSON, and `null`
  left out.

The Android and iOS parts of the Firebase message are set to high priority.

## Where LetsPush differs from AirNotifier

| Topic                        | AirNotifier                         | LetsPush                                               |
|------------------------------|-------------------------------------|--------------------------------------------------------|
| Who owns a device            | The app                             | The access key that registered it                      |
| Requesting an access key     | Checks the site on moodle.net       | Allowed per app by an administrator; no outside check  |
| Push services                | FCM, APNs, WNS and others           | Firebase only                                          |
| Other API areas              | Users, objects, broadcast           | Not implemented                                        |
| Firebase keys                | Stored in its database              | Stored sealed with `LETSPUSH_SECRET`                   |

Error texts and status codes follow AirNotifier where Moodle reads them.

## Changing this API

- A Moodle site is a client you cannot update. Do not rename a route, change a status code or an error
  text that appears above, or stop accepting a body shape, unless a test shows Moodle does not read it.
- Add a test to `moodle_test.go` first for any new shape the plugin starts to send. Build it from the
  plugin's source, not from guesses.
- Put the decision in the gateway when it is a rule, and in `compat` when it is a wire detail.

---

© 2026 Silify
