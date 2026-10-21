package sync

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/nox456/forgesync/internal/shared"
)

type Notion interface {
	ListProjects(ctx context.Context, filter shared.ProjectFilter) ([]shared.Project, error)
	UpsertStory(ctx context.Context, storyInput shared.StoryInput, isDryRun bool, existingStory *shared.Story) (*shared.UpsertResult, error)
	FindStoryByIssue(ctx context.Context, issue shared.Issue, projectId string) (*shared.Story, error)
	ListStoriesByProject(ctx context.Context, projectId string) ([]shared.Story, error)
}

type Provider interface {
	Name() shared.Provider
	FetchAssignedIssues(ctx context.Context, filter string) ([]shared.Issue, error)
}

type Engine struct {
	NotionClient Notion
	Providers    []Provider
}

type EngineRunOptions struct {
	DryRun            bool
	Repo              string
	Team              string
	ProvidersNarrowed bool
}

type ReportError struct {
	Identifier string
	Provider   shared.Provider
	Project    string
	Error      string
}

type Report struct {
	Created   int
	Updated   int
	Skipped   int
	Unchanged int
	Errors    []ReportError
}

type projectKey struct {
	Provider shared.Provider
	Scope    string
}

func NewEngine(notionClient Notion, providers []Provider) *Engine {
	return &Engine{
		NotionClient: notionClient,
		Providers:    providers,
	}
}

func (e *Engine) Run(ctx context.Context, options EngineRunOptions) (*Report, error) {
	slog.Info("Fetching notion projects...")

	var filter shared.ProjectFilter
	if options.Repo != "" {
		filter.Repo = options.Repo
	} else {
		filter.TeamKey = options.Team
	}

	projects, err := e.NotionClient.ListProjects(ctx, filter)
	if err != nil {
		return nil, err
	}

	errors := make([]ReportError, 0)

	projectsLookup := make(map[projectKey]shared.Project, len(projects))

	for _, project := range projects {
		if project.Provider == "" {
			errors = append(errors, ReportError{
				Project: project.Name,
				Error:   "project has no provider",
			})
			continue
		}

		if project.Provider != shared.ProviderGithub && project.Provider != shared.ProviderLinear {
			errors = append(errors, ReportError{
				Project: project.Name,
				Error:   fmt.Sprintf("unknown provider %q", project.Provider),
			})
			continue
		}

		for _, provider := range e.Providers {
			if project.Provider == provider.Name() {
				projectsLookup[projectKey{project.Provider, strings.ToLower(project.GetScope())}] = project
				break
			}
		}
	}

	created := 0
	updated := 0
	skipped := 0
	unchanged := 0

	storiesByProject := make(map[string]map[string]shared.Story)

	for _, project := range projectsLookup {
		stories, err := e.NotionClient.ListStoriesByProject(ctx, project.PageID)
		if err != nil {
			return nil, err
		}

		storiesByIssue := make(map[string]shared.Story)
		for _, story := range stories {
			if _, ok := storiesByIssue[story.Issue]; ok {
				errors = append(errors, ReportError{
					Identifier: story.Issue,
					Project:    project.Name,
					Provider:   project.Provider,
					Error:      "found more than one story for issue",
				})
				continue
			}
			storiesByIssue[story.Issue] = story
		}
		storiesByProject[project.PageID] = storiesByIssue
	}

	var issues []shared.Issue
	for _, p := range e.Providers {
		f := options.Repo
		if p.Name() == shared.ProviderLinear {
			f = options.Team
		}
		got, err := p.FetchAssignedIssues(ctx, f)
		if err != nil {
			return nil, err // Phase 4 turns this into a ReportError
		}
		issues = append(issues, got...)
	}

	slog.Info("Syncing...")
	for _, issue := range issues {
		project, ok := projectsLookup[projectKey{issue.Provider, strings.ToLower(issue.Scope)}]
		if !ok {
			skipped++
			continue
		}

		var existingStory *shared.Story
		if story, ok := storiesByProject[project.PageID][issue.Identifier]; ok {
			existingStory = &story
		}

		// Deciding that a Story needs no write is a sync rule, not an adapter
		// concern, so the engine short-circuits before touching Notion at all.
		if IsSynced(issue, existingStory) {
			unchanged++
			slog.Debug(fmt.Sprintf("[SYNC]: Unchanged - Identifier: %s Title %s Project: %s", issue.Identifier, issue.Title, project.Name))
			continue
		}

		storyInput := IssueToStoryInput(issue, existingStory, project.PageID)
		result, err := e.NotionClient.UpsertStory(ctx, storyInput, options.DryRun, existingStory)
		if err != nil {
			slog.Error(err.Error())
			errors = append(errors, ReportError{
				Identifier: issue.Identifier,
				Error:      err.Error(),
			})
			continue
		}

		if result.Created {
			created++
		} else if result.Updated {
			updated++
		}
		slog.Debug(fmt.Sprintf("[SYNC]: Issue - Identifier: %s Title %s <-> Story - Status: %s Project: %s", issue.Identifier, issue.Title, storyInput.Status, project.Name))
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
