package cli

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// runRoot executes the root command with args the way main does and returns
// what cobra wrote to stdout and stderr, restoring the globals it touches.
func runRoot(t *testing.T, args ...string) (string, string, error) {
	t.Helper()

	prevLogger, prevJSON := slog.Default(), JSONOutput
	usePrinter(t)
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		slog.SetDefault(prevLogger)
		JSONOutput = prevJSON
	})

	var stdout, stderr bytes.Buffer
	rootCmd.SetArgs(args)
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

// update used to silence cobra and rely on Execute to print its errors. Now
// cobra prints every command's error once, so update must not silence it.
func TestUpdateErrorsReachStderr(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		t.Run("a failed update prints its error once, with cobra's prefix and no usage", func(t *testing.T) {
			stdout, stderr, err := runRoot(t, "update")

			if err == nil {
				t.Fatalf("update = nil, want the development build error")
			}
			want := "Error: this is a development build; update by rebuilding from source\n"
			if stderr != want {
				t.Errorf("stderr = %q, want %q", stderr, want)
			}
			if strings.Contains(stdout, "Usage:") {
				t.Errorf("stdout = %q, want no usage after an update error", stdout)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("the --json refusal is printed once too", func(t *testing.T) {
			_, stderr, err := runRoot(t, "update", "--json")

			if err == nil {
				t.Fatalf("update --json = nil, want the --json refusal")
			}
			want := "Error: update does not support --json\n"
			if stderr != want {
				t.Errorf("stderr = %q, want %q", stderr, want)
			}
		})
	})

	// error path: none — every case here is a failing update; what is under
	// test is how that failure reaches the user.
}
