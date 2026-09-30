package support

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launch"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const diagnosticsLogLines = 2000

// SaveZip, when set, writes the diagnostics zip after the user picks a path. Tests use it so the
// native dialog and the real keyring are never touched. Nil uses App's save dialog.
func (s *Service) saveZip(filename string, data []byte) (string, error) {
	if s.SaveZip != nil {
		return s.SaveZip("Save diagnostics", filename, data)
	}
	if s.App == nil {
		return "", errors.New("diagnostics save is not available")
	}
	d := s.App.Dialog.SaveFile()
	d.SetOptions(&application.SaveFileDialogOptions{Title: "Save diagnostics", Filename: filename})
	d.AddFilter("Zip archives", "*.zip")
	d.AddFilter("All files", "*")
	if w := s.App.Window.Current(); w != nil {
		d.AttachToWindow(w)
	}
	path, err := d.PromptForSingleSelection()
	if err != nil || path == "" {
		return path, err
	}
	return path, fsx.WriteFile(path, data, 0o600)
}

func (s *Service) dataDir() (string, error) {
	if s.Dir != nil {
		return s.Dir()
	}
	return datadir.Dir()
}

// SaveDiagnostics writes a zip of redacted app state through the native save dialog.
// gameID and profileID select whose SMAPI log to include; either may be empty.
func (s *Service) SaveDiagnostics(gameID, profileID string) (string, error) {
	data, err := s.bundle(gameID, profileID)
	if err != nil {
		return "", err
	}
	name := "mortar-diagnostics-" + time.Now().Format("2006-01-02") + ".zip"
	return s.saveZip(name, data)
}

type bundleProfile struct {
	Game    string        `json:"game"`
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Entries []bundleEntry `json:"entries"`
}

type bundleEntry struct {
	Source profile.Source `json:"source"`
	Mods   []bundleMod    `json:"mods"`
}

type bundleMod struct {
	Name     string `json:"name"`
	UniqueID string `json:"uniqueId"`
	Version  string `json:"version"`
}

func (s *Service) bundle(gameID, profileID string) ([]byte, error) {
	dir, err := s.dataDir()
	if err != nil {
		return nil, err
	}
	included := []string{
		"build.json: Mortar version, OS, architecture, Go, and Wails/WebKit when known",
		"settings.json: settings with Nexus account fields cleared and API keys removed",
		"profiles.json: profiles with mod names, versions and sources",
		"queue.json: download queue without nxm keys",
	}
	removed := []string{
		"Nexus API key (not read from the keyring; stripped if present in settings.json)",
		"nexusName and nexusUserId",
		"nxm download keys and expiry on queue items",
		"profile notes",
	}

	files := map[string][]byte{
		"build.json":    jsonIndent(buildInfo(s.version)),
		"settings.json": redactSettings(readFile(filepath.Join(dir, "settings.json"))),
		"profiles.json": jsonIndent(collectProfiles(filepath.Join(dir, "profiles"))),
		"queue.json":    redactQueue(readFile(filepath.Join(dir, "queue.json"))),
	}

	logName, logBody := mortarLog(dir, s.recentLines(gameID, profileID))
	if logName != "" {
		files[logName] = []byte(logBody)
		included = append(included, logName+": last "+fmt.Sprint(diagnosticsLogLines)+" lines of Mortar's log or in-memory console")
	}
	if smapi := s.smapiTail(gameID, profileID); smapi != "" {
		files["smapi-latest.txt"] = []byte(smapi)
		included = append(included, "smapi-latest.txt: last "+fmt.Sprint(diagnosticsLogLines)+" lines of the open profile's SMAPI log")
	}

	files["README.txt"] = []byte(diagnosticsReadme(included, removed))
	return zipFiles(files)
}

func (s *Service) recentLines(gameID, profileID string) string {
	if s.RecentLog == nil {
		return ""
	}
	return s.RecentLog(gameID, profileID)
}

func diagnosticsReadme(included, removed []string) string {
	var b strings.Builder
	b.WriteString("Mortar diagnostics bundle\n\nIncluded:\n")
	for _, line := range included {
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("\nRemoved:\n")
	for _, line := range removed {
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func buildInfo(version string) map[string]string {
	info := map[string]string{
		"mortar": version,
		"go":     runtime.Version(),
		"os":     runtime.GOOS,
		"arch":   runtime.GOARCH,
	}
	if v := wailsVersion(); v != "" {
		info["wails"] = v
	}
	if v := webkitVersion(); v != "" {
		info["webkit"] = v
	}
	return info
}

func wailsVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, m := range info.Deps {
		if strings.HasSuffix(m.Path, "wails/v3") || strings.Contains(m.Path, "/wails/v3") {
			return m.Version
		}
	}
	return ""
}

func webkitVersion() string {
	// The embedded WebKit/WebView2 version is not exposed by Wails on every platform.
	return ""
}

func mortarLog(dir, recent string) (string, string) {
	for _, rel := range []string{"mortar.log", filepath.Join("logs", "mortar.log")} {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			continue
		}
		return "mortar-log.txt", lastLines(string(b), diagnosticsLogLines)
	}
	if strings.TrimSpace(recent) == "" {
		return "", ""
	}
	return "console.txt", lastLines(recent, diagnosticsLogLines)
}

func (s *Service) smapiTail(gameID, profileID string) string {
	if gameID == "" || profileID == "" {
		return ""
	}
	g := game.Find(gameID)
	if g == nil {
		return ""
	}
	path, err := g.LogFile()
	if err != nil {
		return ""
	}
	data, err := fsx.ReadFile(path)
	if err != nil {
		return ""
	}
	modsDir, err := s.modsDir(gameID, profileID)
	if err != nil {
		return ""
	}
	text := strings.ToValidUTF8(string(data), "")
	if !launch.LogOwnedBy(text, s.home, modsDir) {
		return ""
	}
	return lastLines(text, diagnosticsLogLines)
}

func lastLines(text string, n int) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

func readFile(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

func jsonIndent(v any) []byte {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return []byte("{}")
	}
	b = append(b, '\n')
	return b
}

func redactSettings(raw []byte) []byte {
	m := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	if m == nil {
		m = map[string]any{}
	}
	m["nexusName"] = ""
	m["nexusUserId"] = 0
	delete(m, "nexusKey")
	delete(m, "apiKey")
	delete(m, "nexusApiKey")
	return jsonIndent(m)
}

func redactQueue(raw []byte) []byte {
	m := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	if m == nil {
		m = map[string]any{"items": []any{}}
	}
	items, _ := m["items"].([]any)
	for _, it := range items {
		obj, ok := it.(map[string]any)
		if !ok {
			continue
		}
		delete(obj, "key")
		delete(obj, "expires")
		delete(obj, "nxmKey")
	}
	if _, ok := m["items"]; !ok {
		m["items"] = []any{}
	}
	return jsonIndent(m)
}

func collectProfiles(root string) []bundleProfile {
	var out []bundleProfile
	games, err := os.ReadDir(root)
	if err != nil {
		return []bundleProfile{}
	}
	for _, g := range games {
		if !g.IsDir() || g.Type()&fs.ModeSymlink != 0 {
			continue
		}
		gameID := g.Name()
		entries, err := os.ReadDir(filepath.Join(root, gameID))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || e.Type()&fs.ModeSymlink != 0 {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(root, gameID, e.Name(), "profile.json"))
			if err != nil {
				continue
			}
			var p profile.Profile
			if json.Unmarshal(raw, &p) != nil {
				continue
			}
			bp := bundleProfile{Game: gameID, ID: p.ID, Name: p.Name}
			for _, ent := range p.Entries {
				be := bundleEntry{Source: ent.Source}
				for _, mod := range ent.Mods {
					be.Mods = append(be.Mods, bundleMod{Name: mod.Name, UniqueID: mod.UniqueID, Version: mod.Version})
				}
				bp.Entries = append(bp.Entries, be)
			}
			out = append(out, bp)
		}
	}
	slices.SortFunc(out, func(a, b bundleProfile) int {
		if a.Game != b.Game {
			return strings.Compare(a.Game, b.Game)
		}
		return strings.Compare(a.ID, b.ID)
	})
	return out
}

func zipFiles(files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := w.Write(files[name]); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
