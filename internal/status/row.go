package status

import (
	"log/slog"

	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
	"github.com/nox456/forgesync/internal/sync"
)

type Row struct {
	IssueNumber int
	IssueTitle  string
	ProjectName *string
	IssueRepo   string
	Status      string
	HasPR       bool
	IsSynced    bool
}

func BuildRow(issue github.Issue, project *notion.Project, existingStory *notion.Story) Row {
	var isSynced bool
	var status string

	if existingStory == nil {
		slog.Debug("Story not found")
		isSynced = false
		status = sync.ComputeStatus(issue, "")
	} else {
		isSynced = true
		status = sync.ComputeStatus(issue, existingStory.Status)
	}

	var projectName *string
	if project != nil {
		projectName = &project.Name
	}

	return Row{
		IssueNumber: issue.Number,
		IssueTitle:  issue.Title,
		ProjectName: projectName,
		IssueRepo:   issue.Repo,
		Status:      status,
		HasPR:       issue.HasLinkedPR,
		IsSynced:    isSynced,
	}
}
