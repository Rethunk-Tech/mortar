package profile

import (
	"cmp"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/installer"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// Converting an archive mod to a REDmod and back is Vortex's redmod.conversion.actions.ts. Vortex rearranges the mod's
// staging folder; Mortar's store items never change, so the entry records the REDmod's module name and the layout is
// moved when the profile's files are laid out.

// generatedDir holds, in the profile's root, the files a conversion generates (a REDmod's info.json).
const generatedDir = ".mortar-generated"

const defaultREDmodVersion = "0.0.1+V2077"

// REDmodSkip is an entry a conversion left alone, and why.
type REDmodSkip struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// REDmodResult is a profile after converting entries, and the ones that could not be.
type REDmodResult struct {
	Profile Profile      `json:"profile"`
	Skipped []REDmodSkip `json:"skipped"`
}

func (s *Store) isCyberpunk(game string) bool {
	info, ok := components.Game(game)
	if !ok {
		return false
	}
	t, has := info.Target(installer.TargetMods)
	return has && t.Layout == components.LayoutCyberpunk
}

func redmodVersionOf(e Entry) string {
	for _, m := range e.Mods {
		if m.Version != "" {
			return m.Version
		}
	}
	return cmp.Or(e.Source.Version, defaultREDmodVersion)
}

// modsLayout is the files of l bound for the mods target, which the conversion moves.
func modsLayout(l installer.Layout) []installer.File {
	var out []installer.File
	for _, f := range l.Files {
		if f.Target == installer.TargetMods {
			out = append(out, f)
		}
	}
	return out
}

// convertedLayout is files with e's converted REDmod laid out: its archives below mods/<module>, and the info.json to
// generate. An entry that is not converted, or no longer converts (an update added files that are not archives), lays
// out as it is.
func (s *Store) convertedLayout(game string, e Entry, files []installer.File) ([]installer.File, *installer.REDmodConversion) {
	if e.REDmod == "" || e.File != "" || !s.isCyberpunk(game) {
		return files, nil
	}
	conv, err := installer.ConvertToREDmod(files, e.REDmod, redmodVersionOf(e))
	if err != nil {
		slog.Warn("a converted REDmod no longer converts, so it is laid out as an archive mod", "mod", entryLabel(e), "error", err)
		return files, nil
	}
	return conv.Files, &conv
}

// writeGenerated writes a generated file of entry key to the profile's generated folder, and returns where.
func writeGenerated(dir, key, content string) (string, error) {
	file := filepath.Join(dir, generatedDir, strings.NewReplacer("/", "_", "\\", "_", "#", "_").Replace(key)+".info.json")
	if have, _, err := safefs.ReadFileIfExists(file); err == nil && string(have) == content {
		return file, nil
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		return "", err
	}
	return file, safefs.WriteFileAtomic(file, []byte(content), 0o644)
}

// plan is what converting e does: the module name it would take, or why it cannot.
func (s *Store) planREDmod(game, id string, e Entry, taken []string) (string, error) {
	if !e.Package || e.IsOverlay() || e.File != "" || e.Tray {
		return "", errors.New("only a mod that is one download of archives can be converted")
	}
	_, l, _, err := s.layoutOf(game, id, e.StoreKey(), e.Fomod)
	if err != nil {
		return "", err
	}
	wanted, err := installer.REDmodModuleName(cmp.Or(e.DisplayName, entryLabel(e)))
	if err != nil {
		return "", err
	}
	module := installer.UniqueREDmodModuleName(wanted, taken)
	if _, err := installer.ConvertToREDmod(modsLayout(l), module, redmodVersionOf(e)); err != nil {
		return "", err
	}
	return module, nil
}

// REDmodChoice is what the mod menu offers an entry of a Cyberpunk 2077 profile: "convert" for an archive mod that can
// be converted, "revert" for one that was, and "" for anything else.
func (s *Store) REDmodChoice(game, id, key string) (string, error) {
	if !s.isCyberpunk(game) {
		return "", nil
	}
	p, err := s.read(game, id)
	if err != nil {
		return "", err
	}
	i := EntryIndex(p.Entries, key)
	if i < 0 {
		return "", nil
	}
	if p.Entries[i].REDmod != "" {
		return "revert", nil
	}
	taken, err := s.REDmodFolders(game, id)
	if err != nil {
		return "", err
	}
	if _, err := s.planREDmod(game, id, p.Entries[i], taken); err == nil {
		return "convert", nil
	}
	return "", nil
}

// ConvertToREDmod converts the named entries' archives to REDmods, each under a folder of its own named for the mod.
// The REDmod DLC must be installed, since a converted mod is loaded by REDmod and nothing else.
func (s *Store) ConvertToREDmod(game, id string, keys []string) (REDmodResult, error) {
	if !s.isCyberpunk(game) {
		return REDmodResult{}, usererr.New(usererr.Invalid, "only Cyberpunk 2077 mods convert to REDmods")
	}
	if s.REDmodReady != nil && !s.REDmodReady(game, id) {
		return REDmodResult{}, usererr.New(usererr.External, "converting to a REDmod needs the free REDmod DLC, which is not installed")
	}
	p, err := s.read(game, id)
	if err != nil {
		return REDmodResult{}, err
	}
	taken, err := s.REDmodFolders(game, id)
	if err != nil {
		return REDmodResult{}, err
	}
	// A converted mod switched off is not laid out, but switching it back on needs its folder free.
	for _, e := range p.Entries {
		if e.REDmod != "" && !slices.Contains(taken, e.REDmod) {
			taken = append(taken, e.REDmod)
		}
	}
	var skipped []REDmodSkip
	modules := map[string]string{}
	var ok []string
	for _, key := range keys {
		i, err := requireEntry(p.Entries, key)
		if err != nil {
			return REDmodResult{}, err
		}
		e := p.Entries[i]
		if e.REDmod != "" {
			skipped = append(skipped, REDmodSkip{Key: key, Name: entryLabel(e), Reason: "it is already a REDmod"})
			continue
		}
		module, err := s.planREDmod(game, id, e, taken)
		if err != nil {
			skipped = append(skipped, REDmodSkip{Key: key, Name: entryLabel(e), Reason: err.Error()})
			continue
		}
		taken = append(taken, module)
		modules[key] = module
		ok = append(ok, key)
	}
	out, err := s.updateEntriesAs(game, id, "", HistoryEvent{}, ok, func(e *Entry, key string) error {
		e.REDmod = modules[key]
		return nil
	})
	return REDmodResult{Profile: out, Skipped: nonNilSkips(skipped)}, err
}

// RevertREDmod puts the named converted entries' archives back where they started.
func (s *Store) RevertREDmod(game, id string, keys []string) (REDmodResult, error) {
	if !s.isCyberpunk(game) {
		return REDmodResult{}, usererr.New(usererr.Invalid, "only Cyberpunk 2077 mods convert to REDmods")
	}
	p, err := s.read(game, id)
	if err != nil {
		return REDmodResult{}, err
	}
	var skipped []REDmodSkip
	var ok []string
	for _, key := range keys {
		i, err := requireEntry(p.Entries, key)
		if err != nil {
			return REDmodResult{}, err
		}
		if p.Entries[i].REDmod == "" {
			skipped = append(skipped, REDmodSkip{Key: key, Name: entryLabel(p.Entries[i]), Reason: "it was not converted from an archive mod"})
			continue
		}
		ok = append(ok, key)
	}
	out, err := s.updateEntries(game, id, ok, func(e *Entry, _ string) error {
		e.REDmod = ""
		return nil
	})
	return REDmodResult{Profile: out, Skipped: nonNilSkips(skipped)}, err
}

func nonNilSkips(in []REDmodSkip) []REDmodSkip {
	if in == nil {
		return []REDmodSkip{}
	}
	return in
}

// ConvertToREDmod is Store.ConvertToREDmod.
func (s *Service) ConvertToREDmod(game, id string, keys []string) (REDmodResult, error) {
	return s.store.ConvertToREDmod(game, id, keys)
}

// RevertREDmod is Store.RevertREDmod.
func (s *Service) RevertREDmod(game, id string, keys []string) (REDmodResult, error) {
	return s.store.RevertREDmod(game, id, keys)
}

// REDmodChoice is Store.REDmodChoice.
func (s *Service) REDmodChoice(game, id, key string) (string, error) {
	return s.store.REDmodChoice(game, id, key)
}

// recordInstallNotes keeps what the Cyberpunk installer noted about key for Problems, which shows it while the mod is
// in the profile.
func (s *Store) recordInstallNotes(game, id, key string, whole []Component, l installer.Layout) error {
	if l.Kind == "" && len(l.Notes) == 0 || !s.isCyberpunk(game) {
		return nil
	}
	dir, err := s.ProfileDir(game, id)
	if err != nil {
		return err
	}
	name := key
	if len(whole) > 0 && whole[0].Name != "" {
		name = whole[0].Name
	}
	return installer.RecordNotes(dir, key, name, l.Notes)
}
