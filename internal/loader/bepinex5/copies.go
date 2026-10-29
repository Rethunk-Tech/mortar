package bepinex5

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/launchplan"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/wineoverride"
)

// A test run's later copies each get a BepInEx root of their own beside Mortar's data, outside the profile (whose
// folder many operations walk), so a copy's config/ is its own and config sync can be tried between copies.

func copyBase(dataDir, gameID, profileID string) string {
	return filepath.Join(dataDir, "test-copies", gameID, profileID)
}

// CopyRoot is where copy n (2 and up) of a test run of the profile roots its BepInEx.
func CopyRoot(dataDir, gameID, profileID string, n int) string {
	return filepath.Join(copyBase(dataDir, gameID, profileID), strconv.Itoa(n))
}

// CopyLogPath is the log copy n of a test run writes.
func CopyLogPath(dataDir, gameID, profileID string, n int) string {
	return filepath.Join(CopyRoot(dataDir, gameID, profileID, n), "BepInEx", "LogOutput.log")
}

// ownState is what a copy keeps for itself.
func ownState(name string) bool {
	return name == "config" || name == "cache" || strings.HasPrefix(name, "LogOutput.log")
}

// BuildCopyRoot lays out root as a BepInEx root for a copy of the profile: every folder of the profile's BepInEx
// but its own state is a link to the profile's, files are copied, and config/ is copied from the profile once and then
// kept, so the copy's settings survive between runs. Links and files are refreshed on every call.
func BuildCopyRoot(profileDir, root string) error {
	src, dst := filepath.Join(profileDir, "BepInEx"), filepath.Join(root, "BepInEx")
	if err := os.MkdirAll(filepath.Join(dst, "cache"), 0o750); err != nil {
		return err
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if ownState(e.Name()) {
			continue
		}
		from, to := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		if err := safefs.RemoveAll(to); err != nil {
			return err
		}
		if e.IsDir() {
			err = safefs.LinkDir(to, from)
		} else {
			err = datadir.CopyFile(from, to)
		}
		if err != nil {
			return err
		}
	}
	cfg := filepath.Join(dst, "config")
	if _, err := os.Stat(cfg); errors.Is(err, fs.ErrNotExist) {
		if err := datadir.CopyTree(filepath.Join(src, "config"), cfg); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// forCopy is the hook that points copy n at its own root: it builds the root and retargets Doorstop's entry assembly,
// in the arguments and, for a native Linux build, in the environment.
func forCopy(p loader.ProfileView, proton bool) func(int, *launchplan.Plan) error {
	return func(n int, c *launchplan.Plan) error {
		data, err := datadir.Dir()
		if err != nil {
			return err
		}
		root := CopyRoot(data, p.Game, filepath.Base(p.Dir), n)
		if err := BuildCopyRoot(p.Dir, root); err != nil {
			return err
		}
		target := hostPath(root, entry(p.Dir), proton)
		for i, a := range c.Args[:max(len(c.Args)-1, 0)] {
			if a == "--doorstop-target" || a == "--doorstop-target-assembly" {
				c.Args[i+1] = target
			}
		}
		for _, k := range []string{"DOORSTOP_TARGET_ASSEMBLY", "DOORSTOP_INVOKE_DLL_PATH"} {
			if _, ok := c.Env[k]; ok {
				c.Env[k] = target
			}
		}
		return nil
	}
}

// ownsCopyTarget is true when t is the entry assembly of one of the profile's test copies, in either path form.
func ownsCopyTarget(t string, prof loader.ProfileView) bool {
	data, err := datadir.Dir()
	if err != nil {
		return false
	}
	base := copyBase(data, prof.Game, filepath.Base(prof.Dir))
	return strings.HasPrefix(t, base+string(filepath.Separator)) || strings.HasPrefix(t, wineoverride.ZPath(base)+`\`)
}
