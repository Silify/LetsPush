# Changelog

## [Unreleased]

## [1.1.0] - 2026-10-25

This release adds Let's Push's own API, a new dashboard, and a more reliable delivery path. It runs two database migrations at startup that cannot be undone: **back up the data file and `LETSPUSH_SECRET` before upgrading.**

### Added
- Added the `/v1` API for new integrations: register and remove devices, send a notification with a title, a body, an image and data, and ask how a delivery ended. It uses a Bearer access key and one JSON error shape. See `docs/11-native-api.md`.
- Added priority, time to live and sound options, and image support, for Android, iOS and the web.
- Added a delivery status for notifications sent through `/v1` (`pending`, `sent`, `failed`, `unregistered`).
- Added a dashboard: an overview, the apps, activity across all apps, and for each app tabs for access keys, devices, activity, the Firebase key, connection details and settings. It has light, dark and system themes and works on a phone.
- Added a list of an app's devices to the dashboard, with the option to remove one.
- Added `X-Request-Id` to every answer and to the server's log, so a request can be followed from a site to the log.
- Added `LETSPUSH_LOG_FORMAT` (`text` or `json`) for the server's own log.
- Added the documentation in `docs/`, and the version shown by `letspush -version` and in the start-up log line.

### Changed
- Changed delivery to wait up to 1.5 seconds for Firebase and answer with the real result. A slower delivery now finishes on its own, so Moodle's two-second limit no longer loses a notification.
- Changed the activity log to include the reason Google gave when Firebase refuses a send, for example `PERMISSION_DENIED`. The reason is not sent back to the calling site.
- Changed Firebase sends to retry up to three times when Google is busy or unreachable, or when its access token was refused.
- Changed an access key's "last used" time to be written at most once a minute.
- Changed how an app's Firebase key is stored, to prepare for more push services (database migration; back up before upgrading).
- Changed the name to "Let's Push" and the logo to a paper plane.

### Fixed
- Fixed an issue where a push could be lost when Moodle gave up waiting before Firebase answered.
- Fixed an issue where the first send after Google's access token expired could fail.

## [1.0.0] - 2026-10-25

### Added
- Initial release: the Moodle-compatible push API, delivery through Firebase Cloud Messaging, and the admin web interface for apps, access keys and the activity log.

[Unreleased]: https://github.com/Silify/LetsPush/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/Silify/LetsPush/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/Silify/LetsPush/releases/tag/v1.0.0
