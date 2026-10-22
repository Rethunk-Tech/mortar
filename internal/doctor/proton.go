package doctor

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/runtime"
	"github.com/Rethunk-Tech/mortar/internal/steam"
)

// OverrideLoader is implemented by a loader that injects through a proxy library next to the game (winhttp for BepInEx,
// version for MelonLoader), which Wine only loads when the prefix overrides it to native. OverrideDLL names the library
// the game needs without its extension.
type OverrideLoader interface{ OverrideDLL(gameID string) string }

// loaders are the loaders that contribute checks, by game id; a game without one skips those checks.
var loaders = map[string]OverrideLoader{}

// RegisterLoader makes a game's loader contribute its checks to the Proton install checks.
func RegisterLoader(gameID string, l OverrideLoader) { loaders[gameID] = l }

// Replaced by tests.
var (
	flatpakOverrides = steam.ShowOverride
	launchOptions    = func(appID string) string {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		st, status := steam.Locate(home)
		if status != steam.Found {
			return ""
		}
		opts, _ := st.LaunchOptions(appID)
		return opts
	}
)

// overrideOption is the Steam launch option that loads the library natively.
func overrideOption(dll string) string { return `WINEDLLOVERRIDES="` + dll + `=n,b" %command%` }

// nativeOverride matches the library set to native in a registry file or a launch option.
func nativeOverride(dll string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)"?` + regexp.QuoteMeta(dll) + `"?\s*=\s*"?(n|native)\b`)
}

func protonChecks(g game.GameInfo) []Check {
	var checks []Check
	for _, in := range g.Installs {
		if in.Runtime != runtime.Proton {
			continue
		}
		inst := runtime.Install{Store: in.Store, Dir: in.Dir, AppID: g.AppID, Platform: in.Platform, GOOS: "linux"}
		compat, ok := runtime.CompatDataDir(inst)
		if !ok {
			continue
		}
		if l, has := loaders[g.ID]; has && l.OverrideDLL(g.ID) != "" {
			checks = append(checks, overrideCheck(g, compat, l.OverrideDLL(g.ID)))
		}
		if in.Store == "flatpak-steam" {
			checks = append(checks, flatpakCheck(g, compat))
		}
	}
	return checks
}

func overrideCheck(g game.GameInfo, compat, dll string) Check {
	c := Check{
		ID: dll + ":" + g.ID, Status: Pass, Detail: g.Name + ": " + dll + " is overridden to native in the Proton prefix",
		Code: "protonOverride", Params: params("game", g.Name, "name", dll),
	}
	reg, err := fsx.ReadFile(filepath.Join(compat, "pfx", "user.reg"))
	if re := nativeOverride(dll); re.Match(reg) || re.MatchString(launchOptions(g.AppID)) {
		return c
	}
	c.Status = Warn
	if err != nil {
		// A direct launch passes the override itself; a launch Steam relays needs a prefix to carry it.
		c.Detail = g.Name + ": Proton has not created its prefix yet, so a Steam launch cannot load the loader"
		c.Code = "protonNoPrefix"
		c.Fix = "start " + g.Name + " once from Steam and quit it, or set Default launch to Direct, or set the Steam launch options to " + overrideOption(dll)
		return c
	}
	c.Detail = g.Name + ": the Proton prefix does not load " + dll + " natively, so its loader will not start"
	c.Code = "protonNotNative"
	c.Fix = "set the Steam launch options to " + overrideOption(dll)
	return c
}

func flatpakCheck(g game.GameInfo, compat string) Check {
	c := Check{
		ID: "flatpakCompatdata:" + g.ID, Status: Pass, Detail: g.Name + ": Flatpak Steam can reach the Proton prefix",
		Code: "flatpakReach", Params: params("game", g.Name),
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.HasPrefix(compat, steam.FlatpakRoot(home)+string(filepath.Separator)) {
		return c
	}
	library := filepath.Dir(filepath.Dir(filepath.Dir(compat)))
	show, err := flatpakOverrides()
	if err != nil || steam.HasFilesystem(show, library) {
		return c
	}
	c.Status = Warn
	c.Detail = g.Name + ": Flatpak Steam cannot see the library that holds its Proton prefix"
	c.Code = "flatpakNoLibrary"
	c.Fix = steam.OverrideCommand(library)
	return c
}
