package update

import "runtime/debug"

const GoInstallCommand = "go install github.com/nox456/forgesync/cmd/forgesync@latest"

func IsGoInstall(info *debug.BuildInfo) bool {
	if info == nil {
		return false
	}

	if info.Main.Path != "github.com/nox456/forgesync" {
		return false
	}

	if info.Main.Version == "" || info.Main.Version == "(devel)" {
		return false
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return false
		}
	}

	return true
}
