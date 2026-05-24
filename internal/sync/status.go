package sync

import "github.com/nox456/forgesync/internal/github"

func ComputeStatus(issue github.Issue) string {
	switch issue.State {
	case "open":
		if issue.HasLinkedPR {
			return "In PR"
		}
		return "In progress"
	case "closed":
		return "Done"
	}

	return "Not started"
}
