package sync

import (
	"fmt"

	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
)

func IssueToStoryInput(issue github.Issue, projectPageId string) notion.StoryInput {
	var finishedDate string
	if issue.ClosedAt != nil {
		finishedDate = issue.ClosedAt.Format("2006-01-02")
	}
	return notion.StoryInput{
		Name:         issue.Title,
		Project:      projectPageId,
		Issue:        fmt.Sprintf("%d", issue.Number),
		Url:          issue.URL,
		Status:       ComputeStatus(issue),
		Labels:       issue.Labels,
		LastWorkedAt: issue.UpdatedAt.Format("2006-01-02"),
		FinishedDate: finishedDate,
	}
}
