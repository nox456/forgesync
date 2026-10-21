package status

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/sync"
)

type projectKey struct {
	Provider shared.Provider
	Scope    string
}

type Notion interface {
	ListProjects(ctx context.Context, filter shared.ProjectFilter) ([]shared.Project, error)
	FindStoryByIssue(ctx context.Context, issue shared.Issue, projectId string) (*shared.Story, error)
}

type Collector struct {
	NotionClient Notion
	Providers    []sync.Provider
}

func NewCollector(notionClient Notion, providers []sync.Provider) *Collector {
	return &Collector{
		NotionClient: notionClient,
		Providers:    providers,
	}
}

func (c *Collector) Collect(ctx context.Context, filter shared.ProjectFilter) ([]Row, error) {
	slog.Info("Fetching notion projects...")
	projects, err := c.NotionClient.ListProjects(ctx, filter)
	if err != nil {
		return nil, err
	}

	projectsLookup := make(map[projectKey]shared.Project, len(projects))

	for _, project := range projects {
		projectsLookup[projectKey{project.Provider, strings.ToLower(project.GetScope())}] = project
	}

	slog.Info("Fetching github issues...")

	var issues []shared.Issue

	for _, p := range c.Providers {
		f := filter.Repo
		if p.Name() == shared.ProviderLinear {
			f = filter.TeamKey
		}
		got, err := p.FetchAssignedIssues(ctx, f)
		if err != nil {
			return nil, err
		}
		issues = append(issues, got...)
	}

	rows := make([]Row, 0)

	slog.Info("Building rows...")
	for _, issue := range issues {
		slog.Debug(fmt.Sprintf("[STATUS]: Building row - Issue %s Scope %s", issue.Identifier, issue.Scope))
		project, ok := projectsLookup[projectKey{issue.Provider, strings.ToLower(issue.Scope)}]
		if !ok {
			slog.Debug(fmt.Sprintf("[STATUS]: Project not found - Issue %s Scope %s", issue.Identifier, issue.Scope))
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
