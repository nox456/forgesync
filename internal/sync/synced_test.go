package sync

import (
	"testing"
	"time"

	"github.com/nox456/forgesync/internal/shared"
)

func TestIsSynced(t *testing.T) {
	updatedAt := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)

	issue := shared.Issue{
		Provider:   shared.ProviderGithub,
		Identifier: "#42",
		Title:      "Add login flow",
		State:      "open",
		Labels:     []string{"feature", "backend"},
		UpdatedAt:  updatedAt,
	}

	// storyWith returns the Story an in-sync issue would have, with a single
	// field changed, so each case proves that field alone breaks the match.
	storyWith := func(mutate func(*shared.Story)) *shared.Story {
		story := shared.Story{
			Name:         "Add login flow",
			Status:       "Not started",
			Labels:       []string{"feature", "backend"},
			LastWorkedAt: "2026-05-20 10:30",
		}
		if mutate != nil {
			mutate(&story)
		}
		return &story
	}

	cases := []struct {
		name  string
		issue shared.Issue
		story *shared.Story
		want  bool
	}{
		{
			name:  "a missing story is never synced",
			issue: issue,
			story: nil,
			want:  false,
		},
		{
			name:  "every compared field matches",
			issue: issue,
			story: storyWith(nil),
			want:  true,
		},
		{
			name:  "a stale title is out of sync",
			issue: issue,
			story: storyWith(func(s *shared.Story) { s.Name = "Add login" }),
			want:  false,
		},
		{
			name:  "stale labels are out of sync",
			issue: issue,
			story: storyWith(func(s *shared.Story) { s.Labels = []string{"feature"} }),
			want:  false,
		},
		{
			name:  "reordered labels count as a difference",
			issue: issue,
			story: storyWith(func(s *shared.Story) { s.Labels = []string{"backend", "feature"} }),
			want:  false,
		},
		{
			name:  "a stale last worked at is out of sync",
			issue: issue,
			story: storyWith(func(s *shared.Story) { s.LastWorkedAt = "2026-05-19 10:30" }),
			want:  false,
		},
		{
			name:  "a manual status the rules preserve stays in sync",
			issue: issue,
			story: storyWith(func(s *shared.Story) { s.Status = "Done" }),
			want:  true,
		},
		{
			name:  "a status the rules would promote is out of sync",
			issue: shared.Issue{Provider: shared.ProviderGithub, Title: issue.Title, State: "open", HasLinkedPR: true, Labels: issue.Labels, UpdatedAt: updatedAt},
			story: storyWith(func(s *shared.Story) { s.Status = "In progress" }),
			want:  false,
		},
		{
			name:  "a closed issue whose story is still not started is out of sync",
			issue: shared.Issue{Provider: shared.ProviderGithub, Title: issue.Title, State: "closed", Labels: issue.Labels, UpdatedAt: updatedAt},
			story: storyWith(nil),
			want:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSynced(tc.issue, tc.story); got != tc.want {
				t.Errorf("IsSynced() = %v, want %v", got, tc.want)
			}
		})
	}
}

// The Linear rule ignores the story's status (D7), so IsSynced must report a
// manual status as a difference whenever the Linear state maps elsewhere.
func TestIsSyncedLinear(t *testing.T) {
	updatedAt := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)

	issueWithState := func(state string, hasLinkedPR bool) shared.Issue {
		return shared.Issue{
			Provider:    shared.ProviderLinear,
			Identifier:  "ENG-7",
			Title:       "Plan the rollout",
			State:       state,
			HasLinkedPR: hasLinkedPR,
			Labels:      []string{"infra"},
			UpdatedAt:   updatedAt,
		}
	}

	storyWithStatus := func(status string) *shared.Story {
		return &shared.Story{
			Name:         "Plan the rollout",
			Status:       status,
			Labels:       []string{"infra"},
			LastWorkedAt: "2026-11-04 09:00",
		}
	}

	cases := []struct {
		name  string
		issue shared.Issue
		story *shared.Story
		want  bool
	}{
		// happy path
		{
			name:  "a story already at the status the Linear state maps to is synced",
			issue: issueWithState("backlog", false),
			story: storyWithStatus("Not started"),
			want:  true,
		},
		{
			name:  "a manual in progress on a backlog issue is out of sync",
			issue: issueWithState("backlog", false),
			story: storyWithStatus("In progress"),
			want:  false,
		},
		// edge cases
		{
			name:  "a manual done on a started issue is out of sync",
			issue: issueWithState("started", false),
			story: storyWithStatus("Done"),
			want:  false,
		},
		{
			name:  "a started issue with a PR is synced only when the story is in PR",
			issue: issueWithState("started", true),
			story: storyWithStatus("In PR"),
			want:  true,
		},
		{
			name:  "a manual cancelled on a completed issue is out of sync",
			issue: issueWithState("completed", false),
			story: storyWithStatus("Cancelled"),
			want:  false,
		},
		// error path: none — IsSynced is a pure comparison.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSynced(tc.issue, tc.story); got != tc.want {
				t.Errorf("IsSynced() = %v, want %v", got, tc.want)
			}
		})
	}
}
