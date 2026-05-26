package cli

import (
	"fmt"

	"github.com/nox456/forgesync/internal/sync"
	"github.com/spf13/cobra"
)

var DryRun bool

func init() {
	syncCmd.Flags().BoolVarP(&DryRun, "dry-run", "d", false, "Dry run - Don't perform any changes, just print the results")
	rootCmd.AddCommand(syncCmd)
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync Notion stories with GitHub issuse (use --dry-run to see what would be changed without actually changing anything)",
	Run: func(cmd *cobra.Command, args []string) {
		engine := sync.NewEngine(NotionClient, GithubClient)

		report, err := engine.Run(cmd.Context(), sync.EngineRunOptions{
			DryRun:     DryRun,
			JSONOutput: JSONOutput,
		})

		if err != nil {
			fmt.Println(err)
			return
		}

		Printer.PrintReport(report)
	},
}
