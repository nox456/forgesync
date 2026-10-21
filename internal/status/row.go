package status

import (
	"log/slog"

	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/sync"
)

type Row struct {
	Identifier  string
	IssueTitle  string
	ProjectName *string
	IssueScope  string
	Status      string
	HasPR       bool
	IsSynced    bool
	Provider    shared.Provider
}

func BuildRow(issue shared.Issue, project *shared.Project, existingStory *shared.Story) Row {
	isSynced := sync.IsSynced(issue, existingStory)
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
		Identifier:  issue.Identifier,
		IssueTitle:  issue.Title,
		ProjectName: projectName,
		IssueScope:  issue.Scope,
		Status:      status,
		HasPR:       issue.HasLinkedPR,
		IsSynced:    isSynced,
		Provider:    issue.Provider,
	}
}
