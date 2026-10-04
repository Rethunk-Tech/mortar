package support

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/doctor"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/picker"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
)

const (
	diagnosticsLogLines = 2000
	diagnosticsRuns     = 3
)

// SaveZip, when set, writes the diagnostics zip after the user picks a path. Tests use it so the
// native dialog and the real keyring are never touched. Nil uses App's save dialog.
func (s *Service) saveZip(filename string, data []byte) (string, error) {
	if s.SaveZip != nil {
		return s.SaveZip("Save diagnostics", filename, data)
	}
	if s.App == nil {
		return "", errors.New("diagnostics save is not available")
	}
	return picker.SaveFile(s.App, "Save diagnostics", filename, "Zip archives", "*.zip", data)
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

// ShowDiagnostics opens the folder holding a diagnostics zip Mortar saved.
func (s *Service) ShowDiagnostics(path string) error {
	if !strings.EqualFold(filepath.Ext(path), ".zip") {
		return errors.New("not a diagnostics zip")
	}
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return errors.New("the diagnostics zip is gone")
	}
	return datadir.Open(filepath.Dir(path))
}

type bundleProfile struct {
	Game       string        `json:"game"`
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	EntryCount int           `json:"entryCount"`
	Entries    []bundleEntry `json:"entries"`
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
		"settings.json: settings with Nexus account fields cleared, API keys removed, and home-directory paths written as ~",
		"profiles.json: profile ids, names, entry counts, mod names, versions and sources",
		"doctor.txt: the checks `mortar doctor` runs, with home-directory paths written as ~",
		"runs.json: the last " + fmt.Sprint(diagnosticsRuns) + " launch runs (outcome, versions, error counts; no log text)",
		"queue.json: download queue without nxm keys",
	}
	removed := []string{
		"Nexus API key (not read from the keyring; stripped if present in settings.json)",
		"nexusName, nexusUserId, and nexusPremium",
		"overlayToken",
		"absolute paths under the home directory (written as ~)",
		"nxm download keys and expiry on queue items",
		"profile notes",
		"any settings field whose name says token, secret, key or password",
	}

	files := map[string][]byte{
		"build.json":      jsonIndent(buildInfo(s.version)),
		settings.FileName: redactSettings(readFile(filepath.Join(dir, settings.FileName)), s.home),
		"profiles.json":   jsonIndent(collectProfiles(filepath.Join(dir, "profiles"))),
		"queue.json":      redactQueue(readFile(filepath.Join(dir, "queue.json"))),
	}

	if report, err := s.Doctor(); err == nil {
		files["doctor.txt"] = []byte(hideHomeIn(doctor.PlainText(report), s.home))
	}
	if runs := recentRuns(filepath.Join(dir, "profiles"), diagnosticsRuns); len(runs) > 0 {
		files["runs.json"] = []byte(hideHomeIn(string(jsonIndent(runs)), s.home))
	}
	logName, logBody := mortarLog(dir, s.recentLines(gameID, profileID))
	if logName != "" {
		files[logName] = []byte(hideHomeIn(logBody, s.home))
		included = append(included, logName+": last "+fmt.Sprint(diagnosticsLogLines)+" lines of Mortar's log or in-memory console, home folder written as ~")
	}
	for _, name := range []string{prevLogName, crashLogName} {
		if b := readFile(filepath.Join(dir, name)); len(b) > 0 {
			files[name] = []byte(hideHomeIn(lastLines(string(b), diagnosticsLogLines), s.home))
			included = append(included, name+": last "+fmt.Sprint(diagnosticsLogLines)+" lines, home folder written as ~")
		}
	}
	if smapi := s.smapiTail(gameID, profileID); smapi != "" {
		files["smapi-latest.txt"] = []byte(hideHomeIn(smapi, s.home))
		included = append(included, "smapi-latest.txt: last "+fmt.Sprint(diagnosticsLogLines)+" lines of the open profile's SMAPI log")
	}

	files["manifest.txt"] = []byte(diagnosticsManifest(included, removed))
	return zipFiles(files)
}

func (s *Service) recentLines(gameID, profileID string) string {
	if s.RecentLog == nil {
		return ""
	}
	return s.RecentLog(gameID, profileID)
}

func diagnosticsManifest(included, removed []string) string {
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
		"mortar":   version,
		"go":       runtime.Version(),
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"portable": strconv.FormatBool(datadir.Portable()),
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
		b, err := fsx.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			continue
		}
		return "mortar.log", lastLines(string(b), diagnosticsLogLines)
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
	b, err := fsx.ReadFile(path)
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

func redactSettings(raw []byte, home string) []byte {
	m := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &m)
	}
	if m == nil {
		m = map[string]any{}
	}
	m["nexusName"] = ""
	m["nexusUserId"] = 0
	delete(m, "nexusPremium")
	dropSecretFields(m)
	redactHomePaths(m, home)
	return jsonIndent(m)
}

// dropSecretFields removes every field, at any depth, whose name marks a credential, so a key added later is
// covered without a list to update.
func dropSecretFields(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			if secretName(k) {
				delete(x, k)
				continue
			}
			dropSecretFields(child)
		}
	case []any:
		for _, child := range x {
			dropSecretFields(child)
		}
	}
}

func secretName(k string) bool {
	k = strings.ToLower(k)
	for _, w := range []string{"token", "secret", "apikey", "nexuskey", "password", "keyring"} {
		if strings.Contains(k, w) {
			return true
		}
	}
	return false
}

func redactHomePaths(v any, home string) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			if s, ok := child.(string); ok {
				x[k] = hideHome(s, home)
				continue
			}
			redactHomePaths(child, home)
		}
	case []any:
		for i, child := range x {
			if s, ok := child.(string); ok {
				x[i] = hideHome(s, home)
				continue
			}
			redactHomePaths(child, home)
		}
	}
}

func hideHome(s, home string) string {
	if home == "" || s == "" {
		return s
	}
	for _, h := range []string{home, filepath.ToSlash(home), filepath.FromSlash(home)} {
		h = strings.TrimRight(h, `/\`)
		if h == "" {
			continue
		}
		if s == h {
			return "~"
		}
		if strings.HasPrefix(s, h+"/") || strings.HasPrefix(s, h+`\`) {
			return "~" + s[len(h):]
		}
	}
	return s
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
	dirs, err := datadir.ProfileDirs(root)
	if err != nil {
		return []bundleProfile{}
	}
	for _, d := range dirs {
		raw, err := fsx.ReadFile(filepath.Join(d.Dir, "profile.json"))
		if err != nil {
			continue
		}
		var p profile.Profile
		if json.Unmarshal(raw, &p) != nil {
			continue
		}
		bp := bundleProfile{Game: d.Game, ID: p.ID, Name: p.Name, EntryCount: len(p.Entries)}
		for _, ent := range p.Entries {
			be := bundleEntry{Source: ent.Source}
			for _, mod := range ent.Mods {
				be.Mods = append(be.Mods, bundleMod{Name: mod.Name, UniqueID: mod.UniqueID, Version: mod.Version})
			}
			bp.Entries = append(bp.Entries, be)
		}
		out = append(out, bp)
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

// recentRuns reads each profile's runs/index.json and returns the n newest runs across all of them. Run records
// carry outcome, versions and counts; the log text stays out.
func recentRuns(profilesRoot string, n int) []map[string]any {
	var all []map[string]any
	dirs, _ := datadir.ProfileDirs(profilesRoot)
	for _, d := range dirs {
		var idx struct {
			Runs []map[string]any `json:"runs"`
		}
		if json.Unmarshal(readFile(filepath.Join(d.Dir, "runs", "index.json")), &idx) != nil {
			continue
		}
		for _, r := range idx.Runs {
			r["game"], r["profile"] = d.Game, d.ID
			all = append(all, r)
		}
	}
	slices.SortFunc(all, func(a, b map[string]any) int {
		as, _ := a["started"].(string)
		bs, _ := b["started"].(string)
		return strings.Compare(bs, as)
	})
	if len(all) > n {
		all = all[:n]
	}
	return all
}
