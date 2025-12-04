package version

import (
	"fmt"
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

var Tag = "unknown"

type BuildInfo struct {
	Tag      string
	Revision string
	GitDirty *bool
}

// GetBuildInfo returns the current build information.
func GetBuildInfo() BuildInfo {
	buildInfo := BuildInfo{
		Tag:      Tag,
		Revision: "unknown",
		GitDirty: nil,
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				buildInfo.Revision = setting.Value
			case "vcs.modified":
				val := setting.Value == "true"
				buildInfo.GitDirty = &val
			}
		}
	}

	return buildInfo
}

func BuildInfoString() string {
	info := GetBuildInfo()

	retval := fmt.Sprintf("Tag: %s\n", info.Tag)
	retval += fmt.Sprintf("Revision: %s\n", info.Revision)

	if info.GitDirty == nil {
		retval += "Git Dirty: Unknown\n"
	} else if *info.GitDirty {
		retval += "Git Dirty: Yes\n"
	}

	return retval
}

func OutputToDebug() {
	info := GetBuildInfo()

	dirty := "no"

	if info.GitDirty == nil {
		dirty = "unknown"
	} else if *info.GitDirty {
		dirty = "yes"
	}

	log.Debug().
		Str("tag", info.Tag).
		Str("revision", info.Revision).
		Str("git_dirty", dirty).
		Msg("Build information")
}
