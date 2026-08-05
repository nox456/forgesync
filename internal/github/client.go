package github

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/go-github/v88/github"
	"github.com/nox456/forgesync/internal/shared"
)

type Client struct {
	Token string
	gh    *github.Client
}

func NewClient(token string) (*Client, error) {
	client, err := github.NewClient(github.WithAuthToken(token))

	if err != nil {
		return nil, err
	}

	return &Client{
		Token: token,
		gh:    client,
	}, nil
}

func (c *Client) FetchAssignedIssues(ctx context.Context, repoName string) ([]shared.Issue, error) {
	var issues []shared.Issue

	issuesResponse := c.gh.Issues.ListAllIssuesIter(ctx, &github.ListAllIssuesOptions{
		State: "all",
		Since: time.Now().AddDate(0, 0, -7),
	})

	for issueResponse, err := range issuesResponse {
		if err != nil {
			return nil, err
		}

		if repoName != "" && !strings.EqualFold(issueResponse.Repository.GetFullName(), repoName) {
			continue
		}

		issueLabels := make([]string, len(issueResponse.Labels))
		for i, label := range issueResponse.Labels {
			issueLabels[i] = *label.Name
		}

		if issueResponse.IsPullRequest() {
			continue
		}

		var closedAt *time.Time
		if issueResponse.ClosedAt != nil {
			closedAt = &issueResponse.ClosedAt.Time
		}

		owner := *issueResponse.Repository.Owner.Login
		repo := *issueResponse.Repository.Name

		hasLinkedPR := false

		if *issueResponse.State == "open" {
			hasLinkedPR, err = c.hasLinkedPR(ctx, owner, repo, *issueResponse.Number)
			if err != nil {
				return nil, err
			}
		}

		issue := shared.Issue{
			Number:      *issueResponse.Number,
			Title:       *issueResponse.Title,
			URL:         *issueResponse.HTMLURL,
			Body:        issueResponse.GetBody(),
			State:       *issueResponse.State,
			Labels:      issueLabels,
			Repo:        fmt.Sprintf("%s/%s", owner, repo),
			UpdatedAt:   (*issueResponse.UpdatedAt).Time,
			CreatedAt:   (*issueResponse.CreatedAt).Time,
			ClosedAt:    closedAt,
			HasLinkedPR: hasLinkedPR,
		}
		slog.Debug(fmt.Sprintf("[GITHUB]: Issue found - Number: %d Title: %s Repo: %s", issue.Number, issue.Title, issue.Repo))
		issues = append(issues, issue)
	}

	return issues, nil
}

// hasLinkedPR reports whether the issue has at least one non-draft pull request
// linked to close it. The REST timeline's "connected" events carry no reference
// to the pull request they linked, so the draft state can only be resolved
// through the GraphQL API, which go-github doesn't wrap.
func (c *Client) hasLinkedPR(ctx context.Context, owner, repo string, number int) (bool, error) {
	query := map[string]any{
		"query": `query($owner: String!, $repo: String!, $number: Int!) {
			repository(owner: $owner, name: $repo) {
				issue(number: $number) {
					closedByPullRequestsReferences(first: 100, includeClosedPrs: true) {
						nodes { number isDraft }
					}
				}
			}
		}`,
		"variables": map[string]any{
			"owner":  owner,
			"repo":   repo,
			"number": number,
		},
	}

	req, err := c.gh.NewRequest(ctx, "POST", "graphql", query)
	if err != nil {
		return false, err
	}

	var response struct {
		Data struct {
			Repository struct {
				Issue struct {
					ClosedByPullRequestsReferences struct {
						Nodes []struct {
							Number  int  `json:"number"`
							IsDraft bool `json:"isDraft"`
						} `json:"nodes"`
					} `json:"closedByPullRequestsReferences"`
				} `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if _, err := c.gh.Do(req, &response); err != nil {
		return false, err
	}

	if len(response.Errors) > 0 {
		return false, fmt.Errorf("github graphql: %s", response.Errors[0].Message)
	}

	for _, pr := range response.Data.Repository.Issue.ClosedByPullRequestsReferences.Nodes {
		if pr.IsDraft {
			slog.Debug(fmt.Sprintf("[GITHUB]: Skipping draft PR #%d linked to %s/%s#%d", pr.Number, owner, repo, number))
			continue
		}
		return true, nil
	}

	return false, nil
}
