# Versioning Conventions

Let's Push follows [Semantic Versioning 2.0.0](https://semver.org/). This document says what a version
number promises for this project and where the number comes from.

## 1. Version Format

```text
MAJOR.MINOR.PATCH
```

- **MAJOR**: incompatible or breaking changes. `1.4.2 → 2.0.0`
- **MINOR**: new backward-compatible functionality. `1.4.2 → 1.5.0`
- **PATCH**: backward-compatible fixes. `1.4.2 → 1.4.3`

Components increase numerically, without leading zeroes. Let's Push has no separate build number: one
version names one source tree, and a Docker image or binary built from it carries that version.

## 2. What the version promises

Let's Push is run by people who cannot always update the things around it, so the promises are about
what those things see. These are its **public contract**:

1. **The compatible API** ([Compatible API](05-compatible-api.md)): routes, headers, request shapes, status
   codes and error texts that Moodle sites rely on.
2. **The native API** ([Native API](11-native-api.md)): its paths, bodies, error codes and statuses. It is
   versioned in its path (`/v1`); a breaking change is a new path (`/v2`) *and* a MAJOR release. Adding an
   optional field, an endpoint or an error code is not breaking.
3. **The settings**: environment variable names, their meaning and their defaults.
4. **The data**: the database file written by one version can be read by the next one (see section 4).

What is **not** part of the contract: the admin JSON API under `/admin/api/` (it belongs to the bundled
web interface, which ships in the same binary), the layout of the Go packages, log wording, and the
activity-log text.

## 3. Version `0.x` and `1.0.0`

Versions below `1.0.0` mean initial development: the contract may still change, and a MINOR release may
contain breaking changes (say so in the changelog). Release `1.0.0` when:

- the compatible API has been used by real Moodle sites without needing changes,
- the settings and the database layout are settled, and
- the project is ready to keep its promises.

After `1.0.0`, breaking changes require MAJOR, new functionality MINOR, fixes PATCH.

## 4. Breaking changes, migrations and deprecation

A change is **breaking** when an existing site, operator or deployment can no longer use the new version
without modification. Examples:

| Change                                                          | Version |
|-----------------------------------------------------------------|---------|
| Changing a status code or error text the compatible API sends    | MAJOR   |
| Removing or renaming an environment variable                     | MAJOR   |
| Raising a requirement an operator must act on (a new required setting) | MAJOR   |
| A new optional setting, route, provider or log field            | MINOR   |
| A bug fix, a retry rule, a faster path                           | PATCH   |
| Deprecating something that still works                           | MINOR   |
| Removing something that was deprecated                           | MAJOR   |

**Database migrations run automatically at startup and cannot be undone.** A release that adds a migration
is a normal MINOR or PATCH as long as the new version can read everything the old one wrote. Going back
to an older release means restoring a backup. Say in the changelog when a release migrates the database,
so operators back up first ([Operations](06-operations.md#4-upgrading)). A migration that loses or
reshapes data an operator can see is breaking.

## 5. Pre-release Versions

Format: `MAJOR.MINOR.PATCH-IDENTIFIER`

```text
1.5.0-alpha.1
1.5.0-beta.1
1.5.0-rc.1
1.5.0
```

A pre-release has lower precedence than its stable version (`1.5.0-rc.1 < 1.5.0`). Use one when someone
else will run an unstable build, for example a Moodle administrator testing a release candidate.

## 6. Source of Truth and Git Tags

**The Git tag is the source of truth:** `vMAJOR.MINOR.PATCH` (for example `v0.2.0`), annotated and
pointing at the exact release commit. Once pushed, a tag and the source behind it are never rewritten;
a fix is a new release (`v0.2.0 → v0.2.1`). See [Git conventions](07-git-conventions.md#5-versioning-and-tags).

The running program reports its version through `main.version`, which defaults to `dev` and is set at
build time:

```bash
go build -ldflags "-X main.version=0.2.0" ./cmd/letspush
letspush -version     # letspush 0.2.0
```

`make build` sets it from the latest Git tag (`git describe`, without the `v`), or from `make build VERSION=…`.
The Docker image takes it from the `VERSION` build argument, which `docker-compose.yml` reads from `.env`
(`VERSION=1.1.0`); without it the image reports `dev`.

`web/package.json` has its own `version` (`0.1.0`). It is not the release version and is not kept in step;
leave it alone.

## 7. Changelog, Dependencies and Environments

Every released version has a `CHANGELOG.md` entry (see
[Changelog conventions](10-changelog-conventions.md)). Versioning chooses the number; the changelog says what
changed.

The version depends on the **compatibility impact** of a change, not on what changed inside. Updating
Go, the SQLite driver or the web tooling is a PATCH unless an operator must act on it.

Do not put environment names in a version. Prefer `Version: 0.2.0, Environment: staging` over
`0.2.0-staging`.

## 8. Recommended Release Flow

```text
Changes merged to develop → release/X.Y.Z → test on a real Moodle and device → update CHANGELOG
→ merge to main → tag vX.Y.Z → build image from the tag → publish
```

Example: `0.1.3` → a new provider → `0.2.0` → tag `v0.2.0` → image built from `v0.2.0`.

If images are published to a registry, tag each with the full version, and (after `1.0.0`) the minor
version; never move a full-version tag. `latest` MAY follow the newest stable release.

## 9. Versioning Rules Summary

| Change                                   | Version                  |
|------------------------------------------|--------------------------|
| Breaking change to the public contract   | `MAJOR` (before 1.0: `MINOR`, flagged) |
| New backward-compatible functionality    | `MINOR`                  |
| Backward-compatible bug or security fix  | `PATCH`                  |
| Deprecating functionality                | `MINOR`                  |
| Removing deprecated functionality        | `MAJOR`                  |
| Unstable build for others to try         | `-alpha`, `-beta`, `-rc` |

## 10. Best Practices

- Use Semantic Versioning for release versions, and the Git tag as the one source of truth.
- Never reuse a released version for different source.
- Mark a release that migrates the database, and tell operators to back up.
- Treat the compatible API's behavior as the contract, and a test as its proof.
- Do not encode environment names into versions.
- Make the running program able to report its version.

## References

- [Semantic Versioning 2.0.0](https://semver.org/)
- [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)

---

© 2026 Silify
