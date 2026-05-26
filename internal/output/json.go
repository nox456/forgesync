package output

import (
	"encoding/json"
	"fmt"

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
	HasLinkedPR bool   `json:"hasLinkedPR"`
}

type Report struct {
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Skipped   int `json:"skipped"`
	Unchanged int `json:"unchanged"`
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

	bytes, err := json.Marshal(struct{ Projects []Project }{Projects: parsedProjects})
	if err != nil {
		fmt.Println(err)
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

	bytes, err := json.Marshal(struct{ Issues []Issue }{Issues: parsedIssues})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(bytes))
}

func (p *JSONPrinter) PrintReport(report *sync.Report) {
	parsedReport := Report{
		Created:   report.Created,
		Updated:   report.Updated,
		Skipped:   report.Skipped,
		Unchanged: report.Unchanged,
	}

	bytes, err := json.Marshal(struct{ Report Report }{Report: parsedReport})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(bytes))
}
