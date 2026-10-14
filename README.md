# forgesync

A small Go CLI that syncs GitHub issues assigned to you into your Notion
workspace, linking each issue to the matching Project in your Project Manager
database.

One-way sync (GitHub → Notion), run on demand. No daemon, no webhooks.

---

## What it does

For every issue assigned to you on GitHub (updated in the last 7 days, pull
requests excluded), `forgesync` will:

1. Look up the issue's repo (`owner/name`) in your **Project Manager** database
   via the `repo` text field.
2. If a matching Project exists, upsert a row in the **Stories** database
   linked to that Project. If no Project matches, the issue is skipped.
3. Keep the Story's `Name`, `Labels`, `Last Worked At`, and `Finished Date` in
   sync with the GitHub issue, mirror the issue body into the Story's page
   content, and compute its `Status` from the issue state
   (see [Status mapping](#status-mapping)). The `URL`, `Issue` number, and
   `Project` relation are set once, when the Story is first created.

The sync key is the pair **(Project, issue number)**: `forgesync` looks up an
existing Story that is linked to the issue's Project *and* carries a matching
`Issue` number, updates it, and otherwise creates one. Running `forgesync sync`
repeatedly is safe and idempotent. A Story whose `Name`, `Status`, `Labels`,
and `Last Worked At` already match is reported as `Unchanged` and left
untouched.

### Status mapping

When a Story is **first created**, its `Status` is derived purely from the
GitHub issue:

| GitHub issue state | Linked PR? | Notion `Status` |
| ------------------ | ---------- | --------------- |
| Open               | no         | `Not started`   |
| Open               | yes        | `In PR`         |
| Closed             | any        | `Done`          |

Any other/unknown state falls back to `Not started`.

Linked-PR detection only runs on **open** issues, so a closed issue lands on
`Done` regardless of the PRs attached to it.

On later syncs the **previous Notion status is folded in**, so most manual
edits survive instead of being overwritten:

- An **open** issue with no status yet (or still `Not started`) lets GitHub
  decide: a linked PR makes it `In PR`, otherwise it stays `Not started`.
- An **open** issue you set to `In progress` is promoted to `In PR` as soon as
  a PR is linked. This is the one case where GitHub overrides a status you set
  by hand.
- Any other manual status on an **open** issue is preserved — `Done`,
  `Cancelled`, `In PR`, or an `In progress` with no PR linked yet.
- A **closed** issue you previously set to `Cancelled` stays `Cancelled`.
- Any other closed issue becomes `Done`.

`forgesync` never writes `In progress` itself. You set it in Notion to mark
that you have started work, and the sync promotes it to `In PR` when a PR
appears — so the option still has to exist in your database.

Linked-PR detection asks GitHub's GraphQL API for the issue's
`closedByPullRequestsReferences` — the PRs linked to it with a closing keyword
(`Fixes #12`) or through the Development sidebar. The issue counts as having a
linked PR when at least one of them is **not a draft**. Closed-but-unmerged PRs
are included in that list, so they count too.

### What it does NOT touch

- The `URL`, `Issue`, and `Project` relation after the Story is created
- Any property not listed above (e.g. `Prioridad` — you own this in Notion)
- The `Created time` property (auto-managed by Notion)
- `Finished Date` while the issue is still open (when the issue is closed, it is
  set from `closed_at`)

---

## Requirements

- Go 1.26 or later
- A GitHub personal access token with `repo` scope (classic) or the equivalent
  fine-grained permissions (`Issues: read`, `Pull requests: read`,
  `Metadata: read` on the repos you care about)
- A Notion internal integration token with access to the Project Manager and
  Stories databases. Create one at
  <https://www.notion.so/profile/integrations> and share both databases with it.

---

## Install

### Install script (Linux & macOS)

The quickest path. It downloads the prebuilt binary for your platform from the
latest GitHub release, verifies its checksum, and installs it system-wide:

```sh
curl -fsSL https://github.com/nox456/forgesync/releases/latest/download/install.sh | sh
```

Pick a specific version or a different install location with environment
variables:

```sh
curl -fsSL https://github.com/nox456/forgesync/releases/latest/download/install.sh \
  | FORGESYNC_VERSION=v0.1.0 FORGESYNC_INSTALL_DIR="$HOME/.local/bin" sh
```

> **Caveats**
>
> - **Linux and macOS only.** The script resolves `linux`/`darwin` and
>   `amd64`/`arm64`. On Windows, use `go install` or download the `.zip` from the
>   [releases page](https://github.com/nox456/forgesync/releases).
> - **It pipes remote code into your shell.** Convenient, but it runs unreviewed.
>   To inspect before running, download it first:
>   ```sh
>   curl -fsSL -o install.sh \
>     https://github.com/nox456/forgesync/releases/latest/download/install.sh
>   less install.sh   # read it
>   sh install.sh     # then run it
>   ```
> - **`sudo` may be requested.** The default install directory is
>   `/usr/local/bin`; if it isn't writable, the script falls back to `sudo`. Set
>   `FORGESYNC_INSTALL_DIR` to a writable path (e.g. `~/.local/bin`) to avoid
>   that — just make sure it's on your `PATH`.
> - **`FORGESYNC_VERSION` controls the binary, not the script URL.** Pinning the
>   *script* URL to a tag does not pin the *binary*; set the env var for a
>   specific version. The default is the latest release.
> - **Needs `curl` (or `wget`) and `tar`.** Checksum verification additionally
>   uses `sha256sum` or `shasum`; if neither is found, verification is skipped
>   with a warning rather than failing.
> - **A system-wide install updates with `sudo`.** `forgesync update` replaces
>   the binary in place, so with the default `/usr/local/bin` it needs
>   `sudo forgesync update` (see [Update forgesync](#update-forgesync)). Installing
>   to `~/.local/bin` avoids that.

### With Go

```sh
go install github.com/nox456/forgesync/cmd/forgesync@latest
```

Replace `@latest` with a tag (e.g. `@v0.1.0`) to pin a version. This compiles
from source, so the resulting binary depends on your local Go toolchain.
`forgesync update` doesn't replace a binary installed this way. It prints this
same `go install` command instead.

### From source

```sh
git clone https://github.com/nox456/forgesync.git
cd forgesync
go build -o forgesync ./cmd/forgesync
```

---

## Configuration

`forgesync` reads configuration from a YAML file at
`$XDG_CONFIG_HOME/forgesync/config.yaml` (commonly
`~/.config/forgesync/config.yaml`, provided `XDG_CONFIG_HOME` is set).
Environment variables prefixed with `FORGESYNC_` override the file's values.

> **Note:** a config file must exist, and every required value must be set,
> before any command runs except `version` and `update`, which never read it.
> `config` included, every other command errors out if it can't find the file,
> even when every value is also provided through environment variables.

### Required values

| Key                  | Env var                         | Description                                       |
| -------------------- | ------------------------------- | ------------------------------------------------- |
| `github_token`       | `FORGESYNC_GITHUB_TOKEN`        | GitHub personal access token                      |
| `notion_token`       | `FORGESYNC_NOTION_TOKEN`        | Notion integration token                          |
| `projects_source_id` | `FORGESYNC_PROJECTS_SOURCE_ID`  | Data source ID of the Project Manager database    |
| `stories_source_id`  | `FORGESYNC_STORIES_SOURCE_ID`   | Data source ID of the Stories database            |

### Example `config.yaml`

```yaml
github_token: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
notion_token: ntn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
projects_source_id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
stories_source_id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

### Overriding with environment variables

```sh
export FORGESYNC_GITHUB_TOKEN=ghp_...
export FORGESYNC_NOTION_TOKEN=ntn_...
export FORGESYNC_PROJECTS_SOURCE_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
export FORGESYNC_STORIES_SOURCE_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

---

## Usage

### Global flags

| Flag        | Description                                                        |
| ----------- | ------------------------------------------------------------------ |
| `--json`    | Print output as JSON. Also discards all logs, `--verbose` included. `update` rejects it |
| `--verbose` | Enable debug-level logging on stderr (no effect with `--json`)     |

`--repo owner/name` is **not** global: only `sync` and `status` accept it. See
those sections below.

### Sanity check — list Projects

Confirms your tokens and IDs work before you sync anything.

```sh
forgesync projects
```

### List assigned issues

Shows the GitHub issues `forgesync` would consider, with their linked-PR status.

```sh
forgesync issues
```

### Status report — what the sync would compute

Reads from both APIs, writes nothing. Prints one row per assigned issue: the
matching Project (or `[NO PROJECT]` when the issue's repo has no Project, in
which case the repo is shown next to the issue number), the issue number and
title, the `Status` the sync would compute, whether a PR is linked, and whether
the Story is already in sync.

```sh
forgesync status
forgesync status --repo owner/name
```

### Dry run — preview what would happen

No writes to Notion. It prints the same summary a real sync would — the
`Created` / `Updated` / `Unchanged` / `Skipped` counts plus any errors — not a
per-issue plan. Add `--verbose` (without `--json`) to see a debug line for each
issue that would be written or is unchanged.

```sh
forgesync sync --dry-run   # or -d
forgesync sync --dry-run --verbose
```

### Real sync

```sh
forgesync sync
forgesync sync --repo owner/name   # restrict to a single repo
```

`--repo owner/name` narrows both sides of the sync: it filters the Notion
`repo` field and the GitHub repo full name, case-insensitively. `projects` and
`issues` don't take it — they always list everything.

### Inspect the loaded configuration

Prints every loaded value, tokens included, in plain text.

```sh
forgesync config
```

### Print the version

Doesn't read the config, so it works right after installing.

```sh
forgesync version
```

### Update forgesync

Replaces the running binary with the latest GitHub release, after you
confirm. The download is checked against the release's `checksums.txt` before
the binary is touched. Doesn't read the config.

```sh
forgesync update
```

```
Update forgesync 0.9.0 → 0.10.0? [y/N] y
Downloading forgesync 0.10.0…
Updated forgesync 0.9.0 → 0.10.0
Release notes: https://github.com/nox456/forgesync/releases/tag/v0.10.0
```

- Only `y` or `yes` (any case) confirms. Enter, any other answer, or an empty
  stdin cancels, so a closed stdin never replaces the binary. To confirm from a
  script, pipe the answer: `echo y | forgesync update`.
- When you're already on the latest release, it says so and doesn't prompt.
- When the install directory isn't writable (the install script's default
  `/usr/local/bin`), it exits 1 and tells you to rerun with
  `sudo forgesync update`. It never runs `sudo` itself. On Windows, rerun from
  a terminal opened with "Run as administrator".
- A binary installed with `go install` isn't replaced. `update` prints the
  `go install …@latest` command to run instead. A local `go build` (version
  `dev`) is refused as well.
- No `--check` flag and no `--json` output. To see whether an update exists
  without installing it, answer `n` at the prompt.

### Update notice

When a newer release exists, every command prints a one-line notice on
**stderr** before its own output:

```
Update available! -> 0.10.0. Run: forgesync update
```

The notice reads a cached check, so it never waits on the network. When the
cache is older than the check interval (6 hours by default), the command
refreshes it in the background, from the unauthenticated GitHub Releases API
(or authenticated, if `GITHUB_TOKEN` is set). The refresh runs alongside the
command and is capped at 2 seconds from its start, so only a fast command on a
slow network waits for it. A failed refresh is silent unless you pass
`--verbose`, and the next command retries it.

- The notice, and the background refresh, only run when **stderr is a
  terminal**. Pipes, redirects, CI logs and editor integrations
  (forgesync.nvim) get neither, and `--json` output on stdout is never touched.
- Shell completion and `dev` builds never show it.
- The cache lives in its own state file, not in `config.yaml`:
  `$XDG_STATE_HOME/forgesync/update-check.yaml` (or
  `~/.local/state/forgesync/update-check.yaml` when `XDG_STATE_HOME` is unset).

```yaml
last_checked_at: 2026-10-13T09:30:00Z
latest_version: 0.10.0
update_check_interval: 6h
```

`update_check_interval` takes a Go duration (`30m`, `12h`, `24h`). Set it to
`0` to turn the background check off. A newer version already in the cache
still prints until you update. A missing or invalid value means `6h`.

---

## How your Notion databases need to be set up

The CLI expects these exact property names. Renaming one does not corrupt your
data, but it does not produce a helpful error either: Notion omits the unknown
property from its response, the decoder leaves the field at its zero value, and
the first read of it panics with `index out of range`. Loud, but a stack trace
rather than a message — see [Limitations](#limitations--known-caveats).

### Project Manager database

| Property | Type   | Required | Purpose                                    |
| -------- | ------ | -------- | ------------------------------------------ |
| `name`   | Title  | yes      | Project name                               |
| `repo`   | Text   | yes      | GitHub repo as `owner/name` (the bridge)   |

### Stories database

| Property         | Type         | Required | Purpose                                                      |
| ---------------- | ------------ | -------- | ------------------------------------------------------------ |
| `Name`           | Title        | yes      | Issue title                                                  |
| `Issue`          | Number       | yes      | GitHub issue number (the sync key)                           |
| `URL`            | URL          | yes      | Link back to the GitHub issue                                |
| `Status`         | Status       | yes      | One of: `Not started`, `In PR`, `In progress`, `Done`, `Cancelled` |
| `Labels`         | Multi-select | yes      | Mirrored from GitHub labels                                  |
| `Last Worked At` | Date         | yes      | Mirrored from the issue's `updated_at`                       |
| `Finished Date`  | Date         | yes      | Set from `closed_at` when the issue is closed                |
| `Project`        | Relation     | yes      | Relation to Project Manager (limit 1)                        |
| `Created time`   | Created time | yes      | Auto-managed by Notion                                       |
| `Prioridad`      | Select       | optional | User-managed; never touched by sync                          |

The `Status` options must exist with exactly those names — Notion rejects a
write to an option it doesn't know, so a database missing `In PR` fails on the
first open issue that has a PR linked.

---

## Project layout

```
forgesync/
├── cmd/forgesync/         # CLI entry point
└── internal/
    ├── cli/               # cobra commands
    ├── config/            # env + file config loading
    ├── github/            # GitHub adapter (REST issues + GraphQL PR links)
    ├── notion/            # Notion adapter (data sources API: reads + writes)
    ├── output/            # text & JSON printers
    ├── shared/            # domain types (Issue, Project, Story, …)
    ├── status/            # read-only report behind `forgesync status`
    ├── sync/              # orchestration & sync rules
    └── update/            # release lookup, update notice, self-update
```

Dependencies only ever point downwards:

```
cli → output → status → sync → github → shared
                  └──────────→ notion → shared
cli → update
```

The graph shows the main path. Two shortcuts sit alongside it: `status` also
imports `github` and `notion` directly, only for the concrete client types in
`NewCollector`, and `sync` imports `github` only for the pure
`NormalizeGithubBody` helper.

`internal/shared` is the leaf: domain types, no imports of our own, no
opinions. The adapters (`internal/github`, `internal/notion`) own their SDKs
and speak `shared` types outward, so swapping an SDK stays a one-package
change — neither one knows a sync exists. `internal/sync` owns every rule
about *what* should happen (`ComputeStatus`, `IsSynced`, `IssueToStoryInput`)
and drives the adapters through interfaces it declares itself. `internal/status`
is a read-only sibling that reuses those rules to preview a run.

`internal/update` imports none of our packages. It's the only package that
imports `go-selfupdate`, the same rule the adapters follow for their SDKs, so
`internal/cli` never sees a library type. The current version is passed in
from `internal/cli`, and the pure parts (`IsNewer`, `ShouldCheck`,
`ShouldNotify`, `IsGoInstall`) take every input as a parameter, the clock
included.

The one decision that could tempt an adapter upwards — "does this Story even
need writing?" — lives in the engine, which checks `IsSynced` before calling
`UpsertStory`. That keeps `internal/notion` a dumb pipe to the API.

---

## Limitations & known caveats

- **7-day window.** Only issues updated in the last 7 days are fetched. Older
  issues are not synced. The window is a hardcoded constant that has already
  moved more than once, which is the argument for making it configurable.
- **A renamed Notion property panics instead of erroring.** The decoders index
  straight into the JSON they expect (`Name.Title[0]`, `Repo.RichText[0]`,
  `Project.Relation[0]`). A renamed or emptied property decodes to a zero
  value, so those reads fail with `index out of range` and a stack trace. If
  you rename a property, rename it back or update the code — the CLI has no
  friendlier path yet.
- **Duplicate-Story errors can be false positives.** Story lookup is keyed by
  (Project, issue number), so the right Story is always the one that syncs.
  The duplicate check, however, compares each Project's Stories against
  **every** assigned issue by number alone. If two of your assigned issues
  share a number across repos — or one of them is in a repo with no Project —
  `sync` reports `found more than one story for issue N` even though nothing
  is duplicated. A real duplicate (two Stories in one Project with the same
  `Issue` number) produces the same error; `sync` keeps the first Story and
  continues with the rest. `status` is stricter: it looks Stories up per
  (Project, issue), so it only sees real duplicates, and aborts on them with
  `found more than one story for issue N`.
- **Rate limits.** Notion limits writes to ~3 req/s. For a typical personal
  workspace this is well under the threshold. If you have hundreds of issues
  you may want to add backoff.
- **Linked-PR detection only sees closing references.** A PR that just
  mentions the issue, without a closing keyword or a Development-sidebar link,
  is never detected. Draft PRs are ignored until they're marked ready. A
  closed-but-unmerged PR still counts as linked. Only the first 100 linked PRs
  are checked.
- **`--json` silences all logs.** With `--json`, nothing is written to stderr
  except a fatal error — not even `--verbose` debug output.
- **Fatal errors are printed twice.** When a command aborts, the error appears
  on stderr twice — once prefixed with `Error:` and followed by the usage text,
  and once plain — and the process exits with status 1. `update` is the
  exception: it prints its error once.
- **`go install` builds never get the update notice.** They report `version`
  as `dev`, and `dev` builds skip the notice. `forgesync update` still points
  them to the `go install …@latest` command.
- **Time zones.** Dates are written as `YYYY-MM-DD HH:MM` in UTC (the timezone
  offset is dropped). Notion displays them in your local time zone, so a value
  can read a few hours — or a day — off from the GitHub timestamp.
- **Manual `Status` edits are mostly preserved.** The sync folds your existing
  Notion status into the result, so a manual status on an open issue normally
  survives. The one exception is `In progress` + a linked PR, which is promoted
  to `In PR`. Otherwise the PR state only decides the status of a Story that
  has none yet (or is still `Not started`): a PR makes it `In PR`, otherwise
  `Not started`. A closed issue you set to `Cancelled` stays `Cancelled`; any
  other closed issue lands on `Done`. Other fields (`Name`, `Labels`,
  `Last Worked At`, `Finished Date`) are always refreshed from GitHub.

---

## Roadmap

- [x] Unit tests on `sync/mapping` and `sync/status`
- [x] Structured logging via `log/slog`
- [x] Optional `--repo owner/name` flag to sync a single repo
- [x] Break the `notion` → `utils` → `sync` dependency
- [x] Update notice and `forgesync update` self-update
- [ ] Configurable fetch window (currently a fixed 7 days)
- [ ] Cache the Projects map between runs
- [ ] Guard the Notion decoders against renamed or missing properties
