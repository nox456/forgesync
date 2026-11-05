package cli

import (
	"fmt"
	"log/slog"

	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/sync"
)

type resolvedProviders struct {
	Active   []shared.Provider
	Repo     string
	TeamKey  string
	Narrowed bool
}

func resolveProviders(hasGithub bool, hasLinear bool, provider string, repo string, team string) (*resolvedProviders, error) {
	if provider != "" && provider != string(shared.ProviderGithub) && provider != string(shared.ProviderLinear) {
		return nil, fmt.Errorf("unknown provider %q", provider)
	}

	if repo != "" && team != "" {
		return nil, fmt.Errorf("--repo and --team can't be combined")
	}

	if repo != "" {

		if provider != "" && provider != string(shared.ProviderGithub) {
			return nil, fmt.Errorf("repo filter only applies to github provider")
		}

		if !hasGithub {
			return nil, fmt.Errorf("github_token is not set")
		}
		return &resolvedProviders{
			Active:   []shared.Provider{shared.ProviderGithub},
			Repo:     repo,
			Narrowed: true,
		}, nil
	}

	if team != "" {

		if provider != "" && provider != string(shared.ProviderLinear) {
			return nil, fmt.Errorf("team filter only applies to linear provider")
		}

		if !hasLinear {
			return nil, fmt.Errorf("linear_token is not set")
		}
		return &resolvedProviders{
			Active:   []shared.Provider{shared.ProviderLinear},
			TeamKey:  team,
			Narrowed: true,
		}, nil
	}

	if provider == string(shared.ProviderGithub) {

		if !hasGithub {
			return nil, fmt.Errorf("github_token is not set")
		}
		return &resolvedProviders{
			Active:   []shared.Provider{shared.ProviderGithub},
			Narrowed: true,
		}, nil
	}

	if provider == string(shared.ProviderLinear) {

		if !hasLinear {
			return nil, fmt.Errorf("linear_token is not set")
		}
		return &resolvedProviders{
			Active:   []shared.Provider{shared.ProviderLinear},
			Narrowed: true,
		}, nil
	}

	activeProviders := make([]shared.Provider, 0, 2)

	if hasGithub {
		activeProviders = append(activeProviders, shared.ProviderGithub)
	}

	if hasLinear {
		activeProviders = append(activeProviders, shared.ProviderLinear)
	}

	return &resolvedProviders{
		Active:   activeProviders,
		Narrowed: false,
	}, nil
}

func getActiveProviders(includeLinear bool) ([]sync.Provider, *resolvedProviders, error) {
	resolved, err := resolveProviders(Config.GitHubToken != "", Config.LinearToken != "", ProviderName, RepoName, TeamKey)
	if err != nil {
		return nil, nil, err
	}

	providers := make([]sync.Provider, 0, len(resolved.Active))

	for _, name := range resolved.Active {
		// TODO(phase-4): drop includeLinear once sync and status apply Linear's status rules
		if name == shared.ProviderLinear && !includeLinear {
			if resolved.Narrowed || len(resolved.Active) == 1 {
				return nil, nil, fmt.Errorf("linear provider is not supported by sync and status until Phase 4")
			}

			slog.Debug("linear provider is not supported by sync and status yet, skipping")
			continue
		}

		providers = append(providers, Providers[name])
	}

	return providers, resolved, nil
}
