// Package steam locates Steam (native or Flatpak), its libraries, installed apps and current account.
package steam

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

// homeOr is home, or the user's own home when none was given: an empty home would make every candidate a path
// relative to the working directory.
func homeOr(home string) string {
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	return home
}

// FlatpakRoot is Flatpak Steam's Steam directory under home.
func FlatpakRoot(home string) string {
	home = homeOr(home)
	return filepath.Join(home, ".var", "app", FlatpakID, ".local", "share", "Steam")
}

// Roots are the native Steam folders searched, the user's own folders first.
func Roots(home string, extra ...string) []string {
	return append(slices.Clone(extra), candidates(homeOr(home))...)
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

// hostOutput runs a command on the host and returns its stdout; tests replace it.
var hostOutput = sandbox.HostOutput

var runFlatpak = func(args ...string) ([]byte, error) {
	if sandbox.InFlatpak() {
		return hostOutput("flatpak", args...)
	}
	return exec.Command("flatpak", args...).Output() //nolint:gosec // G204: binary is flatpak; args are steam --install or --run
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
	// Older clients mark the signed-in account MostRecent; current ones (2026) keep only each account's last sign-in
	// Timestamp, so the newest one is the account in use.
	var best Account
	var bestAt int64 = -1
	for id, u := range users {
		e, ok := u.(map[string]any)
		if !ok {
			continue
		}
		name, _ := vdfKey(e, "AccountName").(string)
		persona, _ := vdfKey(e, "PersonaName").(string)
		acct := Account{ID: id, AccountName: name, PersonaName: persona}
		if vdfKey(e, "MostRecent") == "1" {
			return acct, nil
		}
		ts, _ := vdfKey(e, "Timestamp").(string)
		if at, err := strconv.ParseInt(ts, 10, 64); err == nil && (at > bestAt || at == bestAt && id < best.ID) {
			best, bestAt = acct, at
		}
	}
	if bestAt < 0 {
		return Account{}, errors.New("no Steam account is signed in")
	}
	return best, nil
}

// vdfKey looks a key up the way Steam does, ignoring case: current clients write loginusers.vdf's "timestamp" in
// lower case where older ones wrote "Timestamp".
func vdfKey(m map[string]any, key string) any {
	if v, ok := m[key]; ok {
		return v
	}
	for k, v := range m {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
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

// The VDF parser has no size or depth limit, and a stack overflow cannot be recovered. Real files are under 100 KB
// and nest under 8 levels.
const (
	maxTextVDFBytes = 8 << 20
	maxTextVDFDepth = 32
)

func parseVDF(path string) (map[string]any, error) {
	f, err := fsx.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	body, err := io.ReadAll(io.LimitReader(f, maxTextVDFBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxTextVDFBytes {
		return nil, fmt.Errorf("%s is larger than %d bytes", filepath.Base(path), maxTextVDFBytes)
	}
	depth, err := textVDFDepth(body)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	if depth > maxTextVDFDepth {
		return nil, fmt.Errorf("%s nests deeper than %d levels", filepath.Base(path), maxTextVDFDepth)
	}
	m, err := vdf.NewParser(bytes.NewReader(body)).Parse()
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
	}
	return m, nil
}

// textVDFDepth is the deepest run of unclosed braces over the tokens the parser reads, so a comment or a quoted
// brace never counts; a file scanVDF cannot tokenize is refused.
func textVDFDepth(b []byte) (int, error) {
	toks, err := scanVDF(b)
	if err != nil {
		return 0, err
	}
	depth, deepest := 0, 0
	for _, t := range toks {
		switch {
		case t.kind == '{':
			if depth++; depth > deepest {
				deepest = depth
			}
		case t.kind == '}' && depth > 0:
			depth--
		}
	}
	return deepest, nil
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

// ScreenshotsDir is the folder Steam keeps appID's screenshots in for the MostRecent account.
func (s Steam) ScreenshotsDir(appID string) (string, error) {
	cfg, err := s.userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfg), "760", "remote", appID, "screenshots"), nil
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
