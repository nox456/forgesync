package output

import (
	"strings"
	"testing"

	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/status"
	"github.com/nox456/forgesync/internal/sync"
)

// lineContaining returns the first output line that contains substr.
func lineContaining(t *testing.T, out, substr string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	t.Fatalf("no line contains %q in:\n%s", substr, out)
	return ""
}

// assertSameColumn fails unless headerCell in the header line and rowCell in
// the row line start at the same offset, which tabwriter only produces when
// both lines have the same number of cells before them.
func assertSameColumn(t *testing.T, header, headerCell, row, rowCell string) {
	t.Helper()
	if h, r := strings.Index(header, headerCell), strings.Index(row, rowCell); h != r {
		t.Errorf("%q starts at column %d but %q starts at %d:\n%s\n%s", headerCell, h, rowCell, r, header, row)
	}
}

func TestTextPrintReportErrors(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Run("an issue error names its provider and identifier", func(t *testing.T) {
			report := &sync.Report{Errors: []sync.ReportError{
				{Identifier: "#12", Provider: shared.ProviderGithub, Project: "ForgeSync", Error: "boom"},
			}}

			out := captureStdout(t, func() { NewTextPrinter().PrintReport(report) })

			if !strings.Contains(out, "* github #12: boom") {
				t.Errorf("PrintReport() output missing %q:\n%s", "* github #12: boom", out)
			}
		})

		t.Run("a project error names the project", func(t *testing.T) {
			report := &sync.Report{Errors: []sync.ReportError{
				{Project: "No provider", Error: "project has no provider"},
			}}

			out := captureStdout(t, func() { NewTextPrinter().PrintReport(report) })

			if !strings.Contains(out, "* No provider: project has no provider") {
				t.Errorf("PrintReport() output missing %q:\n%s", "* No provider: project has no provider", out)
			}
		})

		t.Run("a provider error with no project or issue names the provider", func(t *testing.T) {
			report := &sync.Report{Errors: []sync.ReportError{
				{Provider: shared.ProviderLinear, Error: "linear api 401"},
			}}

			out := captureStdout(t, func() { NewTextPrinter().PrintReport(report) })

			if !strings.Contains(out, "* linear: linear api 401") {
				t.Errorf("PrintReport() output missing %q:\n%s", "* linear: linear api 401", out)
			}
			if strings.Contains(out, "* : ") {
				t.Errorf("PrintReport() printed an error with an empty name:\n%s", out)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("the old hard-coded repo prefix is gone", func(t *testing.T) {
			report := &sync.Report{Errors: []sync.ReportError{
				{Identifier: "ENG-7", Provider: shared.ProviderLinear, Error: "boom"},
			}}

			out := captureStdout(t, func() { NewTextPrinter().PrintReport(report) })

			if strings.Contains(out, "forgesync#") {
				t.Errorf("PrintReport() output still contains %q:\n%s", "forgesync#", out)
			}
		})

		t.Run("no errors prints no errors section", func(t *testing.T) {
			out := captureStdout(t, func() { NewTextPrinter().PrintReport(&sync.Report{Created: 2}) })

			if strings.Contains(out, "Errors:") {
				t.Errorf("PrintReport() printed an errors section without errors:\n%s", out)
			}
		})
	})

	// error path: none — PrintReport only formats data it is given.
}

func TestTextPrintTables(t *testing.T) {
	projectName := "ForgeSync"

	t.Run("happy path", func(t *testing.T) {
		t.Run("issues table has a provider column aligned with its rows", func(t *testing.T) {
			issues := []shared.Issue{{Provider: shared.ProviderGithub, Identifier: "#12", Scope: "owner/repo", Title: "FixTitle", State: "open"}}

			out := captureStdout(t, func() { NewTextPrinter().PrintIssues(issues, nil) })

			header := lineContaining(t, out, "Provider")
			row := lineContaining(t, out, "#12")
			assertSameColumn(t, header, "Provider", row, "github")
			assertSameColumn(t, header, "Scope", row, "owner/repo")
			assertSameColumn(t, header, "Title", row, "FixTitle")
		})

		t.Run("status table has a provider column aligned with its rows", func(t *testing.T) {
			rows := []status.Row{{Provider: shared.ProviderGithub, Identifier: "#12", ProjectName: &projectName, IssueTitle: "FixTitle", Status: "InPR"}}

			out := captureStdout(t, func() { NewTextPrinter().PrintStatus(rows, nil) })

			header := lineContaining(t, out, "Provider")
			row := lineContaining(t, out, "#12")
			assertSameColumn(t, header, "Provider", row, "github")
			assertSameColumn(t, header, "Issue", row, "#12")
			assertSameColumn(t, header, "Title", row, "FixTitle")
		})

		t.Run("projects table shows provider and scope", func(t *testing.T) {
			projects := []shared.Project{
				{Name: "ForgeSync", Repo: "owner/repo", Provider: shared.ProviderGithub},
				{Name: "Engineering", TeamKey: "ENG", Provider: shared.ProviderLinear},
			}

			out := captureStdout(t, func() { NewTextPrinter().PrintProjects(projects) })

			header := lineContaining(t, out, "Provider")
			githubRow := lineContaining(t, out, "ForgeSync")
			linearRow := lineContaining(t, out, "Engineering")
			assertSameColumn(t, header, "Provider", githubRow, "github")
			assertSameColumn(t, header, "Scope", githubRow, "owner/repo")
			assertSameColumn(t, header, "Scope", linearRow, "ENG")
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("a status row without a project shows the issue scope next to its identifier", func(t *testing.T) {
			rows := []status.Row{{Provider: shared.ProviderGithub, Identifier: "#3", IssueScope: "owner/unknown", IssueTitle: "Orphan"}}

			out := captureStdout(t, func() { NewTextPrinter().PrintStatus(rows, nil) })

			row := lineContaining(t, out, "#3")
			if !strings.Contains(row, "[NO PROJECT]") || !strings.Contains(row, "#3 (owner/unknown)") {
				t.Errorf("status row = %q, want [NO PROJECT] and %q", row, "#3 (owner/unknown)")
			}
		})
	})

	// error path: none — the tables only format data they are given.
}

func TestTextPrintTableErrors(t *testing.T) {
	projectName := "ForgeSync"

	printers := []struct {
		name  string
		print func(reportErrors []sync.ReportError)
	}{
		{
			name: "issues",
			print: func(reportErrors []sync.ReportError) {
				issues := []shared.Issue{{Provider: shared.ProviderGithub, Identifier: "#12", Scope: "owner/repo", Title: "FixTitle", State: "open"}}
				NewTextPrinter().PrintIssues(issues, reportErrors)
			},
		},
		{
			name: "status",
			print: func(reportErrors []sync.ReportError) {
				rows := []status.Row{{Provider: shared.ProviderGithub, Identifier: "#12", ProjectName: &projectName, IssueTitle: "FixTitle", Status: "InPR"}}
				NewTextPrinter().PrintStatus(rows, reportErrors)
			},
		},
	}

	t.Run("happy path", func(t *testing.T) {
		reportErrors := []sync.ReportError{
			{Provider: shared.ProviderLinear, Error: "linear api 401"},
			{Project: "Team project", Provider: shared.ProviderLinear, Error: "linear token not configured"},
		}

		for _, p := range printers {
			t.Run(p.name+" prints the errors after the table", func(t *testing.T) {
				out := captureStdout(t, func() { p.print(reportErrors) })

				if !strings.Contains(out, "* linear: linear api 401") {
					t.Errorf("output missing %q:\n%s", "* linear: linear api 401", out)
				}
				if !strings.Contains(out, "* Team project: linear token not configured") {
					t.Errorf("output missing %q:\n%s", "* Team project: linear token not configured", out)
				}
				if row, section := strings.Index(out, "#12"), strings.Index(out, "Errors:"); section < row {
					t.Errorf("errors section at %d, want it after the table row at %d:\n%s", section, row, out)
				}
			})
		}
	})

	t.Run("edge cases", func(t *testing.T) {
		empties := []struct {
			name         string
			reportErrors []sync.ReportError
		}{
			{name: "nil errors", reportErrors: nil},
			{name: "an empty errors slice", reportErrors: []sync.ReportError{}},
		}

		for _, p := range printers {
			for _, empty := range empties {
				t.Run(p.name+" prints no errors section for "+empty.name, func(t *testing.T) {
					out := captureStdout(t, func() { p.print(empty.reportErrors) })

					if strings.Contains(out, "Errors:") {
						t.Errorf("printed an errors section without errors:\n%s", out)
					}
				})
			}
		}
	})

	// error path: none — the printers only format data they are given.
}
