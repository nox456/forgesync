package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// storyPage renders one page of a Notion data-source query holding a single
// story per issue number.
func storyPage(issueNumbers []int, nextCursor string) string {
	results := make([]string, 0, len(issueNumbers))

	for _, number := range issueNumbers {
		results = append(results, fmt.Sprintf(`{
			"object": "page",
			"id": "story-%d",
			"properties": {
				"Issue": {"number": %d},
				"Created time": {"created_time": "2026-08-01T09:00:00.000+00:00"},
				"Labels": {"multi_select": [{"name": "bug"}]},
				"Last Worked At": {"date": {"start": "2026-08-02T10:30:00.000+00:00"}},
				"Finished Date": {"date": {"start": ""}},
				"Status": {"status": {"name": "In progress"}},
				"Project": {"relation": [{"id": "p1"}]},
				"URL": {"url": "https://github.com/owner/repo/issues/%d"},
				"Name": {"title": [{"plain_text": "Story %d"}]}
			}
		}`, number, number, number, number))
	}

	hasMore := nextCursor != ""

	return fmt.Sprintf(`{"object": "list", "results": [%s], "has_more": %t, "next_cursor": %q}`,
		joinComma(results), hasMore, nextCursor)
}

func joinComma(parts []string) string {
	var b bytes.Buffer
	for i, p := range parts {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(p)
	}
	return b.String()
}

func TestListStoriesByProjectFollowsPagination(t *testing.T) {
	pages := []string{
		storyPage([]int{1, 2}, "cursor-2"),
		storyPage([]int{3, 4}, "cursor-3"),
		storyPage([]int{5}, ""),
	}

	var sentCursors []string

	client := &Client{
		Token:           "token",
		StoriesSourceId: "stories",
		HTTP: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}

				var payload StoryFilterPayload
				if err := json.Unmarshal(body, &payload); err != nil {
					return nil, err
				}
				sentCursors = append(sentCursors, payload.StartCursor)

				if len(sentCursors) > len(pages) {
					t.Errorf("client requested page %d, only %d exist", len(sentCursors), len(pages))
					return nil, fmt.Errorf("too many requests")
				}

				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(pages[len(sentCursors)-1])),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	stories, err := client.ListStoriesByProject(context.Background(), "p1")
	if err != nil {
		t.Fatalf("ListStoriesByProject() unexpected error: %v", err)
	}

	if len(stories) != 5 {
		t.Fatalf("got %d stories, want 5 across %d pages", len(stories), len(pages))
	}

	for i, want := range []string{"1", "2", "3", "4", "5"} {
		if stories[i].Issue != want {
			t.Errorf("stories[%d].Issue = %q, want %q", i, stories[i].Issue, want)
		}
	}

	// The first request must not carry a cursor; each later one must carry the
	// cursor the previous page returned.
	wantCursors := []string{"", "cursor-2", "cursor-3"}
	if len(sentCursors) != len(wantCursors) {
		t.Fatalf("made %d requests %v, want %d", len(sentCursors), sentCursors, len(wantCursors))
	}
	for i, want := range wantCursors {
		if sentCursors[i] != want {
			t.Errorf("request %d start_cursor = %q, want %q", i+1, sentCursors[i], want)
		}
	}
}

// A page that reports has_more without a cursor must not loop forever.
func TestListStoriesByProjectStopsOnMissingCursor(t *testing.T) {
	requests := 0

	client := &Client{
		Token:           "token",
		StoriesSourceId: "stories",
		HTTP: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if requests > 5 {
					return nil, fmt.Errorf("did not stop paginating")
				}

				page := `{"object":"list","results":[],"has_more":true,"next_cursor":""}`
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(page)),
					Header:     make(http.Header),
				}, nil
			}),
		},
	}

	stories, err := client.ListStoriesByProject(context.Background(), "p1")
	if err != nil {
		t.Fatalf("ListStoriesByProject() unexpected error: %v", err)
	}
	if len(stories) != 0 {
		t.Errorf("got %d stories, want 0", len(stories))
	}
	if requests != 1 {
		t.Errorf("made %d requests, want 1", requests)
	}
}
