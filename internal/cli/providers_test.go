package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nox456/forgesync/internal/config"
	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/sync"
)

// fakeProvider is a sync.Provider whose name and issues are set by the test.
type fakeProvider struct {
	name   shared.Provider
	issues []shared.Issue
	err    error
	// gotFilter records the filter passed to FetchAssignedIssues; fetched
	// tells whether it was called at all.
	gotFilter string
	fetched   bool
}

func (f *fakeProvider) Name() shared.Provider { return f.name }

func (f *fakeProvider) FetchAssignedIssues(ctx context.Context, filter string) ([]shared.Issue, error) {
	f.fetched = true
	f.gotFilter = filter
	return f.issues, f.err
}

// providerFlags holds the values of --provider, --repo and --team.
type providerFlags struct {
	provider, repo, team string
}

// useProviders sets the globals getActiveProviders reads and restores them
// when the test ends.
func useProviders(t *testing.T, cfg *config.Config, clients map[shared.Provider]sync.Provider, flags providerFlags) {
	t.Helper()

	prevConfig, prevProviders := Config, Providers
	prevProvider, prevRepo, prevTeam := ProviderName, RepoName, TeamKey
	t.Cleanup(func() {
		Config, Providers = prevConfig, prevProviders
		ProviderName, RepoName, TeamKey = prevProvider, prevRepo, prevTeam
	})

	Config, Providers = cfg, clients
	ProviderName, RepoName, TeamKey = flags.provider, flags.repo, flags.team
}

// names returns each provider's Name, in order.
func names(providers []sync.Provider) []shared.Provider {
	got := make([]shared.Provider, len(providers))
	for i, p := range providers {
		got[i] = p.Name()
	}
	return got
}

func TestResolveProviders(t *testing.T) {
	github, linear := shared.ProviderGithub, shared.ProviderLinear

	cases := []struct {
		name      string
		hasGithub bool
		hasLinear bool
		flags     providerFlags
		want      *resolvedProviders
		wantErr   string
	}{
		// happy path: rules 6 and 7
		{
			name:      "no flags with both tokens runs both, GitHub first, not narrowed",
			hasGithub: true, hasLinear: true,
			want: &resolvedProviders{Active: []shared.Provider{github, linear}},
		},
		{
			name:      "no flags with only a GitHub token runs GitHub",
			hasGithub: true,
			want:      &resolvedProviders{Active: []shared.Provider{github}},
		},
		{
			name:      "no flags with only a Linear token runs Linear",
			hasLinear: true,
			want:      &resolvedProviders{Active: []shared.Provider{linear}},
		},
		{
			name:      "--repo implies GitHub and passes the repo on",
			hasGithub: true, hasLinear: true,
			flags: providerFlags{repo: "owner/name"},
			want:  &resolvedProviders{Active: []shared.Provider{github}, Repo: "owner/name", Narrowed: true},
		},
		{
			name:      "--team implies Linear and passes the team key on",
			hasGithub: true, hasLinear: true,
			flags: providerFlags{team: "ENG"},
			want:  &resolvedProviders{Active: []shared.Provider{linear}, TeamKey: "ENG", Narrowed: true},
		},
		{
			name:      "--provider github narrows to GitHub",
			hasGithub: true, hasLinear: true,
			flags: providerFlags{provider: "github"},
			want:  &resolvedProviders{Active: []shared.Provider{github}, Narrowed: true},
		},
		{
			name:      "--provider linear narrows to Linear",
			hasGithub: true, hasLinear: true,
			flags: providerFlags{provider: "linear"},
			want:  &resolvedProviders{Active: []shared.Provider{linear}, Narrowed: true},
		},
		// edge cases: agreeing flags, and the order the rules are checked in
		{
			name:      "--repo with --provider github is accepted",
			hasGithub: true, hasLinear: true,
			flags: providerFlags{provider: "github", repo: "owner/name"},
			want:  &resolvedProviders{Active: []shared.Provider{github}, Repo: "owner/name", Narrowed: true},
		},
		{
			name:      "--team with --provider linear is accepted",
			hasGithub: true, hasLinear: true,
			flags: providerFlags{provider: "linear", team: "ENG"},
			want:  &resolvedProviders{Active: []shared.Provider{linear}, TeamKey: "ENG", Narrowed: true},
		},
		{
			name:    "an unknown provider is reported before a --repo and --team conflict",
			flags:   providerFlags{provider: "jira", repo: "owner/name", team: "ENG"},
			wantErr: `unknown provider "jira"`,
		},
		{
			name:    "a --repo and --team conflict is reported before a missing token",
			flags:   providerFlags{repo: "owner/name", team: "ENG"},
			wantErr: "can't be combined",
		},
		{
			name:      "--repo with --provider linear is reported before a missing GitHub token",
			hasLinear: true,
			flags:     providerFlags{provider: "linear", repo: "owner/name"},
			wantErr:   "only applies to github",
		},
		{
			name:      "--team with --provider github is reported before a missing Linear token",
			hasGithub: true,
			flags:     providerFlags{provider: "github", team: "ENG"},
			wantErr:   "only applies to linear",
		},
		{
			name:      "provider names are case-sensitive",
			hasGithub: true, hasLinear: true,
			flags:   providerFlags{provider: "GitHub"},
			wantErr: `unknown provider "GitHub"`,
		},
		// error path: rules 1 to 5
		{
			name:      "an unknown provider is rejected",
			hasGithub: true, hasLinear: true,
			flags:   providerFlags{provider: "jira"},
			wantErr: `unknown provider "jira"`,
		},
		{
			name:      "--repo and --team can't be combined",
			hasGithub: true, hasLinear: true,
			flags:   providerFlags{repo: "owner/name", team: "ENG"},
			wantErr: "can't be combined",
		},
		{
			name:      "--repo only applies to GitHub",
			hasGithub: true, hasLinear: true,
			flags:   providerFlags{provider: "linear", repo: "owner/name"},
			wantErr: "only applies to github",
		},
		{
			name:      "--team only applies to Linear",
			hasGithub: true, hasLinear: true,
			flags:   providerFlags{provider: "github", team: "ENG"},
			wantErr: "only applies to linear",
		},
		{
			name:      "--provider github without a GitHub token names github_token",
			hasLinear: true,
			flags:     providerFlags{provider: "github"},
			wantErr:   "github_token",
		},
		{
			name:      "--provider linear without a Linear token names linear_token",
			hasGithub: true,
			flags:     providerFlags{provider: "linear"},
			wantErr:   "linear_token",
		},
		{
			name:      "--repo without a GitHub token names github_token",
			hasLinear: true,
			flags:     providerFlags{repo: "owner/name"},
			wantErr:   "github_token",
		},
		{
			name:      "--team without a Linear token names linear_token",
			hasGithub: true,
			flags:     providerFlags{team: "ENG"},
			wantErr:   "linear_token",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveProviders(tc.hasGithub, tc.hasLinear, tc.flags.provider, tc.flags.repo, tc.flags.team)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("resolveProviders() = %+v, want an error containing %q", got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("resolveProviders() error = %q, want it to contain %q", err, tc.wantErr)
				}
				if got != nil {
					t.Errorf("resolveProviders() = %+v alongside the error, want nil", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveProviders() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("resolveProviders() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetActiveProviders(t *testing.T) {
	bothTokens := &config.Config{GitHubToken: "gh-token", LinearToken: "lin-token"}

	t.Run("happy path", func(t *testing.T) {
		t.Run("no flags returns every client in provider order, GitHub first", func(t *testing.T) {
			clients := map[shared.Provider]sync.Provider{
				shared.ProviderLinear: &fakeProvider{name: shared.ProviderLinear},
				shared.ProviderGithub: &fakeProvider{name: shared.ProviderGithub},
			}
			useProviders(t, bothTokens, clients, providerFlags{})

			providers, resolved, err := getActiveProviders()
			if err != nil {
				t.Fatalf("getActiveProviders() unexpected error: %v", err)
			}

			if diff := cmp.Diff([]shared.Provider{shared.ProviderGithub, shared.ProviderLinear}, names(providers)); diff != "" {
				t.Errorf("provider order mismatch (-want +got):\n%s", diff)
			}
			if resolved.Narrowed {
				t.Errorf("Narrowed = true, want false for a run without flags")
			}
		})

		t.Run("--repo returns only the GitHub client and the repo filter", func(t *testing.T) {
			github := &fakeProvider{name: shared.ProviderGithub}
			clients := map[shared.Provider]sync.Provider{
				shared.ProviderGithub: github,
				shared.ProviderLinear: &fakeProvider{name: shared.ProviderLinear},
			}
			useProviders(t, bothTokens, clients, providerFlags{repo: "owner/name"})

			providers, resolved, err := getActiveProviders()
			if err != nil {
				t.Fatalf("getActiveProviders() unexpected error: %v", err)
			}

			if len(providers) != 1 || providers[0] != github {
				t.Errorf("providers = %v, want only the GitHub client from the map", names(providers))
			}
			want := &resolvedProviders{Active: []shared.Provider{shared.ProviderGithub}, Repo: "owner/name", Narrowed: true}
			if diff := cmp.Diff(want, resolved); diff != "" {
				t.Errorf("resolved mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("without a Linear client a run with both tokens skips Linear", func(t *testing.T) {
			clients := map[shared.Provider]sync.Provider{
				shared.ProviderGithub: &fakeProvider{name: shared.ProviderGithub},
			}
			useProviders(t, bothTokens, clients, providerFlags{})

			providers, _, err := getActiveProviders()
			if err != nil {
				t.Fatalf("getActiveProviders() unexpected error: %v", err)
			}

			if diff := cmp.Diff([]shared.Provider{shared.ProviderGithub}, names(providers)); diff != "" {
				t.Errorf("providers mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("error path", func(t *testing.T) {
		cases := []struct {
			name    string
			cfg     *config.Config
			flags   providerFlags
			wantErr string
		}{
			{
				name:    "a resolution error is returned",
				cfg:     bothTokens,
				flags:   providerFlags{provider: "jira"},
				wantErr: `unknown provider "jira"`,
			},
			{
				name:    "--provider linear without a Linear client is not available yet",
				cfg:     bothTokens,
				flags:   providerFlags{provider: "linear"},
				wantErr: "linear provider not available yet",
			},
			{
				name:    "--team without a Linear client is not available yet",
				cfg:     bothTokens,
				flags:   providerFlags{team: "ENG"},
				wantErr: "linear provider not available yet",
			},
			{
				name:    "only a Linear token, without a Linear client, is not available yet",
				cfg:     &config.Config{LinearToken: "lin-token"},
				wantErr: "linear provider not available yet",
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				clients := map[shared.Provider]sync.Provider{
					shared.ProviderGithub: &fakeProvider{name: shared.ProviderGithub},
				}
				useProviders(t, tc.cfg, clients, tc.flags)

				providers, resolved, err := getActiveProviders()

				if err == nil {
					t.Fatalf("getActiveProviders() = %v, %+v, want an error containing %q", names(providers), resolved, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("getActiveProviders() error = %q, want it to contain %q", err, tc.wantErr)
				}
				if providers != nil || resolved != nil {
					t.Errorf("getActiveProviders() = %v, %+v alongside the error, want nil, nil", names(providers), resolved)
				}
			})
		}
	})
}
