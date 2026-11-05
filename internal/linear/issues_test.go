package linear

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/nox456/forgesync/internal/shared"
)

var testNow = time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)

// issueJSON renders one assigned-issue node in Linear's response format.
func issueJSON(identifier string) string {
	return fmt.Sprintf(`{"id":"id-%[1]s","identifier":%[1]q,"title":"Issue %[1]s","url":"https://linear.app/acme/issue/%[1]s",
		"description":"","createdAt":"2026-10-01T09:00:00.000Z","updatedAt":"2026-11-02T10:00:00.000Z","completedAt":null,
		"state":{"type":"started"},"team":{"key":"ENG"},"labels":{"nodes":[]},"attachments":{"nodes":[]}}`, identifier)
}

// issuesPage renders one page of viewer.assignedIssues holding the given
// identifiers.
func issuesPage(identifiers []string, hasNextPage bool, endCursor string) string {
	nodes := make([]string, len(identifiers))
	for i, identifier := range identifiers {
		nodes[i] = issueJSON(identifier)
	}

	return fmt.Sprintf(`{"data":{"viewer":{"assignedIssues":{"nodes":[%s],"pageInfo":{"hasNextPage":%t,"endCursor":%q}}}}}`,
		strings.Join(nodes, ","), hasNextPage, endCursor)
}

// sentVariables is the variables object as it travels on the wire. The
// pointers tell an absent key from an empty one.
type sentVariables struct {
	After  *string `json:"after"`
	Filter struct {
		UpdatedAt struct {
			Gte string `json:"gte"`
		} `json:"updatedAt"`
		State struct {
			Type struct {
				Neq string `json:"neq"`
			} `json:"type"`
		} `json:"state"`
		Team *struct {
			Key struct {
				EqIgnoreCase string `json:"eqIgnoreCase"`
			} `json:"key"`
		} `json:"team"`
	} `json:"filter"`
}

// servePages returns a client on a fixed clock that answers each request
// with the next page and records the variables every request carried.
func servePages(t *testing.T, pages []string, sent *[]sentVariables) *Client {
	t.Helper()

	client := newTestClient(func(req *http.Request) (*http.Response, error) {
		var payload struct {
			Variables sentVariables `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			return nil, err
		}
		*sent = append(*sent, payload.Variables)

		if len(*sent) > len(pages) {
			t.Errorf("client requested page %d, only %d exist", len(*sent), len(pages))
			return nil, fmt.Errorf("too many requests")
		}

		return respond(200, pages[len(*sent)-1]), nil
	})
	client.Now = func() time.Time { return testNow }

	return client
}

func identifiers(issues []shared.Issue) []string {
	out := make([]string, len(issues))
	for i, issue := range issues {
		out[i] = issue.Identifier
	}
	return out
}

func TestFetchAssignedIssues(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Run("follows the cursor across pages and returns every issue in order", func(t *testing.T) {
			var sent []sentVariables
			client := servePages(t, []string{
				issuesPage([]string{"ENG-1", "ENG-2"}, true, "cursor-2"),
				issuesPage([]string{"ENG-3"}, true, "cursor-3"),
				issuesPage([]string{"ENG-4"}, false, ""),
			}, &sent)

			issues, err := client.FetchAssignedIssues(context.Background(), "")
			if err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			if diff := cmp.Diff([]string{"ENG-1", "ENG-2", "ENG-3", "ENG-4"}, identifiers(issues)); diff != "" {
				t.Errorf("issues mismatch (-want +got):\n%s", diff)
			}
			if len(sent) != 3 {
				t.Fatalf("made %d requests, want 3", len(sent))
			}
			if sent[0].After != nil {
				t.Errorf("request 1 after = %q, want no after on the first page", *sent[0].After)
			}
			for i, want := range []string{"cursor-2", "cursor-3"} {
				got := sent[i+1].After
				if got == nil {
					t.Errorf("request %d has no after, want %q", i+2, want)
					continue
				}
				if *got != want {
					t.Errorf("request %d after = %q, want %q", i+2, *got, want)
				}
			}
		})

		t.Run("asks for issues updated in the 7 days before the client's clock", func(t *testing.T) {
			var sent []sentVariables
			client := servePages(t, []string{issuesPage(nil, false, "")}, &sent)

			if _, err := client.FetchAssignedIssues(context.Background(), ""); err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			if got := sent[0].Filter.UpdatedAt.Gte; got != "2026-10-28T12:00:00Z" {
				t.Errorf("filter.updatedAt.gte = %q, want %q", got, "2026-10-28T12:00:00Z")
			}
		})

		t.Run("always leaves triage issues out", func(t *testing.T) {
			var sent []sentVariables
			client := servePages(t, []string{issuesPage(nil, false, "")}, &sent)

			if _, err := client.FetchAssignedIssues(context.Background(), ""); err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			if got := sent[0].Filter.State.Type.Neq; got != "triage" {
				t.Errorf("filter.state.type.neq = %q, want %q", got, "triage")
			}
		})

		t.Run("a team key narrows to that team, ignoring case", func(t *testing.T) {
			var sent []sentVariables
			client := servePages(t, []string{issuesPage(nil, false, "")}, &sent)

			if _, err := client.FetchAssignedIssues(context.Background(), "eng"); err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			if sent[0].Filter.Team == nil {
				t.Fatalf("filter.team is absent, want it set for a team key")
			}
			if got := sent[0].Filter.Team.Key.EqIgnoreCase; got != "eng" {
				t.Errorf("filter.team.key.eqIgnoreCase = %q, want %q", got, "eng")
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("without a team key the team filter is left out", func(t *testing.T) {
			var sent []sentVariables
			client := servePages(t, []string{issuesPage(nil, false, "")}, &sent)

			if _, err := client.FetchAssignedIssues(context.Background(), ""); err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			if sent[0].Filter.Team != nil {
				t.Errorf("filter.team = %+v, want it absent without a team key", *sent[0].Filter.Team)
			}
		})

		t.Run("a page that has a next page but no cursor stops", func(t *testing.T) {
			var sent []sentVariables
			client := servePages(t, []string{issuesPage([]string{"ENG-1"}, true, "")}, &sent)

			issues, err := client.FetchAssignedIssues(context.Background(), "")
			if err != nil {
				t.Fatalf("FetchAssignedIssues() unexpected error: %v", err)
			}

			if len(sent) != 1 {
				t.Errorf("made %d requests, want 1", len(sent))
			}
			if diff := cmp.Diff([]string{"ENG-1"}, identifiers(issues)); diff != "" {
				t.Errorf("issues mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("error path", func(t *testing.T) {
		t.Run("an error on a later page returns the error and no issues", func(t *testing.T) {
			var sent []sentVariables
			client := servePages(t, []string{
				issuesPage([]string{"ENG-1"}, true, "cursor-2"),
				`{"data":null,"errors":[{"message":"rate limited"}]}`,
			}, &sent)

			issues, err := client.FetchAssignedIssues(context.Background(), "")

			if err == nil || !strings.Contains(err.Error(), "rate limited") {
				t.Errorf("FetchAssignedIssues() error = %v, want the second page's GraphQL error", err)
			}
			if issues != nil {
				t.Errorf("FetchAssignedIssues() = %v alongside an error, want nil", identifiers(issues))
			}
		})
	})
}

// decodeNode decodes one node from Linear's JSON, so the mapping tests also
// exercise the response struct tags.
func decodeNode(t *testing.T, raw string) issueNode {
	t.Helper()

	var node issueNode
	if err := json.Unmarshal([]byte(raw), &node); err != nil {
		t.Fatalf("decode node: %v", err)
	}
	return node
}

func TestToIssue(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Run("maps a completed issue to shared.Issue", func(t *testing.T) {
			node := decodeNode(t, `{"id":"9f1c","identifier":"ENG-123","title":"Fix login redirect",
				"url":"https://linear.app/acme/issue/ENG-123/fix-login-redirect","description":"Steps to reproduce",
				"createdAt":"2026-10-20T09:00:00.000Z","updatedAt":"2026-11-03T15:30:00.000Z","completedAt":"2026-11-03T15:30:00.000Z",
				"state":{"type":"completed"},"team":{"key":"ENG"},
				"labels":{"nodes":[{"name":"bug"},{"name":"auth"}]},
				"attachments":{"nodes":[{"sourceType":"github","url":"https://github.com/acme/web/pull/42"}]}}`)

			closedAt := time.Date(2026, 11, 3, 15, 30, 0, 0, time.UTC)
			want := shared.Issue{
				Provider:    shared.ProviderLinear,
				Identifier:  "ENG-123",
				Title:       "Fix login redirect",
				URL:         "https://linear.app/acme/issue/ENG-123/fix-login-redirect",
				Body:        "Steps to reproduce",
				State:       "completed",
				Labels:      []string{"bug", "auth"},
				Scope:       "ENG",
				UpdatedAt:   time.Date(2026, 11, 3, 15, 30, 0, 0, time.UTC),
				CreatedAt:   time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC),
				ClosedAt:    &closedAt,
				HasLinkedPR: true,
			}

			if diff := cmp.Diff(want, toIssue(node)); diff != "" {
				t.Errorf("toIssue() mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("a canceled issue has no ClosedAt", func(t *testing.T) {
			node := decodeNode(t, `{"identifier":"ENG-7","createdAt":"2026-10-20T09:00:00Z","updatedAt":"2026-11-03T15:30:00Z",
				"completedAt":null,"state":{"type":"canceled"},"team":{"key":"ENG"},"labels":{"nodes":[]},"attachments":{"nodes":[]}}`)

			if got := toIssue(node).ClosedAt; got != nil {
				t.Errorf("ClosedAt = %v, want nil for a canceled issue", *got)
			}
		})

		t.Run("an issue without labels gets an empty slice, not nil", func(t *testing.T) {
			node := decodeNode(t, `{"identifier":"ENG-8","createdAt":"2026-10-20T09:00:00Z","updatedAt":"2026-11-03T15:30:00Z",
				"state":{"type":"started"},"team":{"key":"ENG"},"labels":{"nodes":[]},"attachments":{"nodes":[]}}`)

			got := toIssue(node).Labels
			if got == nil || len(got) != 0 {
				t.Errorf("Labels = %#v, want an empty, non-nil slice", got)
			}
		})

		t.Run("the body is the description without Linear images or blank lines", func(t *testing.T) {
			node := decodeNode(t, `{"identifier":"ENG-9","createdAt":"2026-10-20T09:00:00Z","updatedAt":"2026-11-03T15:30:00Z",
				"description":"Before\n\n![shot.png](https://uploads.linear.app/a/b/c)\nAfter",
				"state":{"type":"started"},"team":{"key":"ENG"},"labels":{"nodes":[]},"attachments":{"nodes":[]}}`)

			if got := toIssue(node).Body; got != "Before\nAfter" {
				t.Errorf("Body = %q, want %q", got, "Before\nAfter")
			}
		})

		t.Run("a null description becomes an empty body", func(t *testing.T) {
			node := decodeNode(t, `{"identifier":"ENG-10","createdAt":"2026-10-20T09:00:00Z","updatedAt":"2026-11-03T15:30:00Z",
				"description":null,"state":{"type":"started"},"team":{"key":"ENG"},"labels":{"nodes":[]},"attachments":{"nodes":[]}}`)

			if got := toIssue(node).Body; got != "" {
				t.Errorf("Body = %q, want an empty body", got)
			}
		})
	})

	// error path: none — toIssue is total and cannot fail.
}

func TestHasLinkedPR(t *testing.T) {
	cases := []struct {
		name        string
		attachments []attachmentNode
		want        bool
	}{
		// happy path
		{
			name:        "a GitHub pull request counts",
			attachments: []attachmentNode{{SourceType: "github", URL: "https://github.com/acme/web/pull/42"}},
			want:        true,
		},
		{
			name: "a GitHub pull request after other attachments counts",
			attachments: []attachmentNode{
				{SourceType: "slack", URL: "https://acme.slack.com/archives/C01/p1700000000"},
				{SourceType: "github", URL: "https://github.com/acme/web/pull/42"},
			},
			want: true,
		},
		// edge cases
		{
			name:        "a GitHub issue link does not count",
			attachments: []attachmentNode{{SourceType: "github", URL: "https://github.com/acme/web/issues/7"}},
			want:        false,
		},
		{
			name:        "a GitHub commit link does not count",
			attachments: []attachmentNode{{SourceType: "github", URL: "https://github.com/acme/web/commit/3f9a1c2"}},
			want:        false,
		},
		{
			name:        "a non-GitHub source with /pull/ in the URL does not count",
			attachments: []attachmentNode{{SourceType: "gitlab", URL: "https://gitlab.com/acme/web/pull/3"}},
			want:        false,
		},
		{
			name:        "a pull request URL from an unknown source does not count",
			attachments: []attachmentNode{{SourceType: "unknown", URL: "https://github.com/acme/web/pull/42"}},
			want:        false,
		},
		{
			name:        "no attachments means no linked PR",
			attachments: nil,
			want:        false,
		},
		// error path: none — hasLinkedPR is total and cannot fail.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasLinkedPR(tc.attachments); got != tc.want {
				t.Errorf("hasLinkedPR() = %v, want %v", got, tc.want)
			}
		})
	}
}
