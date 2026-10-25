# Naming Conventions

How we name things in LetsPush. It has two parts: the **Go rules** we follow as they are, and a **small
set of project rules** layered on top. The rules are written as patterns, not as a list of today's
types, so they stay valid when files move. When in doubt, open a neighbouring file and copy its shape.

`gofmt` and `go vet` enforce part of this. Run `gofmt -l .` and `make test` to check.

## 1. Go rules

These come from the official guides. Read them once, they are short:

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Go Style Guide (Google)](https://google.github.io/styleguide/go/)

The part that matters day to day:

| What                                          | Style                             | Example                                  |
|-----------------------------------------------|-----------------------------------|------------------------------------------|
| Exported types, functions, methods, fields    | `UpperCamelCase`                  | `Service`, `Register`, `AppByName`       |
| Unexported ones, variables, parameters        | `lowerCamelCase`                  | `deviceOf`, `sendCtx`, `maxBody`         |
| Constants                                     | `UpperCamelCase` or `lowerCamelCase`, never `SCREAMING_CAPS` | `PermAll`, `defaultWait` |
| Packages                                      | short, lowercase, one word, no underscores | `gateway`, `store`, `reqid`     |
| Files                                         | `lowercase`, with `_` for two words | `handler.go`, `payload.go`         |
| Tests                                         | `<file>_test.go`                  | `gateway_test.go`                        |
| Receivers                                     | one or two letters, the same in every method | `s *Service`, `h *Handler`    |

Rules that people often get wrong:

- **Acronyms keep one case:** `ID`, `URL`, `HTTP`, `FCM`, `SQL`, `JSON`: `AppID`, `ProjectID`, `siteURL`,
  never `AppId` or `Url`. (The exception is a name that is already in a fixed external format.)
- **Package names are not repeated in the names inside them.** `gateway.Service`, not
  `gateway.GatewayService`; `store.App`, not `store.StoreApp`. Callers already read the package name.
- **Don't stutter and don't abbreviate** unless the abbreviation is more common than the word
  (`id`, `url`, `ctx`, `err`, `req`, `resp`, `db`, `fcm` are fine).
- **Interfaces are small and named for the role, with no `I` prefix and no `Impl` suffix:** `Provider`,
  `Store`, `Opener`. A one-method interface is often `<Verb>er`. The concrete type has an ordinary name
  (`*store.Store`, `*fcm.Provider`).
- **Booleans read as a statement:** `created`, `allow`, `revoked`; fields and functions like
  `AllowKeyRequests`, `HasCredentials`, `Can(permission)`.
- **Functions that do something are verbs** (`Register`, `Remove`, `deliver`); getters are nouns, without
  `Get` (`app.Name`, not `app.GetName()`).
- **Constructors are `New` when a package has one main type** (`gateway.New`, `fcm.NewClient`), and
  `New<Type>` when there are several.
- **Return `(value, error)`;** the error is last. A function that reports "not found" returns the
  package's sentinel error, not `nil, nil`.

## 2. Project rules

### Packages

- One idea per package, named after it (`gateway`, `store`, `secret`). Never `util`, `common`, `helpers`
  or `types`.
- Adapters are named for what they adapt (`compat`, `admin`, `webui`). A push service is named for the
  service (`fcm`; later `apns`).
- Everything is under `internal/`. See [Project structure](04-project-structure.md).

### Errors

- Sentinel errors are `Err` plus the condition, in the package that owns the idea: `gateway.ErrUnregistered`,
  `store.ErrNotFound`. Compare with `errors.Is`.
- An error type, when it carries data, is named for what it is: `fcm.Error` (a refusal from Google).
  Check with `errors.As`.
- Error strings are lowercase, have no final period and no `error:` prefix, and say what failed:
  `sending to ab12… failed: …`. The compatible API's texts are the exception: they are fixed by
  AirNotifier and Moodle (`Unregistered token`, `Invalid access key`). See
  [Compatible API](05-compatible-api.md).
- The edge translates errors. The gateway never returns an HTTP status or a Moodle text.

### Tests

- Test functions are **sentences** that say the behavior, in UpperCamelCase:
  `TestADeadDeviceFoundAfterSendAnsweredIsStillRemoved`, `TestARefusedTokenIsReplacedAndTheSendRetried`.
  A failing test should read as a broken promise.
- Group variations in one test with `t.Run("…", …)` and a plain sentence, or a table of cases named by a
  map key.
- Fakes are named for what they pretend to be: `fakeProvider`, `fakeSender`, `google` (a fake Google).
- Test helpers call `t.Helper()` and take `*testing.T` first.

### Storage

- Tables are plural `snake_case` (`apps`, `access_keys`, `app_providers`); columns are `snake_case`;
  timestamps are integer Unix seconds named `<event>_at` (`created_at`, `revoked_at`); foreign keys are
  `<table singular>_id` (`app_id`, `key_id`).
- Migration files are `NNN_what_it_does.sql`, numbered in order (`002_app_providers.sql`). A released
  migration is never renamed or edited.
- Provider kinds are short lowercase words stored in `app_providers.kind` (`fcm`), and declared as a
  constant in one place (`store.ProviderFCM`).

### Settings and logs

- Environment variables are `LETSPUSH_` plus `UPPER_SNAKE_CASE`: `LETSPUSH_LOG_FORMAT`.
- `slog` messages are short lowercase phrases (`"delivery failed"`); data goes in key-value pairs with
  `snake_case` keys (`request_id`, `duration_ms`). Pass the request's context so `request_id` is added.
- Activity-log actions are lowercase nouns: `app`, `accesskey`, `token`, `push`, `error`.

### APIs and JSON

- The **compatible API** uses the names AirNotifier and Moodle use (`accesskey`, `device`, `extra`,
  `X-AN-APP-NAME`). Do not "fix" them.
- The **native API** (`/v1/…`) uses `lowerCamelCase` JSON (`requestId`, `createdAt`), plural resource paths
  (`/v1/devices`, `/v1/notifications`) and `snake_case` error codes (`device_unregistered`). Notification
  IDs start with `ntf_`.
- The **admin API** (`/admin/api/…`) uses `lowerCamelCase` JSON (`projectId`, `hasCredentials`) and plural
  resource paths (`/apps/{id}/keys`).
- Go structs map to JSON with tags; the Go field keeps the Go name.

### Web interface

- Svelte components and pages are `PascalCase.svelte` (`AppDetail.svelte`); one component per file.
  Screens live in `pages/`, app-specific pieces in `components/`.
- Plain modules are `camelCase.js` (`api.js`, `format.js`); a module that uses Svelte runes ends in
  `.svelte.js` (`router.svelte.js`).
- Folders under `lib/components/ui/` are named by shadcn-svelte (`alert`, `dialog`…); they are generated,
  so rename a component by regenerating it, not by hand.
- Colors and radii come from the theme variables in `app.css` (`--primary`, `--radius`), through Tailwind
  classes (`bg-primary`), never from a hard-coded color.

### Documents

- Files in `docs/` are `NN-kebab-case.md`, numbered in reading order. The numbers are part of the name;
  renumbering needs the links updated in the same change.

## 3. Renaming and adding names

- **Rename with the tools** (`gopls` rename, or the editor's rename symbol) so every use changes. Keep the
  file name and the main type in step.
- **After renaming anything a client reads** (a JSON field of the admin API, a column, a setting), update
  the web interface, the migration and the docs in the same change. Names in the compatible API do not
  change.
- **Follow the existing name** in the neighbouring package before inventing a new pattern. If a rule here
  feels wrong, change this document in the same pull request as the code, so it stays accurate.

---

© 2026 Silify
