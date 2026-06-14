package github

import (
	"context"
	"fmt"
	"log/slog"
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
	CreatedAt   time.Time
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

	issuesResponse := client.Issues.ListAllIssuesIter(ctx, &github.ListAllIssuesOptions{
		State: "all",
		Since: time.Now().AddDate(0, 0, -30),
	})

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

		hasLinkedPR, err := c.hasConnectedPR(ctx, owner, repo, *issueResponse.Number)
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
			CreatedAt:   (*issueResponse.CreatedAt).Time,
			ClosedAt:    closedAt,
			HasLinkedPR: hasLinkedPR,
		}

		slog.Debug(fmt.Sprintf("[GITHUB]: Issue found - Number: %d Title: %s Repo: %s", issue.Number, issue.Title, issue.Repo))
		issues = append(issues, issue)
	}

	return issues, nil
}

func (c *Client) hasConnectedPR(ctx context.Context, owner, repo string, number int) (bool, error) {
	client, err := github.NewClient(github.WithAuthToken(c.Token))
	if err != nil {
		return false, err
	}

	opts := &github.ListOptions{PerPage: 100}
	connections := 0

	for {
		events, resp, err := client.Issues.ListIssueTimeline(ctx, owner, repo, number, opts)
		if err != nil {
			return false, err
		}

		for _, event := range events {
			switch event.GetEvent() {
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
