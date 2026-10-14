package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/nox456/forgesync/internal/update"
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
	Short:         "Update ForgeSync to the latest version",
	Args:          cobra.NoArgs,
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if JSONOutput {
			return errors.New("update does not support --json")
		}

		if VERSION == "dev" {
			info, _ := debug.ReadBuildInfo()

			if update.IsGoInstall(info) {
				return errors.New("forgesync was installed with go install; update it with:\n\tgo install github.com/nox456/forgesync/cmd/forgesync@latest")
			}

			return errors.New("this is a development build; update by rebuilding from source")
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()

		u, err := update.NewUpdater(VERSION)
		if err != nil {
			return err
		}

		contextWithTimeout, stop := context.WithTimeout(ctx, 15*time.Second)
		defer stop()

		rel, err := u.Latest(contextWithTimeout)
		if err != nil {
			if errors.Is(err, update.ErrNoRelease) {
				return fmt.Errorf("no release is published for %s/%s", runtime.GOOS, runtime.GOARCH)
			}
			return err
		}

		if !update.IsNewer(VERSION, rel.Version) {
			fmt.Printf("forgesync is already up to date (%s)\n", VERSION)
			return nil
		}

		if !confirm(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("Update forgesync %s → %s?", VERSION, rel.Version)) {
			fmt.Println("Update Cancelled")
			return nil
		}

		fmt.Printf("Downloading forgesync %s…\n", rel.Version)

		err = u.Apply(ctx, rel)

		if err != nil {
			var perr *update.PermissionError

			errors.As(err, &perr)

			if perr != nil {
				fmt.Printf("permission denied: cannot write to %s\n", perr.Dir)
				if runtime.GOOS == "windows" {
					fmt.Printf("Rerun from a terminal opened with \"Run as administrator\".\n")
				} else {
					fmt.Printf("Rerun with sudo: sudo forgesync update\n")
				}

				return err
			}

			if errors.Is(err, update.ErrRollbackFailed) {
				fmt.Printf("Update failed and the previous binary could not be put back.\n")
				return err
			}

			return err
		}

		fmt.Printf("Updated forgesync %s → %s\n", VERSION, rel.Version)
		fmt.Printf("Release notes: %s\n", rel.URL)

		return nil
	},
}
