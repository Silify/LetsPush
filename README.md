# LetsPush

A small push notification gateway. Sites and apps send it notifications over a simple HTTP API, and it
delivers them to phones through Firebase Cloud Messaging.

It speaks the push API that Moodle's mobile notifications use, so a Moodle site can point at it with no
plugin and no changes. A native API for other platforms is planned.

**Status:** early development.

## Plan for the first version

- Moodle-compatible endpoints: connectivity check, access keys, device tokens, and sending.
- Firebase Cloud Messaging only, through the HTTP v1 API with a service-account key.
- A tiny admin web interface: apps, access keys, devices and recent pushes.
- One Go binary with the interface embedded, SQLite for storage, a Docker image.

## License

Apache License 2.0. See [LICENSE](LICENSE).
