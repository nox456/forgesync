package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(issuesCmd)
}

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "List issues in GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		issues, err := GithubClient.FetchAssignedIssues(cmd.Context(), "")

		if err != nil {
			return err
		}

		Printer.PrintIssues(issues)
		return nil
	},
}
