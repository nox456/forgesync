package cli

import (
	"fmt"

	"github.com/nox456/forgesync/internal/config"
	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
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
	Short: "Sync",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := config.Load()

		if err != nil {
			fmt.Println(err)
			return
		}

		notionClient := notion.NewClient(config.NotionToken, config.ProjectsSourceId, config.StoriesSourceId)
		githubClient := github.NewClient(config.GitHubToken)

		engine := sync.NewEngine(notionClient, githubClient)

		report, err := engine.Run(cmd.Context(), sync.EngineRunOptions{
			DryRun: DryRun,
		})

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("\n=== Sync Report ===\n")
		fmt.Printf("  Created: %d\n", report.Created)
		fmt.Printf("  Updated: %d\n", report.Updated)
		fmt.Printf("  Skipped: %d\n", report.Skipped)
		fmt.Printf("  Unchanged: %d\n", report.Unchanged)
		fmt.Printf("  Errors:  %d\n", report.Errors)
	},
}
