package update

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver/v3"
)

type State struct {
	LastCheckedAt       time.Time `json:"last_checked_at"`
	LatestVersion       string    `json:"latest_version"`
	UpdateCheckInterval string    `json:"update_check_interval"`
}

func IsNewer(current string, latest string) bool {
	currentVersion, err := semver.NewVersion(current)
	if err != nil {
		return false
	}

	latestVersion, err := semver.NewVersion(latest)
	if err != nil {
		return false
	}

	return currentVersion.LessThan(latestVersion)
}

func ShouldCheck(state State, now time.Time, interval time.Duration) bool {
	if interval <= 0 {
		return false
	}

	if state.LastCheckedAt.After(now) {
		return true
	}

	if now.Sub(state.LastCheckedAt) < interval {
		return false
	}

	return true
}

func NewNotice(latest string) string {
	return fmt.Sprintf("Update available! -> %s. Run: forgesync update", latest)
}
