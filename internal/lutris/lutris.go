// Package lutris finds a game's installs registered in Lutris on Linux.
package lutris

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/gog"
)

const (
	// StoreLutris is an install discovered from Lutris game configs.
	StoreLutris = "lutris"
)

var (
	reRunnerSteam = regexp.MustCompile(`(?m)^\s*runner:\s*steam\s*$`)
	reExe         = regexp.MustCompile(`(?m)^\s*exe:\s*(.+)\s*$`)
	reWorkDir     = regexp.MustCompile(`(?m)^\s*working_dir:\s*(.+)\s*$`)
)

// Game identifies one game to the Lutris locator.
type Game struct {
	// Slug is the game's Lutris slug (stardew-valley).
	Slug string
	// Keyword is a lowercase word an executable path of the game contains, for configs without the slug.
	Keyword string
	// Marker is a file every install of the game holds, at its root or one "game" folder down.
	Marker string
}

// matcher recognises one game's Lutris configs.
type matcher struct {
	g                      Game
	gameSlug, slug, slugIn *regexp.Regexp
}

func newMatcher(g Game) matcher {
	q := regexp.QuoteMeta(g.Slug)
	return matcher{
		g:        g,
		gameSlug: regexp.MustCompile(`(?m)^game_slug:\s*` + q + `\s*$`),
		slug:     regexp.MustCompile(`(?m)^slug:\s*` + q + `(?:-[0-9a-f]+)?\s*$`),
		slugIn:   regexp.MustCompile(`(?m)^\s+slug:\s*` + q + `\s*$`),
	}
}

// Install is one Lutris-sourced game folder.
type Install struct {
	Dir string
}

// Locate finds g's folders from Lutris YAML configs under home and in the user's own config folders.
func Locate(home string, g Game, extra ...string) []Install {
	if runtime.GOOS != "linux" {
		return nil
	}
	m := newMatcher(g)
	var out []Install
	seen := map[string]struct{}{}
	for _, dir := range ConfigDirs(home, extra...) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".yml") {
				continue
			}
			gameDir, err := m.installFromConfig(dir, ent.Name())
			if err != nil || gameDir == "" {
				continue
			}
			if _, ok := seen[gameDir]; ok {
				continue
			}
			seen[gameDir] = struct{}{}
			out = append(out, Install{Dir: gameDir})
		}
	}
	return out
}

// ConfigDirs are the Lutris game-config folders searched, the user's own first.
func ConfigDirs(home string, extra ...string) []string {
	return append(slices.Clone(extra),
		filepath.Join(home, ".local", "share", "lutris", "games"),
		filepath.Join(home, ".var", "app", "net.lutris.Lutris", "data", "lutris", "games"),
		filepath.Join(home, ".config", "lutris", "games"),
	)
}

func (m matcher) installFromConfig(gamesDir, name string) (string, error) {
	b, err := fsx.ReadFile(filepath.Join(gamesDir, name))
	if err != nil {
		return "", err
	}
	return m.installFromYAML(string(b))
}

func (m matcher) installFromYAML(text string) (string, error) {
	if reRunnerSteam.MatchString(text) {
		return "", nil
	}
	if !m.matches(text) {
		return "", nil
	}
	return m.resolveDir(text), nil
}

func (m matcher) matches(text string) bool {
	if m.gameSlug.MatchString(text) || m.slug.MatchString(text) || m.slugIn.MatchString(text) {
		return true
	}
	if m.g.Keyword == "" || !strings.Contains(strings.ToLower(text), m.g.Keyword) {
		return false
	}
	for _, e := range reExe.FindAllStringSubmatch(text, -1) {
		if strings.Contains(strings.ToLower(strings.Trim(e[1], `"'`)), m.g.Keyword) {
			return true
		}
	}
	return false
}

func (m matcher) resolveDir(text string) string {
	if w := reWorkDir.FindStringSubmatch(text); len(w) == 2 {
		if dir := gog.GameDir(strings.Trim(w[1], `"'`), m.g.Marker); dir != "" {
			return dir
		}
	}
	for _, e := range reExe.FindAllStringSubmatch(text, -1) {
		exe := strings.Trim(e[1], `"'`)
		if dir := gog.GameDir(rootOfMarker(exe, m.g.Marker), m.g.Marker); dir != "" {
			return dir
		}
		if dir := gog.GameDir(filepath.Dir(exe), m.g.Marker); dir != "" {
			return dir
		}
		if dir := gog.GameDir(exe, m.g.Marker); dir != "" {
			return dir
		}
	}
	return ""
}

// rootOfMarker is the install root implied by exe when exe is the marker file itself: the marker is a slash path below
// the root ("Game/Bin/TS4_x64.exe"), so exe ends with it, compared without case (a Windows executable path in a Wine prefix may differ in case). It returns "" when exe does not end with the marker.
func rootOfMarker(exe, marker string) string {
	slash := filepath.ToSlash(exe)
	cut := len(slash) - len(marker)
	if cut <= 0 || slash[cut-1] != '/' || !strings.EqualFold(slash[cut:], marker) {
		return ""
	}
	return filepath.FromSlash(slash[:cut-1])
}
