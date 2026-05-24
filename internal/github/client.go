package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v88/github"
)

type Client struct {
	Token string
}

type Issue struct {
	Number      int
	Title       string
	URL         string
	State       string
	Labels      []string
	Repo        string
	UpdatedAt   time.Time
	ClosedAt    *time.Time
	HasLikendPR bool
}

func NewClient(token string) *Client {
	return &Client{
		Token: token,
	}
}

func (c *Client) FetchAssignedIssues(ctx context.Context) ([]Issue, error) {
	client, err := github.NewClient(github.WithAuthToken(c.Token))

	if err != nil {
		return nil, err
	}

	var issues []Issue

	issuesResponse := client.Issues.ListAllIssuesIter(ctx, &github.ListAllIssuesOptions{})

	for issueResponse, err := range issuesResponse {
		if err != nil {
			return nil, err
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

		issue := Issue{
			Number:      *issueResponse.Number,
			Title:       *issueResponse.Title,
			URL:         *issueResponse.HTMLURL,
			State:       *issueResponse.State,
			Labels:      issueLabels,
			Repo:        fmt.Sprintf("%s/%s", *issueResponse.Repository.Owner.Login, *issueResponse.Repository.Name),
			UpdatedAt:   (*issueResponse.UpdatedAt).Time,
			ClosedAt:    closedAt,
			HasLikendPR: issueResponse.PullRequestLinks != nil,
		}

		issues = append(issues, issue)
	}

	return issues, nil
}
