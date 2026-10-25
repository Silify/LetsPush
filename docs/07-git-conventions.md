# Git Conventions

This document is the single source of truth for Git rules and guidelines: branching, commits, pull
requests, merging, versioning, releases, and repository protection.

**Goals**

- Keep the production branch stable.
- Let multiple developers work independently.
- Make development and release processes predictable.
- Make releases easy to track and reproduce.
- Support emergency production fixes.
- Keep the Git history readable.
- Enable automated CI/CD, changelogs, and semantic versioning.

**Keywords:** MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are used in the sense
of [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

---

## Table of Contents

1. [Core Rules](#1-core-rules)
2. [Branching Model](#2-branching-model)
3. [Branch Naming](#3-branch-naming)
4. [Commit Messages](#4-commit-messages)
5. [Versioning and Tags](#5-versioning-and-tags)
6. [Workflows](#6-workflows)
7. [Pull Requests](#7-pull-requests)
8. [Merge Strategy](#8-merge-strategy)
9. [Rebase and History Rules](#9-rebase-and-history-rules)
10. [Repository Protection](#10-repository-protection)
11. [Continuous Integration](#11-continuous-integration)
12. [Critical and Sensitive Changes](#12-critical-and-sensitive-changes)
13. [Issue References](#13-issue-references)
14. [Branch Cleanup](#14-branch-cleanup)
15. [Quick Reference](#15-quick-reference)

---

## 1. Core Rules

1. `main` contains production-ready code.
2. `develop` contains integrated work for the upcoming release.
3. Never commit directly to `main`.
4. Avoid committing directly to `develop`; use Pull Requests.
5. Create features and normal fixes from `develop`.
6. Create releases from `develop`.
7. Create production hotfixes from `main`.
8. Merge release branches into both `main` and `develop`.
9. Merge hotfixes into both `main` and `develop` (and into any active release branch).
10. Tag every production release.
11. Write commit messages following [Conventional Commits](#4-commit-messages).
12. Use Pull Requests for all reviewed integration.
13. Require passing CI before important merges.
14. Delete temporary branches after merging.
15. Never rewrite published history on shared or protected branches.
16. Prefer `--force-with-lease` over `--force` on personal branches.
17. Keep feature branches short-lived.
18. Keep release branches focused on stabilization.
19. Do not introduce unrelated features into an active release branch.
20. Production source code must always be traceable to a Git tag.

---

## 2. Branching Model

### Branch overview

| Branch       | Purpose                                     | Lifetime  |
|--------------|---------------------------------------------|-----------|
| `main`       | Production-ready source code                | Permanent |
| `develop`    | Integration branch for upcoming development | Permanent |
| `feature/*`  | New features and improvements               | Temporary |
| `fix/*`      | Non-production bug fixes                    | Temporary |
| `refactor/*` | Significant internal restructuring          | Temporary |
| `chore/*`    | Maintenance or infrastructure work          | Temporary |
| `release/*`  | Release preparation and stabilization       | Temporary |
| `hotfix/*`   | Urgent production fixes                     | Temporary |

Typical flow:

```text
feature/*  ──►  develop  ──►  release/*  ──►  main  ──►  tag
```

Production hotfix:

```text
main ──► hotfix/*  ──┬──►  main  ──►  tag
                     └──►  develop
```

### Permanent branches

#### `main`

Represents the current production-ready state. Code in `main` MUST always be stable, reviewed,
tested, and releasable.

- Developers MUST NOT commit directly to `main`.
- Changes reach `main` only through `release/*` and `hotfix/*` branches.
- Every production release MUST be marked with a Git tag (e.g. `v1.4.0`).

#### `develop`

The integration branch for the next release. Most work branches start from `develop` and merge back
into it.

- `develop` may contain features not yet released, but SHOULD remain reasonably stable and pass
  required CI checks.
- Developers SHOULD NOT commit directly to `develop`; changes arrive through Pull Requests.

### Temporary branch types

| Type         | Use for                                                           | Created from | Merged into        |
|--------------|-------------------------------------------------------------------|--------------|--------------------|
| `feature/*`  | New user-facing features or significant functionality             | `develop`    | `develop`          |
| `fix/*`      | Bugs found during normal development (not production emergencies) | `develop`    | `develop`          |
| `refactor/*` | Internal restructuring with no new feature or bug fix             | `develop`    | `develop`          |
| `chore/*`    | Maintenance: dependencies, tooling, cleanup, lint rules           | `develop`    | `develop`          |
| `release/*`  | Preparing and stabilizing a specific version                      | `develop`    | `main` + `develop` |
| `hotfix/*`   | Urgent production problems                                        | `main`       | `main` + `develop` |

> Do NOT use `hotfix/*` for normal bugs. It is reserved for urgent issues affecting production.

### Release branches

Release branches prepare a specific version for production.

**Allowed in a release branch**

- Bug fixes
- Version and build number updates
- Release configuration and release notes
- Text and message corrections
- Minor UI corrections
- Test fixes related to the release
- Documentation related to the release

**Not allowed (normally)**

- Large new features
- Major refactoring
- Unrelated dependency migrations
- Architectural changes

New features wait for the next development cycle. Development on `develop` may continue while a
release is being stabilized; those changes are NOT automatically part of the release branch.

After approval, the release branch is merged into **both** `main` and `develop` so stabilization
fixes are never lost.

### Hotfix branches

Examples of hotfix-worthy problems:

- The server crashes or cannot start
- Notifications are not delivered, or go to the wrong devices
- The compatible API stops working for Moodle sites
- Broken sign-in, or an access key that works when it should not
- Data loss or a migration that corrupts data
- Severe security issues
- Firebase keys that can no longer be opened

Hotfixes start from `main`, NOT `develop`. After verification they MUST be merged into `main` and
`develop`. If an active release branch exists, the hotfix SHOULD also be incorporated into it, so
the bug does not reappear in the next release:

```text
hotfix/1.4.1-token-leak
        ├────► main
        ├────► develop
        └────► release/1.5.0   (if one is active)
```

---

## 3. Branch Naming

Branch names SHOULD:

- Use lowercase letters
- Use hyphens between words
- Be short but descriptive
- Describe the purpose of the branch
- Optionally contain an issue/task ID

Formats:

```text
<type>/<description>
<type>/<issue-id>-<description>
```

Release and hotfix formats:

```text
release/<version>
hotfix/<description>
hotfix/<version>-<description>
```

**Good**

```text
feature/apns-provider
feature/142-native-api
fix/unregistered-token-cleanup
refactor/gateway-errors
chore/update-dependencies
release/0.2.0
hotfix/0.2.1-sign-in-loop
```

**Avoid**

```text
new-feature
test
dev2
fix-stuff
my-branch
feature123
final
final-new-2
```

Branch names should communicate intent without requiring telepathy.

---

## 4. Commit Messages

All commits MUST follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
specification.

### Format

```text
<type>[optional scope][!]: <description>

[optional body]

[optional footer(s)]
```

| Part          | Required | Description                                                     |
|---------------|----------|-----------------------------------------------------------------|
| `type`        | Yes      | Category of the change (see below)                              |
| `scope`       | No       | Part of the codebase affected, e.g. `gateway`, `compat`, `web`  |
| `!`           | No       | Marks a **breaking change**                                     |
| `description` | Yes      | Short summary of the change                                     |
| `body`        | No       | Detailed explanation: *why* the change was made                 |
| `footer`      | No       | References (`Refs: #123`), reviewers, or `BREAKING CHANGE: ...` |

### Types

| Type       | Meaning                                  | Version impact |
|------------|------------------------------------------|----------------|
| `feat`     | New feature                              | MINOR          |
| `fix`      | Bug fix                                  | PATCH          |
| `docs`     | Documentation only                       | none           |
| `style`    | Formatting / styling, no behavior change | none           |
| `refactor` | Code restructure, no new feature or fix  | none           |
| `perf`     | Performance improvement                  | PATCH          |
| `test`     | Add or update tests                      | none           |
| `build`    | Build system or dependency changes       | none           |
| `ci`       | CI/CD configuration changes              | none           |
| `chore`    | Maintenance, no production code impact   | none           |
| `revert`   | Revert a previous commit                 | varies         |

Any commit with a breaking change triggers a **MAJOR** bump regardless of type.

### Writing rules

There is no commit hook yet, so these rules are checked in review. A hook is a good first addition
when CI is added ([section 11](#11-continuous-integration)).

- Use the **imperative mood**: "add retry for busy sends", not "added" or "adds".
- Keep the description short (aim for **72 characters or fewer** on the first line).
- Do not end the description with a period.
- Start the description in lowercase (unless it begins with a proper noun or acronym).
- One logical change per commit.
- The body is optional. Add one only when the **why** is not clear from the summary, in one to three
  short lines. Small or self-explanatory commits (a rename, a string fix) need no body.
- Types are case-insensitive, but lowercase is the convention. `BREAKING CHANGE` MUST be uppercase.

### Breaking changes

Indicate with `!`:

```text
feat(compat)!: change the unregistered-token answer
```

or with a footer:

```text
refactor(gateway): split delivery from device lookup

BREAKING CHANGE: providers must now implement the new gateway.Provider interface
```

### Body and footer examples

```text
fix(fcm): retry when Google refuses the access token

An expired token made the first send after an hour fail. The client now
fetches a fresh token and sends again.

Refs: #123
Reviewed-by: Developer Name
```

### Examples by type

```text
feat(gateway): wait up to 1.5 seconds before answering pending
feat(web): show the reason Firebase gave in the activity log

fix(compat): accept a push with no extra object
fix(store): write a key's last use at most once a minute

docs(readme): explain where LetsPush differs from AirNotifier
docs: add the compatible API reference

style: format the Go sources

refactor(gateway): move device registration out of the compat handler
refactor(store): store provider keys in their own table

perf(fcm): parse a service-account key once

test(compat): pin the push the current Moodle plugin sends
test(gateway): a caller that hangs up does not cancel the delivery

build: use Go 1.25 in the Dockerfile
build(deps): update the SQLite driver

ci: run make test on every pull request

chore: ignore local build logs
chore(web): regenerate the shadcn components
```

### Reverts

```text
revert: feat(gateway): wait up to 1.5 seconds before answering pending

Refs: 676104e, a215868
```

### Scopes

Scopes are optional. Use the **package or area** the change is in, from this list:

| Scope     | For                                                       |
|-----------|-----------------------------------------------------------|
| `gateway` | The core: devices, permissions, delivery                  |
| `compat`  | The AirNotifier / Moodle-compatible API                   |
| `native`  | LetsPush's own `/v1` API                                  |
| `admin`   | The admin JSON API (sign-in, apps, keys, log)             |
| `fcm`     | The Firebase provider                                     |
| `store`   | The database and migrations                               |
| `secret`  | Sealing, hashing, keys                                    |
| `config`  | Settings                                                  |
| `server`  | Wiring, middleware, startup                               |
| `web`     | The web interface                                         |
| `readme`, `docs` | Documentation                                      |
| `deps`    | Dependency updates (with `build`)                         |

Keep scopes short, lowercase and based on areas, not on file names. Add a scope here when a new area
appears (a new provider package, say). A change across several areas needs no scope.

---

## 5. Versioning and Tags

### Semantic Versioning

Projects follow [Semantic Versioning](https://semver.org/): `MAJOR.MINOR.PATCH`

| Change                          | Version bump |
|---------------------------------|--------------|
| Breaking change                 | MAJOR        |
| New backward-compatible feature | MINOR        |
| Backward-compatible bug fix     | PATCH        |

```text
1.4.2 → 1.4.3   bug fix
1.4.3 → 1.5.0   new feature
1.5.0 → 2.0.0   breaking change
```

These rules align directly with the commit types in [section 4](#4-commit-messages).

Hotfixes normally increment PATCH (`1.4.0` → `1.4.1`, tag `v1.4.1`).

### Production tags

After a release is merged into `main`, create an **annotated** tag:

```text
v<major>.<minor>.<patch>
```

```bash
git checkout main
git pull
git tag -a v1.4.0 -m "Release v1.4.0"
git push origin v1.4.0
```

Production images and binaries SHOULD be built from a version tag rather than an arbitrary commit, so that
`v1.4.0` uniquely identifies the source used for that release.

### Release candidates (optional)

If QA needs multiple builds before release, candidates may be tagged:

```text
v1.4.0-rc.1
v1.4.0-rc.2
```

The final release is tagged `v1.4.0`. Use this only when it adds useful traceability; do not tag
ordinary development builds.

### Build metadata

LetsPush has no separate build number. The Git tag is the version (`v1.4.0`), and the Docker image or
binary built from it reports that version. See
[Versioning conventions](09-versioning-conventions.md).

---

## 6. Workflows

### 6.1 Feature (and fix / refactor / chore)

1. **Update `develop`**

   ```bash
   git checkout develop
   git pull origin develop
   ```

2. **Create the branch**

   ```bash
   git checkout -b feature/apns-provider
   ```

3. **Implement**, committing with Conventional Commits:

   ```text
   feat(apns): add the APNs provider
   feat(admin): accept an APNs key for an app
   test(apns): add provider tests
   ```

4. **Stay in sync** with `develop` (see [section 9](#9-rebase-and-history-rules)):

   ```bash
   git fetch origin
   git rebase origin/develop
   ```

5. **Push**

   ```bash
   git push -u origin feature/apns-provider
   ```

   After rebasing an already-pushed personal branch:

   ```bash
   git push --force-with-lease
   ```

6. **Open a Pull Request** targeting `develop`. It must pass review and CI before merging.

Never create a branch from an outdated local `develop`. Always pull first.

### 6.2 Release

1. **Create the release branch**

   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b release/1.4.0
   ```

2. **Stabilize**: bug fixes, release notes ([Changelog conventions](10-changelog-conventions.md)), QA
   on a real Moodle with a real device. Only release-critical changes enter the branch.

3. **Approve and merge** into `main` (merge commit; see [section 8](#8-merge-strategy)).

4. **Tag** the release on `main` (`v1.4.0`).

5. **Merge back** into `develop` so release fixes are not lost.

6. **Delete** the release branch.

```text
develop ─────────────────────────────► future development
    │
    └── release/1.4.0 ──► main ──► v1.4.0
                  └─────► develop
```

**Bugs found during release QA** are fixed on the release branch through the normal review process,
e.g. `fix(gateway): remove a dead device found after the answer`. They flow back to `develop` when the release
merges back.

### 6.3 Hotfix

1. **Create from `main`**

   ```bash
   git checkout main
   git pull origin main
   git checkout -b hotfix/1.4.1-sign-in-loop
   ```

2. **Fix and commit**

   ```text
   fix(admin): keep the session after a restart
   ```

3. **Review and test.**

4. **Merge into `main`**, then **tag** `v1.4.1`.

5. **Merge into `develop`** (and into any active `release/*` branch).

6. **Delete** the hotfix branch.

---

## 7. Pull Requests

All important changes use Pull Requests.

### Rules

A Pull Request SHOULD:

- Address one logical change
- Have a clear title
- Include a useful description (what, why, how to test)
- Reference the related issue or task
- Pass required CI checks
- Receive required code review
- Include appropriate tests (a change to the compatible API needs a test in `moodle_test.go`)
- Contain no unrelated changes

Avoid large Pull Requests. Smaller ones are easier to review, test, reason about, revert, and merge
safely. A Pull Request should not become a storage facility for everything a developer happened to
change that week.

### Title convention

PR titles SHOULD follow Conventional Commit syntax. This matters especially with squash merging,
where the PR title becomes the final commit message.

```text
feat(gateway): add the bounded wait
fix(compat): answer 404 for a dead device
refactor(store): move provider keys to their own table
chore: update dependencies
```

### Suggested description template

```markdown
## Summary

What does this change do, and why?

## Changes

- ...

## How to test

1. ...

## Related issues

Refs: #142

## Checklist

- [ ] Tests added/updated (`make test` passes)
- [ ] Documentation updated (README, `docs/`)
- [ ] CHANGELOG entry, if it matters to an operator or a site
- [ ] Migration added, not edited, if the schema changed
- [ ] No unrelated changes
```

---

## 8. Merge Strategy

| Branch type                                   | Strategy             |
|-----------------------------------------------|----------------------|
| `feature/*`, `fix/*`, `refactor/*`, `chore/*` | **Squash and Merge** |
| `release/*`, `hotfix/*`                       | **Merge commit**     |

**Why squash development branches:** a feature branch with many small commits becomes a single,
readable commit on `develop`:

```text
feat(auth): add biometric repository
feat(auth): add biometric service
fix(auth): handle biometric cancellation
test(auth): add biometric tests
        ▼ squash
feat(auth): add biometric authentication
```

**Why merge commits for release/hotfix:** preserving branch history makes it clear what was
released, when, which tag represents it, and which changes belong to it:

```text
Merge branch 'release/1.4.0'
Merge branch 'hotfix/1.4.1-token-refresh'
```

---

## 9. Rebase and History Rules

- Rebasing is **recommended for personal branches** to keep them up to date with `develop`. It
  avoids unnecessary merge commits.
- **Do not casually rebase shared branches.** Coordinate with others before rewriting their history.
- **Never rebase** `main`, `develop`, or `release/*` once they are shared.
- **Never rewrite published production history.**

Synchronize a personal branch:

```bash
git fetch origin
git rebase origin/develop
```

Resolve conflicts on your working branch, never directly on protected branches:

```bash
git add .
git rebase --continue
```

Push after rebasing:

```bash
git push --force-with-lease      # OK on personal branches
```

`--force-with-lease` protects against accidentally overwriting commits pushed by someone else. Do
NOT use plain `--force`. The following MUST be prohibited:

```bash
git push --force origin main
git push --force origin develop
```

---

## 10. Repository Protection

Configure branch protection on the hosting platform (GitHub, GitLab, Bitbucket, etc.).

### `main`

```text
Require pull request:        Yes
Require CI checks:           Yes
Require review:              Yes
Resolve all comments:        Yes
Allow direct push:           No
Allow force push:            No
Allow deletion:              No
```

### `develop`

```text
Require pull request:        Yes
Require CI checks:           Yes
Require review:              Yes (per team policy)
Allow direct push:           No
Allow force push:            No
Allow deletion:              No
```

Also recommended: protect `v*` tags from deletion and overwriting.

---

## 11. Continuous Integration

There is no CI yet. Until there is, run these before opening a pull request:

```bash
gofmt -l .         # prints nothing when the Go sources are formatted
make test          # go vet ./... and go test ./...
cd web && npm run build
```

When CI is added, it SHOULD run, at minimum, on every pull request:

- `gofmt` check
- `go vet` and `go test`
- Building the web interface
- Building the Docker image

Additional checks, as the project grows:

- Security scanning of dependencies
- A run of the compatible API tests against a real Moodle plugin payload set
- A migration check on a copy of a database from the previous release

Release branches SHOULD additionally build the Docker image from the tag.

---

## 12. Critical and Sensitive Changes

Some areas deserve extra review and testing before merge, for example:

- Access keys, permissions, and who may reach which device
- Admin sign-in, sessions and the throttle
- Sealing, hashing and anything in `internal/secret`
- Storing, opening or logging Firebase keys
- Database migrations (they cannot be undone)
- The compatible API's status codes and texts (real Moodle sites depend on them)
- Anything that writes to the log: it must never hold message text, keys or whole device tokens

Pull Requests touching these areas SHOULD receive additional reviewers and targeted testing.
Critical logic MUST NOT be merged solely because the tests are green. Automated tests are useful.
They have not yet developed judgment.

---

## 13. Issue References

Reference the corresponding issue or task wherever possible.

```text
Branch:         feature/142-native-api
Commit footer:  Refs: #142
Pull Request:   feat(native): add the /v1 API (#142)
```

The exact identifier format is the issue number of the project's GitHub repository.

---

## 14. Branch Cleanup

Delete temporary branches after they are merged:

```text
feature/*   fix/*   refactor/*   chore/*   release/*   hotfix/*
```

Permanent branches are `main` and `develop`. Keeping merged branches around adds noise without
value.

**Avoid long-lived feature branches.** They cause more merge conflicts, harder reviews, delayed
integration, and higher regression risk. Split large work into smaller deliverable changes. Protect
incomplete functionality with:

- A setting that is off by default
- Routes that are not mounted yet
- Code that nothing calls yet, kept small

---

## 15. Quick Reference

```bash
# Feature
git checkout develop && git pull origin develop
git checkout -b feature/feature-name

# Normal fix
git checkout develop && git pull origin develop
git checkout -b fix/bug-name

# Release
git checkout develop && git pull origin develop
git checkout -b release/0.2.0

# Hotfix
git checkout main && git pull origin main
git checkout -b hotfix/0.2.1-critical-fix

# Sync a personal branch
git fetch origin
git rebase origin/develop

# Push a rebased personal branch
git push --force-with-lease

# Tag a production release
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

---

## Notes on Adoption

- LetsPush has a single deployable and a small team. Before the first release, work goes to `develop`
  (commits or short branches), and `main` stays empty of unreleased work. Once there are users with
  data to protect, use the full model above. If it proves too heavy, the `develop` branch MAY be
  dropped in favor of short-lived branches off `main` (then `release/*` is optional and hotfixes
  branch from the latest tag). Make that an explicit change to this document.
- This document is a living guideline. Propose changes through a Pull Request like any other change.

---

© 2026 Silify
