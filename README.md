# forgesync

A small Go CLI that syncs the GitHub issues assigned to you into Notion. Each
issue becomes a Story in your Stories database, linked to the matching Project
in your Project Manager database.

One-way sync (GitHub → Notion), run on demand. No daemon, no webhooks.

---

## How it works

For every issue assigned to you on GitHub (updated in the last 7 days, pull
requests excluded), `forgesync sync`:

1. Looks up the issue's repo (`owner/name`) in your **Project Manager**
   database through its `repo` field. If no Project matches, the issue is
   skipped.
2. Creates or updates the issue's Story in your **Stories** database, linked
   to that Project.
3. Keeps the Story's `Name`, `Labels`, `Last Worked At`, and `Finished Date` in
   sync with the issue, mirrors the issue body into the Story's page content,
   and computes its `Status` (see [Status mapping](#status-mapping)). The
   `URL`, `Issue` number, and `Project` relation are set once, when the Story
   is created.

A Story is identified by its **(Project, issue number)** pair, so running
`forgesync sync` again is always safe: existing Stories are updated, missing
ones are created, and a Story whose `Name`, `Status`, `Labels`, and
`Last Worked At` already match is reported as `Unchanged` and left alone.

### Status mapping

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

`forgesync` never writes `In progress` itself. You set it in Notion to mark
that you've started work, and the sync promotes it to `In PR` when a PR
appears.

### What it never changes

- `URL`, `Issue`, and `Project` after the Story is created
- `Finished Date` while the issue is open (it's set from `closed_at` once the
  issue closes)
- `Created time`, which Notion manages, and any property not listed above
  (e.g. `Prioridad` — you own it in Notion)

---

## Requirements

- A GitHub personal access token: classic with the `repo` scope, or
  fine-grained with `Issues: read`, `Pull requests: read`, and
  `Metadata: read` on the repos you care about.
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
| `notion_token`       | `FORGESYNC_NOTION_TOKEN`        | Notion integration token                          |
| `projects_source_id` | `FORGESYNC_PROJECTS_SOURCE_ID`  | Data source ID of the Project Manager database    |
| `stories_source_id`  | `FORGESYNC_STORIES_SOURCE_ID`   | Data source ID of the Stories database            |

All four are required.

```yaml
# ~/.config/forgesync/config.yaml
github_token: ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
notion_token: ntn_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
projects_source_id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
stories_source_id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

```sh
# Overriding with environment variables
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

`--repo owner/name` is **not** global: only `sync` and `status` accept it.

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

Reads from both APIs and writes nothing. Prints one row per assigned issue:
its Project (or `[NO PROJECT]`, with the repo next to the issue number), the
issue number and title, the `Status` the sync would compute, whether a PR is
linked, and whether the Story is already in sync.

```sh
forgesync status
forgesync status --repo owner/name
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
forgesync sync --repo owner/name   # restrict to a single repo
```

`--repo owner/name` narrows both sides of the sync: it filters the Notion
`repo` field and the GitHub repo full name, case-insensitively.

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

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the project layout and how the
packages depend on each other.
