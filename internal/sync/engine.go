package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
)

type Engine struct {
	NotionClient *notion.Client
	GithubClient *github.Client
}

type EngineRunOptions struct {
	DryRun bool
}

type Report struct {
	Created int
	Updated int
	Skipped int
	Errors  int
}

func NewEngine(notionClient *notion.Client, githubClient *github.Client) *Engine {
	return &Engine{
		NotionClient: notionClient,
		GithubClient: githubClient,
	}
}

func (e *Engine) Run(ctx context.Context, options EngineRunOptions) (*Report, error) {
	fmt.Println("Fetching notion projects...")
	projects, err := e.NotionClient.ListProjects()
	if err != nil {
		return nil, err
	}

	projectsMap := make(map[string]notion.Project)

	for _, project := range projects {
		projectsMap[strings.ToLower(project.Repo)] = project
	}

	fmt.Println("Fetching github issues...")
	issues, err := e.GithubClient.FetchAssignedIssues(ctx)
	if err != nil {
		return nil, err
	}

	created := 0
	updated := 0
	skipped := 0
	errors := 0

	fmt.Println("Syncing...")
	for _, issue := range issues {
		project, ok := projectsMap[strings.ToLower(issue.Repo)]
		if !ok {
			skipped++
			continue
		}

		if options.DryRun {
			created++
			continue
		}

		storyInput := IssueToStoryInput(issue, project.PageID)
		isCreated, err := e.NotionClient.UpsertStory(storyInput, issue)
		if err != nil {
			fmt.Println("Error upserting story:", err)
			errors++
			continue
		}

		if isCreated {
			created++
		} else {
			updated++
		}
	}

	report := &Report{
		Created: created,
		Updated: updated,
		Skipped: skipped,
		Errors:  errors,
	}

	return report, nil
}
