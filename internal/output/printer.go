package output

import (
	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/status"
	"github.com/nox456/forgesync/internal/sync"
)

type Printer interface {
	PrintProjects([]shared.Project)
	PrintIssues([]shared.Issue)
	PrintReport(*sync.Report)
	PrintStatus([]status.Row)
}
