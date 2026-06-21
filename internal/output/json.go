package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
	"github.com/nox456/forgesync/internal/sync"
)

type JSONPrinter struct{}

type Project struct {
	Name string `json:"name"`
	Repo string `json:"repo"`
}

type Issue struct {
	Number      int    `json:"number"`
	Repo        string `json:"repo"`
	Title       string `json:"title"`
	HasLinkedPR bool   `json:"has_linked_pr"`
}

type Report struct {
	Created   int      `json:"created"`
	Updated   int      `json:"updated"`
	Skipped   int      `json:"skipped"`
	Unchanged int      `json:"unchanged"`
	Errors    []string `json:"errors"`
}

func NewJSONPrinter() *JSONPrinter {
	return &JSONPrinter{}
}

func (p *JSONPrinter) PrintProjects(projects []notion.Project) {
	parsedProjects := make([]Project, len(projects))

	for i, project := range projects {
		parsedProjects[i] = Project{
			Name: project.Name,
			Repo: project.Repo,
		}
	}

	bytes, err := json.Marshal(struct {
		Projects []Project `json:"projects"`
	}{Projects: parsedProjects})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(bytes))
}

func (p *JSONPrinter) PrintIssues(issues []github.Issue) {
	parsedIssues := make([]Issue, len(issues))

	for i, issue := range issues {
		parsedIssues[i] = Issue{
			Number:      issue.Number,
			Repo:        issue.Repo,
			Title:       issue.Title,
			HasLinkedPR: issue.HasLinkedPR,
		}
	}

	bytes, err := json.Marshal(struct {
		Issues []Issue `json:"issues"`
	}{Issues: parsedIssues})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(bytes))
}

func (p *JSONPrinter) PrintReport(report *sync.Report) {
	var errors []string

	for _, error := range report.Errors {
		errors = append(errors, error.Error)
	}

	parsedReport := Report{
		Created:   report.Created,
		Updated:   report.Updated,
		Skipped:   report.Skipped,
		Unchanged: report.Unchanged,
		Errors:    errors,
	}

	bytes, err := json.Marshal(struct {
		Report Report `json:"report"`
	}{Report: parsedReport})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Print(string(bytes))
}
