// Package lutris finds Stardew Valley installs registered in Lutris on Linux.
package lutris

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	// StoreLutris is a Stardew install discovered from Lutris game configs.
	StoreLutris = "lutris"
	marker      = "Stardew Valley.dll"
)

var (
	reRunnerSteam = regexp.MustCompile(`(?m)^\s*runner:\s*steam\s*$`)
	reGameSlug    = regexp.MustCompile(`(?m)^game_slug:\s*stardew-valley\s*$`)
	reSlug        = regexp.MustCompile(`(?m)^slug:\s*stardew-valley(?:-[0-9a-f]+)?\s*$`)
	reGameSlugIn  = regexp.MustCompile(`(?m)^\s+slug:\s*stardew-valley\s*$`)
	reExe         = regexp.MustCompile(`(?m)^\s*exe:\s*(.+)\s*$`)
	reWorkDir     = regexp.MustCompile(`(?m)^\s*working_dir:\s*(.+)\s*$`)
)

// Install is one Lutris-sourced game folder.
type Install struct {
	Dir string
}

// Locate finds Stardew folders from Lutris YAML configs under home.
func Locate(home string) []Install {
	if runtime.GOOS != "linux" {
		return nil
	}
	var out []Install
	seen := map[string]struct{}{}
	for _, dir := range gameConfigDirs(home) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".yml") {
				continue
			}
			path := filepath.Join(dir, ent.Name())
			gameDir, err := installFromFile(path)
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

func gameConfigDirs(home string) []string {
	return []string{
		filepath.Join(home, ".local", "share", "lutris", "games"),
		filepath.Join(home, ".var", "app", "net.lutris.Lutris", "data", "lutris", "games"),
	}
}

func installFromFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	text := string(b)
	if reRunnerSteam.MatchString(text) {
		return "", nil
	}
	if !matchesStardew(text) {
		return "", nil
	}
	return resolveDir(text), nil
}

func matchesStardew(text string) bool {
	if reGameSlug.MatchString(text) || reSlug.MatchString(text) || reGameSlugIn.MatchString(text) {
		return true
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "stardew") {
		for _, m := range reExe.FindAllStringSubmatch(text, -1) {
			if strings.Contains(strings.ToLower(strings.Trim(m[1], `"'`)), "stardew") {
				return true
			}
		}
	}
	return false
}

func resolveDir(text string) string {
	if m := reWorkDir.FindStringSubmatch(text); len(m) == 2 {
		if dir := gogGameDir(strings.Trim(m[1], `"'`)); dir != "" {
			return dir
		}
	}
	for _, m := range reExe.FindAllStringSubmatch(text, -1) {
		exe := strings.Trim(m[1], `"'`)
		if dir := gogGameDir(filepath.Dir(exe)); dir != "" {
			return dir
		}
		if dir := gogGameDir(exe); dir != "" {
			return dir
		}
	}
	return ""
}

func gogGameDir(dir string) string {
	if dir == "" {
		return ""
	}
	dir = filepath.Clean(dir)
	if hasMarker(dir) {
		return dir
	}
	nested := filepath.Join(dir, "game")
	if hasMarker(nested) {
		return nested
	}
	return ""
}

func hasMarker(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, marker))
	return err == nil && st.Mode().IsRegular()
}
