package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print the configuration of ForgeSync",
	RunE: func(cmd *cobra.Command, args []string) error {
		githubToken := "(not set)"
		if Config.GitHubToken != "" {
			githubToken = Config.GitHubToken
		}

		linearToken := "(not set)"
		if Config.LinearToken != "" {
			linearToken = Config.LinearToken
		}

		fmt.Println("Config loaded:")
		fmt.Println("  GitHub Token:", githubToken)
		fmt.Println("  Linear Token:", linearToken)
		fmt.Println("  Notion Token:", Config.NotionToken)
		fmt.Println("  Projects Source ID:", Config.ProjectsSourceId)
		fmt.Println("  Stories Source ID:", Config.StoriesSourceId)

		return nil
	},
}
