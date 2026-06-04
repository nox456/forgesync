package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of ForgeSync",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("v0.0.1")

		return nil
	},
}
