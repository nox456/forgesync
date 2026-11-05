package cli

import (
	"github.com/nox456/forgesync/internal/shared"
	"github.com/nox456/forgesync/internal/status"
	"github.com/spf13/cobra"
)

func init() {
	statusCmd.Flags().StringVarP(&RepoName, "repo", "", "", "Filter issues from this repo (GitHub, format: owner/repo)")
	statusCmd.Flags().StringVarP(&TeamKey, "team", "", "", "Filter issues from this team (Linear, example: ENG)")
	statusCmd.Flags().StringVarP(&ProviderName, "provider", "", "", "Filter issues from this provider (github or linear)")
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Compute status for a given issue",
	RunE: func(cmd *cobra.Command, args []string) error {
		providers, resolved, err := getActiveProviders()
		if err != nil {
			return err
		}

		collector := status.NewCollector(NotionClient, providers)

		rows, err := collector.Collect(cmd.Context(), shared.ProjectFilter{Repo: resolved.Repo, TeamKey: resolved.TeamKey})

		if err != nil {
			return err
		}

		Printer.PrintStatus(rows)
		return nil
	},
}
