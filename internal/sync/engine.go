package sync

import (
	"context"
	"log/slog"
	"strings"

	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
)

type Notion interface {
	ListProjects(ctx context.Context) ([]notion.Project, error)
	UpsertStory(ctx context.Context, storyInput notion.StoryInput, issue github.Issue, isDryRun bool, existingStory *notion.Story) (*notion.UpsertResult, error)
	FindStoryByIssue(ctx context.Context, issue github.Issue) (*notion.Story, error)
}

type Github interface {
	FetchAssignedIssues(ctx context.Context) ([]github.Issue, error)
}

type Engine struct {
	NotionClient Notion
	GithubClient Github
}

type EngineRunOptions struct {
	DryRun bool
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

func NewEngine(notionClient *notion.Client, githubClient *github.Client) *Engine {
	return &Engine{
		NotionClient: notionClient,
		GithubClient: githubClient,
	}
}

func (e *Engine) Run(ctx context.Context, options EngineRunOptions) (*Report, error) {
	slog.Info("Fetching notion projects...")
	projects, err := e.NotionClient.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	projectsMap := make(map[string]notion.Project)

	for _, project := range projects {
		projectsMap[strings.ToLower(project.Repo)] = project
	}

	slog.Info("Fetching github issues...")
	issues, err := e.GithubClient.FetchAssignedIssues(ctx)
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

		existingStory, err := e.NotionClient.FindStoryByIssue(ctx, issue)

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
