package version

import (
	"fmt"
	"runtime/debug"
)

var Tag = "unknown"

type BuildInfo struct {
	Tag      string
	Revision string
	GitDirty bool
}

// GetBuildInfo returns the current build information.
func GetBuildInfo() BuildInfo {
	buildInfo := BuildInfo{
		Tag:      Tag,
		Revision: "unknown",
		GitDirty: true,
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				buildInfo.Revision = setting.Value
			case "vcs.modified":
				buildInfo.GitDirty = setting.Value == "true"
			}
		}
	}

	return buildInfo
}

func BuildInfoString() string {
	info := GetBuildInfo()

	retval := fmt.Sprintf("Tag: %s\n", info.Tag)
	retval += fmt.Sprintf("Revision: %s\n", info.Revision)

	if info.GitDirty {
		retval += fmt.Sprintf("Git Dirty: %t\n", info.GitDirty)
	}

	return retval
}
