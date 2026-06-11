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
			name:           "open without linked PR and previous not started stays not started",
			issue:          github.Issue{State: "open", HasLinkedPR: false},
			previousStatus: "Not started",
			want:           "Not started",
		},
		{
			name:  "open with linked PR and no previous status is in progress",
			issue: github.Issue{State: "open", HasLinkedPR: true},
			want:  "In progress",
		},
		{
			name:           "open with linked PR and previous not started is in progress",
			issue:          github.Issue{State: "open", HasLinkedPR: true},
			previousStatus: "Not started",
			want:           "In progress",
		},
		{
			name:           "open without linked PR preserves a previous manual status",
			issue:          github.Issue{State: "open", HasLinkedPR: false},
			previousStatus: "In progress",
			want:           "In progress",
		},
		{
			name:           "open preserves a previously done status",
			issue:          github.Issue{State: "open", HasLinkedPR: true},
			previousStatus: "Done",
			want:           "Done",
		},
		{
			name:           "open preserves a previously cancelled status",
			issue:          github.Issue{State: "open", HasLinkedPR: false},
			previousStatus: "Cancelled",
			want:           "Cancelled",
		},
		{
			name:           "closed from not started becomes done",
			issue:          github.Issue{State: "closed", HasLinkedPR: false},
			previousStatus: "Not started",
			want:           "Done",
		},
		{
			name:           "closed from in progress becomes done",
			issue:          github.Issue{State: "closed", HasLinkedPR: true},
			previousStatus: "In progress",
			want:           "Done",
		},
		{
			name:           "closed preserves a previously done status",
			issue:          github.Issue{State: "closed", HasLinkedPR: true},
			previousStatus: "Done",
			want:           "Done",
		},
		{
			name:           "closed preserves a previously cancelled status",
			issue:          github.Issue{State: "closed", HasLinkedPR: false},
			previousStatus: "Cancelled",
			want:           "Cancelled",
		},
		{
			name:  "closed with no previous status becomes done",
			issue: github.Issue{State: "closed", HasLinkedPR: false},
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
