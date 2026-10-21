package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/status"
	"github.com/nox456/forgesync/internal/sync"
)

// captureStdout runs print with os.Stdout redirected to a pipe and returns
// everything it wrote. Printers are built inside print, so the text printer's
// tabwriter also binds to the pipe.
func captureStdout(t *testing.T, print func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}
	prev := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = prev }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, reader)
		done <- buf.String()
	}()

	print()

	writer.Close()
	return <-done
}

// decode unmarshals printed JSON into a generic value so assertions check the
// field names consumers see, not the Go struct that produced them.
func decode(t *testing.T, out string) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	return got
}

func TestJSONPrintReport(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Run("each error becomes one object with only its own fields", func(t *testing.T) {
			report := &sync.Report{
				Created: 1,
				Errors: []sync.ReportError{
					{Identifier: "#12", Provider: shared.ProviderGithub, Project: "ForgeSync", Error: "boom"},
					{Project: "No provider", Error: "project has no provider"},
				},
			}

			got := decode(t, captureStdout(t, func() { NewJSONPrinter().PrintReport(report) }))

			want := map[string]any{"report": map[string]any{
				"created":   1.0,
				"updated":   0.0,
				"skipped":   0.0,
				"unchanged": 0.0,
				"errors": []any{
					map[string]any{"identifier": "#12", "provider": "github", "project": "ForgeSync", "message": "boom"},
					map[string]any{"project": "No provider", "message": "project has no provider"},
				},
			}}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("PrintReport() JSON mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		cases := []struct {
			name   string
			errors []sync.ReportError
		}{
			{name: "no errors marshal as an empty array, not null", errors: nil},
			{name: "an empty errors slice marshals as an empty array", errors: []sync.ReportError{}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				out := captureStdout(t, func() { NewJSONPrinter().PrintReport(&sync.Report{Errors: tc.errors}) })

				if !bytes.Contains([]byte(out), []byte(`"errors":[]`)) {
					t.Errorf("PrintReport() = %s, want it to contain %q", out, `"errors":[]`)
				}
			})
		}
	})

	// error path: none — the report is plain data, so json.Marshal cannot fail.
}

func TestJSONPrintIssues(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Run("issues carry provider, identifier and scope", func(t *testing.T) {
			issues := []shared.Issue{
				{Provider: shared.ProviderGithub, Identifier: "#12", Scope: "owner/repo", Title: "Fix it", HasLinkedPR: true},
				{Provider: shared.ProviderLinear, Identifier: "ENG-7", Scope: "ENG", Title: "Plan it"},
			}

			got := decode(t, captureStdout(t, func() { NewJSONPrinter().PrintIssues(issues) }))

			want := map[string]any{"issues": []any{
				map[string]any{"provider": "github", "identifier": "#12", "scope": "owner/repo", "title": "Fix it", "has_linked_pr": true},
				map[string]any{"provider": "linear", "identifier": "ENG-7", "scope": "ENG", "title": "Plan it", "has_linked_pr": false},
			}}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("PrintIssues() JSON mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("no issues marshal as an empty array", func(t *testing.T) {
			got := decode(t, captureStdout(t, func() { NewJSONPrinter().PrintIssues(nil) }))

			if diff := cmp.Diff(map[string]any{"issues": []any{}}, got); diff != "" {
				t.Errorf("PrintIssues() JSON mismatch (-want +got):\n%s", diff)
			}
		})
	})

	// error path: none — the issues are plain data, so json.Marshal cannot fail.
}

func TestJSONPrintStatus(t *testing.T) {
	projectName := "ForgeSync"

	t.Run("happy path", func(t *testing.T) {
		t.Run("rows carry identifier, scope and provider under the contract names", func(t *testing.T) {
			rows := []status.Row{{
				Identifier:  "#12",
				IssueTitle:  "Fix it",
				ProjectName: &projectName,
				IssueScope:  "owner/repo",
				Status:      "In PR",
				HasPR:       true,
				IsSynced:    true,
				Provider:    shared.ProviderGithub,
			}}

			got := decode(t, captureStdout(t, func() { NewJSONPrinter().PrintStatus(rows) }))

			want := map[string]any{"rows": []any{map[string]any{
				"identifier":   "#12",
				"issue_title":  "Fix it",
				"project_name": "ForgeSync",
				"scope":        "owner/repo",
				"status":       "In PR",
				"has_pr":       true,
				"is_synced":    true,
				"provider":     "github",
			}}}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("PrintStatus() JSON mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("a row without a project has a null project name", func(t *testing.T) {
			rows := []status.Row{{Identifier: "#1", IssueScope: "owner/unknown", Provider: shared.ProviderGithub}}

			got := decode(t, captureStdout(t, func() { NewJSONPrinter().PrintStatus(rows) }))

			row := got["rows"].([]any)[0].(map[string]any)
			value, present := row["project_name"]
			if !present || value != nil {
				t.Errorf("project_name = %v (present %v), want null", value, present)
			}
		})
	})

	// error path: none — the rows are plain data, so json.Marshal cannot fail.
}

func TestJSONPrintProjects(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Run("the scope is the repo for GitHub and the team key for Linear", func(t *testing.T) {
			projects := []shared.Project{
				{PageID: "p1", Name: "ForgeSync", Repo: "owner/repo", Provider: shared.ProviderGithub},
				{PageID: "p2", Name: "Engineering", TeamKey: "ENG", Provider: shared.ProviderLinear},
			}

			got := decode(t, captureStdout(t, func() { NewJSONPrinter().PrintProjects(projects) }))

			want := map[string]any{"projects": []any{
				map[string]any{"name": "ForgeSync", "provider": "github", "scope": "owner/repo"},
				map[string]any{"name": "Engineering", "provider": "linear", "scope": "ENG"},
			}}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("PrintProjects() JSON mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("a project with an empty provider prints an empty provider", func(t *testing.T) {
			projects := []shared.Project{{PageID: "p1", Name: "Broken", TeamKey: "ENG"}}

			got := decode(t, captureStdout(t, func() { NewJSONPrinter().PrintProjects(projects) }))

			project := got["projects"].([]any)[0].(map[string]any)
			if project["provider"] != "" {
				t.Errorf("provider = %v, want empty string", project["provider"])
			}
		})
	})

	// error path: none — the projects are plain data, so json.Marshal cannot fail.
}
