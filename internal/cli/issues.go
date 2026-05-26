package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(issuesCmd)
}

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "List issues in GitHub",
	Run: func(cmd *cobra.Command, args []string) {
		issues, err := GithubClient.FetchAssignedIssues(cmd.Context())

		if err != nil {
			fmt.Println(err)
			return
		}

		Printer.PrintIssues(issues)
	},
}
