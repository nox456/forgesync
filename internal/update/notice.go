package update

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var PendingRefresh *Refresh

func IsNewer(current string, latest string) bool {
	currentVersion, err := semver.NewVersion(current)
	if err != nil {
		return false
	}

	latestVersion, err := semver.NewVersion(latest)
	if err != nil {
		return false
	}

	return currentVersion.LessThan(latestVersion)
}

func ShouldCheck(state State, now time.Time, interval time.Duration) bool {
	if interval <= 0 {
		return false
	}

	if state.LastCheckedAt.After(now) {
		return true
	}

	if now.Sub(state.LastCheckedAt) < interval {
		return false
	}

	return true
}

func NewNotice(latest string) string {
	return fmt.Sprintf("Update available! -> %s. Run: forgesync update", latest)
}

func ShouldNotify(cmd *cobra.Command, version string, stderrIsTerminal bool) bool {
	if version == "dev" {
		return false
	}

	cmdName := cmd.Name()

	if cmdName == cobra.ShellCompRequestCmd || cmdName == cobra.ShellCompNoDescRequestCmd || cmdName == "completion" || (cmd.HasParent() && cmd.Parent().Name() == "completion") {
		return false
	}

	if !stderrIsTerminal {
		return false
	}

	return true
}

func StartUpdateNotice(cmd *cobra.Command, currentVersion string) {
	if !ShouldNotify(cmd, currentVersion, term.IsTerminal(int(os.Stderr.Fd()))) {
		return
	}

	path, err := DefaultPath()

	if err != nil {
		slog.Debug(err.Error())
		return
	}

	finder, err := NewUpdater(currentVersion)
	if err != nil {
		slog.Debug(err.Error())
		return
	}

	notifier := Notifier{
		Current: currentVersion,
		Finder:  finder,
		Store:   &Store{Path: path},
		Out:     os.Stderr,
		Now:     time.Now,
		Timeout: RefreshTimeout,
	}

	refresh, err := notifier.Start()
	if err != nil {
		slog.Debug(err.Error())
		return
	}

	PendingRefresh = refresh
}
