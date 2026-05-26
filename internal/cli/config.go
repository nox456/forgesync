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
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Config loaded:")
		fmt.Println("  GitHub Token:", Config.GitHubToken)
		fmt.Println("  Notion Token:", Config.NotionToken)
		fmt.Println("  Projects Source ID:", Config.ProjectsSourceId)
		fmt.Println("  Stories Source ID:", Config.StoriesSourceId)
	},
}
