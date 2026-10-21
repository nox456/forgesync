package notion

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/nox456/forgesync/internal/shared"
)

// projectRow is one Projects database row as the test wants Notion to return
// it. An empty string renders as an empty rich-text array (or a null select),
// which is how Notion sends a cell nobody filled in.
type projectRow struct {
	id, name, repo, teamKey, provider string
}

func richText(text string) string {
	if text == "" {
		return "[]"
	}
	return fmt.Sprintf(`[{"plain_text": %q}]`, text)
}

func (r projectRow) json() string {
	selectValue := "null"
	if r.provider != "" {
		selectValue = fmt.Sprintf(`{"name": %q}`, r.provider)
	}

	return fmt.Sprintf(`{
		"object": "page",
		"id": %q,
		"properties": {
			"name": {"title": %s},
			"repo": {"rich_text": %s},
			"teamKey": {"rich_text": %s},
			"provider": {"select": %s}
		}
	}`, r.id, richText(r.name), richText(r.repo), richText(r.teamKey), selectValue)
}

// projectsClient returns a client whose single Projects query is answered
// with rows on one page.
func projectsClient(rows ...projectRow) *Client {
	results := make([]string, len(rows))
	for i, row := range rows {
		results[i] = row.json()
	}
	page := fmt.Sprintf(`{"object":"list","has_more":false,"next_cursor":"","results":[%s]}`, joinComma(results))

	return &Client{
		Token:            "token",
		ProjectsSourceId: "projects",
		HTTP: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return jsonResponse(200, page), nil
			}),
		},
	}
}

func TestListProjects(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Run("decodes a GitHub and a Linear project", func(t *testing.T) {
			client := projectsClient(
				projectRow{id: "p-gh", name: "ForgeSync", repo: "nox456/forgesync", provider: "GitHub"},
				projectRow{id: "p-lin", name: "Engineering", teamKey: "ENG", provider: "Linear"},
			)

			got, err := client.ListProjects(context.Background(), shared.ProjectFilter{})
			if err != nil {
				t.Fatalf("ListProjects() unexpected error: %v", err)
			}

			want := []shared.Project{
				{PageID: "p-gh", Name: "ForgeSync", Repo: "nox456/forgesync", Provider: shared.ProviderGithub},
				{PageID: "p-lin", Name: "Engineering", TeamKey: "ENG", Provider: shared.ProviderLinear},
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("ListProjects() mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		cases := []struct {
			name         string
			row          projectRow
			wantProvider shared.Provider
		}{
			{name: "an upper-case option maps to the provider", row: projectRow{id: "p", name: "A", provider: "LINEAR"}, wantProvider: shared.ProviderLinear},
			{name: "a lower-case option maps to the provider", row: projectRow{id: "p", name: "A", provider: "github"}, wantProvider: shared.ProviderGithub},
			{name: "an empty select is kept as an empty provider", row: projectRow{id: "p", name: "A", repo: "owner/repo"}, wantProvider: ""},
			{name: "an unknown option is kept as its raw value", row: projectRow{id: "p", name: "A", provider: "GitLab"}, wantProvider: "GitLab"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				got, err := projectsClient(tc.row).ListProjects(context.Background(), shared.ProjectFilter{})
				if err != nil {
					t.Fatalf("ListProjects() unexpected error: %v", err)
				}
				if len(got) != 1 {
					t.Fatalf("ListProjects() returned %d projects, want 1 (projects must not be dropped here)", len(got))
				}
				if got[0].Provider != tc.wantProvider {
					t.Errorf("Provider = %q, want %q", got[0].Provider, tc.wantProvider)
				}
			})
		}

		t.Run("empty name, repo and team key cells decode as empty strings", func(t *testing.T) {
			got, err := projectsClient(projectRow{id: "p-blank"}).ListProjects(context.Background(), shared.ProjectFilter{})
			if err != nil {
				t.Fatalf("ListProjects() unexpected error: %v", err)
			}

			want := []shared.Project{{PageID: "p-blank"}}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("ListProjects() mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("error path", func(t *testing.T) {
		cases := []struct {
			name   string
			status int
			body   string
		}{
			{name: "an API error status is returned", status: 500, body: `{"message":"internal"}`},
			{name: "a malformed body is returned as an error", status: 200, body: `{"results": [`},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				client := &Client{
					Token:            "token",
					ProjectsSourceId: "projects",
					HTTP: &http.Client{
						Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
							return jsonResponse(tc.status, tc.body), nil
						}),
					},
				}

				got, err := client.ListProjects(context.Background(), shared.ProjectFilter{})
				if err == nil {
					t.Fatalf("ListProjects() = %+v, want an error", got)
				}
				if tc.status >= 400 && !strings.Contains(err.Error(), fmt.Sprint(tc.status)) {
					t.Errorf("error = %q, want it to mention the status code %d", err, tc.status)
				}
			})
		}
	})
}

func TestListProjectsFilters(t *testing.T) {
	rows := []projectRow{
		{id: "p-gh", name: "ForgeSync", repo: "nox456/forgesync", provider: "GitHub"},
		{id: "p-gh2", name: "Other repo", repo: "nox456/other", provider: "GitHub"},
		{id: "p-lin", name: "Engineering", teamKey: "ENG", provider: "Linear"},
		{id: "p-empty", name: "No provider", repo: "nox456/forgesync"},
	}

	cases := []struct {
		name    string
		filter  shared.ProjectFilter
		wantIDs []string
	}{
		// happy path
		{
			name:    "an empty filter returns every project, broken ones included",
			filter:  shared.ProjectFilter{},
			wantIDs: []string{"p-gh", "p-gh2", "p-lin", "p-empty"},
		},
		{
			name:    "a provider filter keeps only that provider",
			filter:  shared.ProjectFilter{Provider: shared.ProviderLinear},
			wantIDs: []string{"p-lin"},
		},
		{
			name:    "a repo filter keeps every project with that repo",
			filter:  shared.ProjectFilter{Repo: "nox456/forgesync"},
			wantIDs: []string{"p-gh", "p-empty"},
		},
		{
			name:    "a team key filter keeps the project with that key",
			filter:  shared.ProjectFilter{TeamKey: "ENG"},
			wantIDs: []string{"p-lin"},
		},
		// edge cases
		{
			name:    "the repo filter ignores case",
			filter:  shared.ProjectFilter{Repo: "NOX456/ForgeSync"},
			wantIDs: []string{"p-gh", "p-empty"},
		},
		{
			name:    "the team key filter ignores case",
			filter:  shared.ProjectFilter{TeamKey: "eng"},
			wantIDs: []string{"p-lin"},
		},
		{
			name:    "provider and repo filters must both match",
			filter:  shared.ProjectFilter{Provider: shared.ProviderGithub, Repo: "nox456/forgesync"},
			wantIDs: []string{"p-gh"},
		},
		{
			name:    "a filter nothing matches returns no projects",
			filter:  shared.ProjectFilter{TeamKey: "OPS"},
			wantIDs: nil,
		},
		// error path: none — filtering only drops rows; Notion errors are covered in TestListProjects.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := projectsClient(rows...).ListProjects(context.Background(), tc.filter)
			if err != nil {
				t.Fatalf("ListProjects() unexpected error: %v", err)
			}

			gotIDs := make([]string, 0, len(got))
			for _, project := range got {
				gotIDs = append(gotIDs, project.PageID)
			}
			if diff := cmp.Diff(tc.wantIDs, gotIDs, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("ListProjects(%+v) ids mismatch (-want +got):\n%s", tc.filter, diff)
			}
		})
	}
}
