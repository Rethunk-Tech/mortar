// Package steam locates a directly installed Steam, its libraries, installed apps and current account.
package steam

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"

	"github.com/andygrunwald/vdf"
)

// Status says whether a usable Steam was found.
type Status string

const (
	Found       Status = "found"
	FlatpakOnly Status = "flatpak-only"
	NotFound    Status = "not-found"
)

// Steam is a directly installed Steam rooted at Root.
type Steam struct {
	Root string
}

// Account is a Steam login from config/loginusers.vdf.
type Account struct {
	ID          string
	AccountName string
	PersonaName string
}

// Locate finds Steam for the given home directory. A Flatpak Steam is reported but never returned.
func Locate(home string) (Steam, Status) {
	for _, root := range candidates(home) {
		if isDir(filepath.Join(root, "steamapps")) {
			return Steam{Root: root}, Found
		}
	}
	if isDir(filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam")) {
		return Steam{}, FlatpakOnly
	}
	return Steam{}, NotFound
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
		if isDir(dir) {
			return dir, nil
		}
	}
	return "", nil
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
	cache := filepath.Join(s.Root, "appcache", "librarycache")
	for _, p := range []string{
		filepath.Join(cache, appID, "library_hero.jpg"),
		filepath.Join(cache, appID+"_library_hero.jpg"),
	} {
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
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

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// steamID64Base is the offset between a SteamID64 and the account number naming its userdata folder.
const steamID64Base = 76561197960265728

// LaunchOptions returns appID's launch options from the MostRecent account's localconfig.vdf, or "" when none are set.
func (s Steam) LaunchOptions(appID string) (string, error) {
	acct, err := s.CurrentAccount()
	if err != nil {
		return "", err
	}
	id64, err := strconv.ParseUint(acct.ID, 10, 64)
	if err != nil || id64 < steamID64Base {
		return "", fmt.Errorf("invalid Steam account id %q", acct.ID)
	}
	path := filepath.Join(s.Root, "userdata", strconv.FormatUint(id64-steamID64Base, 10), "config", "localconfig.vdf")
	m, err := parseVDF(path)
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
