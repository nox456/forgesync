package sync

import (
	"github.com/nox456/forgesync/internal/github"
)

func ComputeStatus(issue github.Issue, previousStatus string) string {
	switch issue.State {
	case "open":
		if previousStatus == "" || previousStatus == "Not started" {
			if issue.HasLinkedPR {
				return "In progress"
			}
			return "Not started"
		}
		return previousStatus
	case "closed":
		if previousStatus == "Not started" || previousStatus == "In progress" {
			return "Done"
		}
		return previousStatus
	}

	return "Not started"
}
