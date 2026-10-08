package update

import (
	"context"
	"errors"
	"fmt"

	"github.com/creativeprojects/go-selfupdate"
)

type Updater struct {
	current string
	lib     *selfupdate.Updater
}

var ErrNoRelease = errors.New("no release found for this platform")

func NewUpdater(current string) (*Updater, error) {
	lib, err := newLibUpdater()
	if err != nil {
		return nil, err
	}
	return &Updater{current: current, lib: lib}, nil
}

func (u *Updater) LatestVersion(ctx context.Context) (string, error) {
	latest, found, err := u.lib.DetectLatest(ctx, selfupdate.NewRepositorySlug("nox456", "forgesync"))

	if err != nil {
		return "", fmt.Errorf("detect latest release: %w", err)
	}

	if !found {
		return "", ErrNoRelease
	}

	return latest.Version(), nil
}
