// Package runtime resolves a game's catalog path templates for the way an install runs: natively on the host or
// as a Windows build inside a Proton prefix.
package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
)

// Runtime ids.
const (
	Native = "native"
	Proton = "proton"
)

// Steam store ids, the stores whose Windows builds run under Proton.
const (
	storeSteam        = "steam"
	storeFlatpakSteam = "flatpak-steam"
)

// Install is the part of a game install a runtime needs.
type Install struct {
	Store, Dir, AppID string
	// Platform is the OS the install's build is for: "windows", "linux" or "darwin".
	Platform string
	// Home is the user's home folder.
	Home string
	// GOOS is the host OS; empty means this process's.
	GOOS string
}

func (i Install) host() string {
	if i.GOOS != "" {
		return i.GOOS
	}
	return goruntime.GOOS
}

// PlatformOf is the platform of the build whose marker file is marker: an .exe marker is a Windows build, anything
// else is built for the host.
func PlatformOf(marker, goos string) string {
	if strings.HasSuffix(strings.ToLower(marker), ".exe") {
		return "windows"
	}
	return goos
}

// Runtime runs an install and knows where its paths live.
type Runtime interface {
	ID() string
	// Detect reports whether this runtime runs inst.
	Detect(inst Install) bool
	// Resolve expands the template for inst's platform into an absolute path.
	Resolve(inst Install, t components.PathTemplate) (string, error)
}

// drivers are tried in order; native runs whatever no other runtime claims.
var drivers = []Runtime{proton{}, native{}}

// IDOf is the id of the runtime that runs inst.
func IDOf(inst Install) string {
	for _, rt := range drivers {
		if rt.Detect(inst) {
			return rt.ID()
		}
	}
	return Native
}

// Resolve expands t for inst through the runtime that runs it.
func Resolve(inst Install, t components.PathTemplate) (string, error) {
	for _, rt := range drivers {
		if rt.Detect(inst) {
			return rt.Resolve(inst, t)
		}
	}
	return native{}.Resolve(inst, t)
}

func template(inst Install, t components.PathTemplate) (string, error) {
	var p string
	switch inst.Platform {
	case "windows":
		p = t.Windows
	case "darwin":
		p = t.Darwin
	case "linux":
		p = t.Linux
	}
	if p == "" {
		return "", fmt.Errorf("no path for a %s build", inst.Platform)
	}
	return p, nil
}

func expand(p string, tokens map[string]string) (string, error) {
	for k, v := range tokens {
		p = strings.ReplaceAll(p, "{"+k+"}", v)
	}
	if strings.ContainsAny(p, "{}") {
		return "", fmt.Errorf("path %q uses a folder this install does not have", p)
	}
	return filepath.Clean(filepath.FromSlash(p)), nil
}

type native struct{}

func (native) ID() string { return Native }

func (native) Detect(inst Install) bool { return inst.Platform == inst.host() }

func (n native) Resolve(inst Install, t components.PathTemplate) (string, error) {
	p, err := template(inst, t)
	if err != nil {
		return "", err
	}
	if !n.Detect(inst) {
		return "", fmt.Errorf("a %s build cannot run natively on %s", inst.Platform, inst.host())
	}
	tokens, err := nativeTokens(inst)
	if err != nil {
		return "", err
	}
	return expand(p, tokens)
}

func nativeTokens(inst Install) (map[string]string, error) {
	home := inst.Home
	tokens := map[string]string{"home": home, "install": inst.Dir, "documents": filepath.Join(home, "Documents")}
	switch inst.host() {
	case "windows":
		tokens["appData"] = envOr("APPDATA", filepath.Join(home, "AppData", "Roaming"))
		tokens["localAppData"] = envOr("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))
		tokens["localLow"] = filepath.Join(home, "AppData", "LocalLow")
	case "darwin":
		tokens["xdgConfig"] = filepath.Join(home, "Library", "Application Support")
		tokens["xdgData"] = tokens["xdgConfig"]
	default:
		switch {
		case inst.Store == storeFlatpakSteam:
			sandbox := filepath.Join(home, ".var", "app", "com.valvesoftware.Steam")
			tokens["xdgConfig"] = filepath.Join(sandbox, ".config")
			tokens["xdgData"] = filepath.Join(sandbox, ".local", "share")
		case os.Getenv("FLATPAK_ID") != "":
			tokens["xdgConfig"] = filepath.Join(home, ".config")
			tokens["xdgData"] = filepath.Join(home, ".local", "share")
		default:
			config := os.Getenv("XDG_CONFIG_HOME")
			if config == "" {
				config = filepath.Join(home, ".config")
			} else if !filepath.IsAbs(config) {
				return nil, errors.New("XDG_CONFIG_HOME is not an absolute path")
			}
			tokens["xdgConfig"] = config
			tokens["xdgData"] = envOr("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
		}
	}
	return tokens, nil
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

type proton struct{}

func (proton) ID() string { return Proton }

// Detect: a Windows build of a Steam game on Linux runs under Proton.
func (proton) Detect(inst Install) bool {
	return inst.host() == "linux" && inst.Platform == "windows" && inst.AppID != "" &&
		(inst.Store == storeSteam || inst.Store == storeFlatpakSteam)
}

// ponytail: the Steam library is found by walking up to its steamapps folder; a Proton prefix that Steam keeps elsewhere needs a setting.
func (p proton) Resolve(inst Install, t components.PathTemplate) (string, error) {
	if !p.Detect(inst) {
		return "", errors.New("install does not run under Proton")
	}
	tmpl := t.Windows
	if tmpl == "" {
		return "", errors.New("no path for a windows build")
	}
	steamapps := inst.Dir
	for filepath.Base(steamapps) != "steamapps" {
		parent := filepath.Dir(steamapps)
		if parent == steamapps {
			return "", fmt.Errorf("%q is not inside a Steam library", inst.Dir)
		}
		steamapps = parent
	}
	user := filepath.Join(steamapps, "compatdata", inst.AppID, "pfx", "drive_c", "users", "steamuser")
	return expand(tmpl, map[string]string{
		"home":         user,
		"install":      inst.Dir,
		"documents":    filepath.Join(user, "Documents"),
		"appData":      filepath.Join(user, "AppData", "Roaming"),
		"localAppData": filepath.Join(user, "AppData", "Local"),
		"localLow":     filepath.Join(user, "AppData", "LocalLow"),
	})
}
