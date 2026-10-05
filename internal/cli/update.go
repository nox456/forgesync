package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use: "update",
	Annotations: map[string]string{
		"group": "internal",
	},
	Short: "Update ForgeSync to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("not implemented")
		return nil
	},
}
