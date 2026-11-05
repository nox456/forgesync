package cli

import (
	"github.com/nox456/forgesync/internal/shared"
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
	Short: "List issues in GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		providers, resolved, err := getActiveProviders()
		if err != nil {
			return err
		}

		var issues []shared.Issue

		for _, p := range providers {
			f := resolved.Repo
			if p.Name() == shared.ProviderLinear {
				f = resolved.TeamKey
			}

			got, err := p.FetchAssignedIssues(cmd.Context(), f)
			if err != nil {
				return err
			}

			issues = append(issues, got...)
		}

		Printer.PrintIssues(issues)
		return nil
	},
}
