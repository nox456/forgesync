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
	fmt.Fprintln(p.printer, "Provider\tName\tScope")
	for _, project := range projects {
		fmt.Fprintf(p.printer, "%s\t%s\t%s\n", project.Provider, project.Name, project.GetScope())
	}

	err := p.printer.Flush()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}

func (p *TextPrinter) PrintIssues(issues []shared.Issue) {
	fmt.Println("==== Issues ====")

	fmt.Fprintln(p.printer, "Provider\tIssue\tScope\tTitle\tState\tHas PR")

	for _, issue := range issues {
		var hasPR string

		if issue.HasLinkedPR {
			hasPR = ""
		} else {
			hasPR = ""
		}

		fmt.Fprintf(p.printer, "%s\t%s\t%s\t%s\t%s\t%s\n",
			issue.Provider,
			issue.Identifier,
			issue.Scope,
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
			switch {
			case error.Identifier != "":
				fmt.Printf("* %s %s: %s\n", error.Provider, error.Identifier, error.Error)
			default:
				fmt.Printf("* %s: %s\n", error.Project, error.Error)
			}
		}
	}
}

func (p *TextPrinter) PrintStatus(rows []status.Row) {
	fmt.Println("==== Status ====")
	fmt.Fprintln(p.printer, "Provider\tProject\tIssue\tTitle\tStatus\tHas PR\tIs Synced")
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

		var issueIdentifier string
		var projectName string

		if row.ProjectName != nil {
			projectName = *row.ProjectName
			issueIdentifier = row.Identifier
		} else {
			projectName = "[NO PROJECT]"
			issueIdentifier = fmt.Sprintf("%s (%s)", row.Identifier, row.IssueScope)
		}

		fmt.Fprintf(p.printer, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", row.Provider, projectName, issueIdentifier, row.IssueTitle, row.Status, hasPR, isSynced)
	}

	err := p.printer.Flush()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
}
