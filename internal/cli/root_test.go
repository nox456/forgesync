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
