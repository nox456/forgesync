# Notion setup

`forgesync` works with two Notion databases: a **Project Manager** database
with one row per project, and a **Stories** database where each GitHub issue
becomes a Story. Share both with your Notion integration and put their data
source IDs in your config (see [Configuration](../README.md#configuration)).

The CLI expects these exact property names. Renaming one does not corrupt your
data, but it does not produce a helpful error either: Notion omits the unknown
property from its response, the decoder leaves the field at its zero value, and
the first read of it panics with `index out of range`. Loud, but a stack trace
rather than a message. An empty value crashes the same way, so every Project
needs a `name` and a `repo`, and every Story needs a `Name`.

## Project Manager database

| Property | Type   | Required | Purpose                                    |
| -------- | ------ | -------- | ------------------------------------------ |
| `name`   | Title  | yes      | Project name                               |
| `repo`   | Text   | yes      | GitHub repo as `owner/name` (the bridge)   |

## Stories database

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
first open issue that has a PR linked. `In progress` has to exist too, even
though `forgesync` never writes it: you set it by hand, and the sync promotes
it to `In PR` when a PR is linked.
