package config

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestShouldLoadConfig(t *testing.T) {
	// error path: none — ShouldLoadConfig is total and has no failure mode.
	cases := []struct {
		name        string
		annotations map[string]string
		want        bool
	}{
		// happy path
		{
			name:        "command in the internal group skips config loading",
			annotations: map[string]string{"group": "internal"},
			want:        false,
		},
		{
			name:        "command with no annotations loads config",
			annotations: nil,
			want:        true,
		},
		// edge cases
		{
			name:        "command with an empty annotations map loads config",
			annotations: map[string]string{},
			want:        true,
		},
		{
			name:        "command in another group loads config",
			annotations: map[string]string{"group": "sync"},
			want:        true,
		},
		{
			name:        "group match is case-sensitive",
			annotations: map[string]string{"group": "Internal"},
			want:        true,
		},
		{
			name:        "empty group value loads config",
			annotations: map[string]string{"group": ""},
			want:        true,
		},
		{
			name:        "internal value under a different key loads config",
			annotations: map[string]string{"category": "internal"},
			want:        true,
		},
		{
			name:        "internal group alongside other annotations skips config loading",
			annotations: map[string]string{"group": "internal", "other": "value"},
			want:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test", Annotations: tc.annotations}

			got := ShouldLoadConfig(cmd)
			if got != tc.want {
				t.Errorf("ShouldLoadConfig() = %v, want %v", got, tc.want)
			}
		})
	}
}
