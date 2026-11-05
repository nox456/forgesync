# Contributing

## Project layout

```
forgesync/
├── cmd/forgesync/         # CLI entry point
└── internal/
    ├── cli/               # cobra commands
    ├── config/            # env + file config loading
    ├── github/            # GitHub adapter (REST issues + GraphQL PR links)
    ├── linear/            # Linear adapter (GraphQL issues, standard library only)
    ├── notion/            # Notion adapter (data sources API: reads + writes)
    ├── output/            # text & JSON printers
    ├── shared/            # domain types (Issue, Project, Story, …)
    ├── status/            # read-only report behind `forgesync status`
    ├── sync/              # orchestration & sync rules
    └── update/            # release lookup, update notice, self-update
```

Dependencies only ever point downwards:

```
cli → output → status → sync → shared
cli → github, linear, notion → shared
cli → config, update
```

Only `internal/cli` imports the adapters. It builds the clients and hands them
to `sync` and `status` through interfaces those packages declare themselves
(`sync.Provider`, `sync.Notion`, `status.Notion`), so neither knows which
concrete client it is driving.

`internal/shared` is the leaf: domain types, no imports of our own, no
opinions. The adapters (`internal/github`, `internal/linear`,
`internal/notion`) own their API clients (`go-github` for GitHub, plain
`net/http` for Linear and Notion) and speak `shared` types outward, so
swapping a client stays a one-package change — none of them knows a sync
exists. Each issue adapter cleans up its own issue bodies before handing them
over. `internal/sync` owns every rule about *what*
should happen (`ComputeStatus`, `IsSynced`, `IssueToStoryInput`,
`ClassifyProjects`). `internal/status` is a read-only sibling that reuses
those rules to preview a run.

`sync.Provider` is the extension point for a new issue source. A provider
implements `Name()` and `FetchAssignedIssues(ctx, filter)` in its own adapter
package. Adding one also means a `shared.Provider` constant, a case in
`ComputeStatus` with its status rules, the known-provider check in
`ClassifyProjects`, a case in the `provider` select mapping in
`notion.ListProjects`, a token in `internal/config`, and the wiring and
`--provider` value in `internal/cli`.

`internal/update` imports none of our packages. It's the only package that
imports `go-selfupdate`, the same rule the adapters follow for their SDKs, so
`internal/cli` never sees a library type. The current version is passed in
from `internal/cli`, and the pure parts (`IsNewer`, `ShouldCheck`,
`ShouldNotify`, `IsGoInstall`) take every input as a parameter, the clock
included.

The one decision that could tempt an adapter upwards — "does this Story even
need writing?" — lives in the engine, which checks `IsSynced` before calling
`UpsertStory`. That keeps `internal/notion` a dumb pipe to the API.
