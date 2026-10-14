package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/nox456/forgesync/internal/config"
	"github.com/nox456/forgesync/internal/github"
	"github.com/nox456/forgesync/internal/notion"
	"github.com/nox456/forgesync/internal/output"
	"github.com/nox456/forgesync/internal/update"
	"github.com/spf13/cobra"
)

var JSONOutput bool
var Printer output.Printer
var Config *config.Config
var GithubClient *github.Client
var NotionClient *notion.Client
var Verbose bool
var RepoName string

var rootCmd = &cobra.Command{
	Use:   "forgesync",
	Short: "ForgeSync is a CLI tool for syncing Notion databases with GitHub repositories",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if Verbose {
			slog.SetLogLoggerLevel(slog.LevelDebug)
		} else {
			slog.SetLogLoggerLevel(slog.LevelInfo)
		}

		if JSONOutput {
			Printer = output.NewJSONPrinter()

			disabledLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
			slog.SetDefault(disabledLogger)
		} else {
			Printer = output.NewTextPrinter()
		}

		if cmd.Name() != "update" {
			update.StartUpdateNotice(cmd, VERSION)
		}

		if !config.ShouldLoadConfig(cmd) {
			return nil
		}

		config, err := config.Load()
		if err != nil {
			return err
		}

		Config = config

		GithubClient, err = github.NewClient(Config.GitHubToken)
		if err != nil {
			return err
		}
		NotionClient = notion.NewClient(Config.NotionToken, Config.ProjectsSourceId, Config.StoriesSourceId)

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Help()

		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()

	if update.PendingRefresh != nil {
		if err := update.PendingRefresh.Wait(); err != nil {
			slog.Debug(err.Error())
		}
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&JSONOutput, "json", "", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "", false, "Verbose output")
}
