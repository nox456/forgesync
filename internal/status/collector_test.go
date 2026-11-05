package status

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/sync"
)

type fakeNotion struct {
	projects    []shared.Project
	projectsErr error
	story       *shared.Story
	storyErr    error
	// gotFilter records the filter passed to ListProjects.
	gotFilter shared.ProjectFilter
}

func (f *fakeNotion) ListProjects(ctx context.Context, filter shared.ProjectFilter) ([]shared.Project, error) {
	f.gotFilter = filter
	return f.projects, f.projectsErr
}

func (f *fakeNotion) FindStoryByIssue(ctx context.Context, issue shared.Issue, projectId string) (*shared.Story, error) {
	return f.story, f.storyErr
}

// fakeProvider is a sync.Provider whose name and issues are set by the test.
type fakeProvider struct {
	name   shared.Provider
	issues []shared.Issue
	err    error
	// gotFilter records the filter passed to FetchAssignedIssues.
	gotFilter string
}

func (f *fakeProvider) Name() shared.Provider { return f.name }

func (f *fakeProvider) FetchAssignedIssues(ctx context.Context, filter string) ([]shared.Issue, error) {
	f.gotFilter = filter
	return f.issues, f.err
}

// quietLogs swallows the collector's slog output so the informational lines
// don't clutter the test output, restoring the previous default afterwards.
func quietLogs(t *testing.T) {
	t.Helper()
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
}

// projectNames returns each row's project name, or "" when the row has none,
// so a test can compare which issue landed in which project in one diff.
func projectNames(rows []Row) map[string]string {
	names := make(map[string]string, len(rows))
	for _, row := range rows {
		if row.ProjectName == nil {
			names[row.Identifier] = ""
			continue
		}
		names[row.Identifier] = *row.ProjectName
	}
	return names
}

// The Notion repo field is typed by hand, so its casing rarely matches the one
// GitHub reports. Matching must be case-insensitive, the same way the sync
// engine does it — otherwise `status` shows [NO PROJECT] for issues `sync`
// syncs without complaint.
func TestCollectMatchesProjectRepoIgnoringCase(t *testing.T) {
	quietLogs(t)

	collector := NewCollector(
		&fakeNotion{
			projects: []shared.Project{
				{PageID: "page-1", Name: "ForgeSync", Repo: "Nox456/ForgeSync", Provider: shared.ProviderGithub},
			},
		},
		[]sync.Provider{&fakeProvider{
			name: shared.ProviderGithub,
			issues: []shared.Issue{
				{Provider: shared.ProviderGithub, Identifier: "#42", Title: "Sync statuses", Scope: "nox456/forgesync", State: "open"},
			},
		}},
	)

	rows, _, err := collector.Collect(context.Background(), shared.ProjectFilter{}, false)
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if len(rows) != 1 {
		t.Fatalf("Collect() returned %d rows, want 1", len(rows))
	}

	if rows[0].ProjectName == nil {
		t.Fatalf("Collect() row has no project, want %q", "ForgeSync")
	}

	if got := *rows[0].ProjectName; got != "ForgeSync" {
		t.Errorf("Collect() project name = %q, want %q", got, "ForgeSync")
	}
}

func TestCollect(t *testing.T) {
	quietLogs(t)

	t.Run("happy path", func(t *testing.T) {
		t.Run("issues from every provider land in their own project", func(t *testing.T) {
			collector := NewCollector(
				&fakeNotion{projects: []shared.Project{
					{PageID: "p-gh", Name: "Repo project", Repo: "owner/repo", Provider: shared.ProviderGithub},
					{PageID: "p-lin", Name: "Team project", TeamKey: "ENG", Provider: shared.ProviderLinear},
				}},
				[]sync.Provider{
					&fakeProvider{name: shared.ProviderGithub, issues: []shared.Issue{
						{Provider: shared.ProviderGithub, Identifier: "#1", Scope: "owner/repo", State: "open"},
					}},
					&fakeProvider{name: shared.ProviderLinear, issues: []shared.Issue{
						{Provider: shared.ProviderLinear, Identifier: "ENG-7", Scope: "ENG", State: "started"},
					}},
				},
			)

			rows, _, err := collector.Collect(context.Background(), shared.ProjectFilter{}, false)
			if err != nil {
				t.Fatalf("Collect() error = %v", err)
			}

			want := map[string]string{"#1": "Repo project", "ENG-7": "Team project"}
			if diff := cmp.Diff(want, projectNames(rows)); diff != "" {
				t.Errorf("issue → project mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("the filter reaches Notion and each provider gets its own value", func(t *testing.T) {
			n := &fakeNotion{}
			g := &fakeProvider{name: shared.ProviderGithub}
			l := &fakeProvider{name: shared.ProviderLinear}
			collector := NewCollector(n, []sync.Provider{g, l})

			filter := shared.ProjectFilter{Repo: "owner/repo", TeamKey: "ENG"}
			if _, _, err := collector.Collect(context.Background(), filter, false); err != nil {
				t.Fatalf("Collect() error = %v", err)
			}

			if diff := cmp.Diff(filter, n.gotFilter); diff != "" {
				t.Errorf("ListProjects filter mismatch (-want +got):\n%s", diff)
			}
			if g.gotFilter != "owner/repo" {
				t.Errorf("GitHub FetchAssignedIssues filter = %q, want %q", g.gotFilter, "owner/repo")
			}
			if l.gotFilter != "ENG" {
				t.Errorf("Linear FetchAssignedIssues filter = %q, want %q", l.gotFilter, "ENG")
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		cases := []struct {
			name    string
			project shared.Project
			issue   shared.Issue
		}{
			{
				name:    "a project of another provider with the same scope does not match",
				project: shared.Project{PageID: "p-lin", Name: "Team project", TeamKey: "owner/repo", Provider: shared.ProviderLinear},
				issue:   shared.Issue{Provider: shared.ProviderGithub, Identifier: "#1", Scope: "owner/repo"},
			},
			{
				name:    "a project with an empty provider never matches",
				project: shared.Project{PageID: "p-empty", Name: "No provider", Repo: "owner/repo"},
				issue:   shared.Issue{Provider: shared.ProviderGithub, Identifier: "#1", Scope: "owner/repo"},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				collector := NewCollector(
					&fakeNotion{projects: []shared.Project{tc.project}},
					[]sync.Provider{&fakeProvider{name: tc.issue.Provider, issues: []shared.Issue{tc.issue}}},
				)

				rows, _, err := collector.Collect(context.Background(), shared.ProjectFilter{}, false)
				if err != nil {
					t.Fatalf("Collect() error = %v", err)
				}

				want := map[string]string{tc.issue.Identifier: ""}
				if diff := cmp.Diff(want, projectNames(rows)); diff != "" {
					t.Errorf("issue → project mismatch, want no project (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("error path", func(t *testing.T) {
		wantErr := errors.New("boom")
		project := shared.Project{PageID: "p1", Name: "Repo project", Repo: "owner/repo", Provider: shared.ProviderGithub}
		issue := shared.Issue{Provider: shared.ProviderGithub, Identifier: "#1", Scope: "owner/repo"}

		cases := []struct {
			name     string
			notion   *fakeNotion
			provider *fakeProvider
		}{
			{
				name:     "a ListProjects error is returned",
				notion:   &fakeNotion{projectsErr: wantErr},
				provider: &fakeProvider{name: shared.ProviderGithub},
			},
			{
				name:     "a fetch error from the only provider is returned",
				notion:   &fakeNotion{projects: []shared.Project{project}},
				provider: &fakeProvider{name: shared.ProviderGithub, err: wantErr},
			},
			{
				name:     "a FindStoryByIssue error is returned",
				notion:   &fakeNotion{projects: []shared.Project{project}, storyErr: wantErr},
				provider: &fakeProvider{name: shared.ProviderGithub, issues: []shared.Issue{issue}},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				collector := NewCollector(tc.notion, []sync.Provider{tc.provider})

				rows, _, err := collector.Collect(context.Background(), shared.ProjectFilter{}, false)
				if !errors.Is(err, wantErr) {
					t.Fatalf("Collect() error = %v, want %v", err, wantErr)
				}
				if rows != nil {
					t.Errorf("Collect() rows = %+v, want nil", rows)
				}
			})
		}
	})
}
