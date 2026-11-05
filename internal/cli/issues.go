package cli

import (
	"errors"
	"log/slog"

	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/sync"
	"github.com/spf13/cobra"
)

func init() {
	issuesCmd.Flags().StringVarP(&RepoName, "repo", "", "", "Filter issues from this repo (GitHub, format: owner/repo)")
	issuesCmd.Flags().StringVarP(&TeamKey, "team", "", "", "Filter issues from this team (Linear, example: ENG)")
	issuesCmd.Flags().StringVarP(&ProviderName, "provider", "", "", "Filter issues from this provider (github or linear)")
	rootCmd.AddCommand(issuesCmd)
}

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "List your assigned GitHub and Linear issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		providers, resolved, err := getActiveProviders(true)
		if err != nil {
			return err
		}

		var issues []shared.Issue
		var fetchErrors []error
		reportErrors := make([]sync.ReportError, 0)

		for _, p := range providers {
			f := resolved.Repo
			if p.Name() == shared.ProviderLinear {
				f = resolved.TeamKey
			}

			got, err := p.FetchAssignedIssues(cmd.Context(), f)
			if err != nil {
				slog.Error(err.Error())
				reportErrors = append(reportErrors, sync.ReportError{
					Provider: p.Name(),
					Error:    err.Error(),
				})
				fetchErrors = append(fetchErrors, err)
				continue
			}

			issues = append(issues, got...)
		}

		if len(fetchErrors) > 0 && len(fetchErrors) == len(providers) {
			return errors.Join(fetchErrors...)
		}

		Printer.PrintIssues(issues, reportErrors)
		return nil
	},
}
