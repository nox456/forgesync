package update

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var notifierNow = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

// fakeFinder stands in for *Updater. A delay respects ctx, so a slow fake returns when the
// context is cancelled. stall blocks until the channel is closed and ignores ctx, so the
// timeout case only ever sees the deadline fire.
type fakeFinder struct {
	latest string
	err    error
	delay  time.Duration
	stall  chan struct{}
	calls  int
}

func (f *fakeFinder) LatestVersion(ctx context.Context) (string, error) {
	f.calls++
	if f.stall != nil {
		<-f.stall
		return "", ctx.Err()
	}
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return f.latest, f.err
}

func newTestNotifier(t *testing.T, current string, finder *fakeFinder) (*Notifier, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &Notifier{
		Current: current,
		Finder:  finder,
		Store:   &Store{Path: filepath.Join(t.TempDir(), "update-check.yaml")},
		Out:     out,
		Now:     func() time.Time { return notifierNow },
		Timeout: time.Second,
	}, out
}

func seedState(t *testing.T, store *Store, state State) {
	t.Helper()
	if err := store.Save(state); err != nil {
		t.Fatalf("seeding state: %v", err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return data
}

func loadState(t *testing.T, store *Store) State {
	t.Helper()
	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	return state
}

func TestNotifierStart(t *testing.T) {
	fresh := notifierNow.Add(-1 * time.Hour)
	stale := notifierNow.Add(-7 * time.Hour)

	// happy path
	t.Run("cached newer version prints exactly the notice line", func(t *testing.T) {
		finder := &fakeFinder{}
		n, out := newTestNotifier(t, "0.9.0", finder)
		seedState(t, n.Store, State{LastCheckedAt: fresh, LatestVersion: "0.10.0", UpdateCheckInterval: "6h"})

		refresh, err := n.Start()
		if err != nil {
			t.Fatalf("Start() returned error: %v", err)
		}
		if refresh != nil {
			t.Errorf("Start() returned a refresh for a fresh state, want nil")
		}
		want := NewNotice("0.10.0") + "\n"
		if out.String() != want {
			t.Errorf("output = %q, want %q", out.String(), want)
		}
	})

	t.Run("stale state starts a refresh that calls the finder once", func(t *testing.T) {
		finder := &fakeFinder{latest: "0.10.0"}
		n, _ := newTestNotifier(t, "0.9.0", finder)
		seedState(t, n.Store, State{LastCheckedAt: stale, LatestVersion: "0.9.0", UpdateCheckInterval: "6h"})

		refresh, err := n.Start()
		if err != nil {
			t.Fatalf("Start() returned error: %v", err)
		}
		if refresh == nil {
			t.Fatalf("Start() returned nil for a stale state, want a refresh")
		}
		if err := refresh.Wait(); err != nil {
			t.Fatalf("Wait() returned error: %v", err)
		}
		if finder.calls != 1 {
			t.Errorf("finder called %d times, want 1", finder.calls)
		}
	})

	// edge cases
	noNotice := []struct {
		name   string
		cached string
	}{
		{name: "equal cached version prints nothing", cached: "0.9.0"},
		{name: "older cached version prints nothing", cached: "0.8.0"},
		{name: "empty cached version prints nothing", cached: ""},
	}
	for _, tc := range noNotice {
		t.Run(tc.name, func(t *testing.T) {
			n, out := newTestNotifier(t, "0.9.0", &fakeFinder{})
			seedState(t, n.Store, State{LastCheckedAt: fresh, LatestVersion: tc.cached, UpdateCheckInterval: "6h"})

			if _, err := n.Start(); err != nil {
				t.Fatalf("Start() returned error: %v", err)
			}
			if out.Len() != 0 {
				t.Errorf("output = %q, want nothing", out.String())
			}
		})
	}

	t.Run("fresh state returns no refresh and never calls the finder", func(t *testing.T) {
		finder := &fakeFinder{latest: "0.10.0"}
		n, _ := newTestNotifier(t, "0.9.0", finder)
		seedState(t, n.Store, State{LastCheckedAt: fresh, LatestVersion: "0.9.0", UpdateCheckInterval: "6h"})

		refresh, err := n.Start()
		if err != nil {
			t.Fatalf("Start() returned error: %v", err)
		}
		if refresh != nil {
			t.Errorf("Start() returned a refresh for a fresh state, want nil")
		}
		if finder.calls != 0 {
			t.Errorf("finder called %d times, want 0", finder.calls)
		}
	})

	t.Run("zero interval never refreshes, even when never checked", func(t *testing.T) {
		finder := &fakeFinder{latest: "0.10.0"}
		n, _ := newTestNotifier(t, "0.9.0", finder)
		seedState(t, n.Store, State{UpdateCheckInterval: "0"})

		refresh, err := n.Start()
		if err != nil {
			t.Fatalf("Start() returned error: %v", err)
		}
		if refresh != nil {
			t.Errorf("Start() returned a refresh with interval 0, want nil")
		}
		if finder.calls != 0 {
			t.Errorf("finder called %d times, want 0", finder.calls)
		}
	})

	t.Run("zero interval still prints a cached newer version", func(t *testing.T) {
		n, out := newTestNotifier(t, "0.9.0", &fakeFinder{})
		seedState(t, n.Store, State{LastCheckedAt: stale, LatestVersion: "0.10.0", UpdateCheckInterval: "0"})

		if _, err := n.Start(); err != nil {
			t.Fatalf("Start() returned error: %v", err)
		}
		want := NewNotice("0.10.0") + "\n"
		if out.String() != want {
			t.Errorf("output = %q, want %q", out.String(), want)
		}
	})

	t.Run("missing file prints nothing and starts a refresh", func(t *testing.T) {
		finder := &fakeFinder{latest: "0.10.0"}
		n, out := newTestNotifier(t, "0.9.0", finder)

		refresh, err := n.Start()
		if err != nil {
			t.Fatalf("Start() returned error: %v", err)
		}
		if refresh == nil {
			t.Fatalf("Start() returned nil on the first run, want a refresh")
		}
		if err := refresh.Wait(); err != nil {
			t.Fatalf("Wait() returned error: %v", err)
		}
		if out.Len() != 0 {
			t.Errorf("output = %q, want nothing", out.String())
		}
	})

	// error path
	t.Run("corrupt state returns the load error, prints nothing and never calls the finder", func(t *testing.T) {
		finder := &fakeFinder{latest: "0.10.0"}
		n, out := newTestNotifier(t, "0.9.0", finder)
		if err := os.WriteFile(n.Store.Path, []byte("last_checked_at: [not a time\n"), 0600); err != nil {
			t.Fatal(err)
		}

		refresh, err := n.Start()
		if err == nil {
			t.Fatalf("Start() error = nil, want the load error")
		}
		if !strings.Contains(err.Error(), n.Store.Path) {
			t.Errorf("Start() error = %q, want it to name the path %q", err, n.Store.Path)
		}
		if refresh != nil {
			t.Errorf("Start() returned a refresh over a corrupt file, want nil")
		}
		if out.Len() != 0 {
			t.Errorf("output = %q, want nothing", out.String())
		}
		if finder.calls != 0 {
			t.Errorf("finder called %d times, want 0", finder.calls)
		}
	})
}

func TestRefreshWait(t *testing.T) {
	stale := notifierNow.Add(-7 * time.Hour)

	// happy path
	t.Run("successful refresh saves now, the new version and the user's interval", func(t *testing.T) {
		n, _ := newTestNotifier(t, "0.9.0", &fakeFinder{latest: "0.10.0"})
		seedState(t, n.Store, State{LastCheckedAt: notifierNow.Add(-13 * time.Hour), LatestVersion: "0.9.0", UpdateCheckInterval: "12h"})

		refresh, err := n.Start()
		if err != nil || refresh == nil {
			t.Fatalf("Start() = %v, %v, want a refresh and no error", refresh, err)
		}
		if err := refresh.Wait(); err != nil {
			t.Fatalf("Wait() returned error: %v", err)
		}
		assertState(t, loadState(t, n.Store), State{LastCheckedAt: notifierNow, LatestVersion: "0.10.0", UpdateCheckInterval: "12h"})
	})

	t.Run("first run writes the 6h interval into the file", func(t *testing.T) {
		n, _ := newTestNotifier(t, "0.9.0", &fakeFinder{latest: "0.10.0"})

		refresh, err := n.Start()
		if err != nil || refresh == nil {
			t.Fatalf("Start() = %v, %v, want a refresh and no error", refresh, err)
		}
		if err := refresh.Wait(); err != nil {
			t.Fatalf("Wait() returned error: %v", err)
		}
		if data := readFile(t, n.Store.Path); !strings.Contains(string(data), "update_check_interval: 6h") {
			t.Errorf("state file = %q, want it to contain update_check_interval: 6h", data)
		}
	})

	// edge cases
	t.Run("no release for the platform bumps last_checked_at and keeps the cached version", func(t *testing.T) {
		n, _ := newTestNotifier(t, "0.9.0", &fakeFinder{err: ErrNoRelease})
		seedState(t, n.Store, State{LastCheckedAt: stale, LatestVersion: "0.10.0", UpdateCheckInterval: "6h"})

		refresh, err := n.Start()
		if err != nil || refresh == nil {
			t.Fatalf("Start() = %v, %v, want a refresh and no error", refresh, err)
		}
		if err := refresh.Wait(); err != nil {
			t.Fatalf("Wait() returned error: %v", err)
		}
		assertState(t, loadState(t, n.Store), State{LastCheckedAt: notifierNow, LatestVersion: "0.10.0", UpdateCheckInterval: "6h"})
	})

	t.Run("wrapped no-release error is treated the same way", func(t *testing.T) {
		n, _ := newTestNotifier(t, "0.9.0", &fakeFinder{err: fmt.Errorf("detect: %w", ErrNoRelease)})
		seedState(t, n.Store, State{LastCheckedAt: stale, LatestVersion: "0.10.0", UpdateCheckInterval: "6h"})

		refresh, err := n.Start()
		if err != nil || refresh == nil {
			t.Fatalf("Start() = %v, %v, want a refresh and no error", refresh, err)
		}
		if err := refresh.Wait(); err != nil {
			t.Fatalf("Wait() returned error: %v", err)
		}
		assertState(t, loadState(t, n.Store), State{LastCheckedAt: notifierNow, LatestVersion: "0.10.0", UpdateCheckInterval: "6h"})
	})

	t.Run("slow finder that answers before the timeout still saves", func(t *testing.T) {
		n, _ := newTestNotifier(t, "0.9.0", &fakeFinder{latest: "0.10.0", delay: 20 * time.Millisecond})
		seedState(t, n.Store, State{LastCheckedAt: stale, LatestVersion: "0.9.0", UpdateCheckInterval: "6h"})

		refresh, err := n.Start()
		if err != nil || refresh == nil {
			t.Fatalf("Start() = %v, %v, want a refresh and no error", refresh, err)
		}
		if err := refresh.Wait(); err != nil {
			t.Fatalf("Wait() returned error: %v", err)
		}
		assertState(t, loadState(t, n.Store), State{LastCheckedAt: notifierNow, LatestVersion: "0.10.0", UpdateCheckInterval: "6h"})
	})

	// error path
	t.Run("timed-out refresh returns ErrRefreshTimeout and leaves the file unchanged", func(t *testing.T) {
		stall := make(chan struct{})
		t.Cleanup(func() { close(stall) })
		n, _ := newTestNotifier(t, "0.9.0", &fakeFinder{latest: "0.10.0", stall: stall})
		n.Timeout = 10 * time.Millisecond
		seedState(t, n.Store, State{LastCheckedAt: stale, LatestVersion: "0.9.0", UpdateCheckInterval: "6h"})
		before := readFile(t, n.Store.Path)

		refresh, err := n.Start()
		if err != nil || refresh == nil {
			t.Fatalf("Start() = %v, %v, want a refresh and no error", refresh, err)
		}
		err = refresh.Wait()
		if !errors.Is(err, ErrRefreshTimeout) {
			t.Errorf("Wait() error = %v, want ErrRefreshTimeout", err)
		}
		if after := readFile(t, n.Store.Path); !bytes.Equal(before, after) {
			t.Errorf("state file changed after a timeout:\nbefore %q\nafter  %q", before, after)
		}
	})

	t.Run("finder error is returned and leaves the file unchanged", func(t *testing.T) {
		networkErr := errors.New("network down")
		n, out := newTestNotifier(t, "0.9.0", &fakeFinder{err: networkErr})
		seedState(t, n.Store, State{LastCheckedAt: stale, LatestVersion: "0.9.0", UpdateCheckInterval: "6h"})
		before := readFile(t, n.Store.Path)

		refresh, err := n.Start()
		if err != nil || refresh == nil {
			t.Fatalf("Start() = %v, %v, want a refresh and no error", refresh, err)
		}
		err = refresh.Wait()
		if !errors.Is(err, networkErr) {
			t.Errorf("Wait() error = %v, want %v", err, networkErr)
		}
		if after := readFile(t, n.Store.Path); !bytes.Equal(before, after) {
			t.Errorf("state file changed after a failed refresh:\nbefore %q\nafter  %q", before, after)
		}
		if out.Len() != 0 {
			t.Errorf("output = %q, want nothing", out.String())
		}
	})

	t.Run("save error is returned to the caller", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}
		n, _ := newTestNotifier(t, "0.9.0", &fakeFinder{latest: "0.10.0"})
		dir := filepath.Dir(n.Store.Path)
		if err := os.Chmod(dir, 0500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(dir, 0700) })

		refresh, err := n.Start()
		if err != nil || refresh == nil {
			t.Fatalf("Start() = %v, %v, want a refresh and no error", refresh, err)
		}
		err = refresh.Wait()
		if err == nil {
			t.Fatalf("Wait() error = nil, want the save error")
		}
		if !strings.Contains(err.Error(), n.Store.Path) {
			t.Errorf("Wait() error = %q, want it to name the path %q", err, n.Store.Path)
		}
	})
}
