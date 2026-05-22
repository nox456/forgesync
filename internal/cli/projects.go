package cli

import (
	"fmt"

	"github.com/nox456/forgesync/internal/config"
	"github.com/nox456/forgesync/internal/notion"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(projectsCmd)
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List projects",
	Run: func(cmd *cobra.Command, args []string) {
		config, err := config.Load()

		if err != nil {
			fmt.Println(err)
			return
		}

		notionClient := notion.NewClient(config.NotionToken, config.ProjectsSourceId, config.StoriesSourceId)

		projects, err := notionClient.ListProjects()

		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Projects:")
		for _, project := range projects {
			fmt.Printf("  %-25s %s\n", project.Name, project.Repo)
		}
	},
}
