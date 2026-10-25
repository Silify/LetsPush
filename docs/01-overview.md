# Overview

## What LetsPush is

LetsPush is a small **push notification gateway**. Websites and apps send it notifications over a
simple HTTP API, and it delivers them to phones and browsers through Firebase Cloud Messaging (FCM).

It speaks the API of [AirNotifier](https://github.com/dcai/airnotifier), the gateway that
**Moodle's mobile notifications** are built for. A Moodle site can therefore send through LetsPush
with no plugin and no change to Moodle: its administrator fills in a few settings. See
[Compatible API](05-compatible-api.md).

AirNotifier is the model for what LetsPush does, not for how. LetsPush is a separate program, written
in Go, with a cleaner structure, tighter security defaults and room to grow: more kinds of sites and more
push services. It also has an API of its own, for new integrations that want to send a title, a body, an
image and data, and to ask how a delivery ended. See [Native API](11-native-api.md).

## Who it is for

- **Moodle administrators** who need a push gateway for their mobile app and want something small to
  run and easy to reason about.
- **Operators** who host a gateway for several sites or apps and want one place to manage them.
- **Developers** who add a push service, a database or an API to LetsPush.

## What it does

- **Receives pushes** over the AirNotifier-compatible HTTP API or its own `/v1` API, and **delivers** them to
  devices through Firebase, to Android, iOS and the web.
- **Several apps in one server.** Each app has its own Firebase key, its own access keys and its own
  devices.
- **Access keys per site**, with permissions (register devices, remove devices, send). A key reaches
  only the devices it registered itself.
- **A web dashboard**, light or dark, to create apps, upload Firebase keys, create and revoke access keys,
  see each app's devices and read its activity log.
- **Reliable delivery.** A busy or unreachable Firebase is retried; a slow one never loses a
  notification; a dead device is removed and reported the way Moodle expects.
- **Traceable.** Every request has an ID that appears in the server log and in the activity log. When
  Firebase refuses a push, the log says why.

## What it is not

- Not a message queue or a broadcast system. It sends one notification to one device per request;
  topics, broadcast and scheduling are not built.
- Not a Moodle plugin. It sits beside Moodle and is reached over HTTP.
- Not a multi-instance service. Storage is one SQLite file, so one server runs at a time.
- Not a user or device directory. It knows device tokens and which key registered them, not who the
  people are.

## Platforms

LetsPush runs anywhere Go or Docker does. What it can deliver to depends on Firebase:

| Device  | Status                                                                |
|---------|-----------------------------------------------------------------------|
| Android | Supported, through Firebase                                           |
| Web     | Supported, through Firebase                                           |
| iOS     | Supported through Firebase; encrypted (data-only) notifications are unverified |

APNs directly, and other push services, are not here yet. See [Architecture](03-architecture.md#10-deliberate-non-goals).

## Technology

| Area             | Choice                                                       |
|------------------|--------------------------------------------------------------|
| Server           | Go, standard library `net/http`                              |
| Storage          | SQLite (`modernc.org/sqlite`, no cgo), one file              |
| Push service     | Firebase Cloud Messaging, HTTP v1 API                        |
| Web interface    | Svelte 5, Vite, Tailwind CSS 4, shadcn-svelte components     |
| Packaging        | One static binary with the interface embedded; a Docker image |
| Logging          | `log/slog`, as text or JSON                                  |

The Go code has one dependency: the SQLite driver. Everything else is the standard library. The exact
versions are in `go.mod` and `web/package.json`.

## Guiding ideas

These drive most decisions; the reasoning is in [Architecture](03-architecture.md).

- **Compatible first.** What Moodle sends and expects is a contract. It is pinned by tests.
- **A small core.** One gateway package does the work. APIs and push services are plugged into it.
- **Add by adding.** A new API or push service is a new package, not edits across the code.
- **Secrets stay secret.** Firebase keys are sealed, access keys are hashed, logs hold no message text
  and no whole device tokens.
- **Simple and boring** over clever. One binary, one file of data, no other service to run.

## Status

LetsPush is early. The Moodle-compatible API and the web interface work and are tested. What is not
done is listed as non-goals in [Architecture](03-architecture.md#10-deliberate-non-goals).

## Documentation map

| Document                                               | Read it to…                                         |
|--------------------------------------------------------|-----------------------------------------------------|
| [Getting started](02-getting-started.md)               | Set up, run, test and connect Moodle                |
| [Architecture](03-architecture.md)                     | Understand the design and its reasons               |
| [Project structure](04-project-structure.md)           | Know where code goes and which imports are allowed  |
| [Compatible API](05-compatible-api.md)                 | See exactly what the Moodle-compatible API accepts  |
| [Native API](11-native-api.md)                         | Send notifications from your own plugin or site     |
| [Operations](06-operations.md)                         | Deploy, back up, upgrade and troubleshoot a server  |
| [Git conventions](07-git-conventions.md)               | Branch, commit and review the way the team does     |
| [Naming conventions](08-naming-conventions.md)         | Name files, types and members consistently          |
| [Versioning conventions](09-versioning-conventions.md) | Version and release LetsPush                        |
| [Changelog conventions](10-changelog-conventions.md)   | Record notable changes for each release             |

---

© 2026 Silify
