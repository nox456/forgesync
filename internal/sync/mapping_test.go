package sync

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
)

func TestIssueToStoryInput(t *testing.T) {
	updatedAt := time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC)
	closedAt := time.Date(2026, 5, 25, 18, 0, 0, 0, time.UTC)

	cases := []struct {
		name          string
		issue         github.Issue
		projectPageId string
		want          notion.StoryInput
	}{
		{
			name: "basic open issue maps fields and leaves finished date empty",
			issue: github.Issue{
				Number:    42,
				Title:     "Add login flow",
				URL:       "https://github.com/owner/repo/issues/42",
				State:     "open",
				Labels:    []string{"feature", "frontend"},
				UpdatedAt: updatedAt,
			},
			projectPageId: "project-page-1",
			want: notion.StoryInput{
				Name:         "Add login flow",
				Project:      "project-page-1",
				Issue:        "42",
				Url:          "https://github.com/owner/repo/issues/42",
				Status:       "In progress",
				Labels:       []string{"feature", "frontend"},
				LastWorkedAt: "2026-05-20",
				FinishedDate: "",
			},
		},
		{
			name: "closed issue with linked PR sets finished date and done status",
			issue: github.Issue{
				Number:      7,
				Title:       "Fix race condition",
				URL:         "https://github.com/owner/repo/issues/7",
				State:       "closed",
				Labels:      []string{"bug"},
				UpdatedAt:   updatedAt,
				ClosedAt:    &closedAt,
				HasLinkedPR: true,
			},
			projectPageId: "project-page-2",
			want: notion.StoryInput{
				Name:         "Fix race condition",
				Project:      "project-page-2",
				Issue:        "7",
				Url:          "https://github.com/owner/repo/issues/7",
				Status:       "Done",
				Labels:       []string{"bug"},
				LastWorkedAt: "2026-05-20",
				FinishedDate: "2026-05-25",
			},
		},
		{
			name: "empty labels are preserved as-is",
			issue: github.Issue{
				Number:    1,
				Title:     "No labels yet",
				URL:       "https://github.com/owner/repo/issues/1",
				State:     "open",
				Labels:    []string{},
				UpdatedAt: updatedAt,
			},
			projectPageId: "project-page-3",
			want: notion.StoryInput{
				Name:         "No labels yet",
				Project:      "project-page-3",
				Issue:        "1",
				Url:          "https://github.com/owner/repo/issues/1",
				Status:       "In progress",
				Labels:       []string{},
				LastWorkedAt: "2026-05-20",
				FinishedDate: "",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IssueToStoryInput(tc.issue, tc.projectPageId)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("IssueToStoryInput mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
