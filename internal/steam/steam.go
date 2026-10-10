// Package steam locates Steam (native or Flatpak), its libraries, installed apps and current account.
package steam

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/sandbox"

	"github.com/andygrunwald/vdf"
)

// Status says whether a usable Steam was found.
type Status string

const (
	Found       Status = "found"
	FlatpakOnly Status = "flatpak-only"
	NotFound    Status = "not-found"
)

// Kind is how Steam is installed.
type Kind string

const (
	KindNative  Kind = ""
	KindFlatpak Kind = "flatpak"
)

// FlatpakID is Flathub's Steam application id.
const FlatpakID = "com.valvesoftware.Steam"

// Steam is a Steam install rooted at Root.
type Steam struct {
	Root string
	Kind Kind
}

// Account is a Steam login from config/loginusers.vdf.
type Account struct {
	ID          string
	AccountName string
	PersonaName string
}

// FlatpakRoot is Flatpak Steam's Steam directory under home.
func FlatpakRoot(home string) string {
	return filepath.Join(home, ".var", "app", FlatpakID, ".local", "share", "Steam")
}

// Roots are the native Steam folders searched, the user's own folders first.
func Roots(home string, extra ...string) []string {
	return append(slices.Clone(extra), candidates(home)...)
}

// LocateAll returns every usable Steam, native first (the user's own folders before the usual ones), then Flatpak.
func LocateAll(home string, extra ...string) []Steam {
	var out []Steam
	for _, root := range Roots(home, extra...) {
		if fsx.IsDir(filepath.Join(root, "steamapps")) {
			out = append(out, Steam{Root: root, Kind: KindNative})
		}
	}
	if fp := FlatpakRoot(home); fsx.IsDir(filepath.Join(fp, "steamapps")) {
		out = append(out, Steam{Root: fp, Kind: KindFlatpak})
	}
	return out
}

// Locate finds Steam for the given home directory. Native Steam wins when both exist.
func Locate(home string, extra ...string) (Steam, Status) {
	if all := LocateAll(home, extra...); len(all) > 0 {
		return all[0], Found
	}
	if fsx.IsDir(FlatpakRoot(home)) {
		return Steam{}, FlatpakOnly
	}
	return Steam{}, NotFound
}

// OverrideCommand is the exact flatpak override that grants Steam read-only access to dataDir.
func OverrideCommand(dataDir string) string {
	return "flatpak override --user --filesystem=" + dataDir + ":ro " + FlatpakID
}

// HasFilesystem reports whether `flatpak override --user --show` grants dataDir.
func HasFilesystem(show, dataDir string) bool {
	dataDir = filepath.Clean(dataDir)
	for line := range strings.SplitSeq(show, "\n") {
		key, val, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "filesystems") {
			continue
		}
		for part := range strings.SplitSeq(val, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			path, _, _ := strings.Cut(part, ":")
			path = strings.TrimSpace(path)
			switch path {
			case "home", "host":
				return true
			}
			if fsx.SamePath(path, dataDir) {
				return true
			}
		}
	}
	return false
}

var runFlatpak = func(args ...string) ([]byte, error) {
	if sandbox.InFlatpak() {
		return sandbox.HostOutput("flatpak", args...)
	}
	return exec.Command("flatpak", args...).Output()
}

// ShowOverride is `flatpak override --user --show` for Steam. Tests replace runFlatpak.
func ShowOverride() (string, error) {
	out, err := runFlatpak("override", "--user", "--show", FlatpakID)
	return string(out), err
}

// GrantFilesystem runs the user override that grants Steam read-only access to dataDir.
func GrantFilesystem(dataDir string) error {
	_, err := runFlatpak("override", "--user", "--filesystem="+dataDir+":ro", FlatpakID)
	return err
}

// Libraries returns the library folder paths listed in libraryfolders.vdf.
func (s Steam) Libraries() ([]string, error) {
	m, err := parseVDF(filepath.Join(s.Root, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		return nil, err
	}
	folders, ok := m["libraryfolders"].(map[string]any)
	if !ok {
		return nil, errors.New("libraryfolders.vdf has no libraryfolders section")
	}
	var paths []string
	for _, entry := range folders {
		if e, ok := entry.(map[string]any); ok {
			if p, ok := e["path"].(string); ok {
				paths = append(paths, p)
			}
		}
	}
	return paths, nil
}

// InstallDir returns the install directory of appID, or "" when it is not installed.
func (s Steam) InstallDir(appID string) (string, error) {
	libs, err := s.Libraries()
	if err != nil {
		return "", err
	}
	for _, lib := range libs {
		m, err := parseVDF(filepath.Join(lib, "steamapps", "appmanifest_"+appID+".acf"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		state, _ := m["AppState"].(map[string]any)
		name, _ := state["installdir"].(string)
		if name == "" {
			continue
		}
		dir := filepath.Join(lib, "steamapps", "common", name)
		if fsx.IsDir(dir) {
			return dir, nil
		}
	}
	return "", nil
}

// BuildID is the Steam build of appID installed in dir (a folder of its library's steamapps/common), the version Steam
// itself tracks; "" when dir is not in a Steam library or the app has no manifest.
func BuildID(dir, appID string) string {
	if !filepath.IsLocal(appID) {
		return ""
	}
	apps := filepath.Dir(filepath.Dir(dir))
	m, err := parseVDF(filepath.Join(apps, "appmanifest_"+appID+".acf"))
	if err != nil {
		return ""
	}
	state, _ := m["AppState"].(map[string]any)
	id, _ := state["buildid"].(string)
	return id
}

// CurrentAccount returns the login marked MostRecent in config/loginusers.vdf.
func (s Steam) CurrentAccount() (Account, error) {
	m, err := parseVDF(filepath.Join(s.Root, "config", "loginusers.vdf"))
	if err != nil {
		return Account{}, err
	}
	users, _ := m["users"].(map[string]any)
	for id, u := range users {
		e, ok := u.(map[string]any)
		if !ok || e["MostRecent"] != "1" {
			continue
		}
		name, _ := e["AccountName"].(string)
		persona, _ := e["PersonaName"].(string)
		return Account{ID: id, AccountName: name, PersonaName: persona}, nil
	}
	return Account{}, errors.New("no Steam account is marked MostRecent")
}

// HeroArt returns the path of appID's library_hero.jpg, or "" when Steam has not cached it.
func (s Steam) HeroArt(appID string) string {
	if !filepath.IsLocal(appID) {
		return ""
	}
	cache := filepath.Join(s.Root, "appcache", "librarycache")
	// Newer Steam clients keep the art one folder deeper, under a content-hash name.
	nested, _ := filepath.Glob(filepath.Join(cache, appID, "*", "library_hero.jpg"))
	for _, p := range append([]string{
		filepath.Join(cache, appID, "library_hero.jpg"),
		filepath.Join(cache, appID+"_library_hero.jpg"),
	}, nested...) {
		if fsx.IsFile(p) {
			return p
		}
	}
	return ""
}

func parseVDF(path string) (map[string]any, error) {
	f, err := fsx.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	m, err := vdf.NewParser(f).Parse()
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return m, nil
}

// steamID64Base is the offset between a SteamID64 and the account number naming its userdata folder.
const steamID64Base = 76561197960265728

// userConfigDir is the MostRecent account's userdata config folder, where localconfig.vdf and shortcuts.vdf live.
func (s Steam) userConfigDir() (string, error) {
	acct, err := s.CurrentAccount()
	if err != nil {
		return "", err
	}
	id64, err := strconv.ParseUint(acct.ID, 10, 64)
	if err != nil || id64 < steamID64Base {
		return "", fmt.Errorf("invalid Steam account id %q", acct.ID)
	}
	return filepath.Join(s.Root, "userdata", strconv.FormatUint(id64-steamID64Base, 10), "config"), nil
}

// LaunchOptions returns appID's launch options from the MostRecent account's localconfig.vdf, or "" when none are set.
func (s Steam) LaunchOptions(appID string) (string, error) {
	dir, err := s.userConfigDir()
	if err != nil {
		return "", err
	}
	m, err := parseVDF(filepath.Join(dir, "localconfig.vdf"))
	if err != nil {
		return "", err
	}
	node := m
	for _, key := range []string{"UserLocalConfigStore", "Software", "Valve", "Steam", "apps", appID} {
		next, ok := child(node, key)
		if !ok {
			return "", nil
		}
		node = next
	}
	for k, v := range node {
		if s, ok := v.(string); ok && strings.EqualFold(k, "LaunchOptions") {
			return s, nil
		}
	}
	return "", nil
}

// child finds a nested section by key, ignoring case: Steam writes "Software" and "software" in different versions.
func child(m map[string]any, key string) (map[string]any, bool) {
	for k, v := range m {
		if sub, ok := v.(map[string]any); ok && strings.EqualFold(k, key) {
			return sub, true
		}
	}
	return nil, false
}
