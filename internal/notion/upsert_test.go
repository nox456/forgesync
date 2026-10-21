package notion

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nox456/forgesync/internal/shared"
)

// sentRequest is one request the client made, with its JSON body decoded
// generically so a test can assert the exact shape Notion receives.
type sentRequest struct {
	method string
	path   string
	body   map[string]any
}

// upsertClient returns a client that records every request and answers each
// one with status.
func upsertClient(t *testing.T, status int, sent *[]sentRequest) *Client {
	t.Helper()
	return &Client{
		Token:           "token",
		StoriesSourceId: "stories",
		HTTP: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				raw, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				var body map[string]any
				if err := json.Unmarshal(raw, &body); err != nil {
					return nil, err
				}
				*sent = append(*sent, sentRequest{method: req.Method, path: req.URL.Path, body: body})
				return jsonResponse(status, `{"object":"page"}`), nil
			}),
		},
	}
}

func storyInputFor(identifier string) shared.StoryInput {
	return shared.StoryInput{
		Name:         "Add login flow",
		Project:      "p1",
		Issue:        identifier,
		Url:          "https://github.com/owner/repo/issues/12",
		Status:       "Not started",
		Labels:       []string{"feature"},
		LastWorkedAt: "2026-05-20 10:30",
	}
}

// property returns properties.<name> from a decoded request body, or nil.
func property(body map[string]any, name string) any {
	properties, _ := body["properties"].(map[string]any)
	return properties[name]
}

func TestUpsertStory(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Run("creating a story writes Issue as rich text holding the identifier", func(t *testing.T) {
			var sent []sentRequest
			client := upsertClient(t, 200, &sent)

			result, err := client.UpsertStory(context.Background(), storyInputFor("#12"), false, nil)
			if err != nil {
				t.Fatalf("UpsertStory() unexpected error: %v", err)
			}

			if diff := cmp.Diff(&shared.UpsertResult{Created: true}, result); diff != "" {
				t.Errorf("UpsertStory() result mismatch (-want +got):\n%s", diff)
			}
			if len(sent) != 1 {
				t.Fatalf("made %d requests, want 1", len(sent))
			}
			if sent[0].method != "POST" || sent[0].path != "/v1/pages" {
				t.Errorf("request = %s %s, want POST /v1/pages", sent[0].method, sent[0].path)
			}

			wantIssue := map[string]any{
				"rich_text": []any{
					map[string]any{"text": map[string]any{"content": "#12"}},
				},
			}
			if diff := cmp.Diff(wantIssue, property(sent[0].body, "Issue")); diff != "" {
				t.Errorf("properties.Issue mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("updating a story patches its page", func(t *testing.T) {
			var sent []sentRequest
			client := upsertClient(t, 200, &sent)

			result, err := client.UpsertStory(context.Background(), storyInputFor("#12"), false, &shared.Story{PageID: "story-12", Issue: "#12"})
			if err != nil {
				t.Fatalf("UpsertStory() unexpected error: %v", err)
			}

			if diff := cmp.Diff(&shared.UpsertResult{Updated: true}, result); diff != "" {
				t.Errorf("UpsertStory() result mismatch (-want +got):\n%s", diff)
			}
			if len(sent) != 1 {
				t.Fatalf("made %d requests, want 1 (no body to replace)", len(sent))
			}
			if sent[0].method != "PATCH" || sent[0].path != "/v1/pages/story-12" {
				t.Errorf("request = %s %s, want PATCH /v1/pages/story-12", sent[0].method, sent[0].path)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("an update leaves the Issue property alone", func(t *testing.T) {
			var sent []sentRequest
			client := upsertClient(t, 200, &sent)

			if _, err := client.UpsertStory(context.Background(), storyInputFor("#12"), false, &shared.Story{PageID: "story-12"}); err != nil {
				t.Fatalf("UpsertStory() unexpected error: %v", err)
			}

			if len(sent) != 1 {
				t.Fatalf("made %d requests, want 1", len(sent))
			}
			if got := property(sent[0].body, "Issue"); got != nil {
				t.Errorf("properties.Issue = %v, want it absent on update", got)
			}
		})

		t.Run("a Linear identifier is written verbatim", func(t *testing.T) {
			var sent []sentRequest
			client := upsertClient(t, 200, &sent)

			if _, err := client.UpsertStory(context.Background(), storyInputFor("ENG-123"), false, nil); err != nil {
				t.Fatalf("UpsertStory() unexpected error: %v", err)
			}

			wantIssue := map[string]any{
				"rich_text": []any{
					map[string]any{"text": map[string]any{"content": "ENG-123"}},
				},
			}
			if diff := cmp.Diff(wantIssue, property(sent[0].body, "Issue")); diff != "" {
				t.Errorf("properties.Issue mismatch (-want +got):\n%s", diff)
			}
		})

		cases := []struct {
			name     string
			existing *shared.Story
			want     *shared.UpsertResult
		}{
			{name: "a dry-run create sends nothing and reports created", existing: nil, want: &shared.UpsertResult{Created: true}},
			{name: "a dry-run update sends nothing and reports updated", existing: &shared.Story{PageID: "story-12"}, want: &shared.UpsertResult{Updated: true}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var sent []sentRequest
				client := upsertClient(t, 200, &sent)

				result, err := client.UpsertStory(context.Background(), storyInputFor("#12"), true, tc.existing)
				if err != nil {
					t.Fatalf("UpsertStory() unexpected error: %v", err)
				}

				if diff := cmp.Diff(tc.want, result); diff != "" {
					t.Errorf("UpsertStory() result mismatch (-want +got):\n%s", diff)
				}
				if len(sent) != 0 {
					t.Errorf("made %d requests in a dry run, want 0", len(sent))
				}
			})
		}
	})

	t.Run("error path", func(t *testing.T) {
		cases := []struct {
			name     string
			existing *shared.Story
		}{
			{name: "a rejected create returns an error and no result", existing: nil},
			{name: "a rejected update returns an error and no result", existing: &shared.Story{PageID: "story-12"}},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				var sent []sentRequest
				client := upsertClient(t, 400, &sent)

				result, err := client.UpsertStory(context.Background(), storyInputFor("#12"), false, tc.existing)
				if err == nil {
					t.Fatalf("UpsertStory() = %+v, want an error", result)
				}
				if result != nil {
					t.Errorf("UpsertStory() result = %+v, want nil", result)
				}
			})
		}
	})
}
