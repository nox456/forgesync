package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(projectsCmd)
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects in Notion",
	RunE: func(cmd *cobra.Command, args []string) error {
		projects, err := NotionClient.ListProjects(cmd.Context(), "")

		if err != nil {
			return err
		}

		Printer.PrintProjects(projects)
		return nil
	},
}
