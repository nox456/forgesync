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
	RunE:          runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	if JSONOutput {
		return errors.New("update does not support --json")
	}

	if VERSION == "dev" {
		info, _ := debug.ReadBuildInfo()

		if update.IsGoInstall(info) {
			return fmt.Errorf("forgesync was installed with go install; update it with:\n  %s", update.GoInstallCommand)
		}

		return errors.New("this is a development build; update by rebuilding from source")
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()

	u, err := update.NewUpdater(VERSION)
	if err != nil {
		return err
	}

	detectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rel, err := u.Latest(detectCtx)
	if err != nil {
		if errors.Is(err, update.ErrNoRelease) {
			return fmt.Errorf("no release is published for %s/%s", runtime.GOOS, runtime.GOARCH)
		}
		return err
	}

	out := cmd.OutOrStdout()

	if !update.IsNewer(VERSION, rel.Version) {
		fmt.Fprintf(out, "forgesync is already up to date (%s)\n", VERSION)
		return nil
	}

	if !confirm(cmd.InOrStdin(), out, fmt.Sprintf("Update forgesync %s → %s?", VERSION, rel.Version)) {
		fmt.Fprintln(out, "Update cancelled.")
		return nil
	}

	fmt.Fprintf(out, "Downloading forgesync %s…\n", rel.Version)

	if err := u.Apply(ctx, rel); err != nil {
		var perr *update.PermissionError
		if errors.As(err, &perr) {
			hint := "Rerun with sudo: sudo forgesync update"
			if runtime.GOOS == "windows" {
				hint = `Rerun from a terminal opened with "Run as administrator".`
			}
			return fmt.Errorf("permission denied: cannot write to %s\n%s", perr.Dir, hint)
		}

		if errors.Is(err, update.ErrRollbackFailed) {
			return fmt.Errorf("%w\nReinstall forgesync with install.sh: https://github.com/nox456/forgesync#install", err)
		}

		return err
	}

	fmt.Fprintf(out, "Updated forgesync %s → %s\n", VERSION, rel.Version)
	fmt.Fprintf(out, "Release notes: %s\n", rel.URL)

	return nil
}
