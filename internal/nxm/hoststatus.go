package nxm

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// Host status values for NativeHostStatus rows.
const (
	HostOK         = "ok"
	HostMissing    = "missing"
	HostStale      = "stale"
	HostUnreadable = "unreadable"
)

// HostStatus is one installed browser's native-messaging host manifest state.
type HostStatus struct {
	Browser      string `json:"browser"`
	ManifestPath string `json:"manifestPath"`
	State        string `json:"state"`
}

func manifestState(exe, path string) string {
	b, err := fsx.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return HostMissing
	}
	if err != nil {
		return HostUnreadable
	}
	var m struct {
		Path string `json:"path"`
	}
	if json.Unmarshal(b, &m) != nil || m.Path == "" {
		return HostUnreadable
	}
	if !exeMatches(exe, m.Path) {
		return HostStale
	}
	return HostOK
}

func exeMatches(want, got string) bool {
	want = filepath.Clean(want)
	got = filepath.Clean(got)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(want, got)
	}
	return want == got
}

// RepairNativeHosts rewrites native-messaging host manifests for installed browsers.
func RepairNativeHosts(exe string) error {
	if skipXdgMime() {
		return nil
	}
	return repairNativeHosts(exe)
}

// StatusForExecutable reports native host manifests for browsers found on this system.
func StatusForExecutable(exe string) []HostStatus {
	if skipXdgMime() {
		return nil
	}
	return statusForExecutable(exe)
}
