package sync

import (
	"fmt"

	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
)

func IssueToStoryInput(issue github.Issue, existingStory *notion.Story, projectPageId string) notion.StoryInput {
	var finishedDate string
	if issue.ClosedAt != nil {
		finishedDate = issue.ClosedAt.Format("2006-01-02 15:04")
	}

	var previousStatus string

	if existingStory != nil {
		previousStatus = existingStory.Status
	}

	return notion.StoryInput{
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
