package status

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/nox456/forgesync/internal/shared"
)

func TestBuildRow(t *testing.T) {
	quietLogs(t)

	updatedAt := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)
	projectName := "ForgeSync"
	project := &shared.Project{PageID: "p1", Name: projectName, Repo: "owner/repo", Provider: shared.ProviderGithub}

	issue := shared.Issue{
		Provider:    shared.ProviderGithub,
		Identifier:  "#12",
		Title:       "Sync statuses",
		Scope:       "owner/repo",
		State:       "open",
		HasLinkedPR: true,
		Labels:      []string{"feature"},
		UpdatedAt:   updatedAt,
	}

	t.Run("happy path", func(t *testing.T) {
		t.Run("a matched issue carries its identifier, scope and provider", func(t *testing.T) {
			got := BuildRow(issue, project, nil)

			want := Row{
				Identifier:  "#12",
				IssueTitle:  "Sync statuses",
				ProjectName: &projectName,
				IssueScope:  "owner/repo",
				Status:      "In PR",
				HasPR:       true,
				IsSynced:    false,
				Provider:    shared.ProviderGithub,
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("BuildRow() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("an in-sync story marks the row synced", func(t *testing.T) {
			story := &shared.Story{
				Name:         "Sync statuses",
				Status:       "In PR",
				Labels:       []string{"feature"},
				LastWorkedAt: "2026-05-20 10:30",
			}

			got := BuildRow(issue, project, story)

			if !got.IsSynced {
				t.Errorf("BuildRow().IsSynced = false, want true")
			}
			if got.Status != "In PR" {
				t.Errorf("BuildRow().Status = %q, want %q", got.Status, "In PR")
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("an issue without a project has a nil project name but keeps its scope", func(t *testing.T) {
			got := BuildRow(issue, nil, nil)

			if got.ProjectName != nil {
				t.Errorf("BuildRow().ProjectName = %q, want nil", *got.ProjectName)
			}
			if got.IssueScope != "owner/repo" {
				t.Errorf("BuildRow().IssueScope = %q, want %q", got.IssueScope, "owner/repo")
			}
			if got.Provider != shared.ProviderGithub {
				t.Errorf("BuildRow().Provider = %q, want %q", got.Provider, shared.ProviderGithub)
			}
		})

		t.Run("an existing manual status is carried over for an open GitHub issue", func(t *testing.T) {
			got := BuildRow(issue, project, &shared.Story{Status: "Done"})

			if got.Status != "Done" {
				t.Errorf("BuildRow().Status = %q, want %q", got.Status, "Done")
			}
		})
	})

	// error path: none — BuildRow is a total mapping with no failure mode.
}
