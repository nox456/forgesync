package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/nox456/forgesync/internal/config"
	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/status"
	"github.com/nox456/forgesync/internal/sync"
)

// fakePrinter is an output.Printer that records the issues it was given.
type fakePrinter struct {
	issues  []shared.Issue
	printed bool
}

func (p *fakePrinter) PrintProjects([]shared.Project) {}

func (p *fakePrinter) PrintIssues(issues []shared.Issue) {
	p.printed = true
	p.issues = issues
}

func (p *fakePrinter) PrintReport(*sync.Report) {}

func (p *fakePrinter) PrintStatus([]status.Row) {}

// usePrinter swaps the global Printer for a fakePrinter until the test ends.
func usePrinter(t *testing.T) *fakePrinter {
	t.Helper()

	prev := Printer
	t.Cleanup(func() { Printer = prev })

	printer := &fakePrinter{}
	Printer = printer
	return printer
}

// runIssues runs the issues command's RunE with a background context.
func runIssues(t *testing.T) error {
	t.Helper()
	issuesCmd.SetContext(context.Background())
	return issuesCmd.RunE(issuesCmd, nil)
}

func TestIssuesCommand(t *testing.T) {
	bothTokens := &config.Config{GitHubToken: "gh-token", LinearToken: "lin-token"}
	githubIssue := shared.Issue{Provider: shared.ProviderGithub, Identifier: "#12", Scope: "owner/name"}
	linearIssue := shared.Issue{Provider: shared.ProviderLinear, Identifier: "ENG-7", Scope: "ENG"}

	t.Run("happy path", func(t *testing.T) {
		t.Run("no flags prints every provider's issues, GitHub first", func(t *testing.T) {
			github := &fakeProvider{name: shared.ProviderGithub, issues: []shared.Issue{githubIssue}}
			linear := &fakeProvider{name: shared.ProviderLinear, issues: []shared.Issue{linearIssue}}
			useProviders(t, bothTokens, map[shared.Provider]sync.Provider{
				shared.ProviderGithub: github,
				shared.ProviderLinear: linear,
			}, providerFlags{})
			printer := usePrinter(t)

			if err := runIssues(t); err != nil {
				t.Fatalf("issues RunE unexpected error: %v", err)
			}

			if diff := cmp.Diff([]shared.Issue{githubIssue, linearIssue}, printer.issues); diff != "" {
				t.Errorf("printed issues mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("--repo fetches only GitHub, filtered by that repo", func(t *testing.T) {
			github := &fakeProvider{name: shared.ProviderGithub, issues: []shared.Issue{githubIssue}}
			linear := &fakeProvider{name: shared.ProviderLinear, issues: []shared.Issue{linearIssue}}
			useProviders(t, bothTokens, map[shared.Provider]sync.Provider{
				shared.ProviderGithub: github,
				shared.ProviderLinear: linear,
			}, providerFlags{repo: "owner/name"})
			printer := usePrinter(t)

			if err := runIssues(t); err != nil {
				t.Fatalf("issues RunE unexpected error: %v", err)
			}

			if github.gotFilter != "owner/name" {
				t.Errorf("GitHub filter = %q, want %q", github.gotFilter, "owner/name")
			}
			if linear.fetched {
				t.Errorf("Linear was fetched on a --repo run")
			}
			if diff := cmp.Diff([]shared.Issue{githubIssue}, printer.issues); diff != "" {
				t.Errorf("printed issues mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("--team fetches only Linear, filtered by that team key", func(t *testing.T) {
			github := &fakeProvider{name: shared.ProviderGithub, issues: []shared.Issue{githubIssue}}
			linear := &fakeProvider{name: shared.ProviderLinear, issues: []shared.Issue{linearIssue}}
			useProviders(t, bothTokens, map[shared.Provider]sync.Provider{
				shared.ProviderGithub: github,
				shared.ProviderLinear: linear,
			}, providerFlags{team: "ENG"})
			printer := usePrinter(t)

			if err := runIssues(t); err != nil {
				t.Fatalf("issues RunE unexpected error: %v", err)
			}

			if linear.gotFilter != "ENG" {
				t.Errorf("Linear filter = %q, want %q", linear.gotFilter, "ENG")
			}
			if github.fetched {
				t.Errorf("GitHub was fetched on a --team run")
			}
			if diff := cmp.Diff([]shared.Issue{linearIssue}, printer.issues); diff != "" {
				t.Errorf("printed issues mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("providers with no issues print an empty list", func(t *testing.T) {
			useProviders(t, bothTokens, map[shared.Provider]sync.Provider{
				shared.ProviderGithub: &fakeProvider{name: shared.ProviderGithub},
				shared.ProviderLinear: &fakeProvider{name: shared.ProviderLinear},
			}, providerFlags{})
			printer := usePrinter(t)

			if err := runIssues(t); err != nil {
				t.Fatalf("issues RunE unexpected error: %v", err)
			}

			if !printer.printed || len(printer.issues) != 0 {
				t.Errorf("printed = %v with %d issues, want an empty list printed", printer.printed, len(printer.issues))
			}
		})
	})

	t.Run("error path", func(t *testing.T) {
		t.Run("a provider error is returned and nothing is printed", func(t *testing.T) {
			fetchErr := errors.New("github is down")
			github := &fakeProvider{name: shared.ProviderGithub, err: fetchErr}
			linear := &fakeProvider{name: shared.ProviderLinear, issues: []shared.Issue{linearIssue}}
			useProviders(t, bothTokens, map[shared.Provider]sync.Provider{
				shared.ProviderGithub: github,
				shared.ProviderLinear: linear,
			}, providerFlags{})
			printer := usePrinter(t)

			err := runIssues(t)

			if !errors.Is(err, fetchErr) {
				t.Errorf("issues RunE error = %v, want %v", err, fetchErr)
			}
			if linear.fetched {
				t.Errorf("Linear was fetched after GitHub failed, want the first error returned")
			}
			if printer.printed {
				t.Errorf("issues were printed after a provider error")
			}
		})

		t.Run("a flag conflict fails before any provider is fetched", func(t *testing.T) {
			github := &fakeProvider{name: shared.ProviderGithub}
			linear := &fakeProvider{name: shared.ProviderLinear}
			useProviders(t, bothTokens, map[shared.Provider]sync.Provider{
				shared.ProviderGithub: github,
				shared.ProviderLinear: linear,
			}, providerFlags{repo: "owner/name", team: "ENG"})
			printer := usePrinter(t)

			err := runIssues(t)

			if err == nil {
				t.Fatalf("issues RunE = nil, want the --repo and --team error")
			}
			if github.fetched || linear.fetched {
				t.Errorf("a provider was fetched (GitHub %v, Linear %v), want none", github.fetched, linear.fetched)
			}
			if printer.printed {
				t.Errorf("issues were printed after a flag error")
			}
		})
	})
}
