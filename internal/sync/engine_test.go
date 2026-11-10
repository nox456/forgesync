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
	"github.com/nox456/forgesync/internal/shared"
)

// upsertCall records the arguments a fakeNotion received on UpsertStory.
type upsertCall struct {
	storyInput    shared.StoryInput
	isDryRun      bool
	existingStory *shared.Story
}

type fakeNotion struct {
	projects    []shared.Project
	projectsErr error
	// stories holds the stories each project page returns, keyed by project
	// page id.
	stories    map[string][]shared.Story
	storiesErr error
	// upsert returns the result for a given story input. It is only invoked
	// for issues whose provider and scope match an eligible project.
	upsert func(storyInput shared.StoryInput) (*shared.UpsertResult, error)
	// findStory returns the existing story for an issue. When nil, no story
	// is found (nil, nil).
	findStory func(issue shared.Issue) (*shared.Story, error)
	calls     []upsertCall
	// gotFilter records the filter passed to ListProjects.
	gotFilter shared.ProjectFilter
	// storiesRequested records every project page id ListStoriesByProject
	// was called with.
	storiesRequested []string
}

func (f *fakeNotion) ListProjects(ctx context.Context, filter shared.ProjectFilter) ([]shared.Project, error) {
	f.gotFilter = filter
	return f.projects, f.projectsErr
}

func (f *fakeNotion) FindStoryByIssue(ctx context.Context, issue shared.Issue, projectId string) (*shared.Story, error) {
	if f.findStory == nil {
		return nil, nil
	}
	return f.findStory(issue)
}

func (f *fakeNotion) ListStoriesByProject(ctx context.Context, projectId string) ([]shared.Story, error) {
	f.storiesRequested = append(f.storiesRequested, projectId)
	if f.storiesErr != nil {
		return nil, f.storiesErr
	}
	return f.stories[projectId], nil
}

func (f *fakeNotion) UpsertStory(ctx context.Context, storyInput shared.StoryInput, isDryRun bool, existingStory *shared.Story) (*shared.UpsertResult, error) {
	f.calls = append(f.calls, upsertCall{storyInput: storyInput, isDryRun: isDryRun, existingStory: existingStory})
	return f.upsert(storyInput)
}

// fakeProvider is a sync.Provider whose name and issues are set by the test.
type fakeProvider struct {
	name   shared.Provider
	issues []shared.Issue
	err    error
	// gotFilter records the filter passed to FetchAssignedIssues; fetched
	// tells whether it was called at all.
	gotFilter string
	fetched   bool
}

func (f *fakeProvider) Name() shared.Provider { return f.name }

func (f *fakeProvider) FetchAssignedIssues(ctx context.Context, filter string) ([]shared.Issue, error) {
	f.fetched = true
	f.gotFilter = filter
	return f.issues, f.err
}

func fakeGithub(issues ...shared.Issue) *fakeProvider {
	return &fakeProvider{name: shared.ProviderGithub, issues: issues}
}

func fakeLinear(issues ...shared.Issue) *fakeProvider {
	return &fakeProvider{name: shared.ProviderLinear, issues: issues}
}

// githubIssue builds the issue the GitHub adapter returns for an identifier
// and repo.
func githubIssue(identifier, repo string) shared.Issue {
	return shared.Issue{Provider: shared.ProviderGithub, Identifier: identifier, Scope: repo}
}

// githubProject builds a Notion project bound to a GitHub repo.
func githubProject(pageID, repo string) shared.Project {
	return shared.Project{PageID: pageID, Name: repo, Repo: repo, Provider: shared.ProviderGithub}
}

// linearIssue builds the issue the Linear adapter returns for an identifier,
// team key and workflow state type.
func linearIssue(identifier, team, state string) shared.Issue {
	return shared.Issue{Provider: shared.ProviderLinear, Identifier: identifier, Scope: team, State: state}
}

// linearProject builds a Notion project bound to a Linear team key.
func linearProject(pageID, teamKey string) shared.Project {
	return shared.Project{PageID: pageID, Name: teamKey, TeamKey: teamKey, Provider: shared.ProviderLinear}
}

// storyFor builds the Notion story a project returns for a given issue
// identifier.
func storyFor(identifier string) shared.Story {
	return shared.Story{
		PageID: "story-" + identifier,
		Issue:  identifier,
	}
}

// syncedStoryFor builds a story that already matches its issue in every field
// IsSynced compares, so the engine reports it as Unchanged without upserting.
func syncedStoryFor(issue shared.Issue) shared.Story {
	return shared.Story{
		PageID:       "story-" + issue.Identifier,
		Issue:        issue.Identifier,
		Name:         issue.Title,
		Status:       ComputeStatus(issue, ""),
		Labels:       issue.Labels,
		LastWorkedAt: issue.UpdatedAt.Format(storyDateLayout),
	}
}

func created() *shared.UpsertResult { return &shared.UpsertResult{Created: true} }
func updated() *shared.UpsertResult { return &shared.UpsertResult{Updated: true} }

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

	// syncedIssue's story already matches it, so the engine must count it as
	// Unchanged on its own rather than asking Notion to work that out.
	syncedIssue := shared.Issue{
		Provider:   shared.ProviderGithub,
		Identifier: "#3",
		Title:      "Already in sync",
		Scope:      "owner/repo-b",
		State:      "open",
		UpdatedAt:  time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC),
	}

	cases := []struct {
		name     string
		projects []shared.Project
		stories  map[string][]shared.Story
		issues   []shared.Issue
		upsert   func(storyInput shared.StoryInput) (*shared.UpsertResult, error)
		want     *Report
	}{
		{
			name: "counts created, updated and unchanged results",
			projects: []shared.Project{
				githubProject("p1", "owner/repo-a"),
				githubProject("p2", "owner/repo-b"),
			},
			stories: map[string][]shared.Story{
				"p1": {storyFor("#1"), storyFor("#2")},
				"p2": {syncedStoryFor(syncedIssue)},
			},
			issues: []shared.Issue{
				githubIssue("#1", "owner/repo-a"),
				githubIssue("#2", "owner/repo-a"),
				syncedIssue,
			},
			upsert: func(storyInput shared.StoryInput) (*shared.UpsertResult, error) {
				switch storyInput.Issue {
				case "#1":
					return created(), nil
				case "#2":
					return updated(), nil
				default:
					t.Fatalf("UpsertStory called for issue %s, which is already in sync", storyInput.Issue)
					return nil, nil
				}
			},
			want: &Report{Created: 1, Updated: 1, Unchanged: 1},
		},
		{
			name:     "skips issues without a matching project",
			projects: []shared.Project{githubProject("p1", "owner/known")},
			stories:  map[string][]shared.Story{"p1": {storyFor("#1")}},
			issues: []shared.Issue{
				githubIssue("#1", "owner/known"),
				githubIssue("#2", "owner/unknown"),
				githubIssue("#3", "owner/also-unknown"),
			},
			upsert: func(storyInput shared.StoryInput) (*shared.UpsertResult, error) {
				return created(), nil
			},
			want: &Report{Created: 1, Skipped: 2},
		},
		{
			name:     "no issues produces an empty report",
			projects: []shared.Project{githubProject("p1", "owner/repo")},
			issues:   nil,
			upsert: func(storyInput shared.StoryInput) (*shared.UpsertResult, error) {
				t.Fatalf("UpsertStory should not be called when there are no issues")
				return nil, nil
			},
			want: &Report{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine := NewEngine(
				&fakeNotion{projects: tc.projects, stories: tc.stories, upsert: tc.upsert},
				[]Provider{fakeGithub(tc.issues...)},
			)

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

// TestEngineRunClassifiesProjects pins step 2 of Run: a project without a
// usable provider is reported and skipped before its stories are loaded, and
// the run carries on for every other project.
func TestEngineRunClassifiesProjects(t *testing.T) {
	quietLogs(t)

	valid := githubProject("p-ok", "owner/repo")

	t.Run("happy path", func(t *testing.T) {
		t.Run("a valid project syncs next to a misconfigured one", func(t *testing.T) {
			n := &fakeNotion{
				projects: []shared.Project{
					{PageID: "p-empty", Name: "No provider", Repo: "owner/repo"},
					valid,
				},
				stories: map[string][]shared.Story{"p-ok": {storyFor("#1")}},
				upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
					return updated(), nil
				},
			}
			engine := NewEngine(n, []Provider{fakeGithub(githubIssue("#1", "owner/repo"))})

			got, err := engine.Run(context.Background(), EngineRunOptions{})
			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			want := &Report{
				Updated: 1,
				Errors:  []ReportError{{Project: "No provider", Error: "project has no provider"}},
			}
			if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		cases := []struct {
			name       string
			project    shared.Project
			wantErrors []ReportError
		}{
			{
				name:       "an empty provider is reported and skipped",
				project:    shared.Project{PageID: "p-bad", Name: "Empty", Repo: "owner/other"},
				wantErrors: []ReportError{{Project: "Empty", Error: "project has no provider"}},
			},
			{
				name:       "an unknown provider is reported with its quoted value",
				project:    shared.Project{PageID: "p-bad", Name: "Typo", Repo: "owner/other", Provider: "gitlab"},
				wantErrors: []ReportError{{Project: "Typo", Error: `unknown provider "gitlab"`}},
			},
			{
				name:       "a provider the engine was not given is reported as not configured",
				project:    shared.Project{PageID: "p-bad", Name: "Linear team", TeamKey: "ENG", Provider: shared.ProviderLinear},
				wantErrors: []ReportError{{Project: "Linear team", Provider: shared.ProviderLinear, Error: "linear token not configured"}},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				n := &fakeNotion{
					projects: []shared.Project{tc.project, valid},
					stories:  map[string][]shared.Story{"p-ok": {storyFor("#1")}},
					upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
						return updated(), nil
					},
				}
				engine := NewEngine(n, []Provider{fakeGithub(githubIssue("#1", "owner/repo"))})

				got, err := engine.Run(context.Background(), EngineRunOptions{})
				if err != nil {
					t.Fatalf("Run() unexpected error: %v", err)
				}

				if diff := cmp.Diff(tc.wantErrors, got.Errors, cmpopts.EquateEmpty()); diff != "" {
					t.Errorf("Run() errors mismatch (-want +got):\n%s", diff)
				}
				if diff := cmp.Diff([]string{"p-ok"}, n.storiesRequested); diff != "" {
					t.Errorf("stories loaded for (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("error path", func(t *testing.T) {
		t.Run("every project misconfigured still returns a report, not an error", func(t *testing.T) {
			n := &fakeNotion{
				projects: []shared.Project{
					{PageID: "p1", Name: "A"},
					{PageID: "p2", Name: "B", Provider: "jira"},
				},
				upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
					t.Fatalf("UpsertStory should not be called without an eligible project")
					return nil, nil
				},
			}
			engine := NewEngine(n, []Provider{fakeGithub(githubIssue("#1", "owner/repo"))})

			got, err := engine.Run(context.Background(), EngineRunOptions{})
			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			want := &Report{
				Skipped: 1,
				Errors: []ReportError{
					{Project: "A", Error: "project has no provider"},
					{Project: "B", Error: `unknown provider "jira"`},
				},
			}
			if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
			}
		})
	})
}

// TestEngineRunMatchesEveryStoryInAProject guards against the story cache being
// rebuilt on every match, which used to leave only the last matched issue in the
// map and made every other issue look storyless (and therefore skipped).
func TestEngineRunMatchesEveryStoryInAProject(t *testing.T) {
	quietLogs(t)

	n := &fakeNotion{
		projects: []shared.Project{githubProject("p1", "owner/repo")},
		stories:  map[string][]shared.Story{"p1": {storyFor("#15"), storyFor("#16")}},
		upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
			return updated(), nil
		},
	}
	engine := NewEngine(n, []Provider{fakeGithub(
		githubIssue("#15", "owner/repo"),
		githubIssue("#16", "owner/repo"),
	)})

	got, err := engine.Run(context.Background(), EngineRunOptions{})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	want := &Report{Updated: 2}
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
	}

	syncedStories := make([]string, 0, len(n.calls))
	for _, call := range n.calls {
		syncedStories = append(syncedStories, call.storyInput.Issue)
	}
	if diff := cmp.Diff([]string{"#15", "#16"}, syncedStories); diff != "" {
		t.Errorf("synced issues mismatch (-want +got):\n%s", diff)
	}
}

// TestEngineRunPassesNilStoryForNewIssues guards the create path: an issue with
// no story in the project must reach UpsertStory with a nil existingStory. A
// pointer to the zero value is not nil, so it would send the issue down the
// update path against an empty page id.
func TestEngineRunPassesNilStoryForNewIssues(t *testing.T) {
	quietLogs(t)

	existing := storyFor("#1")

	n := &fakeNotion{
		projects: []shared.Project{githubProject("p1", "owner/repo")},
		stories:  map[string][]shared.Story{"p1": {existing}},
		upsert: func(storyInput shared.StoryInput) (*shared.UpsertResult, error) {
			if storyInput.Issue == "#1" {
				return updated(), nil
			}
			return created(), nil
		},
	}
	engine := NewEngine(n, []Provider{fakeGithub(
		githubIssue("#1", "owner/repo"), // has a story
		githubIssue("#2", "owner/repo"), // brand new
	)})

	got, err := engine.Run(context.Background(), EngineRunOptions{})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	want := &Report{Updated: 1, Created: 1}
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
	}

	if len(n.calls) != 2 {
		t.Fatalf("got %d upsert calls, want 2", len(n.calls))
	}
	if diff := cmp.Diff(&existing, n.calls[0].existingStory); diff != "" {
		t.Errorf("issue #1 existingStory mismatch (-want +got):\n%s", diff)
	}
	if n.calls[1].existingStory != nil {
		t.Errorf("issue #2 existingStory = %+v, want nil", n.calls[1].existingStory)
	}
}

// A story whose Issue holds the bare number (a workspace that skipped the
// Phase 0 rewrite) must not match the "#N" identifier the adapter produces.
func TestEngineRunMatchesStoriesByIdentifierOnly(t *testing.T) {
	quietLogs(t)

	n := &fakeNotion{
		projects: []shared.Project{githubProject("p1", "owner/repo")},
		stories:  map[string][]shared.Story{"p1": {{PageID: "story-legacy", Issue: "12"}}},
		upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
			return created(), nil
		},
	}
	engine := NewEngine(n, []Provider{fakeGithub(githubIssue("#12", "owner/repo"))})

	if _, err := engine.Run(context.Background(), EngineRunOptions{}); err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	if len(n.calls) != 1 {
		t.Fatalf("got %d upsert calls, want 1", len(n.calls))
	}
	if n.calls[0].existingStory != nil {
		t.Errorf("existingStory = %+v, want nil: %q must not match %q", n.calls[0].existingStory, "12", "#12")
	}
}

// TestEngineRunReportsDuplicateStories verifies that two stories pointing at the
// same issue are reported once, with the project and provider they belong to,
// and the first one still syncs.
func TestEngineRunReportsDuplicateStories(t *testing.T) {
	quietLogs(t)

	first := shared.Story{PageID: "story-a", Issue: "#16"}
	second := shared.Story{PageID: "story-b", Issue: "#16"}

	n := &fakeNotion{
		projects: []shared.Project{githubProject("p1", "owner/repo")},
		stories:  map[string][]shared.Story{"p1": {first, second}},
		upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
			return updated(), nil
		},
	}
	engine := NewEngine(n, []Provider{fakeGithub(githubIssue("#16", "owner/repo"))})

	got, err := engine.Run(context.Background(), EngineRunOptions{})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	want := &Report{
		Updated: 1,
		Errors: []ReportError{{
			Identifier: "#16",
			Provider:   shared.ProviderGithub,
			Project:    "owner/repo",
			Error:      "found more than one story for issue",
		}},
	}
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
	}
	if len(n.calls) != 1 {
		t.Fatalf("got %d upsert calls, want 1", len(n.calls))
	}
	if diff := cmp.Diff(&first, n.calls[0].existingStory); diff != "" {
		t.Errorf("existingStory mismatch, want the first story (-want +got):\n%s", diff)
	}
}

// Stories with an empty Issue are hand-written and linked to no issue, so they
// are neither reported as duplicates of each other nor matched to an issue.
func TestEngineRunIgnoresStoriesWithoutIssue(t *testing.T) {
	quietLogs(t)

	n := &fakeNotion{
		projects: []shared.Project{githubProject("p1", "owner/repo")},
		stories: map[string][]shared.Story{"p1": {
			{PageID: "manual-a", Issue: ""},
			{PageID: "manual-b", Issue: ""},
			{PageID: "manual-c", Issue: "   "},
			{PageID: "manual-d", Issue: "   "},
		}},
		upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
			return created(), nil
		},
	}
	engine := NewEngine(n, []Provider{fakeGithub(githubIssue("#1", "owner/repo"))})

	got, err := engine.Run(context.Background(), EngineRunOptions{})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	want := &Report{Created: 1}
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
	}
	if len(n.calls) != 1 {
		t.Fatalf("got %d upsert calls, want 1", len(n.calls))
	}
	if n.calls[0].existingStory != nil {
		t.Errorf("existingStory = %+v, want nil: an unlinked story must not match an issue", n.calls[0].existingStory)
	}
}

// An upsert error is recorded with the issue's Identifier and Provider (plan
// steps 2.3 and 2.4, sync loop) and the remaining issues still sync.
func TestEngineRunRecordsUpsertErrorsAndKeepsProcessing(t *testing.T) {
	quietLogs(t)

	n := &fakeNotion{
		projects: []shared.Project{githubProject("p1", "owner/repo")},
		stories:  map[string][]shared.Story{"p1": {storyFor("#1"), storyFor("#2"), storyFor("#3")}},
		upsert: func(storyInput shared.StoryInput) (*shared.UpsertResult, error) {
			if storyInput.Issue == "#2" {
				return nil, errors.New("boom")
			}
			return created(), nil
		},
	}
	engine := NewEngine(n, []Provider{fakeGithub(
		githubIssue("#1", "owner/repo"),
		githubIssue("#2", "owner/repo"),
		githubIssue("#3", "owner/repo"),
	)})

	got, err := engine.Run(context.Background(), EngineRunOptions{})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	want := &Report{
		Created: 2,
		Errors:  []ReportError{{Identifier: "#2", Provider: shared.ProviderGithub, Error: "boom"}},
	}
	// Project is left out of the comparison: the plan names Identifier and
	// Provider for this error, and Project is optional here.
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty(), cmpopts.IgnoreFields(ReportError{}, "Project")); diff != "" {
		t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
	}
}

// TestEngineRunSkipsUpsertForSyncedStories pins the decision that used to live
// inside the Notion adapter: an in-sync Story is counted as Unchanged by the
// engine itself, without a round trip to Notion, dry run or not.
func TestEngineRunSkipsUpsertForSyncedStories(t *testing.T) {
	quietLogs(t)

	issue := shared.Issue{
		Provider:   shared.ProviderGithub,
		Identifier: "#7",
		Title:      "Nothing to do",
		Scope:      "owner/repo",
		State:      "open",
		Labels:     []string{"chore"},
		UpdatedAt:  time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC),
	}

	for _, dryRun := range []bool{true, false} {
		n := &fakeNotion{
			projects: []shared.Project{githubProject("p1", "owner/repo")},
			stories:  map[string][]shared.Story{"p1": {syncedStoryFor(issue)}},
			upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
				t.Fatalf("UpsertStory called for an already-synced issue (dryRun=%v)", dryRun)
				return nil, nil
			},
		}
		engine := NewEngine(n, []Provider{fakeGithub(issue)})

		got, err := engine.Run(context.Background(), EngineRunOptions{DryRun: dryRun})
		if err != nil {
			t.Fatalf("Run() unexpected error: %v", err)
		}

		want := &Report{Unchanged: 1}
		if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Run(dryRun=%v) report mismatch (-want +got):\n%s", dryRun, diff)
		}
		if len(n.calls) != 0 {
			t.Errorf("UpsertStory called %d times, want 0", len(n.calls))
		}
	}
}

func TestEngineRunListProjectsError(t *testing.T) {
	quietLogs(t)

	wantErr := errors.New("list projects failed")
	g := fakeGithub()
	engine := NewEngine(&fakeNotion{projectsErr: wantErr}, []Provider{g})

	report, err := engine.Run(context.Background(), EngineRunOptions{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	if report != nil {
		t.Errorf("Run() report = %+v, want nil", report)
	}
	if g.fetched {
		t.Errorf("FetchAssignedIssues should not run after a ListProjects error")
	}
}

func TestEngineRunListStoriesError(t *testing.T) {
	quietLogs(t)

	wantErr := errors.New("list stories failed")
	engine := NewEngine(
		&fakeNotion{projects: []shared.Project{githubProject("p1", "owner/repo")}, storiesErr: wantErr},
		[]Provider{fakeGithub(githubIssue("#1", "owner/repo"))},
	)

	report, err := engine.Run(context.Background(), EngineRunOptions{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}
	if report != nil {
		t.Errorf("Run() report = %+v, want nil", report)
	}
}

func TestEngineRunFetchIssuesError(t *testing.T) {
	quietLogs(t)

	wantErr := errors.New("fetch issues failed")
	failing := fakeGithub()
	failing.err = wantErr
	engine := NewEngine(
		&fakeNotion{projects: []shared.Project{githubProject("p1", "owner/repo")}},
		[]Provider{failing},
	)

	report, err := engine.Run(context.Background(), EngineRunOptions{})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	want := &Report{Errors: []ReportError{{Provider: shared.ProviderGithub, Error: wantErr.Error()}}}
	if diff := cmp.Diff(want, report, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
	}
}

func TestEngineRunBuildsStoryInputAndPassesDryRun(t *testing.T) {
	quietLogs(t)

	updatedAt := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)
	issue := shared.Issue{
		Provider:   shared.ProviderGithub,
		Identifier: "#42",
		Title:      "Add login flow",
		URL:        "https://github.com/owner/repo/issues/42",
		State:      "open",
		Labels:     []string{"feature"},
		Scope:      "owner/repo",
		UpdatedAt:  updatedAt,
	}

	existingStory := storyFor(issue.Identifier)

	for _, dryRun := range []bool{true, false} {
		n := &fakeNotion{
			projects: []shared.Project{githubProject("page-1", "owner/repo")},
			stories:  map[string][]shared.Story{"page-1": {existingStory}},
			upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
				return created(), nil
			},
		}
		engine := NewEngine(n, []Provider{fakeGithub(issue)})

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
		wantInput := IssueToStoryInput(issue, &existingStory, "page-1")
		if diff := cmp.Diff(wantInput, call.storyInput); diff != "" {
			t.Errorf("UpsertStory storyInput mismatch (-want +got):\n%s", diff)
		}
		if call.storyInput.Issue != "#42" {
			t.Errorf("UpsertStory storyInput.Issue = %q, want %q", call.storyInput.Issue, "#42")
		}
	}
}

func TestEngineRunMatchesRepoCaseInsensitively(t *testing.T) {
	quietLogs(t)

	n := &fakeNotion{
		projects: []shared.Project{githubProject("p1", "Owner/Repo-A")},
		stories:  map[string][]shared.Story{"p1": {storyFor("#1")}},
		upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
			return created(), nil
		},
	}
	engine := NewEngine(n, []Provider{fakeGithub(githubIssue("#1", "owner/repo-a"))})

	got, err := engine.Run(context.Background(), EngineRunOptions{})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	want := &Report{Created: 1}
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
	}
}

// The project lookup is keyed on provider + scope, so a GitHub repo and a
// Linear team key that happen to share a value never match each other's issues.
func TestEngineRunKeysProjectsOnProviderAndScope(t *testing.T) {
	quietLogs(t)

	n := &fakeNotion{
		projects: []shared.Project{
			{PageID: "p-gh", Name: "GitHub side", Repo: "shared", Provider: shared.ProviderGithub},
			{PageID: "p-lin", Name: "Linear side", TeamKey: "SHARED", Provider: shared.ProviderLinear},
		},
		upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
			return created(), nil
		},
	}
	engine := NewEngine(n, []Provider{
		fakeGithub(githubIssue("#1", "shared")),
		fakeLinear(shared.Issue{Provider: shared.ProviderLinear, Identifier: "SHARED-1", Scope: "SHARED"}),
	})

	got, err := engine.Run(context.Background(), EngineRunOptions{})
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	want := &Report{Created: 2}
	if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
	}

	gotProjects := map[string]string{}
	for _, call := range n.calls {
		gotProjects[call.storyInput.Issue] = call.storyInput.Project
	}
	wantProjects := map[string]string{"#1": "p-gh", "SHARED-1": "p-lin"}
	if diff := cmp.Diff(wantProjects, gotProjects); diff != "" {
		t.Errorf("issue → project page mismatch (-want +got):\n%s", diff)
	}
}

// TestEngineRunForwardsFilters verifies that each filter option reaches
// ListProjects in the filter's own field, and that every provider is fetched
// with the value it understands: the repo for GitHub, the team key for Linear.
// An omitted flag forwards as an empty string, which the clients treat as
// "no filter".
func TestEngineRunForwardsFilters(t *testing.T) {
	quietLogs(t)

	cases := []struct {
		name          string
		options       EngineRunOptions
		wantFilter    shared.ProjectFilter
		wantGithubArg string
		wantLinearArg string
	}{
		{
			name:          "a repo filter goes to Notion and GitHub",
			options:       EngineRunOptions{Repo: "owner/repo-a"},
			wantFilter:    shared.ProjectFilter{Repo: "owner/repo-a"},
			wantGithubArg: "owner/repo-a",
			wantLinearArg: "",
		},
		{
			name:          "a team filter goes to Notion and Linear",
			options:       EngineRunOptions{Team: "ENG"},
			wantFilter:    shared.ProjectFilter{TeamKey: "ENG"},
			wantGithubArg: "",
			wantLinearArg: "ENG",
		},
		{
			name:          "no flag forwards empty filters",
			options:       EngineRunOptions{},
			wantFilter:    shared.ProjectFilter{},
			wantGithubArg: "",
			wantLinearArg: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := &fakeNotion{
				projects: []shared.Project{githubProject("p1", "owner/repo-a")},
				upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
					return created(), nil
				},
			}
			g := fakeGithub()
			l := fakeLinear()
			engine := NewEngine(n, []Provider{g, l})

			if _, err := engine.Run(context.Background(), tc.options); err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.wantFilter, n.gotFilter); diff != "" {
				t.Errorf("ListProjects filter mismatch (-want +got):\n%s", diff)
			}
			if g.gotFilter != tc.wantGithubArg {
				t.Errorf("GitHub FetchAssignedIssues filter = %q, want %q", g.gotFilter, tc.wantGithubArg)
			}
			if l.gotFilter != tc.wantLinearArg {
				t.Errorf("Linear FetchAssignedIssues filter = %q, want %q", l.gotFilter, tc.wantLinearArg)
			}
		})
	}
}

// A failing provider is reported and skipped (D2): the other provider's issues
// still sync and Run returns a report, not an error.
func TestEngineRunIsolatesProviderFailures(t *testing.T) {
	quietLogs(t)

	failing := func(provider *fakeProvider, message string) *fakeProvider {
		provider.err = errors.New(message)
		return provider
	}

	cases := []struct {
		name         string
		providers    []Provider
		want         *Report
		wantUpserted []string
	}{
		// happy path
		{
			name: "GitHub still syncs when Linear fails",
			providers: []Provider{
				fakeGithub(githubIssue("#1", "owner/repo")),
				failing(fakeLinear(), "linear api 401: Authentication required"),
			},
			want: &Report{
				Created: 1,
				Errors:  []ReportError{{Provider: shared.ProviderLinear, Error: "linear api 401: Authentication required"}},
			},
			wantUpserted: []string{"#1"},
		},
		// edge cases
		{
			name: "Linear still syncs when GitHub, fetched first, fails",
			providers: []Provider{
				failing(fakeGithub(), "github is down"),
				fakeLinear(linearIssue("ENG-1", "ENG", "started")),
			},
			want: &Report{
				Created: 1,
				Errors:  []ReportError{{Provider: shared.ProviderGithub, Error: "github is down"}},
			},
			wantUpserted: []string{"ENG-1"},
		},
		// error path
		{
			name: "every provider failing is reported once each and nothing is upserted",
			providers: []Provider{
				failing(fakeGithub(), "github is down"),
				failing(fakeLinear(), "linear is down"),
			},
			want: &Report{Errors: []ReportError{
				{Provider: shared.ProviderGithub, Error: "github is down"},
				{Provider: shared.ProviderLinear, Error: "linear is down"},
			}},
			wantUpserted: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := &fakeNotion{
				projects: []shared.Project{githubProject("p-gh", "owner/repo"), linearProject("p-lin", "ENG")},
				upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
					return created(), nil
				},
			}

			got, err := NewEngine(n, tc.providers).Run(context.Background(), EngineRunOptions{})
			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
			}
			upserted := make([]string, 0, len(n.calls))
			for _, call := range n.calls {
				upserted = append(upserted, call.storyInput.Issue)
			}
			if diff := cmp.Diff(tc.wantUpserted, upserted, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("upserted issues mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// When a flag narrows the run (O4), projects of the providers left out are
// skipped without an error and without loading their stories.
func TestEngineRunNarrowedSkipsOtherProvidersQuietly(t *testing.T) {
	quietLogs(t)

	t.Run("happy path", func(t *testing.T) {
		t.Run("a Linear project on a GitHub-only narrowed run is skipped without an error", func(t *testing.T) {
			n := &fakeNotion{
				projects: []shared.Project{githubProject("p-gh", "owner/repo"), linearProject("p-lin", "ENG")},
				upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
					return created(), nil
				},
			}
			engine := NewEngine(n, []Provider{fakeGithub(githubIssue("#1", "owner/repo"))})

			got, err := engine.Run(context.Background(), EngineRunOptions{ProvidersNarrowed: true})
			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			if diff := cmp.Diff(&Report{Created: 1}, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]string{"p-gh"}, n.storiesRequested); diff != "" {
				t.Errorf("stories loaded for (-want +got):\n%s", diff)
			}
		})

		t.Run("a GitHub project on a Linear-only narrowed run is skipped without an error", func(t *testing.T) {
			n := &fakeNotion{
				projects: []shared.Project{githubProject("p-gh", "owner/repo"), linearProject("p-lin", "ENG")},
				upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
					return created(), nil
				},
			}
			engine := NewEngine(n, []Provider{fakeLinear(linearIssue("ENG-1", "ENG", "started"))})

			got, err := engine.Run(context.Background(), EngineRunOptions{Team: "ENG", ProvidersNarrowed: true})
			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			if diff := cmp.Diff(&Report{Created: 1}, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]string{"p-lin"}, n.storiesRequested); diff != "" {
				t.Errorf("stories loaded for (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("narrowing still reports projects with no provider or an unknown one", func(t *testing.T) {
			n := &fakeNotion{
				projects: []shared.Project{
					githubProject("p-gh", "owner/repo"),
					{PageID: "p-empty", Name: "Empty", Repo: "owner/other"},
					{PageID: "p-typo", Name: "Typo", Repo: "owner/other", Provider: "gitlab"},
					linearProject("p-lin", "ENG"),
				},
				upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
					return created(), nil
				},
			}
			engine := NewEngine(n, []Provider{fakeGithub()})

			got, err := engine.Run(context.Background(), EngineRunOptions{ProvidersNarrowed: true})
			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			want := []ReportError{
				{Project: "Empty", Error: "project has no provider"},
				{Project: "Typo", Error: `unknown provider "gitlab"`},
			}
			if diff := cmp.Diff(want, got.Errors); diff != "" {
				t.Errorf("Run() errors mismatch (-want +got):\n%s", diff)
			}
		})
	})

	// error path: none — skipping a project can't fail; Notion errors are
	// covered by TestEngineRunListProjectsError and TestEngineRunListStoriesError.
}

// Linear issues go through the engine with Linear's status rules: a manual
// status is overwritten (D7), and Finished Date comes only from completedAt.
func TestEngineRunAppliesLinearRules(t *testing.T) {
	quietLogs(t)

	updatedAt := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)
	completedAt := time.Date(2026, 11, 4, 17, 30, 0, 0, time.UTC)

	issueIn := func(state string) shared.Issue {
		issue := linearIssue("ENG-1", "ENG", state)
		issue.Title = "Plan the rollout"
		issue.UpdatedAt = updatedAt
		return issue
	}
	// storyAt returns a story matching issue in every field IsSynced compares
	// except its status, so the status alone decides whether the engine writes.
	storyAt := func(issue shared.Issue, status string) shared.Story {
		story := syncedStoryFor(issue)
		story.Status = status
		return story
	}

	backlog := issueIn("backlog")
	started := issueIn("started")
	canceled := issueIn("canceled")
	completed := issueIn("completed")
	completed.ClosedAt = &completedAt

	type written struct {
		Project      string
		Status       string
		FinishedDate string
	}

	cases := []struct {
		name    string
		teamKey string
		issue   shared.Issue
		stories []shared.Story
		want    written
	}{
		// happy path
		{
			name:    "a manual in progress is overwritten with the backlog issue's not started",
			teamKey: "ENG",
			issue:   backlog,
			stories: []shared.Story{storyAt(backlog, "In progress")},
			want:    written{Project: "p-lin", Status: "Not started"},
		},
		{
			name:    "a completed issue is written as done with its completion date",
			teamKey: "ENG",
			issue:   completed,
			want:    written{Project: "p-lin", Status: "Done", FinishedDate: "2026-11-04 17:30"},
		},
		// edge cases
		{
			name:    "a manual done is overwritten when the issue is started again",
			teamKey: "ENG",
			issue:   started,
			stories: []shared.Story{storyAt(started, "Done")},
			want:    written{Project: "p-lin", Status: "In progress"},
		},
		{
			name:    "a canceled issue without a completion date has no finished date",
			teamKey: "ENG",
			issue:   canceled,
			want:    written{Project: "p-lin", Status: "Cancelled"},
		},
		{
			name:    "an issue from team ENG matches a project whose team key is eng",
			teamKey: "eng",
			issue:   backlog,
			want:    written{Project: "p-lin", Status: "Not started"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n := &fakeNotion{
				projects: []shared.Project{linearProject("p-lin", tc.teamKey)},
				stories:  map[string][]shared.Story{"p-lin": tc.stories},
				upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
					return updated(), nil
				},
			}

			if _, err := NewEngine(n, []Provider{fakeLinear(tc.issue)}).Run(context.Background(), EngineRunOptions{}); err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			if len(n.calls) != 1 {
				t.Fatalf("UpsertStory called %d times, want 1", len(n.calls))
			}
			input := n.calls[0].storyInput
			got := written{Project: input.Project, Status: input.Status, FinishedDate: input.FinishedDate}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("story written mismatch (-want +got):\n%s", diff)
			}
		})
	}

	// edge case: the override only writes when the status actually differs.
	t.Run("a story already at the status the Linear state maps to is left unchanged", func(t *testing.T) {
		n := &fakeNotion{
			projects: []shared.Project{linearProject("p-lin", "ENG")},
			stories:  map[string][]shared.Story{"p-lin": {storyAt(backlog, "Not started")}},
			upsert: func(shared.StoryInput) (*shared.UpsertResult, error) {
				t.Fatalf("UpsertStory called for a story already at its Linear status")
				return nil, nil
			},
		}

		got, err := NewEngine(n, []Provider{fakeLinear(backlog)}).Run(context.Background(), EngineRunOptions{})
		if err != nil {
			t.Fatalf("Run() unexpected error: %v", err)
		}

		if diff := cmp.Diff(&Report{Unchanged: 1}, got, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Run() report mismatch (-want +got):\n%s", diff)
		}
	})

	// error path: none — an upsert failure is covered by
	// TestEngineRunRecordsUpsertErrorsAndKeepsProcessing.
}
