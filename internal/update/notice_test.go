package update

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		name    string
		current string
		latest  string
		want    bool
	}{
		// happy path
		{name: "newer latest is newer", current: "v1.0.0", latest: "v1.1.0", want: true},
		{name: "equal versions are not newer", current: "v1.1.0", latest: "v1.1.0", want: false},
		{name: "older latest is not newer", current: "v1.2.0", latest: "v1.1.0", want: false},
		// edge cases
		{name: "versions without the v prefix compare", current: "1.0.0", latest: "1.0.1", want: true},
		{name: "mixed v prefix compares by version", current: "v1.0.0", latest: "1.0.1", want: true},
		{name: "mixed v prefix on equal versions is not newer", current: "1.0.0", latest: "v1.0.0", want: false},
		{name: "minor versions compare numerically, not lexically", current: "v0.9.0", latest: "v0.10.0", want: true},
		{name: "release is newer than its pre-release", current: "v1.0.0-rc.1", latest: "v1.0.0", want: true},
		{name: "pre-release is not newer than its release", current: "v1.0.0", latest: "v1.0.0-rc.1", want: false},
		// error path: unparseable versions never report an update
		{name: "dev current version is never outdated", current: "dev", latest: "v1.0.0", want: false},
		{name: "dev latest version is never newer", current: "v1.0.0", latest: "dev", want: false},
		{name: "garbage current version returns false", current: "not-a-version", latest: "v1.0.0", want: false},
		{name: "garbage latest version returns false", current: "v1.0.0", latest: "not-a-version", want: false},
		{name: "empty latest version from a never-checked state returns false", current: "v1.0.0", latest: "", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsNewer(tc.current, tc.latest)
			if got != tc.want {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.want)
			}
		})
	}
}

func TestShouldCheck(t *testing.T) {
	// error path: none — ShouldCheck is total; every odd input degrades to a boolean.
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	interval := 6 * time.Hour

	cases := []struct {
		name     string
		state    State
		interval time.Duration
		want     bool
	}{
		// happy path
		{name: "never checked state checks now", state: State{}, interval: interval, want: true},
		{name: "check inside the interval is skipped", state: State{LastCheckedAt: now.Add(-1 * time.Hour)}, interval: interval, want: false},
		{name: "check past the interval runs", state: State{LastCheckedAt: now.Add(-7 * time.Hour)}, interval: interval, want: true},
		// edge cases
		{name: "check exactly at the interval runs", state: State{LastCheckedAt: now.Add(-interval)}, interval: interval, want: true},
		{name: "check one nanosecond before the interval is skipped", state: State{LastCheckedAt: now.Add(-interval + time.Nanosecond)}, interval: interval, want: false},
		{name: "last check in the future degrades to check now", state: State{LastCheckedAt: now.Add(24 * time.Hour)}, interval: interval, want: true},
		{name: "zero interval disables the check on a never checked state", state: State{}, interval: 0, want: false},
		{name: "zero interval disables the check on a stale state", state: State{LastCheckedAt: now.Add(-30 * 24 * time.Hour)}, interval: 0, want: false},
		{name: "zero interval disables the check on a future state", state: State{LastCheckedAt: now.Add(24 * time.Hour)}, interval: 0, want: false},
		{name: "negative interval disables the check", state: State{}, interval: -1 * time.Hour, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ShouldCheck(tc.state, now, tc.interval)
			if got != tc.want {
				t.Errorf("ShouldCheck(%v, %v, %v) = %v, want %v", tc.state.LastCheckedAt, now, tc.interval, got, tc.want)
			}
		})
	}
}

func TestNewNotice(t *testing.T) {
	// error path: none — NewNotice only formats a string.
	cases := []struct {
		name     string
		latest   string
		wantPart string
	}{
		// happy path
		{name: "notice suggests running forgesync update", latest: "v1.2.0", wantPart: "forgesync update"},
		{name: "notice names the latest version", latest: "v1.2.0", wantPart: "v1.2.0"},
		// edge cases
		{name: "notice keeps a pre-release tag intact", latest: "v2.0.0-rc.1", wantPart: "v2.0.0-rc.1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NewNotice(tc.latest)
			if !strings.Contains(got, tc.wantPart) {
				t.Errorf("NewNotice(%q) = %q, want it to contain %q", tc.latest, got, tc.wantPart)
			}
		})
	}
}

func TestShouldNotify(t *testing.T) {
	command := func(name string) func() *cobra.Command {
		return func() *cobra.Command { return &cobra.Command{Use: name} }
	}
	completionChild := func(shell string) func() *cobra.Command {
		return func() *cobra.Command {
			parent := &cobra.Command{Use: "completion"}
			child := &cobra.Command{Use: shell}
			parent.AddCommand(child)
			return child
		}
	}
	exempt := func() *cobra.Command {
		return &cobra.Command{Use: "version", Annotations: map[string]string{"group": "internal"}}
	}

	cases := []struct {
		name     string
		cmd      func() *cobra.Command
		version  string
		terminal bool
		want     bool
	}{
		// happy path
		{name: "normal command on a terminal with a real version notifies", cmd: command("status"), version: "0.9.0", terminal: true, want: true},
		{name: "exempt version command still notifies", cmd: exempt, version: "0.9.0", terminal: true, want: true},
		// edge cases: shell completion never notifies, even on a terminal
		{name: "__complete request is skipped", cmd: command(cobra.ShellCompRequestCmd), version: "0.9.0", terminal: true, want: false},
		{name: "__completeNoDesc request is skipped", cmd: command(cobra.ShellCompNoDescRequestCmd), version: "0.9.0", terminal: true, want: false},
		{name: "completion command is skipped", cmd: command("completion"), version: "0.9.0", terminal: true, want: false},
		{name: "completion zsh subcommand is skipped", cmd: completionChild("zsh"), version: "0.9.0", terminal: true, want: false},
		{name: "completion bash subcommand is skipped", cmd: completionChild("bash"), version: "0.9.0", terminal: true, want: false},
		{name: "command named like a shell outside completion still notifies", cmd: command("bash"), version: "0.9.0", terminal: true, want: true},
		// error path: inputs that must never touch the network or the state file
		{name: "dev build never notifies", cmd: command("status"), version: "dev", terminal: true, want: false},
		{name: "stderr that is not a terminal never notifies", cmd: command("status"), version: "0.9.0", terminal: false, want: false},
		{name: "dev build off a terminal never notifies", cmd: command("status"), version: "dev", terminal: false, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.cmd()
			got := ShouldNotify(cmd, tc.version, tc.terminal)
			if got != tc.want {
				t.Errorf("ShouldNotify(%q, %q, %v) = %v, want %v", cmd.CommandPath(), tc.version, tc.terminal, got, tc.want)
			}
		})
	}
}
