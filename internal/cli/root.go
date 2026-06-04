package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/nox456/forgesync/internal/config"
	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
	"github.com/nox456/forgesync/internal/output"
	"github.com/spf13/cobra"
)

var JSONOutput bool
var Printer output.Printer
var Config *config.Config
var GithubClient *github.Client
var NotionClient *notion.Client
var Verbose bool

var rootCmd = &cobra.Command{
	Use:   "forgesync",
	Short: "ForgeSync is a CLI tool for syncing Notion databases with GitHub repositories",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		config, err := config.Load()
		if err != nil {
			return err
		}

		Config = config

		GithubClient = github.NewClient(Config.GitHubToken)
		NotionClient = notion.NewClient(Config.NotionToken, Config.ProjectsSourceId, Config.StoriesSourceId)

		if JSONOutput {
			Printer = output.NewJSONPrinter()
		} else {
			Printer = output.NewTextPrinter()
		}

		if Verbose {
			slog.SetLogLoggerLevel(slog.LevelDebug)
		} else {
			slog.SetLogLoggerLevel(slog.LevelInfo)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()

		return nil
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
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "", false, "Verbose output")
}
