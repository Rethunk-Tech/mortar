package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/meta"
)

const importedProfileName = "Imported mods"

const (
	outcomeImported = "imported"
	outcomeSkipped  = "skipped"
	outcomeFailed   = "failed"
)

const configFileName = "config.json"

// GameModPreview is one row PreviewGameMods shows: a mod to copy, or a skipped or failed folder.
type GameModPreview struct {
	UniqueID   string `json:"uniqueID,omitempty"`
	Name       string `json:"name"`
	Version    string `json:"version,omitempty"`
	Source     string `json:"source,omitempty"`
	NexusModID int    `json:"nexusModID,omitempty"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
	Disabled   bool   `json:"disabled,omitempty"`
}

// GameModsPreview is the list PreviewGameMods returns, including skips and failures.
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

// ExternalMod is one folder selected from an external mod manager's profile.
type ExternalMod struct {
	SourcePath string `json:"sourcePath"`
	UniqueID   string `json:"uniqueID"`
	Enabled    bool   `json:"enabled"`
}

type gameModFolder struct {
	dir      string
	label    string
	disabled bool
	mods     []manifest.Mod
	source   Source
}

type gameModSlot struct {
	folder     gameModFolder
	ready      bool
	outcome    GameModOutcome
	hasOutcome bool
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

func classifyFolder(dir, name string) (gameModSlot, bool) {
	label := strings.TrimLeft(name, ".")
	if bundledFolder(name) != "" {
		return gameModSlot{}, false
	}
	out := GameModOutcome{Name: label}
	m, hasManifest, err := readTopManifest(dir)
	if err != nil && hasManifest {
		out.Status, out.Reason = outcomeFailed, "The manifest is invalid"
		return gameModSlot{outcome: out, hasOutcome: true}, true
	}
	var mods []manifest.Mod
	if hasManifest {
		mods = []manifest.Mod{{Manifest: m, Folder: "."}}
	} else {
		found, scanErr := manifest.Scan(dir)
		if scanErr != nil {
			out.Status, out.Reason = outcomeFailed, scanErr.Error()
			return gameModSlot{outcome: out, hasOutcome: true}, true
		}
		if len(found) == 0 {
			if _, err := fsx.ReadFile(filepath.Join(dir, manifest.FileName)); err == nil {
				out.Status, out.Reason = outcomeFailed, "The manifest is invalid"
				return gameModSlot{outcome: out, hasOutcome: true}, true
			}
			out.Status, out.Reason = outcomeFailed, "No SMAPI mod was found"
			return gameModSlot{outcome: out, hasOutcome: true}, true
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
		return gameModSlot{}, false
	}
	return gameModSlot{
		folder: gameModFolder{
			dir: dir, label: label, disabled: strings.HasPrefix(name, "."),
			mods: mods, source: sourceFromMods(mods, label),
		},
		ready: true,
	}, true
}

func modVersion(f gameModFolder, id string) string {
	for _, m := range f.mods {
		if strings.EqualFold(m.UniqueID, id) {
			return m.Version
		}
	}
	return ""
}

func preferFolder(a, b gameModFolder, id string) bool {
	if a.disabled != b.disabled {
		return !a.disabled
	}
	va, vb := modVersion(a, id), modVersion(b, id)
	if c, ok := meta.CompareVersions(va, vb); ok && c != 0 {
		return c > 0
	}
	return false
}

func resolveDuplicates(slots []gameModSlot) {
	winner := map[string]int{}
	for i, s := range slots {
		if !s.ready {
			continue
		}
		for _, m := range s.folder.mods {
			id := strings.ToLower(m.UniqueID)
			j, ok := winner[id]
			if !ok || preferFolder(s.folder, slots[j].folder, m.UniqueID) {
				winner[id] = i
			}
		}
	}
	type skip struct {
		i      int
		reason string
	}
	var losers []skip
	seen := map[int]bool{}
	for i, s := range slots {
		if !s.ready {
			continue
		}
		for _, m := range s.folder.mods {
			id := strings.ToLower(m.UniqueID)
			w := winner[id]
			if w == i || seen[i] {
				continue
			}
			seen[i] = true
			losers = append(losers, skip{i, fmt.Sprintf("same mod as %s", slots[w].folder.label)})
		}
	}
	for _, l := range losers {
		slots[l.i].ready = false
		slots[l.i].hasOutcome = true
		slots[l.i].outcome = GameModOutcome{Name: slots[l.i].folder.label, Status: outcomeSkipped, Reason: l.reason}
	}
}

func scanGameMods(modsDir string) ([]gameModSlot, error) {
	entries, err := os.ReadDir(modsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(modsDir)
	if err != nil {
		return nil, err
	}
	var slots []gameModSlot
	for _, it := range entries {
		child := filepath.Join(modsDir, it.Name())
		if !datadir.RealDirUnder(root, child) {
			continue
		}
		slot, keep := classifyFolder(child, it.Name())
		if !keep {
			continue
		}
		slots = append(slots, slot)
	}
	resolveDuplicates(slots)
	return slots, nil
}

func previewMod(f gameModFolder, m manifest.Mod) GameModPreview {
	name := m.Name
	if name == "" {
		name = m.UniqueID
	}
	return GameModPreview{
		UniqueID: m.UniqueID, Name: name, Version: m.Version, Source: sourceLabel(f.source),
		NexusModID: f.source.ModID, Status: outcomeImported, Disabled: f.disabled,
	}
}

func previewFrom(slots []gameModSlot) GameModsPreview {
	var mods []GameModPreview
	for _, s := range slots {
		if s.ready {
			for _, m := range s.folder.mods {
				mods = append(mods, previewMod(s.folder, m))
			}
			continue
		}
		if s.hasOutcome {
			mods = append(mods, GameModPreview{
				Name: s.outcome.Name, Status: s.outcome.Status, Reason: s.outcome.Reason,
				Disabled: s.folder.disabled,
			})
		}
	}
	if mods == nil {
		mods = []GameModPreview{}
	}
	return GameModsPreview{Mods: mods}
}

func outcomesFrom(slots []gameModSlot) []GameModOutcome {
	var out []GameModOutcome
	for _, s := range slots {
		if s.hasOutcome {
			out = append(out, s.outcome)
		}
	}
	return out
}

// PreviewGameMods lists mods that ImportGameMods would copy from modsDir. It reads only.
func (s *Store) PreviewGameMods(modsDir string) (GameModsPreview, error) {
	slots, err := scanGameMods(modsDir)
	if err != nil {
		return GameModsPreview{}, err
	}
	return previewFrom(slots), nil
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

func (s *Store) importFolder(game, id string, f gameModFolder) (string, error) {
	key, err := s.items.AddHashedDir(game, f.dir)
	if err != nil {
		return "", err
	}
	if _, err := s.AddEntry(game, id, key, f.source); err != nil {
		return "", err
	}
	if f.disabled {
		for _, m := range f.mods {
			if _, err := s.SetModEnabled(game, id, key, m.UniqueID, false); err != nil {
				return "", err
			}
		}
	}
	for _, m := range f.mods {
		dest, err := s.ModFolder(game, id, key, m.UniqueID)
		if err != nil {
			return "", err
		}
		if err := carryConfig(f.dir, dest, m.Folder); err != nil {
			return "", err
		}
	}
	return key, nil
}

// ImportGameMods copies each importable folder under modsDir into the store and a new "Imported mods" profile.
// It never writes to modsDir.
func (s *Store) ImportGameMods(game, modsDir string) (GameModsResult, error) {
	slots, err := scanGameMods(modsDir)
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
	created, err = s.SetOrigin(game, created.ID, OriginGameMods, "")
	if err != nil {
		return GameModsResult{}, errors.Join(err, s.Delete(game, created.ID))
	}
	s.setHistoryQuiet(created.ID, true)
	defer s.setHistoryQuiet(created.ID, false)
	res := GameModsResult{Profile: created, Outcomes: outcomesFrom(slots)}
	for _, o := range res.Outcomes {
		switch o.Status {
		case outcomeSkipped:
			res.Skipped++
		case outcomeFailed:
			res.Failed++
		}
	}
	for _, slot := range slots {
		if !slot.ready {
			continue
		}
		outcome := GameModOutcome{Name: slot.folder.label, Status: outcomeImported}
		if _, err := s.importFolder(game, created.ID, slot.folder); err != nil {
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
	if res.Imported > 0 {
		if err := s.recordSnapshot(game, created.ID, historyImported, fmt.Sprintf("Imported %d mods", res.Imported), res.Imported); err != nil {
			return res, err
		}
	}
	return res, nil
}

// ImportExternalMods copies the selected external-manager folders into an existing profile.
func (s *Store) ImportExternalMods(game, id string, mods []ExternalMod) error {
	enabled := make(map[string]bool, len(mods))
	paths := make(map[string]bool, len(mods))
	for _, mod := range mods {
		if mod.SourcePath == "" {
			continue
		}
		paths[filepath.Clean(mod.SourcePath)] = true
		enabled[strings.ToLower(mod.UniqueID)] = mod.Enabled
	}
	var slots []gameModSlot
	for path := range paths {
		name := filepath.Base(path)
		slot, keep := classifyFolder(path, name)
		if keep {
			slots = append(slots, slot)
		}
	}
	resolveDuplicates(slots)
	for _, slot := range slots {
		if !slot.ready {
			continue
		}
		key, err := s.importFolder(game, id, slot.folder)
		if err != nil {
			return err
		}
		for _, mod := range slot.folder.mods {
			if want, ok := enabled[strings.ToLower(mod.UniqueID)]; ok {
				if _, err := s.SetModEnabled(game, id, key, mod.UniqueID, want); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
