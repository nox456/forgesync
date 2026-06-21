package output

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/status"
	"github.com/nox456/forgesync/internal/sync"
)

type TextPrinter struct {
	printer *tabwriter.Writer
}

func NewTextPrinter() *TextPrinter {
	return &TextPrinter{
		printer: tabwriter.NewWriter(os.Stdout, 10, 0, 2, ' ', 0),
	}
}

func (p *TextPrinter) PrintProjects(projects []shared.Project) {
	fmt.Println("==== Projects ====")
	fmt.Fprintln(p.printer, "Name\tRepo")
	for _, project := range projects {
		fmt.Fprintf(p.printer, "%s\t%s\n", project.Name, project.Repo)
	}

	err := p.printer.Flush()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}

func (p *TextPrinter) PrintIssues(issues []shared.Issue) {
	fmt.Println("==== Issues ====")

	fmt.Fprintln(p.printer, "#\tRepo\tTitle\tState\tHas PR")

	for _, issue := range issues {
		var hasPR string

		if issue.HasLinkedPR {
			hasPR = ""
		} else {
			hasPR = ""
		}

		fmt.Fprintf(p.printer, "%d\t%s\t%s\t%s\t%s\n",
			issue.Number,
			issue.Repo,
			issue.Title,
			issue.State,
			hasPR,
		)
	}

	err := p.printer.Flush()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}

func (p *TextPrinter) PrintReport(report *sync.Report) {
	fmt.Println("\nSync Complete")
	fmt.Println("--------------")
	fmt.Fprintln(p.printer, " Created\t", report.Created)
	fmt.Fprintln(p.printer, " Updated\t", report.Updated)
	fmt.Fprintln(p.printer, " Unchanged\t", report.Unchanged)
	fmt.Fprintln(p.printer, " Skipped\t", report.Skipped)

	err := p.printer.Flush()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if len(report.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, error := range report.Errors {
			fmt.Printf("* forgesync#%d: %s\n", error.IssueNumber, error.Error)
		}
	}
}

func (p *TextPrinter) PrintStatus(rows []status.Row) {
	fmt.Println("==== Status ====")
	fmt.Fprintln(p.printer, "Project\tIssue\tTitle\tStatus\tHas PR\tIs Synced")
	for _, row := range rows {
		var hasPR string
		if row.HasPR {
			hasPR = ""
		} else {
			hasPR = ""
		}

		var isSynced string
		if row.IsSynced {
			isSynced = ""
		} else {
			isSynced = ""
		}

		var issueNumber string
		var projectName string

		if row.ProjectName != nil {
			projectName = *row.ProjectName
			issueNumber = fmt.Sprintf("%d", row.IssueNumber)
		} else {
			projectName = "[NO PROJECT]"
			issueNumber = fmt.Sprintf("%d (%s)", row.IssueNumber, row.IssueRepo)
		}

		fmt.Fprintf(p.printer, "%s\t%s\t%s\t%s\t%s\t%s\n", projectName, issueNumber, row.IssueTitle, row.Status, hasPR, isSynced)
	}

	err := p.printer.Flush()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}
