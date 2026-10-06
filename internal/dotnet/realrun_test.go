package dotnet

import (
	"flag"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// profileFlag names a profile folder the Lethal Company matrix regress launched (scripts/regress-bepinex.sh).
var profileFlag = flag.String("bepinex-profile", "", "a launched BepInEx profile folder for TestRealBepInExRun")

var (
	loadingLine = regexp.MustCompile(`^\[Info *: *BepInEx\] Loading \[(.+) (\S+)\]\s*$`)
	skippedLine = regexp.MustCompile(`\] (?:Skipping|Could not load) \[(.+) (\S+)\]`)
	countLine   = regexp.MustCompile(`\] (\d+) plugins? to load`)
)

// sameVersion compares versions the way System.Version reads them, so "1.02" and "1.2" are one version.
func sameVersion(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	if len(as) != len(bs) {
		return false
	}
	for i := range as {
		x, errA := strconv.Atoi(as[i])
		y, errB := strconv.Atoi(bs[i])
		if errA != nil || errB != nil || x != y {
			return false
		}
	}
	return true
}

// TestRealBepInExRun compares the plugins Mortar reads from a profile's deployed BepInEx/plugins with what BepInEx
// loaded in that profile's last run: every plugin BepInEx loaded is one Mortar found, every plugin Mortar found was
// loaded or named as skipped, and BepInEx's own count matches its Loading and skipped lines. It runs only when
// -bepinex-profile is given.
func TestRealBepInExRun(t *testing.T) {
	dir := *profileFlag
	if dir == "" {
		t.Skip("pass -bepinex-profile with a launched BepInEx profile folder")
	}
	raw, err := fsx.ReadFile(filepath.Join(dir, "BepInEx", "LogOutput.log"))
	if err != nil {
		t.Fatal(err)
	}
	type plugin struct{ name, version string }
	var loaded, skipped []plugin
	count := -1
	for line := range strings.Lines(string(raw)) {
		line = strings.TrimRight(line, "\r\n")
		if m := loadingLine.FindStringSubmatch(line); m != nil {
			loaded = append(loaded, plugin{m[1], m[2]})
		} else if m := skippedLine.FindStringSubmatch(line); m != nil {
			skipped = append(skipped, plugin{m[1], m[2]})
		} else if m := countLine.FindStringSubmatch(line); m != nil {
			count, _ = strconv.Atoi(m[1])
		}
	}
	// BepInEx counts a plugin it then skips as a duplicate among those to load.
	if count != len(loaded)+len(skipped) {
		t.Errorf("BepInEx counted %d plugins to load, then logged %d Loading and %d skipped lines", count, len(loaded), len(skipped))
	}
	found := PluginsIn(filepath.Join(dir, "BepInEx", "plugins"))
	in := func(list []plugin, p Plugin) bool {
		for _, l := range list {
			if l.name == p.Name && sameVersion(l.version, p.Version) {
				return true
			}
		}
		return false
	}
	for _, l := range loaded {
		if !slices.ContainsFunc(found, func(p Plugin) bool { return in([]plugin{l}, p) }) {
			t.Errorf("BepInEx loaded [%s %s], which Mortar found in no DLL", l.name, l.version)
		}
	}
	for _, p := range found {
		if !in(loaded, p) && !in(skipped, p) {
			t.Errorf("Mortar found %s (%s %s), which BepInEx neither loaded nor skipped", p.GUID, p.Name, p.Version)
		}
	}
	t.Logf("%d plugins found, %d loaded, %d skipped", len(found), len(loaded), len(skipped))
}
