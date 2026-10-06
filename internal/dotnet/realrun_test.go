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

type runPlugin struct{ name, version string }

// parseRun reads a LogOutput.log: the plugins BepInEx loaded, the ones it skipped or could not load, and the count it
// announced, -1 when it announced none.
func parseRun(raw string) (loaded, skipped []runPlugin, count int) {
	count = -1
	for line := range strings.Lines(raw) {
		line = strings.TrimRight(line, "\r\n")
		if m := loadingLine.FindStringSubmatch(line); m != nil {
			loaded = append(loaded, runPlugin{m[1], m[2]})
		} else if m := skippedLine.FindStringSubmatch(line); m != nil {
			skipped = append(skipped, runPlugin{m[1], m[2]})
		} else if m := countLine.FindStringSubmatch(line); m != nil {
			count, _ = strconv.Atoi(m[1])
		}
	}
	return loaded, skipped, count
}

// The matrix run that failed counted 69 plugins to load with 68 Loading lines: the 69th was an older copy BepInEx
// skipped, which still counts among those to load.
func TestParseRunCountsASkippedCopy(t *testing.T) {
	loaded, skipped, count := parseRun("[Info   :   BepInEx] 3 plugins to load\r\n" +
		"[Info   :   BepInEx] Loading [LateCompany 1.0.12]\r\n" +
		"[Warning:   BepInEx] Skipping [LateCompany 1.0.10] because a newer version exists (LateCompany 1.0.12)\r\n" +
		"[Info   :   BepInEx] Loading [Mirage 1.9.0]\r\n")
	if count != len(loaded)+len(skipped) || len(skipped) != 1 || skipped[0] != (runPlugin{"LateCompany", "1.0.10"}) {
		t.Fatalf("count %d, loaded %v, skipped %v", count, loaded, skipped)
	}
}

// TestRealBepInExRun compares the plugins Mortar reads from a profile's deployed BepInEx/plugins with what BepInEx
// loaded in that profile's last run: every plugin BepInEx loaded is one Mortar found, every plugin Mortar found was
// loaded or named as skipped, BepInEx's own count matches its Loading and skipped lines, and LoadOrder predicts the
// order of the Loading lines. It runs only when
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
	loaded, skipped, count := parseRun(string(raw))
	// BepInEx counts a plugin it then skips as a duplicate among those to load.
	if count != len(loaded)+len(skipped) {
		t.Errorf("BepInEx counted %d plugins to load, then logged %d Loading and %d skipped lines", count, len(loaded), len(skipped))
	}
	found := PluginsIn(filepath.Join(dir, "BepInEx", "plugins"))
	in := func(list []runPlugin, p Plugin) bool {
		for _, l := range list {
			if l.name == p.Name && sameVersion(l.version, p.Version) {
				return true
			}
		}
		return false
	}
	for _, l := range loaded {
		if !slices.ContainsFunc(found, func(p Plugin) bool { return in([]runPlugin{l}, p) }) {
			t.Errorf("BepInEx loaded [%s %s], which Mortar found in no DLL", l.name, l.version)
		}
	}
	for _, p := range found {
		if !in(loaded, p) && !in(skipped, p) {
			t.Errorf("Mortar found %s (%s %s), which BepInEx neither loaded nor skipped", p.GUID, p.Name, p.Version)
		}
	}
	order := LoadOrder(ScanEach(filepath.Join(dir, "BepInEx", "plugins")))
	var predicted []runPlugin
	for _, p := range order {
		if in(loaded, p.Plugin) {
			predicted = append(predicted, runPlugin{p.Name, p.Version})
		}
	}
	if len(predicted) != len(loaded) {
		t.Errorf("Mortar orders %d of the %d plugins BepInEx loaded", len(predicted), len(loaded))
	}
	for i := range min(len(predicted), len(loaded)) {
		if predicted[i].name != loaded[i].name {
			t.Errorf("load order differs at %d: Mortar predicts %s, BepInEx loaded %s", i+1, predicted[i].name, loaded[i].name)
			break
		}
	}
	t.Logf("%d plugins found, %d loaded, %d skipped", len(found), len(loaded), len(skipped))
}
