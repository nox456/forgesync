package update

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/creativeprojects/go-selfupdate"

	selfupdateapply "github.com/creativeprojects/go-selfupdate/update"
)

type Updater struct {
	current string
	lib     *selfupdate.Updater
}

type Release struct {
	Version string
	URL     string
	rel     *selfupdate.Release
}

type PermissionError struct {
	Dir string
	Err error
}

func (e *PermissionError) Error() string { return fmt.Sprintf("cannot write to %s: %v", e.Dir, e.Err) }
func (e *PermissionError) Unwrap() error { return e.Err }

var ErrNoRelease = errors.New("no release found for this platform")
var ErrRollbackFailed = errors.New("update failed and the previous binary could not be restored")

func NewUpdater(current string) (*Updater, error) {
	lib, err := newLibUpdater()
	if err != nil {
		return nil, err
	}
	return &Updater{current: current, lib: lib}, nil
}

func (u *Updater) Latest(ctx context.Context) (*Release, error) {
	latest, found, err := u.lib.DetectLatest(ctx, repository)

	if err != nil {
		return nil, fmt.Errorf("detect latest release: %w", err)
	}

	if !found {
		return nil, ErrNoRelease
	}

	return &Release{
		Version: latest.Version(),
		URL:     latest.URL,
		rel:     latest,
	}, nil
}

func (u *Updater) LatestVersion(ctx context.Context) (string, error) {
	r, err := u.Latest(ctx)
	if err != nil {
		return "", err
	}
	return r.Version, nil
}

func (u *Updater) Apply(ctx context.Context, r *Release) error {
	if r == nil || r.rel == nil {
		return fmt.Errorf("apply update: %w", selfupdate.ErrInvalidRelease)
	}

	exe, err := selfupdate.ExecutablePath()

	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}

	err = u.lib.UpdateTo(ctx, r.rel, exe)

	if err == nil {
		return nil
	}

	if rerr := selfupdateapply.RollbackError(err); rerr != nil {
		return fmt.Errorf("%w (%s may be missing): %w", ErrRollbackFailed, exe, rerr)
	}

	if errors.Is(err, fs.ErrPermission) {
		return &PermissionError{Dir: filepath.Dir(exe), Err: err}
	}

	return fmt.Errorf("apply update: %w", err)
}
