package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nox456/forgesync/internal/config"
)

// captureStdout runs print with os.Stdout redirected to a pipe and returns
// everything it wrote.
func captureStdout(t *testing.T, print func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}
	prev := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = prev }()

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, reader)
		done <- buf.String()
	}()

	print()

	writer.Close()
	return <-done
}

func TestConfigCommand(t *testing.T) {
	// error path: none — RunE only prints the config that was already loaded.
	cases := []struct {
		name      string
		cfg       *config.Config
		wantLines []string
	}{
		// happy path
		{
			name:      "both tokens are printed",
			cfg:       &config.Config{GitHubToken: "gh-token", LinearToken: "lin-token"},
			wantLines: []string{"  GitHub Token: gh-token", "  Linear Token: lin-token"},
		},
		// edge cases: an empty token shows which provider is unavailable
		{
			name:      "an empty Linear token prints (not set)",
			cfg:       &config.Config{GitHubToken: "gh-token"},
			wantLines: []string{"  GitHub Token: gh-token", "  Linear Token: (not set)"},
		},
		{
			name:      "an empty GitHub token prints (not set)",
			cfg:       &config.Config{LinearToken: "lin-token"},
			wantLines: []string{"  GitHub Token: (not set)", "  Linear Token: lin-token"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prev := Config
			t.Cleanup(func() { Config = prev })
			Config = tc.cfg

			var runErr error
			out := captureStdout(t, func() { runErr = configCmd.RunE(configCmd, nil) })

			if runErr != nil {
				t.Fatalf("config RunE unexpected error: %v", runErr)
			}
			for _, line := range tc.wantLines {
				if !strings.Contains(out, line+"\n") {
					t.Errorf("config output missing line %q:\n%s", line, out)
				}
			}
		})
	}
}
