package profile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"
)

const importedProfileName = "Imported mods"

const (
	outcomeImported = "imported"
	outcomeSkipped  = "skipped"
	outcomeFailed   = "failed"
)

const configFileName = "config.json"

// GameModPreview is one mod that ImportGameMods would copy from the game's Mods folder.
type GameModPreview struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

// GameModsPreview is the list ImportGameMods would copy.
type GameModsPreview struct {
	Mods []GameModPreview `json:"mods"`
}

// GameModOutcome is one top-level folder after ImportGameMods.
type GameModOutcome struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// GameModsResult is the profile ImportGameMods created and per-folder results.
type GameModsResult struct {
	Profile  Profile          `json:"profile"`
	Outcomes []GameModOutcome `json:"outcomes"`
	Imported int              `json:"imported"`
	Skipped  int              `json:"skipped"`
	Failed   int              `json:"failed"`
}

type gameModFolder struct {
	dir      string
	label    string
	disabled bool
	mods     []manifest.Mod
	source   Source
}

func bundledFolder(name string) string {
	switch strings.ToLower(strings.TrimLeft(name, ".")) {
	case "consolecommands":
		return "SMAPI's bundled Console Commands"
	case "savebackup":
		return "SMAPI's bundled Save Backup"
	case "mortarsmapibridge":
		return "Mortar's SMAPI bridge"
	}
	return ""
}

func bundledUniqueID(id string) bool {
	switch strings.ToLower(id) {
	case "smapi.consolecommands", "smapi.savebackup", "rethunk.mortarsmapibridge":
		return true
	}
	return false
}

func nexusUpdateKey(key string) (int, bool) {
	site, rest, ok := strings.Cut(key, ":")
	if !ok || !strings.EqualFold(strings.TrimSpace(site), "nexus") {
		return 0, false
	}
	rest, _, _ = strings.Cut(rest, "@")
	n, err := strconv.Atoi(strings.TrimSpace(rest))
	return n, err == nil
}

func githubUpdateKey(key string) (string, bool) {
	site, rest, ok := strings.Cut(key, ":")
	rest = strings.TrimSpace(rest)
	return rest, ok && strings.EqualFold(strings.TrimSpace(site), "github") && strings.Count(rest, "/") == 1
}

func sourceFromMods(mods []manifest.Mod, folder string) Source {
	for _, m := range mods {
		for _, k := range m.UpdateKeys {
			if n, ok := nexusUpdateKey(k); ok {
				return Source{Kind: KindNexus, Name: folder, ModID: n, Version: m.Version}
			}
		}
	}
	for _, m := range mods {
		for _, k := range m.UpdateKeys {
			if repo, ok := githubUpdateKey(k); ok {
				return Source{Kind: KindGitHub, Name: folder, Repo: repo, Version: m.Version}
			}
		}
	}
	return Source{Kind: KindLocal, Name: folder}
}

func sourceLabel(s Source) string {
	switch s.Kind {
	case KindNexus:
		return "Nexus"
	case KindGitHub:
		return "GitHub"
	default:
		return "Local"
	}
}

func readTopManifest(dir string) (manifest.Manifest, bool, error) {
	b, err := fsx.ReadFile(filepath.Join(dir, manifest.FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return manifest.Manifest{}, false, nil
	}
	if err != nil {
		return manifest.Manifest{}, true, err
	}
	m, err := manifest.Parse(b)
	return m, true, err
}

func classifyFolder(dir, name string) (gameModFolder, GameModOutcome, bool) {
	label := strings.TrimLeft(name, ".")
	out := GameModOutcome{Name: label}
	if reason := bundledFolder(name); reason != "" {
		out.Status, out.Reason = outcomeSkipped, reason
		return gameModFolder{}, out, false
	}
	m, hasManifest, err := readTopManifest(dir)
	if err != nil && hasManifest {
		out.Status, out.Reason = outcomeFailed, "The manifest is invalid"
		return gameModFolder{}, out, false
	}
	var mods []manifest.Mod
	if hasManifest {
		mods = []manifest.Mod{{Manifest: m, Folder: "."}}
	} else {
		found, scanErr := manifest.Scan(dir)
		if scanErr != nil {
			out.Status, out.Reason = outcomeFailed, scanErr.Error()
			return gameModFolder{}, out, false
		}
		if len(found) == 0 {
			if _, err := fsx.ReadFile(filepath.Join(dir, manifest.FileName)); err == nil {
				out.Status, out.Reason = outcomeFailed, "The manifest is invalid"
				return gameModFolder{}, out, false
			}
			out.Status, out.Reason = outcomeFailed, "No SMAPI mod was found"
			return gameModFolder{}, out, false
		}
		mods = found
	}
	allBundled := true
	for _, mod := range mods {
		if !bundledUniqueID(mod.UniqueID) {
			allBundled = false
			break
		}
	}
	if allBundled {
		out.Status, out.Reason = outcomeSkipped, "SMAPI's bundled mods or Mortar's bridge"
		return gameModFolder{}, out, false
	}
	return gameModFolder{
		dir: dir, label: label, disabled: strings.HasPrefix(name, "."),
		mods: mods, source: sourceFromMods(mods, label),
	}, GameModOutcome{}, true
}

func scanGameMods(modsDir string) (ready []gameModFolder, outcomes []GameModOutcome, err error) {
	entries, err := os.ReadDir(modsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	for _, it := range entries {
		if !it.IsDir() {
			continue
		}
		dir := filepath.Join(modsDir, it.Name())
		folder, outcome, ok := classifyFolder(dir, it.Name())
		if !ok {
			outcomes = append(outcomes, outcome)
			continue
		}
		ready = append(ready, folder)
	}
	return ready, outcomes, nil
}

func previewFrom(ready []gameModFolder) GameModsPreview {
	var mods []GameModPreview
	for _, f := range ready {
		src := sourceLabel(f.source)
		for _, m := range f.mods {
			name := m.Name
			if name == "" {
				name = m.UniqueID
			}
			mods = append(mods, GameModPreview{Name: name, Version: m.Version, Source: src})
		}
	}
	if mods == nil {
		mods = []GameModPreview{}
	}
	return GameModsPreview{Mods: mods}
}

// PreviewGameMods lists mods that ImportGameMods would copy from modsDir. It reads only.
func (s *Store) PreviewGameMods(modsDir string) (GameModsPreview, error) {
	ready, _, err := scanGameMods(modsDir)
	if err != nil {
		return GameModsPreview{}, err
	}
	return previewFrom(ready), nil
}

func carryConfig(srcRoot, destDir, folder string) error {
	rel := folder
	if rel == "." {
		rel = ""
	}
	from := filepath.Join(srcRoot, filepath.FromSlash(rel), configFileName)
	b, err := fsx.ReadFile(from)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return fsx.WriteFile(filepath.Join(destDir, configFileName), b, 0o600)
}

func (s *Store) importFolder(game, id string, f gameModFolder) error {
	key, err := s.items.AddHashedDir(game, f.dir)
	if err != nil {
		return err
	}
	if _, err := s.AddEntry(game, id, key, f.source); err != nil {
		return err
	}
	if f.disabled {
		for _, m := range f.mods {
			if _, err := s.SetModEnabled(game, id, key, m.UniqueID, false); err != nil {
				return err
			}
		}
	}
	for _, m := range f.mods {
		dest, err := s.ModFolder(game, id, key, m.UniqueID)
		if err != nil {
			return err
		}
		if err := carryConfig(f.dir, dest, m.Folder); err != nil {
			return err
		}
	}
	return nil
}

// ImportGameMods copies each importable folder under modsDir into the store and a new "Imported mods" profile.
// It never writes to modsDir.
func (s *Store) ImportGameMods(game, modsDir string) (GameModsResult, error) {
	ready, outcomes, err := scanGameMods(modsDir)
	if err != nil {
		return GameModsResult{}, err
	}
	existing, err := s.List(game)
	if err != nil {
		return GameModsResult{}, err
	}
	taken := make([]string, len(existing))
	for i, p := range existing {
		taken[i] = p.Name
	}
	created, err := s.Create(game, UniqueName(taken, importedProfileName))
	if err != nil {
		return GameModsResult{}, err
	}
	res := GameModsResult{Profile: created, Outcomes: outcomes}
	for _, o := range outcomes {
		switch o.Status {
		case outcomeSkipped:
			res.Skipped++
		case outcomeFailed:
			res.Failed++
		}
	}
	for _, f := range ready {
		outcome := GameModOutcome{Name: f.label, Status: outcomeImported}
		if err := s.importFolder(game, created.ID, f); err != nil {
			outcome.Status, outcome.Reason = outcomeFailed, err.Error()
			if ie, ok := errors.AsType[*InstallError](err); ok {
				outcome.Reason = ie.Msg
			}
			res.Failed++
		} else {
			res.Imported++
		}
		res.Outcomes = append(res.Outcomes, outcome)
	}
	p, err := s.read(game, created.ID)
	if err != nil {
		return res, err
	}
	res.Profile = p
	if res.Outcomes == nil {
		res.Outcomes = []GameModOutcome{}
	}
	return res, nil
}
