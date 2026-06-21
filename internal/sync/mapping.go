package sync

import (
	"fmt"

	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/shared"
)

func IssueToStoryInput(issue shared.Issue, existingStory *shared.Story, projectPageId string) shared.StoryInput {
	var finishedDate string
	if issue.ClosedAt != nil {
		finishedDate = issue.ClosedAt.Format("2006-01-02 15:04")
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
		LastWorkedAt: issue.UpdatedAt.Format("2006-01-02 15:04"),
		FinishedDate: finishedDate,
	}
}
