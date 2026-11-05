package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/status"
	"github.com/nox456/forgesync/internal/sync"
)

type JSONPrinter struct{}

type Project struct {
	Name     string `json:"name"`
	Scope    string `json:"scope"`
	Provider string `json:"provider"`
}

type Issue struct {
	Identifier  string `json:"identifier"`
	Scope       string `json:"scope"`
	Title       string `json:"title"`
	HasLinkedPR bool   `json:"has_linked_pr"`
	Provider    string `json:"provider"`
}

type Error struct {
	Provider   string `json:"provider,omitempty"`
	Project    string `json:"project,omitempty"`
	Identifier string `json:"identifier,omitempty"`
	Message    string `json:"message"`
}

type Report struct {
	Created   int     `json:"created"`
	Updated   int     `json:"updated"`
	Skipped   int     `json:"skipped"`
	Unchanged int     `json:"unchanged"`
	Errors    []Error `json:"errors"`
}

type StatusRow struct {
	IssueIdentifier string  `json:"identifier"`
	IssueTitle      string  `json:"issue_title"`
	ProjectName     *string `json:"project_name"`
	IssueScope      string  `json:"scope"`
	Status          string  `json:"status"`
	HasPR           bool    `json:"has_pr"`
	IsSynced        bool    `json:"is_synced"`
	Provider        string  `json:"provider"`
}

func NewJSONPrinter() *JSONPrinter {
	return &JSONPrinter{}
}

func (p *JSONPrinter) PrintProjects(projects []shared.Project) {
	parsedProjects := make([]Project, len(projects))

	for i, project := range projects {
		parsedProjects[i] = Project{
			Name:     project.Name,
			Scope:    project.GetScope(),
			Provider: string(project.Provider),
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

func (p *JSONPrinter) PrintIssues(issues []shared.Issue, reportErrors []sync.ReportError) {
	parsedIssues := make([]Issue, len(issues))

	for i, issue := range issues {
		parsedIssues[i] = Issue{
			Identifier:  issue.Identifier,
			Scope:       issue.Scope,
			Title:       issue.Title,
			HasLinkedPR: issue.HasLinkedPR,
			Provider:    string(issue.Provider),
		}
	}

	bytes, err := json.Marshal(struct {
		Issues []Issue `json:"issues"`
		Errors []Error `json:"errors"`
	}{Issues: parsedIssues, Errors: parseErrors(reportErrors)})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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
		Errors:    parseErrors(report.Errors),
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

func (p *JSONPrinter) PrintStatus(rows []status.Row, reportErrors []sync.ReportError) {
	parsedRows := make([]StatusRow, len(rows))

	for i, row := range rows {
		parsedRows[i] = StatusRow{
			IssueIdentifier: row.Identifier,
			IssueTitle:      row.IssueTitle,
			Status:          row.Status,
			IssueScope:      row.IssueScope,
			ProjectName:     row.ProjectName,
			HasPR:           row.HasPR,
			IsSynced:        row.IsSynced,
			Provider:        string(row.Provider),
		}
	}

	bytes, err := json.Marshal(struct {
		Rows   []StatusRow `json:"rows"`
		Errors []Error     `json:"errors"`
	}{Rows: parsedRows, Errors: parseErrors(reportErrors)})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Print(string(bytes))
}

func parseErrors(reportErrors []sync.ReportError) []Error {
	errors := make([]Error, 0, len(reportErrors))

	for _, error := range reportErrors {
		errors = append(errors, Error{
			Provider:   string(error.Provider),
			Project:    error.Project,
			Identifier: error.Identifier,
			Message:    error.Error,
		})
	}

	return errors
}
