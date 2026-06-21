package status

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
	"github.com/nox456/forgesync/internal/shared"
)

type Notion interface {
	ListProjects(ctx context.Context, repoName string) ([]shared.Project, error)
	FindStoryByIssue(ctx context.Context, issue shared.Issue, projectId string) (*shared.Story, error)
}

type Github interface {
	FetchAssignedIssues(ctx context.Context, repoName string) ([]shared.Issue, error)
}

type Collector struct {
	NotionClient Notion
	GithubClient Github
}

func NewCollector(notionClient *notion.Client, githubClient *github.Client) *Collector {
	return &Collector{
		NotionClient: notionClient,
		GithubClient: githubClient,
	}
}

func (c *Collector) Collect(ctx context.Context, repoName string) ([]Row, error) {
	slog.Info("Fetching notion projects...")
	projects, err := c.NotionClient.ListProjects(ctx, repoName)
	if err != nil {
		return nil, err
	}

	projectsMap := make(map[string]shared.Project)

	for _, project := range projects {
		projectsMap[project.Repo] = project
	}

	slog.Info("Fetching github issues...")
	issues, err := c.GithubClient.FetchAssignedIssues(ctx, repoName)
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0)

	slog.Info("Building rows...")
	for _, issue := range issues {
		slog.Debug(fmt.Sprintf("[STATUS]: Building row - IssueNumber %d Repo %s", issue.Number, issue.Repo))
		project, ok := projectsMap[issue.Repo]
		if !ok {
			slog.Debug(fmt.Sprintf("[STATUS]: Project not found - IssueNumber %d Repo %s", issue.Number, issue.Repo))
			rows = append(rows, BuildRow(issue, nil, nil))
			continue
		}

		existingStory, err := c.NotionClient.FindStoryByIssue(ctx, issue, project.PageID)
		if err != nil {
			return nil, err
		}

		rows = append(rows, BuildRow(issue, &project, existingStory))
	}
	return rows, nil
}
