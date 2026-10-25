# Native API

Let's Push's own API, under `/v1`. Use it for anything new: a Moodle plugin of your own, another website, a
script. It sends a title, a body, an image and any data you like, and tells you how the delivery ended.
The AirNotifier-compatible API for Moodle's stock plugin is separate; see
[Compatible API](05-compatible-api.md). The code is in `internal/native`.

## Quick example

Create an app and an access key in the web interface, then:

```bash
KEY=<access key>
HOST=https://push.example.com

# 1. Register a device (the app does this once, or you do it for it)
curl -X POST $HOST/v1/devices -H "Authorization: Bearer $KEY" \
  -d '{"token": "<fcm token>", "platform": "android"}'

# 2. Send
curl -X POST $HOST/v1/notifications -H "Authorization: Bearer $KEY" -d '{
  "to": {"token": "<fcm token>"},
  "title": "Assignment graded",
  "body": "Your essay was graded",
  "image": "https://school.example/img/a.png",
  "data": {"course": "12", "deepLink": "/mod/assign/5"},
  "options": {"priority": "high", "ttl": 3600, "sound": "default"}
}'
# 202 {"id": "ntf_5b3c…", "status": "sent"}

# 3. Ask later how it ended
curl $HOST/v1/notifications/ntf_5b3c… -H "Authorization: Bearer $KEY"
```

## Authentication

Every request carries the access key as a Bearer token:

```text
Authorization: Bearer <access key>
```

The key identifies the app, so there is no app-name header. It has the same three permissions as in the
compatible API: **create token**, **delete token** and **send notification**. A key reaches only the
devices it registered itself.

Use the same key for both APIs if you like; keys are not tied to one of them.

## Endpoints

| Request                          | Purpose                                         | Success                              |
|----------------------------------|-------------------------------------------------|--------------------------------------|
| `POST /v1/devices`               | Register a device                               | `201` new, `200` already registered  |
| `DELETE /v1/devices/{token}`     | Remove a device                                 | `204`                                |
| `POST /v1/notifications`         | Send a notification to one device               | `202`                                |
| `GET /v1/notifications/{id}`     | How a notification you sent ended               | `200`                                |

Bodies are JSON, at most 64 KB. **Unknown fields are an error**, so a misspelt option is not silently
ignored.

### Register a device

```json
{ "token": "<device token>", "platform": "android", "channel": "default" }
```

`platform` is required: lowercase letters, digits, `-` or `_`, up to 32 characters, such as `android`,
`ios` or `web` (case is ignored). `channel` is optional and defaults to `default`. The answer is
`{"platform": "android", "created": true}`; registering the same device again changes nothing and answers
`200` with `"created": false`.

A device that is not registered yet is also registered by the first send to it, if the key may create
devices and `to.platform` is given.

### Send a notification

```json
{
  "to": { "token": "<device token>", "platform": "android" },
  "title": "…",
  "body": "…",
  "image": "https://…",
  "data": { "key": "value" },
  "options": { "priority": "high", "ttl": 3600, "sound": "default" }
}
```

| Field               | Required | Meaning                                                                         |
|---------------------|----------|---------------------------------------------------------------------------------|
| `to.token`          | yes      | The device token, 1 to 4096 characters                                          |
| `to.platform`       | no       | Registers the device on first use                                               |
| `title`, `body`     | one of `title`, `body` or `data` | What the person sees                                   |
| `image`             | no       | An `https` address of at most 2048 characters. Needs a `title` or a `body`      |
| `data`              | no       | Text values only. The app receives them as the message's data                   |
| `options.priority`  | no       | `normal` or `high`. Default `high`: reach the phone at once                     |
| `options.ttl`       | no       | Seconds Firebase keeps trying if the device is off, 1 to 2419200 (28 days)      |
| `options.sound`     | no       | A sound name; `default` is the device's usual one                               |

**Data only.** Leave out `title` and `body` and send `data` alone: the message carries no notification, and
the app decides what to show. This is how encrypted or silent messages work.

**Limits.** Title, body, image and all data keys and values together may be at most 4000 bytes. Data keys
must not be empty, must not be `from`, `notification` or `message_type`, and must not start with `google`
or `gcm`; Firebase keeps those.

The answer is `202` with `{"id": "ntf_…", "status": "sent"}` or `"pending"`:

- **`sent`**: Firebase accepted the message. That means "handed to the push service", not "the person has
  seen it".
- **`pending`**: Firebase had not answered after about 1.5 seconds. The delivery carries on, with
  retries, and `GET /v1/notifications/{id}` tells you how it ends.

A device the push service no longer knows is a `404` with `device_unregistered`, and Let's Push removes it.
Treat that as "forget this token".

### Ask how a notification ended

`GET /v1/notifications/{id}` answers:

```json
{ "id": "ntf_…", "status": "sent", "requestId": "9f2c…", "createdAt": "2026-10-25T12:00:00Z", "updatedAt": "2026-10-25T12:00:01Z" }
```

`status` is one of `pending`, `sent`, `failed` (the push service refused it or could not be reached) and
`unregistered` (it no longer knows the device). The reason for a failure is in the app's activity log, which
the administrator can read; `requestId` is how they find it. Only the key that sent a notification can ask
about it, and the record is kept for the log retention period (30 days by default). It holds no message text.

## Errors

Every error has one shape:

```json
{ "error": { "code": "invalid_request", "message": "options.priority must be \"normal\" or \"high\"", "requestId": "9f2c…" } }
```

`code` is stable and meant for programs; `message` is for people. `requestId` is the same as the
`X-Request-Id` header of the answer; give it to the administrator when asking about a failure.

| Status | `code`                | When                                                                         |
|--------|-----------------------|------------------------------------------------------------------------------|
| `400`  | `invalid_request`     | A body or field is not valid. The message says which                         |
| `401`  | `missing_key`         | No `Authorization: Bearer` header                                            |
| `401`  | `invalid_key`         | Unknown or revoked key                                                       |
| `403`  | `forbidden`           | The key lacks the permission                                                 |
| `404`  | `not_found`           | No such endpoint, device or notification (for this key)                      |
| `404`  | `unknown_device`      | The key has not registered the device and may not now                        |
| `404`  | `device_unregistered` | The push service no longer knows the device; it was removed                  |
| `502`  | `delivery_failed`     | The push service refused the message or could not be reached                 |
| `503`  | `no_provider`         | The app has no working key for this kind of device                           |
| `500`  | `internal`            | A bug or a storage failure                                                   |

Firebase's own explanation never appears in an answer; it goes to the activity log.

## Using it from a Moodle plugin

A plugin of your own can send straight to Let's Push instead of going through Moodle's messaging. You need:

1. **A Let's Push app and an access key** for your mobile app's Firebase project.
2. **The device tokens.** Moodle's mobile app registers them in Moodle's `user_devices` table, where
   `pushid` is the token and `platform` the device type, filtered by the app's ID. Read them from there.
3. **A call to `/v1/notifications`** for each device, for example with PHP's cURL:

```php
function myplugin_push(string $token, string $platform, string $title, string $body, array $data = []): ?string {
    $payload = json_encode([
        'to'    => ['token' => $token, 'platform' => $platform],
        'title' => $title,
        'body'  => $body,
        'data'  => array_map('strval', $data),   // data values must be text
    ]);
    $ch = curl_init(get_config('local_myplugin', 'pushserver') . '/v1/notifications');
    curl_setopt_array($ch, [
        CURLOPT_POST           => true,
        CURLOPT_POSTFIELDS     => $payload,
        CURLOPT_HTTPHEADER     => [
            'Authorization: Bearer ' . get_config('local_myplugin', 'pushkey'),
            'Content-Type: application/json',
        ],
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_TIMEOUT        => 5,
    ]);
    $answer = json_decode(curl_exec($ch), true);
    $status = curl_getinfo($ch, CURLINFO_RESPONSE_CODE);
    curl_close($ch);

    if ($status === 404 && ($answer['error']['code'] ?? '') === 'device_unregistered') {
        // The device is gone: delete this token from your records.
    }
    return $answer['id'] ?? null;     // keep the id if you want to ask how it ended
}
```

This is a sketch, not part of Let's Push; adapt it to the plugin's own settings and error handling. Send from
a task (cron or ad hoc), not while a page is loading: a delivery can take a second or more.

If you only need Moodle's usual notification (the site name as title, the message as body), you do not need
this: send a normal Moodle message and the stock `message_airnotifier` plugin forwards it through the
[compatible API](05-compatible-api.md).

## Compatibility promise

`/v1` is part of Let's Push's public contract ([Versioning conventions](09-versioning-conventions.md#2-what-the-version-promises)).
Adding an optional field, an endpoint or an error code is not a breaking change, so write your client to
ignore fields it does not know and to treat an unknown `code` like a general failure. Removing or changing
the meaning of something above is breaking and means `/v2`.

## What it does not do yet

- Sending to more than one device in a call, to a topic, or to everyone.
- Scheduling a notification for later.
- Action buttons, notification channels and badges.
- Listing devices or notifications.

These can be added to `/v1` without breaking it.

---

© 2026 Silify
