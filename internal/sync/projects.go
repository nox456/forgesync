package sync

import (
	"fmt"
	"log/slog"

	"github.com/nox456/forgesync/internal/shared"
)

func ClassifyProjects(projects []shared.Project, providers []Provider, narrowed bool) ([]shared.Project, []ReportError) {
	active := make(map[shared.Provider]bool, len(providers))
	for _, provider := range providers {
		active[provider.Name()] = true
	}

	eligible := make([]shared.Project, 0, len(projects))
	problems := make([]ReportError, 0)

	for _, project := range projects {
		switch {
		case project.Provider == "":
			problems = append(problems, ReportError{
				Project: project.Name,
				Error:   "project has no provider",
			})
		case project.Provider != shared.ProviderGithub && project.Provider != shared.ProviderLinear:
			problems = append(problems, ReportError{
				Project: project.Name,
				Error:   fmt.Sprintf("unknown provider %q", project.Provider),
			})
		case active[project.Provider]:
			eligible = append(eligible, project)
		case narrowed:
			slog.Debug(fmt.Sprintf("[PROJECTS]: Skipped, provider not selected - Name: %s Provider: %s", project.Name, project.Provider))
		default:
			problems = append(problems, ReportError{
				Project:  project.Name,
				Provider: project.Provider,
				Error:    fmt.Sprintf("%s token not configured", project.Provider),
			})
		}
	}

	return eligible, problems
}
