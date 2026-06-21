package status

import (
	"log/slog"

	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/sync"
	"github.com/nox456/forgesync/internal/utils"
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

func BuildRow(issue shared.Issue, project *shared.Project, existingStory *shared.Story) Row {
	isSynced := utils.IsSynced(issue, existingStory)
	var status string

	if existingStory == nil {
		slog.Debug("Story not found")
		status = sync.ComputeStatus(issue, "")
	} else {
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
