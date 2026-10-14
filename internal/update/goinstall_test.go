package update

import (
	"runtime/debug"
	"testing"
)

func TestIsGoInstall(t *testing.T) {
	const module = "github.com/nox456/forgesync"

	build := func(path, version string, settings ...debug.BuildSetting) *debug.BuildInfo {
		return &debug.BuildInfo{
			Main:     debug.Module{Path: path, Version: version},
			Settings: settings,
		}
	}

	cases := []struct {
		name string
		info *debug.BuildInfo
		want bool
	}{
		// happy path
		{name: "go install from the module cache is a go install build", info: build(module, "v0.10.0"), want: true},
		{name: "pseudo-version from the module cache is a go install build", info: build(module, "v0.10.1-0.20261001120000-abcdef123456"), want: true},
		{name: "local build stamped with vcs.revision is not a go install build", info: build(module, "v0.10.0", debug.BuildSetting{Key: "vcs.revision", Value: "abcdef1"}), want: false},
		// edge cases
		{name: "devel version is not a go install build", info: build(module, "(devel)"), want: false},
		{name: "empty version is not a go install build", info: build(module, ""), want: false},
		{name: "vcs.revision among other settings still marks a local build", info: build(module, "v0.10.0", debug.BuildSetting{Key: "-ldflags", Value: "-s -w"}, debug.BuildSetting{Key: "vcs", Value: "git"}, debug.BuildSetting{Key: "vcs.revision", Value: "abcdef1"}), want: false},
		{name: "non-vcs settings alone keep a go install build", info: build(module, "v0.10.0", debug.BuildSetting{Key: "-ldflags", Value: "-s -w"}, debug.BuildSetting{Key: "CGO_ENABLED", Value: "0"}), want: true},
		// error path: build info that isn't forgesync's, or isn't there
		{name: "nil build info is not a go install build", info: nil, want: false},
		{name: "another module path is not a go install build", info: build("github.com/someone/else", "v0.10.0"), want: false},
		{name: "a fork's module path is not a go install build", info: build("github.com/nox456/forgesync-fork", "v0.10.0"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsGoInstall(tc.info)
			if got != tc.want {
				t.Errorf("IsGoInstall() = %v, want %v", got, tc.want)
			}
		})
	}
}
