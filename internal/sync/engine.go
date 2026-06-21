package sync

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/nox456/forgesync/internal/shared"
)

type Notion interface {
	ListProjects(ctx context.Context, repoName string) ([]shared.Project, error)
	UpsertStory(ctx context.Context, storyInput shared.StoryInput, issue shared.Issue, isDryRun bool, existingStory *shared.Story) (*shared.UpsertResult, error)
	FindStoryByIssue(ctx context.Context, issue shared.Issue, projectId string) (*shared.Story, error)
}

type Github interface {
	FetchAssignedIssues(ctx context.Context, repoName string) ([]shared.Issue, error)
}

type Engine struct {
	NotionClient Notion
	GithubClient Github
}

type EngineRunOptions struct {
	DryRun     bool
	RepoFilter string
}

type ReportError struct {
	IssueNumber int
	Error       string
}

type Report struct {
	Created   int
	Updated   int
	Skipped   int
	Unchanged int
	Errors    []ReportError
}

func NewEngine(notionClient Notion, githubClient Github) *Engine {
	return &Engine{
		NotionClient: notionClient,
		GithubClient: githubClient,
	}
}

func (e *Engine) Run(ctx context.Context, options EngineRunOptions) (*Report, error) {
	slog.Info("Fetching notion projects...")
	projects, err := e.NotionClient.ListProjects(ctx, options.RepoFilter)
	if err != nil {
		return nil, err
	}

	projectsMap := make(map[string]shared.Project)

	for _, project := range projects {
		projectsMap[strings.ToLower(project.Repo)] = project
	}

	slog.Info("Fetching github issues...")
	issues, err := e.GithubClient.FetchAssignedIssues(ctx, options.RepoFilter)
	if err != nil {
		return nil, err
	}

	created := 0
	updated := 0
	skipped := 0
	unchanged := 0
	errors := make([]ReportError, 0)

	slog.Info("Syncing...")
	for _, issue := range issues {
		project, ok := projectsMap[strings.ToLower(issue.Repo)]
		if !ok {
			skipped++
			continue
		}

		existingStory, err := e.NotionClient.FindStoryByIssue(ctx, issue, project.PageID)

		if err != nil {
			return nil, err
		}

		storyInput := IssueToStoryInput(issue, existingStory, project.PageID)
		result, err := e.NotionClient.UpsertStory(ctx, storyInput, issue, options.DryRun, existingStory)
		if err != nil {
			slog.Error(err.Error())
			errors = append(errors, ReportError{
				IssueNumber: issue.Number,
				Error:       err.Error(),
			})
			continue
		}

		if result.Created {
			created++
		} else if result.Updated {
			updated++
		} else if result.Unchanged {
			unchanged++
		}
		slog.Debug(fmt.Sprintf("[SYNC]: Issue - Number: %d Title %s <-> Story - Status: %s Project: %s", issue.Number, issue.Title, storyInput.Status, project.Name))
	}

	report := &Report{
		Created:   created,
		Updated:   updated,
		Skipped:   skipped,
		Unchanged: unchanged,
		Errors:    errors,
	}

	return report, nil
}
