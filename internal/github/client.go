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
	HasLinkedPR bool
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

		owner := *issueResponse.Repository.Owner.Login
		repo := *issueResponse.Repository.Name

		hasLinkedPR, err := hasConnectedPR(ctx, client, owner, repo, *issueResponse.Number)
		if err != nil {
			return nil, err
		}

		issue := Issue{
			Number:      *issueResponse.Number,
			Title:       *issueResponse.Title,
			URL:         *issueResponse.HTMLURL,
			State:       *issueResponse.State,
			Labels:      issueLabels,
			Repo:        fmt.Sprintf("%s/%s", owner, repo),
			UpdatedAt:   (*issueResponse.UpdatedAt).Time,
			ClosedAt:    closedAt,
			HasLinkedPR: hasLinkedPR,
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

func hasConnectedPR(ctx context.Context, client *github.Client, owner, repo string, number int) (bool, error) {
	opts := &github.ListOptions{PerPage: 100}
	connections := 0

	for {
		events, resp, err := client.Issues.ListIssueTimeline(ctx, owner, repo, number, opts)
		if err != nil {
			return false, err
		}

		for _, event := range events {
			if event.Event == nil {
				continue
			}
			switch *event.Event {
			case "connected":
				connections++
			case "disconnected":
				connections--
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return connections > 0, nil
}
