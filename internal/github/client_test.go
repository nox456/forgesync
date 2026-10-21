package github

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	gh "github.com/google/go-github/v88/github"
	"github.com/nox456/forgesync/internal/shared"
)

// issueJSON renders one entry of GitHub's GET /issues response.
func issueJSON(number int, state, repoOwner, repoName, body string, extra string) string {
	closedAt := "null"
	if state == "closed" {
		closedAt = `"2026-05-25T18:00:00Z"`
	}
	return fmt.Sprintf(`{
		"number": %d,
		"title": "Issue %d",
		"html_url": "https://github.com/%s/%s/issues/%d",
		"body": %q,
		"state": %q,
		"labels": [{"name": "bug"}],
		"repository": {"name": %q, "full_name": "%s/%s", "owner": {"login": %q}},
		"created_at": "2026-05-01T09:00:00Z",
		"updated_at": "2026-05-20T10:30:00Z",
		"closed_at": %s
		%s
	}`, number, number, repoOwner, repoName, number, body, state, repoName, repoOwner, repoName, repoOwner, closedAt, extra)
}

// linkedPRsJSON renders the GraphQL answer hasLinkedPR reads.
func linkedPRsJSON(drafts ...bool) string {
	nodes := make([]string, len(drafts))
	for i, draft := range drafts {
		nodes[i] = fmt.Sprintf(`{"number": %d, "isDraft": %t}`, 100+i, draft)
	}
	return fmt.Sprintf(`{"data":{"repository":{"issue":{"closedByPullRequestsReferences":{"nodes":[%s]}}}}}`,
		strings.Join(nodes, ","))
}

// testClient points a Client at a local server that answers GET /issues with
// issues and POST /graphql with graphql. An empty graphql fails the test if
// the client asks for linked PRs at all.
func testClient(t *testing.T, issuesStatus int, issues []string, graphql string) *Client {
	t.Helper()

	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/issues":
			w.WriteHeader(issuesStatus)
			fmt.Fprintf(w, "[%s]", strings.Join(issues, ","))
		case r.Method == http.MethodPost && r.URL.Path == "/graphql":
			if graphql == "" {
				t.Errorf("unexpected linked-PR query")
			}
			fmt.Fprint(w, graphql)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	baseURL := server.URL + "/"
	ghClient, err := gh.NewClient(gh.WithURLs(&baseURL, &baseURL))
	if err != nil {
		t.Fatalf("github.NewClient() error: %v", err)
	}

	return &Client{Token: "token", gh: ghClient}
}

func TestClientName(t *testing.T) {
	if got := (&Client{}).Name(); got != shared.ProviderGithub {
		t.Errorf("Name() = %q, want %q", got, shared.ProviderGithub)
	}
}

func TestFetchAssignedIssues(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Run("a closed issue maps to a provider-aware shared.Issue", func(t *testing.T) {
			client := testClient(t, 200,
				[]string{issueJSON(12, "closed", "Nox456", "ForgeSync", "Fix it\n\nplease", "")},
				"")

			got, err := client.FetchAssignedIssues(context.Background(), "")
			if err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			closedAt := time.Date(2026, 5, 25, 18, 0, 0, 0, time.UTC)
			want := []shared.Issue{{
				Provider:    shared.ProviderGithub,
				Identifier:  "#12",
				Title:       "Issue 12",
				URL:         "https://github.com/Nox456/ForgeSync/issues/12",
				Body:        "Fix it\nplease",
				State:       "closed",
				Labels:      []string{"bug"},
				Scope:       "Nox456/ForgeSync",
				CreatedAt:   time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 5, 20, 10, 30, 0, 0, time.UTC),
				ClosedAt:    &closedAt,
				HasLinkedPR: false,
			}}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("FetchAssignedIssues() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("an open issue with a ready PR has a linked PR", func(t *testing.T) {
			client := testClient(t, 200,
				[]string{issueJSON(7, "open", "owner", "repo", "", "")},
				linkedPRsJSON(false))

			got, err := client.FetchAssignedIssues(context.Background(), "")
			if err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			if len(got) != 1 {
				t.Fatalf("got %d issues, want 1", len(got))
			}
			if !got[0].HasLinkedPR {
				t.Errorf("HasLinkedPR = false, want true")
			}
			if got[0].Identifier != "#7" {
				t.Errorf("Identifier = %q, want %q", got[0].Identifier, "#7")
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("a draft-only PR does not count as linked", func(t *testing.T) {
			client := testClient(t, 200,
				[]string{issueJSON(7, "open", "owner", "repo", "", "")},
				linkedPRsJSON(true))

			got, err := client.FetchAssignedIssues(context.Background(), "")
			if err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			if len(got) != 1 || got[0].HasLinkedPR {
				t.Errorf("issues = %+v, want one issue without a linked PR", got)
			}
		})

		t.Run("pull requests in the list are skipped", func(t *testing.T) {
			client := testClient(t, 200,
				[]string{
					issueJSON(1, "closed", "owner", "repo", "", ""),
					issueJSON(2, "closed", "owner", "repo", "", `, "pull_request": {"url": "https://api.github.com/repos/owner/repo/pulls/2"}`),
				},
				"")

			got, err := client.FetchAssignedIssues(context.Background(), "")
			if err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			gotIDs := make([]string, len(got))
			for i, issue := range got {
				gotIDs[i] = issue.Identifier
			}
			if diff := cmp.Diff([]string{"#1"}, gotIDs); diff != "" {
				t.Errorf("identifiers mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("the repo filter keeps only that repo, ignoring case", func(t *testing.T) {
			client := testClient(t, 200,
				[]string{
					issueJSON(1, "closed", "Owner", "Repo-A", "", ""),
					issueJSON(2, "closed", "owner", "repo-b", "", ""),
				},
				"")

			got, err := client.FetchAssignedIssues(context.Background(), "owner/repo-a")
			if err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			if len(got) != 1 || got[0].Identifier != "#1" || got[0].Scope != "Owner/Repo-A" {
				t.Errorf("issues = %+v, want only #1 with scope Owner/Repo-A", got)
			}
		})
	})

	t.Run("error path", func(t *testing.T) {
		t.Run("a failed issues request is returned as an error", func(t *testing.T) {
			client := testClient(t, 500, nil, "")

			got, err := client.FetchAssignedIssues(context.Background(), "")
			if err == nil {
				t.Fatalf("FetchAssignedIssues() = %+v, want an error", got)
			}
			if got != nil {
				t.Errorf("FetchAssignedIssues() issues = %+v, want nil", got)
			}
		})

		t.Run("a GraphQL error while checking linked PRs is returned", func(t *testing.T) {
			client := testClient(t, 200,
				[]string{issueJSON(7, "open", "owner", "repo", "", "")},
				`{"errors":[{"message":"rate limited"}]}`)

			got, err := client.FetchAssignedIssues(context.Background(), "")
			if err == nil {
				t.Fatalf("FetchAssignedIssues() = %+v, want an error", got)
			}
			if !strings.Contains(err.Error(), "rate limited") {
				t.Errorf("error = %q, want it to carry the GraphQL message", err)
			}
		})
	})
}
