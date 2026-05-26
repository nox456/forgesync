package cli

import (
	"fmt"
	"os"

	"github.com/nox456/forgesync/internal/output"
	"github.com/spf13/cobra"
)

var JSONOutput bool
var Printer output.Printer

var rootCmd = &cobra.Command{
	Use:   "forgesync",
	Short: "ForgeSync is a CLI tool for syncing Notion databases with GitHub repositories",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if JSONOutput {
			Printer = output.NewJSONPrinter()
		} else {
			Printer = output.NewTextPrinter()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello, world!")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&JSONOutput, "json", "", false, "Output in JSON format")
}
