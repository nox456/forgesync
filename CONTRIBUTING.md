# Contributing

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
