package sync

import (
	"fmt"

	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/shared"
)

// storyDateLayout is the format every date written to a Story uses. IsSynced
// compares against it too, so the two must stay in step.
const storyDateLayout = "2006-01-02 15:04"

func IssueToStoryInput(issue shared.Issue, existingStory *shared.Story, projectPageId string) shared.StoryInput {
	var finishedDate string
	if issue.ClosedAt != nil {
		finishedDate = issue.ClosedAt.Format(storyDateLayout)
	}

	var previousStatus string

	if existingStory != nil {
		previousStatus = existingStory.Status
	}

	return shared.StoryInput{
		Name:         issue.Title,
		Project:      projectPageId,
		Issue:        fmt.Sprintf("%d", issue.Number),
		Url:          issue.URL,
		Body:         github.NormalizeGithubBody(issue.Body),
		Status:       ComputeStatus(issue, previousStatus),
		Labels:       issue.Labels,
		LastWorkedAt: issue.UpdatedAt.Format(storyDateLayout),
		FinishedDate: finishedDate,
	}
}
