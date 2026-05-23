package cli

import (
	"fmt"

	"github.com/nox456/forgesync/internal/config"
	"github.com/nox456/forgesync/internal/github"
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

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Issues:")
		for _, issue := range issues {
			fmt.Printf("  %-30s [%s] %s\n", issue.Repo, issue.State, issue.Title)
		}
	},
}
