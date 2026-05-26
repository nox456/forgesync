package cli

import (
	"fmt"

	"github.com/nox456/forgesync/internal/config"
	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(issuesCmd)
}

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "List issues",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := config.Load()

		if err != nil {
			fmt.Println(err)
			return
		}

		githubClient := github.NewClient(config.GitHubToken)

		issues, err := githubClient.FetchAssignedIssues(cmd.Context())

		printer := output.NewTextPrinter()

		if err != nil {
			fmt.Println(err)
			return
		}

		printer.PrintIssues(issues)
	},
}
