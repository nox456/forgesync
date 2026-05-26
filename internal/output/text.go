package output

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
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

func (p *TextPrinter) PrintProjects(projects []notion.Project) {
	fmt.Println("==== Projects ====")
	fmt.Fprintln(p.printer, "Name\tRepo")
	for _, project := range projects {
		fmt.Fprintf(p.printer, "%s\t%s\n", project.Name, project.Repo)
	}

	err := p.printer.Flush()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func (p *TextPrinter) PrintIssues(issues []github.Issue) {
	fmt.Println("==== Issues ====")

	fmt.Fprintln(p.printer, "#\tRepo\tTitle\tHas PR")

	for _, issue := range issues {
		var hasPR string

		if issue.HasLinkedPR {
			hasPR = ""
		} else {
			hasPR = ""
		}

		fmt.Fprintf(p.printer, "%d\t%s\t%s\t%s\n",
			issue.Number,
			issue.Repo,
			issue.Title,
			hasPR,
		)
	}

	err := p.printer.Flush()
	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
		return
	}

	if len(report.Errors) > 0 {
		fmt.Println("\nErrors:")
		for _, error := range report.Errors {
			fmt.Printf("* forgesync#%d: %s\n", error.IssueNumber, error.Error)
		}
	}
}
