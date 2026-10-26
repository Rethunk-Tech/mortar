package runtime

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

// MortarProton runs a Windows build Steam did not install (a folder, a Heroic or Lutris copy) on Linux through a
// Proton build Steam already has, in a prefix Mortar keeps for the game, as Heroic and Lutris start Proton outside
// Steam.
const MortarProton = "mortar-proton"

type mortarProton struct{}

func (mortarProton) ID() string { return MortarProton }

// Detect: a Windows build on Linux that neither Steam nor a Bottles bottle runs.
func (mortarProton) Detect(inst Install) bool {
	return inst.host() == "linux" && inst.Platform == "windows" && inst.Prefix == "" && inst.Game != "" &&
		inst.Store != storeSteam && inst.Store != storeFlatpakSteam
}

// ProtonPrefix is the folder Mortar keeps inst's Proton prefix in (Proton's STEAM_COMPAT_DATA_PATH; Wine's prefix is
// its pfx folder).
func ProtonPrefix(inst Install) (string, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "prefixes", inst.Game), nil
}

func (m mortarProton) Resolve(inst Install, t components.PathTemplate) (string, error) {
	if !m.Detect(inst) {
		return "", errors.New("install does not run under Mortar's Proton")
	}
	if t.Windows == "" {
		return "", errors.New("no path for a windows build")
	}
	root, err := ProtonPrefix(inst)
	if err != nil {
		return "", err
	}
	user := filepath.Join(root, "pfx", "drive_c", "users", "steamuser")
	return expand(t.Windows, map[string]string{
		"home":         user,
		"install":      inst.Dir,
		"documents":    filepath.Join(user, "Documents"),
		"appData":      filepath.Join(user, "AppData", "Roaming"),
		"localAppData": filepath.Join(user, "AppData", "Local"),
		"localLow":     filepath.Join(user, "AppData", "LocalLow"),
	})
}

// ProtonBuild is a Proton a native Steam has: its folder (holding the proton script) and that Steam's root.
type ProtonBuild struct {
	Dir, SteamRoot string
}

var protonVersion = regexp.MustCompile(`^Proton (\d+)\.(\d+)`)

// protonRank orders Proton folder names, best first: the newest numbered Valve release, then Experimental, then any
// other build (GE-Proton and friends) by name, newest first.
func protonRank(a, b string) int {
	ma, mb := protonVersion.FindStringSubmatch(a), protonVersion.FindStringSubmatch(b)
	switch {
	case ma != nil && mb != nil:
		for i := 1; i <= 2; i++ {
			x, _ := strconv.Atoi(ma[i])
			y, _ := strconv.Atoi(mb[i])
			if x != y {
				return y - x
			}
		}
		return 0
	case ma != nil:
		return -1
	case mb != nil:
		return 1
	}
	ea, eb := strings.Contains(a, "Experimental"), strings.Contains(b, "Experimental")
	if ea != eb {
		if ea {
			return -1
		}
		return 1
	}
	return strings.Compare(padNumbers(b), padNumbers(a))
}

var digits = regexp.MustCompile(`\d+`)

// padNumbers zero-pads each run of digits so names compare by their numbers: GE-Proton10 after GE-Proton9.
func padNumbers(s string) string {
	return digits.ReplaceAllStringFunc(s, func(d string) string { return strings.Repeat("0", max(0, 20-len(d))) + d })
}

// ProtonBuilds are the Proton builds a native Steam on this computer has, in its libraries' steamapps/common or its
// compatibilitytools.d, best first. Flatpak Steam's builds need Flatpak's runtime, so they are left out.
func ProtonBuilds(home string) []ProtonBuild {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	var found []ProtonBuild
	for _, st := range steam.LocateAll(home) {
		if st.Kind != steam.KindNative {
			continue
		}
		dirs := []string{filepath.Join(st.Root, "compatibilitytools.d")}
		libs, _ := st.Libraries()
		for _, lib := range append([]string{st.Root}, libs...) {
			dirs = append(dirs, filepath.Join(lib, "steamapps", "common"))
		}
		for _, d := range dirs {
			entries, _ := os.ReadDir(d)
			for _, e := range entries {
				p := filepath.Join(d, e.Name())
				if fsx.IsFile(filepath.Join(p, "proton")) && !slices.ContainsFunc(found, func(b ProtonBuild) bool { return b.Dir == p }) {
					found = append(found, ProtonBuild{Dir: p, SteamRoot: st.Root})
				}
			}
		}
	}
	slices.SortStableFunc(found, func(a, b ProtonBuild) int { return protonRank(filepath.Base(a.Dir), filepath.Base(b.Dir)) })
	return found
}

// FindProton is the build named want (its folder name) when Steam has it, else the best one.
func FindProton(home, want string) (ProtonBuild, bool) {
	found := ProtonBuilds(home)
	if i := slices.IndexFunc(found, func(b ProtonBuild) bool { return filepath.Base(b.Dir) == want }); i >= 0 {
		return found[i], true
	}
	if len(found) == 0 {
		return ProtonBuild{}, false
	}
	return found[0], true
}

// ProtonCommand is the argv and the environment that run exe with args through Proton in inst's prefix, which it
// creates when missing. A launcher's own Proton prefix is used as it is, and so is Steam's for a Steam copy (a direct
// start of it, such as several copies at once, which Steam's own start cannot do).
func ProtonCommand(inst Install, exe string, args ...string) (argv, env []string, err error) {
	root, own := CompatDataOf(inst)
	if !own && inst.Store == storeSteam {
		root, own = CompatDataDir(inst)
	}
	switch {
	case own:
	case (launcherPrefix{}).Detect(inst):
		return nil, nil, errors.New("the launcher's Wine prefix is not a Proton prefix: start the game from its launcher")
	case !(mortarProton{}).Detect(inst):
		return nil, nil, errors.New("install does not run under Mortar's Proton")
	}
	build, ok := FindProton(inst.Home, inst.Proton)
	if !ok {
		return nil, nil, ErrNoProton
	}
	if !own {
		if root, err = ProtonPrefix(inst); err != nil {
			return nil, nil, err
		}
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, nil, err
	}
	argv = append([]string{filepath.Join(build.Dir, "proton"), "waitforexitandrun", exe}, args...)
	env = []string{"STEAM_COMPAT_DATA_PATH=" + root, "STEAM_COMPAT_CLIENT_INSTALL_PATH=" + build.SteamRoot}
	return argv, env, nil
}

// ErrNoProton means no native Steam on this computer has a Proton build to run a Windows game with.
var ErrNoProton = errors.New("no Proton build was found in Steam")
