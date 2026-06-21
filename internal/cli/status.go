package cli

import (
	"github.com/nox456/forgesync/internal/status"
	"github.com/spf13/cobra"
)

func init() {
	statusCmd.Flags().StringVarP(&RepoName, "repo", "", "", "Only sync issues from this repo (format: owner/repo)")
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Compute status for a given issue",
	RunE: func(cmd *cobra.Command, args []string) error {
		collector := status.NewCollector(NotionClient, GithubClient)

		rows, err := collector.Collect(cmd.Context(), RepoName)

		if err != nil {
			return err
		}

		Printer.PrintStatus(rows)
		return nil
	},
}
