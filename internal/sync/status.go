package sync

import (
	"github.com/nox456/forgesync/internal/github"
)

func ComputeStatus(issue github.Issue, previousStatus string) string {
	switch issue.State {
	case "open":
		if issue.HasLinkedPR {
			return "In PR"
		}
		if previousStatus == "" {
			return "Not started"
		}
		return previousStatus
	case "closed":
		if previousStatus == "Cancelled" {
			return "Cancelled"
		}
		if issue.HasLinkedPR {
			return "Done"
		}
		return "Cancelled"
	}

	return "Not started"
}
