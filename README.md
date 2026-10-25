<p align="center"><img src="assets/logo.svg" alt="Let's Push" width="192" height="192"></p>

# Let's Push

A small push notification gateway. Your sites and apps send it notifications over a simple HTTP API, and
it delivers them to Android, iOS and web devices through Firebase Cloud Messaging.

It speaks the API of [AirNotifier](https://github.com/dcai/airnotifier), so a **Moodle** site can send
through it with no plugin and no change to Moodle.

**Status:** early. The Moodle-compatible API and the admin interface work and are tested.

- One Go binary with the web interface inside. SQLite for storage, no other service to run.
- Firebase keys are stored encrypted; access keys are stored only as hashes.
- Each site can only reach the devices it registered itself.

## Quick start

You need Docker and a Firebase project with a service-account key.

```bash
cp .env.example .env     # set LETSPUSH_SECRET and the first admin's email and password
docker compose up -d --build
```

Open <http://127.0.0.1:8080>, sign in, create an app with your Firebase key and an access key, and enter
the values in Moodle. Put a reverse proxy with HTTPS in front before real use.

## Documentation

Everything else is in [`docs/`](docs/01-overview.md): [getting started](docs/02-getting-started.md),
[architecture](docs/03-architecture.md), the [compatible API](docs/05-compatible-api.md),
[operations](docs/06-operations.md), and the project's conventions.

## Credits

Let's Push is a separate program with its own design and storage. It follows the HTTP API of AirNotifier
(by Dongsheng Cai and contributors, BSD licensed) and works with Moodle's `message_airnotifier` plugin. It is
not affiliated with or endorsed by either project; the names belong to their owners.

## License

Apache License 2.0. See [LICENSE](LICENSE).
