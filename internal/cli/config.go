package cli

import (
	"fmt"

	"github.com/nox456/forgesync/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print the configuration of ForgeSync",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := config.Load()

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Config loaded:")
		fmt.Println("\tGitHub Token:", config.GitHubToken)
		fmt.Println("\tNotion Token:", config.NotionToken)
		fmt.Println("\tProjects Source ID:", config.ProjectsSourceId)
		fmt.Println("\tStories Source ID:", config.StoriesSourceId)
	},
}
