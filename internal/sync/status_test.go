package sync

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nox456/forgesync/internal/shared"
)

func TestComputeStatus(t *testing.T) {
	cases := []struct {
		name           string
		issue          shared.Issue
		previousStatus string
		want           string
	}{
		{
			name:  "open without linked PR and no previous status defaults to not started",
			issue: shared.Issue{Provider: shared.ProviderGithub, State: "open", HasLinkedPR: false},
			want:  "Not started",
		},
		{
			name:           "open without linked PR and previous not started stays not started",
			issue:          shared.Issue{Provider: shared.ProviderGithub, State: "open", HasLinkedPR: false},
			previousStatus: "Not started",
			want:           "Not started",
		},
		{
			name:  "open with linked PR and no previous status is in PR",
			issue: shared.Issue{Provider: shared.ProviderGithub, State: "open", HasLinkedPR: true},
			want:  "In PR",
		},
		{
			name:           "open with linked PR and previous not started is in PR",
			issue:          shared.Issue{Provider: shared.ProviderGithub, State: "open", HasLinkedPR: true},
			previousStatus: "Not started",
			want:           "In PR",
		},
		{
			name:           "open without linked PR preserves a previous manual status",
			issue:          shared.Issue{Provider: shared.ProviderGithub, State: "open", HasLinkedPR: false},
			previousStatus: "In progress",
			want:           "In progress",
		},
		{
			name:           "open preserves a previously done status",
			issue:          shared.Issue{Provider: shared.ProviderGithub, State: "open", HasLinkedPR: true},
			previousStatus: "Done",
			want:           "Done",
		},
		{
			name:           "open preserves a previously cancelled status",
			issue:          shared.Issue{Provider: shared.ProviderGithub, State: "open", HasLinkedPR: false},
			previousStatus: "Cancelled",
			want:           "Cancelled",
		},
		{
			name:           "closed from not started becomes done",
			issue:          shared.Issue{Provider: shared.ProviderGithub, State: "closed", HasLinkedPR: false},
			previousStatus: "Not started",
			want:           "Done",
		},
		{
			name:           "closed from in progress becomes done",
			issue:          shared.Issue{Provider: shared.ProviderGithub, State: "closed", HasLinkedPR: true},
			previousStatus: "In progress",
			want:           "Done",
		},
		{
			name:           "closed preserves a previously done status",
			issue:          shared.Issue{Provider: shared.ProviderGithub, State: "closed", HasLinkedPR: true},
			previousStatus: "Done",
			want:           "Done",
		},
		{
			name:           "closed preserves a previously cancelled status",
			issue:          shared.Issue{Provider: shared.ProviderGithub, State: "closed", HasLinkedPR: false},
			previousStatus: "Cancelled",
			want:           "Cancelled",
		},
		{
			name:  "closed with no previous status becomes done",
			issue: shared.Issue{Provider: shared.ProviderGithub, State: "closed", HasLinkedPR: false},
			want:  "Done",
		},
		{
			name:  "unknown state falls back to not started",
			issue: shared.Issue{Provider: shared.ProviderGithub, State: "", HasLinkedPR: true},
			want:  "Not started",
		},
		{
			name:           "an unset provider falls back to not started even when open with a PR",
			issue:          shared.Issue{State: "open", HasLinkedPR: true},
			previousStatus: "In progress",
			want:           "Not started",
		},
		{
			name:           "an unset provider falls back to not started even when closed",
			issue:          shared.Issue{State: "closed"},
			previousStatus: "In progress",
			want:           "Not started",
		},
		{
			name:  "an unknown provider falls back to not started",
			issue: shared.Issue{Provider: "gitlab", State: "closed", HasLinkedPR: true},
			want:  "Not started",
		},
		// error path: none — ComputeStatus is total; every input maps to a status.
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

func TestComputeStatusLinear(t *testing.T) {
	linear := func(state string, hasLinkedPR bool) shared.Issue {
		return shared.Issue{Provider: shared.ProviderLinear, State: state, HasLinkedPR: hasLinkedPR}
	}

	cases := []struct {
		name           string
		issue          shared.Issue
		previousStatus string
		want           string
	}{
		// happy path: one case per row of the plan's mapping table
		{name: "backlog is not started", issue: linear("backlog", false), want: "Not started"},
		{name: "unstarted is not started", issue: linear("unstarted", false), want: "Not started"},
		{name: "started with a linked PR is in PR", issue: linear("started", true), want: "In PR"},
		{name: "started without a linked PR is in progress", issue: linear("started", false), want: "In progress"},
		{name: "completed is done", issue: linear("completed", false), want: "Done"},
		{name: "canceled is cancelled", issue: linear("canceled", false), want: "Cancelled"},
		// edge cases: the previous status never carries over (D7)
		{
			name:           "started overrides a previous done instead of keeping it",
			issue:          linear("started", false),
			previousStatus: "Done",
			want:           "In progress",
		},
		{
			name:           "completed overrides a previous cancelled instead of keeping it",
			issue:          linear("completed", false),
			previousStatus: "Cancelled",
			want:           "Done",
		},
		{
			name:           "backlog overrides a manual in progress",
			issue:          linear("backlog", false),
			previousStatus: "In progress",
			want:           "Not started",
		},
		{
			name:           "canceled overrides a previous done",
			issue:          linear("canceled", false),
			previousStatus: "Done",
			want:           "Cancelled",
		},
		{
			name:           "started with a linked PR overrides a previous not started",
			issue:          linear("started", true),
			previousStatus: "Not started",
			want:           "In PR",
		},
		{name: "a linked PR does not matter once completed", issue: linear("completed", true), want: "Done"},
		{name: "a linked PR does not matter while unstarted", issue: linear("unstarted", true), want: "Not started"},
		{name: "an unknown state is not started", issue: linear("triage", true), previousStatus: "In PR", want: "Not started"},
		{name: "an empty state is not started", issue: linear("", false), want: "Not started"},
		{name: "state types are matched exactly, not case-insensitively", issue: linear("Started", false), want: "Not started"},
		// error path: none — the Linear rule is total; every state maps to a status.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeStatus(tc.issue, tc.previousStatus)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("ComputeStatus(%+v, %q) mismatch (-want +got):\n%s", tc.issue, tc.previousStatus, diff)
			}
		})
	}
}
