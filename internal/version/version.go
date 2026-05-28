package version

import (
	"runtime"
	"runtime/debug"
	"strings"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
	Go      string `json:"go"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
}

func Current() Info {
	return Info{
		Version: effectiveVersion(),
		Commit:  commit,
		Date:    date,
		Go:      runtime.Version(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
}

func (Info) OutputSafe() {}

func effectiveVersion() string {
	if version != "" && version != "dev" {
		return strings.TrimPrefix(version, "v")
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		if version == "" {
			return "dev"
		}
		return version
	}
	return strings.TrimPrefix(info.Main.Version, "v")
}
