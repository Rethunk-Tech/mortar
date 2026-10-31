// Package appversion reads Mortar's version from build/config.yml, the one place it is set, and names the running build.
package appversion

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
)

// infoVersion matches `version: "x"` indented under `info:`; the file's top-level `version: '3'` is the
// Taskfile schema version, not the app's.
var infoVersion = regexp.MustCompile(`(?m)^info:\n(?:[ \t]+.*\n)*?[ \t]+version:[ \t]*"([^"]+)"`)

// AppID is Mortar's application id: the Flatpak id, the desktop-entry basename and the single-instance id.
const AppID = "tech.rethunk.Mortar"

// SteamBackupSuffix is the suffix of the one-time copy Mortar takes of a Steam file before its first edit; it stays
// what 0.3.8 wrote so an existing backup is found and never copied over.
const SteamBackupSuffix = ".mortar.bak"

// UserAgent names Mortar to a site.
func UserAgent(version string) string {
	if version == "" {
		return "Mortar"
	}
	return "Mortar/" + version
}

// FromConfig returns info.version from the contents of build/config.yml.
func FromConfig(config []byte) (string, error) {
	m := infoVersion.FindSubmatch(config)
	if m == nil {
		return "", errors.New("build/config.yml has no info.version")
	}
	return string(m[1]), nil
}

// Build names this exact build, so a cache another build wrote is known to be stale: the commit for a clean tree,
// with the executable's size and time added otherwise, since every local build shares the commit.
var Build = sync.OnceValue(func() string {
	rev, modified := vcs()
	if rev != "" && modified == "false" {
		return rev
	}
	if exe, err := os.Executable(); err == nil {
		if fi, err := os.Stat(exe); err == nil {
			return fmt.Sprintf("%s+%d.%d", rev, fi.Size(), fi.ModTime().UnixNano())
		}
	}
	return rev
})

// vcs is Go's VCS stamp: the commit, and "true" or "false" for whether the tree had uncommitted changes ("" when the
// build has no stamp).
func vcs() (rev, modified string) {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				modified = s.Value
			}
		}
	}
	return rev, modified
}

// stamped is set with -ldflags -X by the release build, which turns off Go's own VCS stamp (-buildvcs=false).
var stamped string

// Commit is the short commit this binary was built from, with "-modified" when the tree had uncommitted changes, so
// two builds of one version can be told apart; empty when the build carries no VCS stamp.
var Commit = sync.OnceValue(func() string {
	if stamped != "" {
		return stamped
	}
	rev, modified := vcs()
	if len(rev) > 8 {
		rev = rev[:8]
	}
	if rev != "" && modified == "true" {
		rev += "-modified"
	}
	return rev
})

// pastRelease is set with -ldflags -X to the number of commits since the release tag of the configured version; empty
// or 0 for the tagged release itself.
var pastRelease string

// PastRelease is whether this build comes from commits made after its configured version was tagged, so it already
// holds code the next release will announce.
func PastRelease() bool {
	n, err := strconv.Atoi(pastRelease)
	return err == nil && n > 0
}

// NextMinor is version with its minor number raised by one and its patch zeroed; a version that is not x.y.z comes
// back as it was.
func NextMinor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return version
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return version
	}
	if _, err := strconv.Atoi(parts[2]); err != nil {
		return version
	}
	parts[1], parts[2] = strconv.Itoa(minor+1), "0"
	return strings.Join(parts, ".")
}

// ForCatalog is the version the catalog's minVersion entries are compared with: a build past its release reads
// entries for the next release, a patch or a minor, since the code they need is in it. The version shown and compared for updates is
// untouched.
func ForCatalog(version string) string {
	if PastRelease() {
		return NextMinor(version)
	}
	return version
}
