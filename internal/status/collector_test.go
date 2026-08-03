package status

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/nox456/forgesync/internal/shared"
)

type fakeNotion struct {
	projects []shared.Project
	story    *shared.Story
}

func (f *fakeNotion) ListProjects(ctx context.Context, repoName string) ([]shared.Project, error) {
	return f.projects, nil
}

func (f *fakeNotion) FindStoryByIssue(ctx context.Context, issue shared.Issue, projectId string) (*shared.Story, error) {
	return f.story, nil
}

type fakeGithub struct {
	issues []shared.Issue
}

func (f *fakeGithub) FetchAssignedIssues(ctx context.Context, repoName string) ([]shared.Issue, error) {
	return f.issues, nil
}

// quietLogs swallows the collector's slog output so the informational lines
// don't clutter the test output, restoring the previous default afterwards.
func quietLogs(t *testing.T) {
	t.Helper()
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
}

// The Notion repo field is typed by hand, so its casing rarely matches the one
// GitHub reports. Matching must be case-insensitive, the same way the sync
// engine does it — otherwise `status` shows [NO PROJECT] for issues `sync`
// syncs without complaint.
func TestCollectMatchesProjectRepoIgnoringCase(t *testing.T) {
	quietLogs(t)

	collector := &Collector{
		NotionClient: &fakeNotion{
			projects: []shared.Project{
				{PageID: "page-1", Name: "ForgeSync", Repo: "Nox456/ForgeSync"},
			},
		},
		GithubClient: &fakeGithub{
			issues: []shared.Issue{
				{Number: 42, Title: "Sync statuses", Repo: "nox456/forgesync", State: "open"},
			},
		},
	}

	rows, err := collector.Collect(context.Background(), "")
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
