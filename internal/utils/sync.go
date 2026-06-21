package utils

import (
	"slices"

	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/sync"
)

func IsSynced(issue shared.Issue, existingStory *shared.Story) bool {
	if existingStory == nil {
		return false
	}

	hasSameName := existingStory.Name == issue.Title
	hasSameStatus := existingStory.Status == sync.ComputeStatus(issue, existingStory.Status)
	hasSameLabels := slices.Equal(existingStory.Labels, issue.Labels)
	hasSameLastWorkedAt := existingStory.LastWorkedAt == issue.UpdatedAt.Format("2006-01-02 15:04")

	return hasSameName && hasSameStatus && hasSameLabels && hasSameLastWorkedAt
}
