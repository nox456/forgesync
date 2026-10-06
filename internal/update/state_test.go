package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func assertState(t *testing.T, got State, want State) {
	t.Helper()
	if !got.LastCheckedAt.Equal(want.LastCheckedAt) || got.LatestVersion != want.LatestVersion || got.UpdateCheckInterval != want.UpdateCheckInterval {
		t.Errorf("state = %+v, want %+v", got, want)
	}
}

func TestDefaultPath(t *testing.T) {
	cases := []struct {
		name     string
		xdg      string
		unsetXDG bool
		home     string
		want     string
		wantErr  bool
	}{
		// happy path
		{name: "absolute XDG_STATE_HOME is used", xdg: "/var/xdg-state", home: "/home/tester", want: "/var/xdg-state/forgesync/update-check.yaml"},
		{name: "unset XDG_STATE_HOME falls back to ~/.local/state", unsetXDG: true, home: "/home/tester", want: "/home/tester/.local/state/forgesync/update-check.yaml"},
		// edge cases
		{name: "relative XDG_STATE_HOME is ignored and falls back to ~/.local/state", xdg: "relative/state", home: "/home/tester", want: "/home/tester/.local/state/forgesync/update-check.yaml"},
		{name: "empty XDG_STATE_HOME falls back to ~/.local/state", xdg: "", home: "/home/tester", want: "/home/tester/.local/state/forgesync/update-check.yaml"},
		// error path
		{name: "unknown home directory returns an error", unsetXDG: true, home: "", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_STATE_HOME", tc.xdg)
			if tc.unsetXDG {
				os.Unsetenv("XDG_STATE_HOME")
			}
			t.Setenv("HOME", tc.home)

			got, err := DefaultPath()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("DefaultPath() = %q, nil, want an error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("DefaultPath() returned error: %v", err)
			}
			if got != filepath.FromSlash(tc.want) {
				t.Errorf("DefaultPath() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStoreLoad(t *testing.T) {
	checkedAt := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		setup   func(t *testing.T, path string)
		want    State
		wantErr bool
	}{
		// happy path
		{
			name:  "file with every field loads them all",
			setup: writeFile("last_checked_at: 2026-10-05T12:00:00Z\nlatest_version: v1.2.0\nupdate_check_interval: 30m\n"),
			want:  State{LastCheckedAt: checkedAt, LatestVersion: "v1.2.0", UpdateCheckInterval: "30m"},
		},
		{
			name:  "missing file loads as the zero state without an error",
			setup: func(t *testing.T, path string) {},
			want:  State{},
		},
		// edge cases
		{
			name:  "missing interval falls back to 6h and keeps the other fields",
			setup: writeFile("last_checked_at: 2026-10-05T12:00:00Z\nlatest_version: v1.2.0\n"),
			want:  State{LastCheckedAt: checkedAt, LatestVersion: "v1.2.0", UpdateCheckInterval: "6h"},
		},
		{
			name:  "invalid interval falls back to 6h and keeps the other fields",
			setup: writeFile("last_checked_at: 2026-10-05T12:00:00Z\nlatest_version: v1.2.0\nupdate_check_interval: abc\n"),
			want:  State{LastCheckedAt: checkedAt, LatestVersion: "v1.2.0", UpdateCheckInterval: "6h"},
		},
		{
			name:  "zero interval is kept so the user can disable the check",
			setup: writeFile("latest_version: v1.2.0\nupdate_check_interval: \"0\"\n"),
			want:  State{LatestVersion: "v1.2.0", UpdateCheckInterval: "0"},
		},
		{
			name:  "empty file loads as the zero state with the 6h interval",
			setup: writeFile(""),
			want:  State{UpdateCheckInterval: "6h"},
		},
		// error path
		{
			name:    "corrupt YAML returns the zero state and an error",
			setup:   writeFile("last_checked_at: [\n"),
			want:    State{},
			wantErr: true,
		},
		{
			name:    "wrongly typed field returns the zero state, not a half-decoded one",
			setup:   writeFile("latest_version: v1.2.0\nlast_checked_at: hello\n"),
			want:    State{},
			wantErr: true,
		},
		{
			name: "path that is a directory returns the zero state and an error",
			setup: func(t *testing.T, path string) {
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			},
			want:    State{},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "update-check.yaml")
			tc.setup(t, path)
			store := &Store{Path: path}

			got, err := store.Load()
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Load() error = nil, want an error")
				}
				if !strings.Contains(err.Error(), path) {
					t.Errorf("Load() error = %q, want it to name the path %q", err, path)
				}
			} else if err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}
			assertState(t, got, tc.want)
		})
	}
}

func writeFile(content string) func(t *testing.T, path string) {
	return func(t *testing.T, path string) {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStoreSave(t *testing.T) {
	state := State{
		LastCheckedAt:       time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC),
		LatestVersion:       "v1.2.0",
		UpdateCheckInterval: "30m",
	}

	// happy path
	t.Run("saved state loads back unchanged, including a user-set interval", func(t *testing.T) {
		store := &Store{Path: filepath.Join(t.TempDir(), "update-check.yaml")}

		if err := store.Save(state); err != nil {
			t.Fatalf("Save() returned error: %v", err)
		}
		got, err := store.Load()
		if err != nil {
			t.Fatalf("Load() returned error: %v", err)
		}
		assertState(t, got, state)
	})

	t.Run("second save replaces the first", func(t *testing.T) {
		store := &Store{Path: filepath.Join(t.TempDir(), "update-check.yaml")}
		newer := State{
			LastCheckedAt:       time.Date(2026, 10, 6, 8, 30, 0, 0, time.UTC),
			LatestVersion:       "v1.3.0",
			UpdateCheckInterval: "12h",
		}

		if err := store.Save(state); err != nil {
			t.Fatalf("first Save() returned error: %v", err)
		}
		if err := store.Save(newer); err != nil {
			t.Fatalf("second Save() returned error: %v", err)
		}
		got, err := store.Load()
		if err != nil {
			t.Fatalf("Load() returned error: %v", err)
		}
		assertState(t, got, newer)
	})

	// edge cases
	t.Run("missing parent directories are created", func(t *testing.T) {
		store := &Store{Path: filepath.Join(t.TempDir(), "state", "forgesync", "update-check.yaml")}

		if err := store.Save(state); err != nil {
			t.Fatalf("Save() returned error: %v", err)
		}
		got, err := store.Load()
		if err != nil {
			t.Fatalf("Load() returned error: %v", err)
		}
		assertState(t, got, state)
	})

	t.Run("successful save leaves only the state file in the directory", func(t *testing.T) {
		dir := t.TempDir()
		store := &Store{Path: filepath.Join(dir, "update-check.yaml")}

		if err := store.Save(state); err != nil {
			t.Fatalf("Save() returned error: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 || entries[0].Name() != "update-check.yaml" {
			t.Errorf("directory holds %v, want only update-check.yaml", entryNames(entries))
		}
	})

	// error path
	t.Run("unwritable directory returns an error and leaves no temp file", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}
		dir := t.TempDir()
		if err := os.Chmod(dir, 0500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(dir, 0700) })
		path := filepath.Join(dir, "update-check.yaml")
		store := &Store{Path: path}

		err := store.Save(state)
		if err == nil {
			t.Fatalf("Save() error = nil, want an error")
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("Save() error = %q, want it to name the path %q", err, path)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Errorf("directory holds %v, want it empty", entryNames(entries))
		}
	})
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}
