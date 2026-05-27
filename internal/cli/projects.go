package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(projectsCmd)
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects in Notion",
	Run: func(cmd *cobra.Command, args []string) {
		projects, err := NotionClient.ListProjects(cmd.Context())

		if err != nil {
			fmt.Println(err)
			return
		}

		Printer.PrintProjects(projects)
	},
}
