package output

import (
	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
	"github.com/nox456/forgesync/internal/sync"
)

type Printer interface {
	PrintProjects([]notion.Project)
	PrintIssues([]github.Issue)
	PrintReport(*sync.Report)
}
