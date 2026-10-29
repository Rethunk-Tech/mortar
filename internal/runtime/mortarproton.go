package runtime

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/go-gamelocator/steam"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
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
	return expand(t.Windows, userTokens(user, inst.Dir))
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
	tool := ""
	if inst.Proton == "" && inst.Store == storeSteam {
		tool = Version(inst)
	}
	build, ok := steam.PickProton(steam.ProtonBuilds(inst.Home, inst.SteamRoots...), inst.Proton, tool)
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
