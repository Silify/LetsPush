# Changelog Conventions

How Let's Push records what changed between releases. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Version numbers come from the
[versioning conventions](09-versioning-conventions.md), commits from the
[Git conventions](07-git-conventions.md).

The changelog is a **curated, human-readable record of notable changes per release**. It answers one
question: *what changed between this release and the previous one?* It does not replace Git history,
pull requests or release notes.

## 1. Where it lives

`CHANGELOG.md` in the repository root. Newest release first, with an `[Unreleased]` section on top.

`CHANGELOG.md` starts with the entry for `1.0.0` (`Initial release`). From then on, follow this document.

## 2. Structure

```markdown
# Changelog

## [Unreleased]

### Added
- Added ...

## [0.2.0] - 2026-11-05

### Added
- Added ...

### Fixed
- Fixed ...

[Unreleased]: https://github.com/Silify/LetsPush/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Silify/LetsPush/compare/v0.1.0...v0.2.0
```

- Version heading: `## [MAJOR.MINOR.PATCH] - YYYY-MM-DD`, using the **actual release date**.
- List only the stable version (`0.2.0`), not pre-releases (`0.2.0-rc.1`). Their changes appear under the
  final release.
- Include only categories that have entries.
- The link list at the bottom lets each version heading link to its comparison on GitHub.
- A release that was withdrawn is marked `## [0.2.1] - 2026-11-09 [YANKED]`.

## 3. Categories

| Category     | Use for                                      |
|--------------|----------------------------------------------|
| `Added`      | New features                                 |
| `Changed`    | Changes to existing behavior                 |
| `Deprecated` | Features that still work but will be removed |
| `Removed`    | Features that are gone                       |
| `Fixed`      | Bug fixes                                    |
| `Security`   | Security-related changes                     |

Commit types map to categories like this:

| Commit type                                            | Changelog                                            |
|--------------------------------------------------------|------------------------------------------------------|
| `feat`                                                 | `Added`                                              |
| `fix`                                                  | `Fixed` (or `Security`)                              |
| `perf`                                                 | `Changed`                                            |
| Breaking change (`!` or `BREAKING CHANGE`)             | `Changed` or `Removed`, marked **Breaking**          |
| `docs`, `style`, `refactor`, `test`, `build`, `ci`, `chore` | Usually omitted                                 |
| `revert`                                               | Depends on what was reverted                         |

## 4. What belongs in it

Include a change if it matters to someone **outside the code**:

- **Operators:** a new or changed setting, a database migration (say so, so they back up), a changed
  requirement (Go, Node, Docker), a change in how to deploy or upgrade.
- **Moodle administrators and sites:** anything the compatible API does differently, a new status or
  message, a fix to delivery they would notice.
- **Administrators of the web interface:** new screens, new options, visible fixes.
- **Developers:** a changed build or test command, a new package other code can plug into, a documented
  structural change.

Leave out anything only the implementation cares about: renames, formatting, minor refactors, tests,
dependency bumps with no visible effect, internal file moves. Tests that pin the compatible API are worth
a line only when they reveal a behavior change. Git history already has them.

## 5. Writing entries

- One entry per meaningful change, in plain language, starting with a past-tense verb.
- Say **what changed and why it matters**, not how it was built.
- Write for the reader, not the committer. Don't paste commit messages.

```markdown
### Added
- Added `LETSPUSH_LOG_FORMAT` to write the server's log as JSON.
- Added a request ID to every answer (`X-Request-Id`) and to the server's log.

### Changed
- Changed the activity log to include Google's reason when Firebase refuses a push.

### Fixed
- Fixed an issue where a push could be lost when the caller timed out.
```

Prefer `Fixed an issue where a push could be lost when the caller timed out.` over
`Moved delivery onto its own context in gateway.Send.`

## 6. Breaking changes

Call them out so nobody has to guess. Mark them **Breaking** and say what the reader must do:

```markdown
### Removed
- **Breaking:** Removed the `LETSPUSH_DB_PATH` setting. Use `LETSPUSH_DB` before upgrading.
```

A breaking change needs a MAJOR version under the versioning conventions (before 1.0.0, a MINOR one, and
said so here). Never hide one in a vague
entry like `Updated configuration handling.`

## 7. Security entries

Keep them short and safe to publish. Say that something was improved, not how it could have been
exploited, and never include credentials, attack details or private report contents.

```markdown
### Security
- Improved validation of the request ID sent by callers.
```

## 8. When to write it

- **With the change.** Add the entry under `[Unreleased]` in the same pull request that makes the
  change, if it is changelog-worthy. This avoids reconstructing history at release time.
- **At release.** On the release branch, rename `[Unreleased]` to the new version and date, add a fresh
  empty `[Unreleased]`, and update the link list. This happens before the release is tagged, as in the
  [release flow](09-versioning-conventions.md#8-recommended-release-flow).
- **Hotfixes** get an entry like any other release.

Released entries are stable: do not rewrite them to reflect later changes. Ship a new version instead.
Fix only genuine typos or errors.

Tools can draft entries from commits and pull requests, but a person must review the result. Automation
saves typing; it does not decide what is meaningful.

## 9. Changelog vs. release notes

The changelog is the lasting, per-version history. Release notes are a message for one release (store
listing text, highlights, known issues, migration steps). They can reuse changelog entries, but the
changelog is not automatically the release notes.

A release that **migrates the database** says so in its entry, for example
`Changed how an app's Firebase key is stored (database migration; back up before upgrading).`

## 10. Checklist

```text
[ ] Entries sit under the right category.
[ ] A migration is mentioned, with a reminder to back up.
[ ] Each entry is a meaningful, plain-language change, not a commit message.
[ ] Breaking changes are marked and say what to do.
[ ] Security entries are safe to publish.
[ ] Version and date are correct, and match the Git tag (vMAJOR.MINOR.PATCH).
[ ] [Unreleased] was reset and the comparison links updated.
```

## References

- [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)

---

© 2026 Silify
