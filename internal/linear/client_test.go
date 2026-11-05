package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/nox456/forgesync/internal/shared"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// respond builds an HTTP response with the given status and body.
func respond(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

// newTestClient returns a client whose requests go to transport instead of
// the network.
func newTestClient(transport roundTripFunc) *Client {
	client := NewClient("lin_api_test")
	client.HTTP = &http.Client{Transport: transport}
	return client
}

type viewerResult struct {
	Viewer struct {
		ID string `json:"id"`
	} `json:"viewer"`
}

const viewerQuery = "query { viewer { id } }"

func TestRequest(t *testing.T) {
	// sendOne runs one successful request and returns what the client sent.
	sendOne := func(t *testing.T) (*http.Request, []byte) {
		t.Helper()

		var sent *http.Request
		var sentBody []byte

		client := newTestClient(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			sent, sentBody = req, body
			return respond(200, `{"data":{"viewer":{"id":"user-1"}}}`), nil
		})

		var result viewerResult
		if err := client.request(context.Background(), viewerQuery, map[string]any{"first": 50}, &result); err != nil {
			t.Fatalf("request() unexpected error: %v", err)
		}
		return sent, sentBody
	}

	t.Run("happy path", func(t *testing.T) {
		t.Run("posts to the GraphQL endpoint with a JSON content type", func(t *testing.T) {
			req, _ := sendOne(t)

			if req.Method != http.MethodPost {
				t.Errorf("method = %q, want POST", req.Method)
			}
			if got := req.URL.String(); got != "https://api.linear.app/graphql" {
				t.Errorf("URL = %q, want the Linear GraphQL endpoint", got)
			}
			if got := req.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
		})

		t.Run("sends the raw API key without a Bearer prefix", func(t *testing.T) {
			req, _ := sendOne(t)

			if got := req.Header.Get("Authorization"); got != "lin_api_test" {
				t.Errorf("Authorization = %q, want the raw key %q with no Bearer prefix", got, "lin_api_test")
			}
		})

		t.Run("sends the query and variables as the JSON body", func(t *testing.T) {
			_, body := sendOne(t)

			var payload struct {
				Query     string         `json:"query"`
				Variables map[string]any `json:"variables"`
			}
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("request body is not JSON: %v\n%s", err, body)
			}

			if payload.Query != viewerQuery {
				t.Errorf("query = %q, want %q", payload.Query, viewerQuery)
			}
			if payload.Variables["first"] != float64(50) {
				t.Errorf("variables.first = %v, want 50", payload.Variables["first"])
			}
		})

		t.Run("decodes data into the result", func(t *testing.T) {
			client := newTestClient(func(*http.Request) (*http.Response, error) {
				return respond(200, `{"data":{"viewer":{"id":"user-1"}}}`), nil
			})

			var result viewerResult
			if err := client.request(context.Background(), viewerQuery, nil, &result); err != nil {
				t.Fatalf("request() unexpected error: %v", err)
			}

			if result.Viewer.ID != "user-1" {
				t.Errorf("viewer.id = %q, want %q", result.Viewer.ID, "user-1")
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("an error status with an empty errors array reports the raw body", func(t *testing.T) {
			client := newTestClient(func(*http.Request) (*http.Response, error) {
				return respond(400, `{"errors":[]}`), nil
			})

			err := client.request(context.Background(), viewerQuery, nil, &viewerResult{})

			if err == nil || !strings.Contains(err.Error(), `linear api 400: {"errors":[]}`) {
				t.Errorf("request() error = %v, want the status and the raw body", err)
			}
		})

		t.Run("only the first of several GraphQL errors is reported", func(t *testing.T) {
			client := newTestClient(func(*http.Request) (*http.Response, error) {
				return respond(200, `{"data":null,"errors":[{"message":"first problem"},{"message":"second problem"}]}`), nil
			})

			err := client.request(context.Background(), viewerQuery, nil, &viewerResult{})

			if err == nil || err.Error() != "linear graphql: first problem" {
				t.Errorf("request() error = %v, want %q", err, "linear graphql: first problem")
			}
		})
	})

	t.Run("error path", func(t *testing.T) {
		t.Run("a network error is wrapped and keeps its cause", func(t *testing.T) {
			netErr := errors.New("connection refused")
			client := newTestClient(func(*http.Request) (*http.Response, error) {
				return nil, netErr
			})

			err := client.request(context.Background(), viewerQuery, nil, &viewerResult{})

			if !errors.Is(err, netErr) {
				t.Errorf("request() error = %v, want it to wrap %v", err, netErr)
			}
			if err == nil || !strings.HasPrefix(err.Error(), "linear request:") {
				t.Errorf("request() error = %v, want the %q prefix", err, "linear request:")
			}
		})

		cases := []struct {
			name    string
			status  int
			body    string
			wantErr []string
		}{
			{
				name:    "GraphQL errors on a 200 return the first message",
				status:  200,
				body:    `{"data":{"viewer":{"id":"user-1"}},"errors":[{"message":"Cannot query field \"foo\""}]}`,
				wantErr: []string{`linear graphql: Cannot query field "foo"`},
			},
			{
				name:    "GraphQL errors on a 400 report the message, not just the status",
				status:  400,
				body:    `{"errors":[{"message":"Variable \"$first\" got invalid value"}]}`,
				wantErr: []string{"linear api 400", `Variable "$first" got invalid value`},
			},
			{
				name:    "a non-JSON error body reports the status and the raw body",
				status:  502,
				body:    `<html>Bad Gateway</html>`,
				wantErr: []string{"linear api 502", "<html>Bad Gateway</html>"},
			},
			{
				name:    "a non-JSON success body is a decode error",
				status:  200,
				body:    `not json`,
				wantErr: []string{"linear response:"},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				client := newTestClient(func(*http.Request) (*http.Response, error) {
					return respond(tc.status, tc.body), nil
				})

				var result viewerResult
				err := client.request(context.Background(), viewerQuery, nil, &result)

				if err == nil {
					t.Fatalf("request() = nil, want an error containing %q", tc.wantErr)
				}
				for _, want := range tc.wantErr {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("request() error = %q, want it to contain %q", err, want)
					}
				}
				if result.Viewer.ID != "" {
					t.Errorf("result was decoded (viewer.id = %q) alongside an error, want it untouched", result.Viewer.ID)
				}
			})
		}
	})
}

func TestName(t *testing.T) {
	// edge cases: none — Name takes no input.
	// error path: none — Name can't fail.
	if got := NewClient("lin_api_test").Name(); got != shared.ProviderLinear {
		t.Errorf("Name() = %q, want %q", got, shared.ProviderLinear)
	}
}
