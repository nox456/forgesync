package sync

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nox456/forgesync/internal/github"
)

func TestComputeStatus(t *testing.T) {
	cases := []struct {
		name           string
		issue          github.Issue
		previousStatus string
		want           string
	}{
		{
			name:  "open without linked PR and no previous status defaults to not started",
			issue: github.Issue{State: "open", HasLinkedPR: false},
			want:  "Not started",
		},
		{
			name:           "open without linked PR preserves a previous manual status",
			issue:          github.Issue{State: "open", HasLinkedPR: false},
			previousStatus: "In progress",
			want:           "In progress",
		},
		{
			name:  "open with linked PR is in PR",
			issue: github.Issue{State: "open", HasLinkedPR: true},
			want:  "In PR",
		},
		{
			name:  "closed without linked PR is cancelled",
			issue: github.Issue{State: "closed", HasLinkedPR: false},
			want:  "Cancelled",
		},
		{
			name:  "closed with linked PR takes precedence as done",
			issue: github.Issue{State: "closed", HasLinkedPR: true},
			want:  "Done",
		},
		{
			name:  "unknown state falls back to not started",
			issue: github.Issue{State: "", HasLinkedPR: true},
			want:  "Not started",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeStatus(tc.issue, tc.previousStatus)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ComputeStatus(%+v) mismatch (-want +got):\n%s", tc.issue, diff)
			}
		})
	}
}
