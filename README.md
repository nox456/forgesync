# forgesync

A small Go CLI that syncs GitHub issues assigned to you into your Notion
workspace, linking each issue to the matching Project in your Project Manager
database.

One-way sync (GitHub → Notion), run on demand. No daemon, no webhooks.

---

## What it does

For every issue assigned to you on GitHub (updated in the last 30 days, pull
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

The sync key is the GitHub **issue number**: `forgesync` looks up an existing
Story by its `Issue` number and updates it, otherwise it creates one. Running
`forgesync sync` repeatedly is safe and idempotent. A Story whose `Name`,
`Status`, `Labels`, and `Last Worked At` already match is reported as
`Unchanged` and left untouched.

### Status mapping

When a Story is **first created**, its `Status` is derived purely from the
GitHub issue:

| GitHub issue state | Linked PR? | Notion `Status` |
| ------------------ | ---------- | --------------- |
| Open               | no         | `Not started`   |
| Open               | yes        | `In progress`   |
| Closed             | yes        | `Done`          |
| Closed             | no         | `Done`          |

Any other/unknown state falls back to `Not started`.

On later syncs the **previous Notion status is folded in**, so some manual
edits survive instead of being overwritten:

- An **open** issue keeps whatever status the Story already has, so a manual
  `In progress`, `Done`, or `Cancelled` is preserved — a linked PR does **not**
  override a manual status. Only when the Story has no status yet (or is still
  `Not started`) does the issue decide: a linked PR makes it `In progress`,
  otherwise it stays `Not started`.
- A **closed** issue you previously set to `Cancelled` stays `Cancelled`, even
  if a PR is later linked.
- Any other closed issue becomes `Done`, whether or not it has a linked PR.

Linked-PR detection walks the issue's REST timeline
(`ListIssueTimeline`) and counts `connected` minus `disconnected` events; a
positive total means a PR is linked.

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

### With Go

```sh
go install github.com/nox456/forgesync/cmd/forgesync@latest
```

Replace `@latest` with a tag (e.g. `@v0.1.0`) to pin a version. This compiles
from source, so the resulting binary depends on your local Go toolchain.

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

> **Note:** a config file must exist — the CLI errors out if it cannot find
> one, even when every value is also provided through environment variables.

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
projects_source_id: 2fbb053d-a558-8039-9bd5-000bec9a0d57
stories_source_id: 362b053d-a558-80b7-95fb-000bc69d7347
```

### Overriding with environment variables

```sh
export FORGESYNC_GITHUB_TOKEN=ghp_...
export FORGESYNC_NOTION_TOKEN=ntn_...
export FORGESYNC_PROJECTS_SOURCE_ID=2fbb053d-a558-8039-9bd5-000bec9a0d57
export FORGESYNC_STORIES_SOURCE_ID=362b053d-a558-80b7-95fb-000bc69d7347
```

---

## Usage

### Global flags

| Flag        | Description                          |
| ----------- | ------------------------------------ |
| `--json`    | Print output as JSON                 |
| `--verbose` | Enable debug-level logging           |

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

### Dry run — preview what would happen

No writes to Notion. Just prints the planned actions.

```sh
forgesync sync --dry-run   # or -d
```

### Real sync

```sh
forgesync sync
```

### Inspect the loaded configuration

```sh
forgesync config
```

### Print the version

```sh
forgesync version
```

---

## How your Notion databases need to be set up

The CLI expects specific property names. If any are renamed, sync will fail
loudly rather than silently corrupt data.

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
| `Status`         | Status       | yes      | One of: `Not started`, `In progress`, `Done`, `Cancelled` |
| `Labels`         | Multi-select | yes      | Mirrored from GitHub labels                                  |
| `Last Worked At` | Date         | yes      | Mirrored from the issue's `updated_at`                       |
| `Finished Date`  | Date         | yes      | Set from `closed_at` when the issue is closed                |
| `Project`        | Relation     | yes      | Relation to Project Manager (limit 1)                        |
| `Created time`   | Created time | yes      | Auto-managed by Notion                                       |
| `Prioridad`      | Select       | optional | User-managed; never touched by sync                          |

---

## Project layout

```
forgesync/
├── cmd/forgesync/         # CLI entry point
└── internal/
    ├── cli/               # cobra commands
    ├── config/            # env + file config loading
    ├── github/            # GitHub adapter (issues + REST timeline)
    ├── notion/            # Notion adapter (data sources API: reads + writes)
    ├── output/            # text & JSON printers
    └── sync/              # orchestration & status mapping rules
```

Each `internal/` package is independent: `internal/sync` only depends on
`internal/github` and `internal/notion` through small domain types, never on
the underlying SDKs directly. Swapping an SDK is a one-package change.

---

## Limitations & known caveats

- **30-day window.** Only issues updated in the last 30 days are fetched.
  Older issues are not synced.
- **Issue numbers must be unique across synced repos.** The sync key is the
  GitHub issue number alone. If two of your synced repos share an issue number,
  the sync errors out (`found more than one story for issue N`) or updates the
  wrong Story.
- **Rate limits.** Notion limits writes to ~3 req/s. For a typical personal
  workspace this is well under the threshold. If you have hundreds of issues
  you may want to add backoff.
- **Linked-PR detection is heuristic.** A PR that isn't surfaced as a
  `connected` timeline event (for example, one that only mentions the issue
  without a closing keyword) may not be detected.
- **Time zones.** Dates are written as `YYYY-MM-DD HH:MM` in UTC (the timezone
  offset is dropped). Notion displays them in your local time zone, so a value
  can read a few hours — or a day — off from the GitHub timestamp.
- **Manual `Status` edits are largely preserved.** The sync folds your existing
  Notion status into the result, so any manual status on an open issue survives
  — a linked PR no longer overrides it. The PR state only decides the status of
  a Story that has none yet (or is still `Not started`): a PR makes it
  `In progress`, otherwise `Not started`. A closed issue you set to `Cancelled`
  stays `Cancelled`; any other closed issue lands on `Done`. Other fields
  (`Name`, `Labels`, `Last Worked At`, `Finished Date`) are always refreshed
  from GitHub.

---

## Roadmap

- [x] Unit tests on `sync/mapping` and `sync/status`
- [x] Structured logging via `log/slog`
- [ ] Configurable fetch window (currently a fixed 30 days)
- [ ] Cache the Projects map between runs
- [ ] Optional `--repo owner/name` flag to sync a single repo
