package sync

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
)

// upsertCall records the arguments a fakeNotion received on UpsertStory.
type upsertCall struct {
	storyInput notion.StoryInput
	issue      github.Issue
	isDryRun   bool
}

type fakeNotion struct {
	projects    []notion.Project
	projectsErr error
	// upsert returns the result for a given issue. It is only invoked for
	// issues whose repo matches a project.
	upsert func(issue github.Issue) (*notion.UpsertResult, error)
	calls  []upsertCall
}

func (f *fakeNotion) ListProjects(ctx context.Context) ([]notion.Project, error) {
	return f.projects, f.projectsErr
}

func (f *fakeNotion) UpsertStory(ctx context.Context, storyInput notion.StoryInput, issue github.Issue, isDryRun bool) (*notion.UpsertResult, error) {
	f.calls = append(f.calls, upsertCall{storyInput: storyInput, issue: issue, isDryRun: isDryRun})
	return f.upsert(issue)
}

type fakeGithub struct {
	issues []github.Issue
	err    error
}

func (f *fakeGithub) FetchAssignedIssues(ctx context.Context) ([]github.Issue, error) {
	return f.issues, f.err
}

func created() *notion.UpsertResult   { return &notion.UpsertResult{Created: true} }
func updated() *notion.UpsertResult   { return &notion.UpsertResult{Updated: true} }
func unchanged() *notion.UpsertResult { return &notion.UpsertResult{Unchanged: true} }

// quietLogs silences slog output for the duration of a test so the Run logs
// don't clutter the test output, restoring the previous default afterwards.
func quietLogs(t *testing.T) {
	t.Helper()
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
}

func TestEngineRun(t *testing.T) {
	quietLogs(t)

	cases := []struct {
		name     string
		projects []notion.Project
		issues   []github.Issue
		upsert   func(issue github.Issue) (*notion.UpsertResult, error)
		want     *Report
	}{
		{
			name: "counts created, updated and unchanged results",
			projects: []notion.Project{
				{PageID: "p1", Repo: "owner/repo-a"},
				{PageID: "p2", Repo: "owner/repo-b"},
			},
			issues: []github.Issue{
				{Number: 1, Repo: "owner/repo-a"},
				{Number: 2, Repo: "owner/repo-a"},
				{Number: 3, Repo: "owner/repo-b"},
			},
			upsert: func(issue github.Issue) (*notion.UpsertResult, error) {
				switch issue.Number {
				case 1:
					return created(), nil
				case 2:
					return updated(), nil
				default:
					return unchanged(), nil
				}
			},
			want: &Report{Created: 1, Updated: 1, Unchanged: 1},
		},
		{
			name:     "skips issues without a matching project",
			projects: []notion.Project{{PageID: "p1", Repo: "owner/known"}},
			issues: []github.Issue{
				{Number: 1, Repo: "owner/known"},
				{Number: 2, Repo: "owner/unknown"},
				{Number: 3, Repo: "owner/also-unknown"},
			},
			upsert: func(issue github.Issue) (*notion.UpsertResult, error) {
				return created(), nil
			},
			want: &Report{Created: 1, Skipped: 2},
		},
		{
			name:     "records upsert errors and keeps processing",
			projects: []notion.Project{{PageID: "p1", Repo: "owner/repo"}},
			issues: []github.Issue{
				{Number: 1, Repo: "owner/repo"},
				{Number: 2, Repo: "owner/repo"},
				{Number: 3, Repo: "owner/repo"},
			},
			upsert: func(issue github.Issue) (*notion.UpsertResult, error) {
				if issue.Number == 2 {
					return nil, errors.New("boom")
				}
				return created(), nil
			},
			want: &Report{
				Created: 2,
				Errors:  []ReportError{{IssueNumber: 2, Error: "boom"}},
			},
		},
		{
			name:     "no issues produces an empty report",
			projects: []notion.Project{{PageID: "p1", Repo: "owner/repo"}},
			issues:   nil,
			upsert: func(issue github.Issue) (*notion.UpsertResult, error) {
				t.Fatalf("UpsertStory should not be called when there are no issues")
				return nil, nil
			},
			want: &Report{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := &Engine{
				NotionClient: &fakeNotion{projects: tc.projects, upsert: tc.upsert},
				GithubClient: &fakeGithub{issues: tc.issues},
			}

			got, err := engine.Run(context.Background(), EngineRunOptions{})
			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEngineRunListProjectsError(t *testing.T) {
	quietLogs(t)

	wantErr := errors.New("list projects failed")
	g := &fakeGithub{}
	engine := &Engine{
		NotionClient: &fakeNotion{projectsErr: wantErr},
		GithubClient: g,
	}

	report, err := engine.Run(context.Background(), EngineRunOptions{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	if report != nil {
		t.Errorf("Run() report = %+v, want nil", report)
	}
	if g.issues != nil {
		// FetchAssignedIssues must not be reached once listing projects fails.
		t.Errorf("FetchAssignedIssues should not run after a ListProjects error")
	}
}

func TestEngineRunFetchIssuesError(t *testing.T) {
	quietLogs(t)

	wantErr := errors.New("fetch issues failed")
	engine := &Engine{
		NotionClient: &fakeNotion{projects: []notion.Project{{PageID: "p1", Repo: "owner/repo"}}},
		GithubClient: &fakeGithub{err: wantErr},
	}

	report, err := engine.Run(context.Background(), EngineRunOptions{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	if report != nil {
		t.Errorf("Run() report = %+v, want nil", report)
	}
}

func TestEngineRunBuildsStoryInputAndPassesDryRun(t *testing.T) {
	quietLogs(t)

	updatedAt := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)
	issue := github.Issue{
		Number:    42,
		Title:     "Add login flow",
		URL:       "https://github.com/owner/repo/issues/42",
		State:     "open",
		Labels:    []string{"feature"},
		Repo:      "owner/repo",
		UpdatedAt: updatedAt,
	}

	for _, dryRun := range []bool{true, false} {
		n := &fakeNotion{
			projects: []notion.Project{{PageID: "page-1", Repo: "owner/repo"}},
			upsert: func(issue github.Issue) (*notion.UpsertResult, error) {
				return created(), nil
			},
		}
		engine := &Engine{NotionClient: n, GithubClient: &fakeGithub{issues: []github.Issue{issue}}}

		if _, err := engine.Run(context.Background(), EngineRunOptions{DryRun: dryRun}); err != nil {
			t.Fatalf("Run() unexpected error: %v", err)
		}

		if len(n.calls) != 1 {
			t.Fatalf("UpsertStory called %d times, want 1", len(n.calls))
		}
		call := n.calls[0]

		if call.isDryRun != dryRun {
			t.Errorf("UpsertStory isDryRun = %v, want %v", call.isDryRun, dryRun)
		}
		if diff := cmp.Diff(issue, call.issue); diff != "" {
			t.Errorf("UpsertStory issue mismatch (-want +got):\n%s", diff)
		}
		wantInput := IssueToStoryInput(issue, "page-1")
		if diff := cmp.Diff(wantInput, call.storyInput); diff != "" {
			t.Errorf("UpsertStory storyInput mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestEngineRunMatchesRepoCaseInsensitively(t *testing.T) {
	quietLogs(t)

	n := &fakeNotion{
		projects: []notion.Project{{PageID: "p1", Repo: "Owner/Repo-A"}},
		upsert: func(issue github.Issue) (*notion.UpsertResult, error) {
			return created(), nil
		},
	}
	engine := &Engine{
		NotionClient: n,
		GithubClient: &fakeGithub{issues: []github.Issue{{Number: 1, Repo: "owner/repo-a"}}},
	}

	got, err := engine.Run(context.Background(), EngineRunOptions{})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	want := &Report{Created: 1}
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
	}
}
