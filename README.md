# forgesync

A small Go CLI that syncs the GitHub and Linear issues assigned to you into
Notion. Each issue becomes a Story in your Stories database, linked to the
matching Project in your Project Manager database.

One-way sync (GitHub and Linear → Notion), run on demand. No daemon, no
webhooks.

---

## How it works

For every issue assigned to you and updated in the last 7 days, on GitHub
(pull requests excluded) and on Linear (triage issues excluded),
`forgesync sync`:

1. Looks up the issue's Project in your **Project Manager** database. Each
   Project's `provider` says where its issues come from: a GitHub issue matches
   the Project whose `repo` is its repo (`owner/name`), and a Linear issue
   matches the Project whose `teamKey` is its team's key (e.g. `ENG`). Both
   comparisons ignore case. If no Project matches, the issue is skipped.
2. Creates or updates the issue's Story in your **Stories** database, linked
   to that Project.
3. Keeps the Story's `Name`, `Labels`, `Last Worked At`, and `Finished Date` in
   sync with the issue, mirrors the issue body into the Story's page content,
   and computes its `Status` (see [Status mapping](#status-mapping)). Images
   uploaded to Linear are left out of the body: they need your Linear login and
   would show as broken in Notion. The `URL`, `Issue` identifier (`#12` on
   GitHub, `ENG-123` on Linear), and `Project` relation are set once, when the
   Story is created.

A Story is identified by its **(Project, issue identifier)** pair, so running
`forgesync sync` again is always safe: existing Stories are updated, missing
ones are created, and a Story whose `Name`, `Status`, `Labels`, and
`Last Worked At` already match is reported as `Unchanged` and left alone.

Only the providers you have a token for are used (see
[Configuration](#configuration)). Problems don't stop the run; they are
listed under `Errors` at the end:

- A provider that fails (an expired token, an API outage). The other provider
  still syncs.
- A Project with no `provider`, an unknown one, or a provider you have no token
  for. Its issues are skipped.

### Status mapping

GitHub and Linear follow different rules. GitHub keeps most statuses you set
by hand in Notion; Linear always overwrites them, because the Linear issue
already has a status of its own.

#### GitHub

A new Story gets its `Status` from the GitHub issue alone:

| GitHub issue state | Linked PR? | Notion `Status` |
| ------------------ | ---------- | --------------- |
| Open               | no         | `Not started`   |
| Open               | yes        | `In PR`         |
| Closed             | any        | `Done`          |

On later syncs the Story's current status is taken into account, so most
manual edits you make in Notion survive:

- **Open issue, no status or `Not started`:** GitHub decides — `In PR` with a
  linked PR, `Not started` without one.
- **Open issue, `In progress`:** promoted to `In PR` as soon as a PR is linked.
  This is the only case where GitHub overrides a status you set by hand.
- **Open issue, any other status** (`Done`, `Cancelled`, `In PR`): kept.
- **Closed issue:** becomes `Done`, unless you set it to `Cancelled`.

A PR counts as linked when it closes the issue; draft PRs don't count.

`forgesync` never writes `In progress` for a GitHub issue. You set it in
Notion to mark that you've started work, and the sync promotes it to `In PR`
when a PR appears.

#### Linear

Every sync takes the `Status` from the Linear issue's workflow state type,
whatever the Story says:

| Linear state type     | Linked PR? | Notion `Status` |
| --------------------- | ---------- | --------------- |
| Backlog, Unstarted    | any        | `Not started`   |
| Started               | no         | `In progress`   |
| Started               | yes        | `In PR`         |
| Completed             | any        | `Done`          |
| Canceled              | any        | `Cancelled`     |

The state type is the category a workflow state belongs to, so a custom state
like "In Review" counts as Started. A status you edit by hand in Notion is
overwritten on the next sync, so change it in Linear instead.

A PR counts as linked when a GitHub pull request is attached to the Linear
issue, drafts included.

### What it never changes

- `URL`, `Issue`, and `Project` after the Story is created
- `Finished Date` while the issue is open. It's set from GitHub's `closed_at`
  once the issue closes, or from Linear's `completedAt` once it's completed. A
  canceled Linear issue never gets one.
- `Created time`, which Notion manages, and any property not listed above
  (e.g. `Prioridad` — you own it in Notion)

---

## Requirements

- At least one of:
  - A GitHub personal access token: classic with the `repo` scope, or
    fine-grained with `Issues: read`, `Pull requests: read`, and
    `Metadata: read` on the repos you care about.
  - A Linear personal API key, created under **Settings → Account → Security
    & access** (<https://linear.app/settings/account/security>).
- A Notion internal integration token, created at
  <https://www.notion.so/profile/integrations>, with your Project Manager and
  Stories databases shared with it.
- Both databases set up with the properties `forgesync` expects — see
  [Notion setup](docs/NOTION_SETUP.md).
- Go 1.26.3 or later, only if you install with `go install` or build from
  source.

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
> - **Linux and macOS only** (`amd64` and `arm64`). On Windows, use
>   `go install` or download the `.zip` from the
>   [releases page](https://github.com/nox456/forgesync/releases).
> - **It pipes remote code into your shell.** To read the script before
>   running it:
>   ```sh
>   curl -fsSL -o install.sh \
>     https://github.com/nox456/forgesync/releases/latest/download/install.sh
>   less install.sh   # read it
>   sh install.sh     # then run it
>   ```
> - **It may ask for `sudo`.** The default install directory, `/usr/local/bin`,
>   is usually not writable. Set `FORGESYNC_INSTALL_DIR` to a writable
>   directory on your `PATH` (e.g. `~/.local/bin`) to avoid it — which also
>   spares you `sudo` on every [update](#update-forgesync).
> - **`FORGESYNC_VERSION` pins the binary.** Pinning the script URL to a tag
>   does not; the default is the latest release.
> - **Needs `curl` (or `wget`) and `tar`.** The checksum check uses
>   `sha256sum` or `shasum`, and is skipped with a warning if neither is found.

### With Go

```sh
go install github.com/nox456/forgesync/cmd/forgesync@latest
```

Replace `@latest` with a tag (e.g. `@v0.1.0`) to pin a version.
`forgesync update` can't replace a binary installed this way, so rerun this
command to update.

### From source

```sh
git clone https://github.com/nox456/forgesync.git
cd forgesync
go build -o forgesync ./cmd/forgesync
```

---

## Configuration

`forgesync` reads its configuration from
`$XDG_CONFIG_HOME/forgesync/config.yaml` (commonly
`~/.config/forgesync/config.yaml`, provided `XDG_CONFIG_HOME` is set).
Environment variables prefixed with `FORGESYNC_` override the file's values.

> **Note:** every command except `version` and `update` needs the file to
> exist, even when every value comes from environment variables, and errors
> out if a required value is missing.

| Key                  | Env var                         | Description                                       |
| -------------------- | ------------------------------- | ------------------------------------------------- |
| `github_token`       | `FORGESYNC_GITHUB_TOKEN`        | GitHub personal access token                      |
| `linear_token`       | `FORGESYNC_LINEAR_TOKEN`        | Linear personal API key                           |
| `notion_token`       | `FORGESYNC_NOTION_TOKEN`        | Notion integration token                          |
| `projects_source_id` | `FORGESYNC_PROJECTS_SOURCE_ID`  | Data source ID of the Project Manager database    |
| `stories_source_id`  | `FORGESYNC_STORIES_SOURCE_ID`   | Data source ID of the Stories database            |

`notion_token`, `projects_source_id`, and `stories_source_id` are required,
plus at least one of `github_token` and `linear_token`.

```yaml
# ~/.config/forgesync/config.yaml
github_token: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
linear_token: lin_api_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx  # either token, or both
notion_token: ntn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
projects_source_id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
stories_source_id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

```sh
# Overriding with environment variables
export FORGESYNC_GITHUB_TOKEN=ghp_...
export FORGESYNC_LINEAR_TOKEN=lin_api_...
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

### Choosing providers

`sync`, `status`, and `issues` use every provider you have a token for. Three
flags, accepted only by those commands, narrow a run:

| Flag         | Value                         | Effect                                              |
| ------------ | ----------------------------- | --------------------------------------------------- |
| `--provider` | `github` or `linear`          | Only that provider                                  |
| `--repo`     | `owner/name`                  | Only that GitHub repo (implies `--provider github`) |
| `--team`     | a Linear team key, e.g. `ENG` | Only that Linear team (implies `--provider linear`) |

`--repo` and `--team` ignore case and filter both sides: the issues fetched
and the Notion Projects. When a flag narrows the run, Projects of the other
provider are skipped quietly instead of being reported.

These are rejected before any API call:

- `--repo` together with `--team`
- `--repo` with `--provider linear`, or `--team` with `--provider github`
- a `--provider` other than `github` or `linear`
- a flag for a provider you have no token for

A provider that fails is listed under `Errors` after the output, and the other
provider still runs. `status` and `issues` exit with an error only when every
provider fails.

### Sanity check — list Projects

Confirms your tokens and IDs work before you sync anything.

```sh
forgesync projects
```

### List assigned issues

Shows the issues `forgesync` would consider: provider, identifier, repo or
team key, title, state, and whether a PR is linked.

```sh
forgesync issues
forgesync issues --team ENG
```

### Status report — what the sync would compute

Reads from Notion and your providers and writes nothing. Prints one row per
assigned issue: its provider, its Project (or `[NO PROJECT]`, with the repo or
team key next to the identifier), the identifier and title, the `Status` the
sync would compute, whether a PR is linked, and whether the Story is already in
sync. Problems, such as a failing provider or a misconfigured Project, are
listed after the table.

```sh
forgesync status
forgesync status --repo owner/name
forgesync status --provider linear
```

### Dry run — preview what would happen

Writes nothing to Notion. Prints the same summary a real sync would — the
`Created` / `Updated` / `Unchanged` / `Skipped` counts plus any errors — not a
per-issue plan. Add `--verbose` (without `--json`) for a debug line per issue
that would be written or is unchanged.

```sh
forgesync sync --dry-run   # or -d
forgesync sync --dry-run --verbose
```

### Real sync

```sh
forgesync sync
forgesync sync --repo owner/name   # restrict to a single GitHub repo
forgesync sync --team ENG          # restrict to a single Linear team
```

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

- When the install directory isn't writable (the install script's default
  `/usr/local/bin`), it exits 1 and tells you to rerun with
  `sudo forgesync update`. It never runs `sudo` itself. On Windows, rerun from
  a terminal opened with "Run as administrator".
- A binary installed with `go install` isn't replaced. `update` prints the
  `go install …@latest` command to run instead. A local `go build` (version
  `dev`) is refused as well.

### JSON output

With `--json`, `projects`, `issues`, `status`, and `sync` print one JSON
object on stdout. From v1.0.0 these fields are a stable contract: removing or
retyping one is a breaking change. The table lists field names only:

| Command    | Shape                                                                                                                       |
| ---------- | --------------------------------------------------------------------------------------------------------------------------- |
| `projects` | `{"projects": [{"name", "scope", "provider"}]}`                                                                             |
| `issues`   | `{"issues": [{"identifier", "scope", "title", "has_linked_pr", "provider"}], "errors": [...]}`                              |
| `status`   | `{"rows": [{"identifier", "issue_title", "project_name", "scope", "status", "has_pr", "is_synced", "provider"}], "errors": [...]}` |
| `sync`     | `{"report": {"created", "updated", "skipped", "unchanged", "errors": [...]}}`                                               |

- `identifier` is a string: `#12` on GitHub, `ENG-123` on Linear.
- `provider` is `github` or `linear`.
- `scope` is the repo (`owner/name`) for GitHub and the team key for Linear.
- `project_name` is `null` when no Project matches the issue.
- `errors` is always an array, `[]` when there are none. Each error is
  `{"provider", "project", "identifier", "message"}`: `message` is always
  there, the other three only when they apply. A failing provider has just
  `provider` and `message`; a misconfigured Project has `project`.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the project layout and how the
packages depend on each other.
