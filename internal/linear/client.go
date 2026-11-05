package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nox456/forgesync/internal/shared"
)

const linearEndpoint = "https://api.linear.app/graphql"
const linearPageSize = 50

type Client struct {
	Token string
	HTTP  *http.Client
	Now   func() time.Time
}

func NewClient(token string) *Client {
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 10,
			MaxIdleConns:        10,
		},
		Timeout: time.Second * 30,
	}
	return &Client{
		Token: token,
		HTTP:  httpClient,
		Now:   time.Now,
	}
}

func (c *Client) Name() shared.Provider {
	return shared.ProviderLinear
}

func (c *Client) request(ctx context.Context, query string, variables any, result any) error {
	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", linearEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", c.Token)
	req.Header.Add("Content-Type", "application/json")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("linear request: %w", err)
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("linear response: %w", err)
	}

	var response graphqlResponse

	if res.StatusCode >= 400 {
		if json.Unmarshal(body, &response) == nil && len(response.Errors) > 0 {
			return fmt.Errorf("linear api %d: %s", res.StatusCode, response.Errors[0].Message)
		}
		return fmt.Errorf("linear api %d: %s", res.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("linear response: %w", err)
	}

	if len(response.Errors) > 0 {
		return fmt.Errorf("linear graphql: %s", response.Errors[0].Message)
	}

	if err := json.Unmarshal(response.Data, result); err != nil {
		return fmt.Errorf("linear response: %w", err)
	}

	return nil
}
