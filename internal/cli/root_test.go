package cli

import (
	"testing"

	"github.com/nox456/forgesync/internal/config"
	"github.com/spf13/cobra"
)

func TestCommandsConfigExemption(t *testing.T) {
	// error path: none — the check is a pure lookup on each command's annotations.
	cases := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		// happy path: exempt commands run without a config
		{name: "version runs without a config", cmd: versionCmd, want: false},
		{name: "update runs without a config", cmd: updateCmd, want: false},
		// edge cases: every command that uses Config or the clients still loads them
		{name: "root loads config", cmd: rootCmd, want: true},
		{name: "config loads config", cmd: configCmd, want: true},
		{name: "projects loads config", cmd: projectsCmd, want: true},
		{name: "issues loads config", cmd: issuesCmd, want: true},
		{name: "status loads config", cmd: statusCmd, want: true},
		{name: "sync loads config", cmd: syncCmd, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := config.ShouldLoadConfig(tc.cmd)
			if got != tc.want {
				t.Errorf("ShouldLoadConfig(%q) = %v, want %v", tc.cmd.Name(), got, tc.want)
			}
		})
	}
}

// hasFlag reports whether cmd accepts the flag, whether registered on cmd
// itself or inherited from a parent's persistent flags.
func hasFlag(cmd *cobra.Command, name string) bool {
	return cmd.Flags().Lookup(name) != nil || cmd.InheritedFlags().Lookup(name) != nil
}

func TestProviderFlags(t *testing.T) {
	// error path: none — flags are registered in init and can't fail at runtime.
	cases := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		// happy path: the provider commands accept the three flags
		{name: "sync", cmd: syncCmd, want: true},
		{name: "status", cmd: statusCmd, want: true},
		{name: "issues", cmd: issuesCmd, want: true},
		// edge cases: commands without providers reject them
		{name: "projects", cmd: projectsCmd, want: false},
		{name: "config", cmd: configCmd, want: false},
		{name: "version", cmd: versionCmd, want: false},
	}

	for _, tc := range cases {
		for _, flag := range []string{"provider", "repo", "team"} {
			t.Run(tc.name+" --"+flag, func(t *testing.T) {
				if got := hasFlag(tc.cmd, flag); got != tc.want {
					t.Errorf("%s accepts --%s = %v, want %v", tc.name, flag, got, tc.want)
				}
			})
		}
	}
}
