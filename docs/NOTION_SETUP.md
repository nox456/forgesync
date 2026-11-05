# Notion setup

`forgesync` works with two Notion databases: a **Project Manager** database
with one row per project, and a **Stories** database where each GitHub or
Linear issue becomes a Story. Share both with your Notion integration and put
their data source IDs in your config (see
[Configuration](../README.md#configuration)).

The CLI expects these exact property names. A renamed or empty property reads
as empty, so check `forgesync projects` and `forgesync status` after changing
the schema:

- A Project without a `provider` is listed under `Errors` and skipped.
- A Project whose `repo` or `teamKey` is empty matches no issues, so its issues
  are counted as skipped.
- Notion rejects a write to a Stories property that doesn't exist. Each
  rejected issue is listed under `Errors`.

Coming from v0.12.0? `Issue` changed from Number to Text, and Projects need
the new `provider` and `teamKey` properties. The
[v1.0.0 release notes](releases/v1.0.0.md#notion-schema) walk through the
migration.

## Project Manager database

| Property   | Type   | Required        | Purpose                                                  |
| ---------- | ------ | --------------- | -------------------------------------------------------- |
| `name`     | Title  | yes             | Project name                                             |
| `provider` | Select | yes             | `GitHub` or `Linear`: where the Project's issues come from |
| `repo`     | Text   | GitHub projects | GitHub repo as `owner/name` (the bridge)                 |
| `teamKey`  | Text   | Linear projects | Linear team key, e.g. `ENG` (the bridge)                 |

A Project follows one provider. The `provider` options are matched ignoring
case, and so are `repo` and `teamKey`. If you change a team's key in Linear,
update `teamKey` by hand: until you do, that Project's issues are skipped.

## Stories database

| Property         | Type         | Required | Purpose                                                      |
| ---------------- | ------------ | -------- | ------------------------------------------------------------ |
| `Name`           | Title        | yes      | Issue title                                                  |
| `Issue`          | Text         | yes      | Issue identifier, `#12` or `ENG-123` (the sync key)          |
| `URL`            | URL          | yes      | Link back to the GitHub or Linear issue                      |
| `Status`         | Status       | yes      | One of: `Not started`, `In PR`, `In progress`, `Done`, `Cancelled` |
| `Labels`         | Multi-select | yes      | Mirrored from the issue's labels                             |
| `Last Worked At` | Date         | yes      | Mirrored from the issue's last update                        |
| `Finished Date`  | Date         | yes      | Set when the issue is closed (GitHub) or completed (Linear)  |
| `Project`        | Relation     | yes      | Relation to Project Manager (limit 1)                        |
| `Created time`   | Created time | yes      | Auto-managed by Notion                                       |
| `Priority`       | Select       | optional | User-managed; never touched by sync                          |

The `Status` options must exist with exactly those names — Notion rejects a
write to an option it doesn't know, so a database missing `In PR` fails on the
first open issue that has a PR linked. `In progress` has to exist too:
`forgesync` writes it for started Linear issues, and on GitHub issues you set
it by hand and the sync promotes it to `In PR` when a PR is linked.
