package sync

import (
	"github.com/nox456/forgesync/internal/shared"
)

const (
	statusNotStarted = "Not started"
	statusInProgress = "In progress"
	statusInPR       = "In PR"
	statusDone       = "Done"
	statusCancelled  = "Cancelled"
)

func ComputeStatus(issue shared.Issue, previousStatus string) string {
	switch issue.Provider {
	case shared.ProviderGithub:
		return computeGithubStatus(issue, previousStatus)
	case shared.ProviderLinear:
		return computeLinearStatus(issue)
	}

	return statusNotStarted
}

func computeGithubStatus(issue shared.Issue, previousStatus string) string {
	switch issue.State {
	case "open":
		if previousStatus == "" || previousStatus == statusNotStarted {
			if issue.HasLinkedPR {
				return statusInPR
			}
			return statusNotStarted
		}
		if previousStatus == statusInProgress && issue.HasLinkedPR {
			return statusInPR
		}
		return previousStatus
	case "closed":
		if previousStatus == statusCancelled {
			return statusCancelled
		}
		return statusDone
	}

	return statusNotStarted
}

func computeLinearStatus(issue shared.Issue) string {
	switch issue.State {
	case "started":
		if issue.HasLinkedPR {
			return statusInPR
		}
		return statusInProgress
	case "completed":
		return statusDone
	case "canceled":
		return statusCancelled
	}

	return statusNotStarted
}
