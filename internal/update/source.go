package update

import (
	"fmt"

	"github.com/creativeprojects/go-selfupdate"
)

var repository = selfupdate.ParseSlug("nox456/forgesync")

func newLibUpdater() (*selfupdate.Updater, error) {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("github release source: %w", err)
	}
	return selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
}
