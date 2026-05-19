# forgesync

A small Go CLI that syncs GitHub issues assigned to you into your Notion
workspace, linking each issue to the matching Project in your Project Manager
database.

One-way sync (GitHub → Notion), run on demand. No daemon, no webhooks.

---

## What it does

For every issue assigned to you on GitHub, `forgesync` will:

1. Look up the issue's repo (`owner/name`) in your **Project Manager** database
   via the `repo` text field.
2. If a matching Project exists, upsert a row in the **Stories** database
   linked to that Project. If no Project matches, the issue is skipped.
3. Keep the Story's `Name`, `URL`, `Labels`, `Status`, `Last Worked At`, and
   `Finished Date` in sync with the GitHub issue.

The sync key is the pair `(Project, Issue number)` — running `forgesync sync`
repeatedly is safe and idempotent.

### Status mapping

| GitHub state              | Notion `Status` |
| ------------------------- | --------------- |
| Closed                    | `Done`          |
| Open, with a linked PR    | `In PR`         |
| Open, no linked PR        | `In progress`   |

Linked-PR detection uses GitHub's GraphQL `timelineItems` (`CONNECTED_EVENT`
and `CROSS_REFERENCED_EVENT`).

### What it does NOT touch

- The issue body / page content
- The `Prioridad` property (you own this in Notion)
- The `Created time` property (auto-managed by Notion)
- `Finished Date` if already set (manual edits are preserved)

---

## Requirements

- Go 1.22 or later
- A GitHub personal access token with `repo` scope (classic) or the equivalent
  fine-grained permissions (`Issues: read`, `Pull requests: read`,
  `Metadata: read` on the repos you care about)
- A Notion internal integration token with access to the Project Manager and
  Stories databases. Create one at
  <https://www.notion.so/profile/integrations> and share both databases with it.

---

## Install

```sh
go install github.com/yourname/forgesync/cmd/forgesync@latest
```

Or build from source:

```sh
git clone https://github.com/yourname/forgesync.git
cd forgesync
go build -o forgesync ./cmd/forgesync
```

---

## Configuration

`forgesync` reads configuration from, in order of precedence:

1. Environment variables (highest precedence)
2. `$XDG_CONFIG_HOME/forgesync/config.yaml` (or `~/.config/forgesync/config.yaml`)
3. `./forgesync.yaml` in the working directory

### Required values

| Key                       | Env var                              | Description                                  |
| ------------------------- | ------------------------------------ | -------------------------------------------- |
| `github_token`            | `FORGESYNC_GITHUB_TOKEN`             | GitHub personal access token                 |
| `notion_token`            | `FORGESYNC_NOTION_TOKEN`             | Notion integration token                     |
| `projects_data_source_id` | `FORGESYNC_PROJECTS_DATA_SOURCE_ID`  | UUID of the Project Manager database         |
| `stories_data_source_id`  | `FORGESYNC_STORIES_DATA_SOURCE_ID`   | UUID of the Stories database                 |

### Example `config.yaml`

```yaml
github_token: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
notion_token: ntn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
projects_data_source_id: 2fbb053d-a558-8039-9bd5-000bec9a0d57
stories_data_source_id: 362b053d-a558-80b7-95fb-000bc69d7347
```

### Example env-only setup

```sh
export FORGESYNC_GITHUB_TOKEN=ghp_...
export FORGESYNC_NOTION_TOKEN=ntn_...
export FORGESYNC_PROJECTS_DATA_SOURCE_ID=2fbb053d-a558-8039-9bd5-000bec9a0d57
export FORGESYNC_STORIES_DATA_SOURCE_ID=362b053d-a558-80b7-95fb-000bc69d7347
```

---

## Usage

### Sanity check — list Projects

Confirms your tokens and IDs work before you sync anything.

```sh
forgesync projects list
```

### Dry run — preview what would happen

No writes to Notion. Just prints the planned actions.

```sh
forgesync sync --dry-run
```

### Real sync

```sh
forgesync sync
```

### Verbose output

```sh
forgesync sync --verbose
```

---

## How your Notion databases need to be set up

The CLI expects specific property names. If any are renamed, sync will fail
loudly rather than silently corrupt data.

### Project Manager database

| Property | Type   | Required | Purpose                                    |
| -------- | ------ | -------- | ------------------------------------------ |
| `Nombre` | Title  | yes      | Project name                               |
| `repo`   | Text   | yes      | GitHub repo as `owner/name` (the bridge)   |

### Stories database

| Property         | Type         | Required | Purpose                                       |
| ---------------- | ------------ | -------- | --------------------------------------------- |
| `Name`           | Title        | yes      | Issue title                                   |
| `Issue`          | Number       | yes      | GitHub issue number (part of the sync key)    |
| `URL`            | URL          | yes      | Link back to the GitHub issue                 |
| `Status`         | Status       | yes      | One of: `Not started`, `In progress`, `In PR`, `Done` |
| `Labels`         | Multi-select | yes      | Mirrored from GitHub labels                   |
| `Last Worked At` | Date         | yes      | Mirrored from the issue's `updated_at`        |
| `Finished Date`  | Date         | yes      | Set from `closed_at` when issue is closed     |
| `⌨️ Project`     | Relation     | yes      | Relation to Project Manager (limit 1)         |
| `Prioridad`      | Select       | optional | User-managed; never touched by sync           |

---

## Project layout

```
forgesync/
├── cmd/forgesync/         # CLI entry point
└── internal/
    ├── config/            # env + file config loading
    ├── github/            # GitHub adapter (REST + GraphQL)
    ├── notion/            # Notion adapter (reads + writes)
    ├── sync/              # orchestration & status mapping rules
    └── cli/               # cobra commands
```

Each `internal/` package is independent: `internal/sync` only depends on
`internal/github` and `internal/notion` through small domain types, never on
the underlying SDKs directly. Swapping an SDK is a one-package change.

---

## Limitations & known caveats

- **Rate limits.** Notion limits writes to ~3 req/s. For a typical personal
  workspace this is well under the threshold. If you have hundreds of issues
  you may want to add backoff.
- **Linked-PR detection is heuristic.** A PR that doesn't use a closing
  keyword (`Closes #123`) may not be detected via `timelineItems`.
- **Time zones.** All timestamps are stored in UTC. Notion displays them in
  your local time zone.
- **Manual `Status` edits in Notion are overwritten.** If you set a Story to
  `Done` manually but the GitHub issue is still open, the next sync will
  revert it. (This is intentional — GitHub is the source of truth.)

---

## Roadmap

- [ ] Unit tests on `sync/mapping` and `sync/status`
- [ ] Structured logging via `log/slog`
- [ ] `--since` flag to limit fetches to recently updated issues
- [ ] Cache the Projects map between runs
- [ ] Optional `--repo owner/name` flag to sync a single repo

