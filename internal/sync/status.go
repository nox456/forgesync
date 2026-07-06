package sync

import (
	"github.com/nox456/forgesync/internal/shared"
)

func ComputeStatus(issue shared.Issue, previousStatus string) string {
	switch issue.State {
	case "open":
		if previousStatus == "" || previousStatus == "Not started" {
			if issue.HasLinkedPR {
				return "In PR"
			}
			return "Not started"
		}
		if previousStatus == "In progress" && issue.HasLinkedPR {
			return "In PR"
		}
		return previousStatus
	case "closed":
		if previousStatus == "Cancelled" {
			return "Cancelled"
		}
		return "Done"
	}

	return "Not started"
}
