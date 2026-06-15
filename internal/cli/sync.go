package cli

import (
	"github.com/nox456/forgesync/internal/sync"
	"github.com/spf13/cobra"
)

var DryRun bool
var repoName string

func init() {
	syncCmd.Flags().BoolVarP(&DryRun, "dry-run", "d", false, "Dry run - Don't perform any changes, just print the results")
	syncCmd.Flags().StringVarP(&repoName, "repo", "", "", "Only sync issues from this repo (format: owner/repo)")
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Notion stories with GitHub issuse (use --dry-run to see what would be changed without actually changing anything)",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine := sync.NewEngine(NotionClient, GithubClient)

		report, err := engine.Run(cmd.Context(), sync.EngineRunOptions{
			DryRun: DryRun,
			RepoFilter: repoName,
		})

		if err != nil {
			return err
		}

		Printer.PrintReport(report)
		return nil
	},
}
