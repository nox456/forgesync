package sync

import (
	"slices"

	"github.com/nox456/forgesync/internal/shared"
)

// IsSynced reports whether a Story already matches its GitHub issue in every
// field the sync writes back on an update. Status is compared through
// ComputeStatus so a folded-in manual status (an `In progress` waiting on a PR,
// say) counts as a difference only when the rules would actually change it.
func IsSynced(issue shared.Issue, existingStory *shared.Story) bool {
	if existingStory == nil {
		return false
	}

	hasSameName := existingStory.Name == issue.Title
	hasSameStatus := existingStory.Status == ComputeStatus(issue, existingStory.Status)
	hasSameLabels := slices.Equal(existingStory.Labels, issue.Labels)
	hasSameLastWorkedAt := existingStory.LastWorkedAt == issue.UpdatedAt.Format(storyDateLayout)

	return hasSameName && hasSameStatus && hasSameLabels && hasSameLastWorkedAt
}
