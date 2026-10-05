package profile

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/fomod"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// FomodAsk is a FOMOD wizard the window shows before files land in the profile.
type FomodAsk struct {
	Key        string                         `json:"key"`
	Source     Source                         `json:"source"`
	OldKey     string                         `json:"oldKey,omitempty"`
	ModuleName string                         `json:"moduleName"`
	Steps      []FomodStep                    `json:"steps"`
	Choices    map[string]map[string][]string `json:"choices,omitempty"`
	// Changed means a new version's install options no longer fit the entry's saved choices.
	Changed bool `json:"changed,omitempty"`
}

type FomodStep struct {
	Name   string       `json:"name"`
	Groups []FomodGroup `json:"groups"`
}

type FomodGroup struct {
	Name    string        `json:"name"`
	Type    string        `json:"type"`
	Plugins []FomodPlugin `json:"plugins"`
}

type FomodPlugin struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Type        string `json:"type"`
}

// NeedChoicesError means the store item has a FOMOD config and the given (or stored) choices do not match it.
type NeedChoicesError struct {
	Ask FomodAsk
}

func (e *NeedChoicesError) Error() string { return "this mod has install options" }

func (s *Store) reuseFomod() bool {
	if s.settings == nil {
		return true
	}
	return s.settings.Get().ReuseFomod()
}

func (s *Store) fileIndex(modsDir string) fomod.FileIndex {
	states := map[string]string{}
	_ = filepath.WalkDir(modsDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		root := filepath.Clean(modsDir)
		clean := filepath.Clean(p)
		sep := string(os.PathSeparator)
		if clean != root && !strings.HasPrefix(clean, root+sep) {
			return fmt.Errorf("mod file %q is not under %s", p, modsDir)
		}
		rel := strings.TrimPrefix(clean, root+sep)
		name := filepath.Base(p)
		inactive := false
		for part := range strings.SplitSeq(filepath.ToSlash(rel), "/") {
			if strings.HasPrefix(part, ".") {
				inactive = true
				break
			}
		}
		prev := states[name]
		if !inactive {
			states[name] = fomod.FileActive
			return nil
		}
		if prev != fomod.FileActive {
			states[name] = fomod.FileInactive
		}
		return nil
	})
	return func(name string) string {
		if s := states[name]; s != "" {
			return s
		}
		if s := states[filepath.Base(name)]; s != "" {
			return s
		}
		return fomod.FileMissing
	}
}

func (s *Store) fomodEval(gameID string, files fomod.FileIndex) fomod.EvalContext {
	return fomod.EvalContext{Files: files, GameVersion: s.installedGameVersion(gameID)}
}

func (s *Store) installedGameVersion(gameID string) string {
	if game.Find(gameID) == nil {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	sett, err := settings.Open()
	if err != nil {
		return ""
	}
	dir, err := game.InstallDir(home, sett.Get(), gameID)
	if err != nil || dir == "" {
		return ""
	}
	st, err := game.LoaderStatus(gameID, "", dir, sett.Get().Loaders)
	if err != nil {
		return ""
	}
	return st.GameVersion
}

func (s *Store) fomodOf(game, key string) (fomod.Config, bool, error) {
	dir, err := s.items.Path(game, key)
	if err != nil {
		return fomod.Config{}, false, err
	}
	cfg, _, ok, err := fomod.Open(dir)
	return cfg, ok, err
}

func askFrom(cfg fomod.Config, key string, source Source, oldKey string, choices map[string]map[string][]string, eval fomod.EvalContext) FomodAsk {
	flags := fomod.FlagsFrom(cfg, choices, eval)
	ask := FomodAsk{Key: key, Source: source, OldKey: oldKey, ModuleName: cfg.ModuleName, Choices: choices}
	for _, st := range fomod.VisibleSteps(cfg, flags, eval) {
		step := FomodStep{Name: st.Name}
		for _, g := range st.Groups {
			gr := FomodGroup{Name: g.Name, Type: g.Type}
			for _, p := range g.Plugins {
				gr.Plugins = append(gr.Plugins, FomodPlugin{
					Name: p.Name, Description: p.Description, Image: p.Image,
					Type: fomod.PluginType(p, flags, eval),
				})
			}
			step.Groups = append(step.Groups, gr)
		}
		ask.Steps = append(ask.Steps, step)
	}
	return ask
}

func (s *Store) fomodAsk(game, id, key string, source Source, oldKey string, choices map[string]map[string][]string) (FomodAsk, bool, error) {
	cfg, ok, err := s.fomodOf(game, key)
	if err != nil || !ok {
		return FomodAsk{}, false, err
	}
	modsDir, err := s.ModsDir(game, id)
	if err != nil {
		return FomodAsk{}, false, err
	}
	eval := s.fomodEval(game, s.fileIndex(modsDir))
	if fomod.Match(cfg, choices, eval) && s.reuseFomod() {
		return FomodAsk{}, false, nil
	}
	return askFrom(cfg, key, source, oldKey, choices, eval), true, nil
}

// replayAsk is fomodAsk for choices saved against oldKey's config. It also asks when key shows a group to pick from
// that oldKey did not have, and then prefills the wizard with the saved choices that still exist.
func (s *Store) replayAsk(game, id, key string, source Source, oldKey string, choices map[string]map[string][]string) (FomodAsk, bool, error) {
	cfg, ok, err := s.fomodOf(game, key)
	if err != nil || !ok {
		return FomodAsk{}, false, err
	}
	old, _, err := s.fomodOf(game, oldKey)
	if err != nil {
		old = cfg
	}
	modsDir, err := s.ModsDir(game, id)
	if err != nil {
		return FomodAsk{}, false, err
	}
	eval := s.fomodEval(game, s.fileIndex(modsDir))
	if fomod.Match(cfg, choices, eval) && !fomod.Unanswered(old, cfg, choices, eval) && s.reuseFomod() {
		return FomodAsk{}, false, nil
	}
	ask := askFrom(cfg, key, source, oldKey, fomod.Keep(cfg, choices), eval)
	ask.Changed = true
	return ask, true, nil
}

// installAsk is the wizard installing key needs. When key replaces one entry of the profile and the user gave no
// choices, that entry's saved choices replay: the returned source carries them.
func (s *Store) installAsk(game, id, key string, source Source) (Source, FomodAsk, bool, error) {
	if source.fomod != nil {
		ask, need, err := s.fomodAsk(game, id, key, source, "", source.fomodMap())
		return source, ask, need, err
	}
	if _, ok, err := s.fomodOf(game, key); err != nil || !ok {
		return source, FomodAsk{}, false, err
	}
	prev, ok, err := s.replacing(game, id, key)
	if err != nil {
		return source, FomodAsk{}, false, err
	}
	if !ok {
		ask, need, err := s.fomodAsk(game, id, key, source, "", nil)
		return source, ask, need, err
	}
	ask, need, err := s.replayAsk(game, id, key, source, prev.Key, prev.Fomod)
	if err != nil || need {
		return source, ask, need, err
	}
	return source.WithFomod(cloneFomod(prev.Fomod)), FomodAsk{}, false, nil
}

// replacing is the one entry holding a mod anywhere in the store item key, under any choice of its options.
func (s *Store) replacing(game, id, key string) (Entry, bool, error) {
	root, err := s.items.Path(game, key)
	if err != nil {
		return Entry{}, false, err
	}
	found, err := manifest.Scan(root)
	if err != nil {
		return Entry{}, false, err
	}
	p, err := s.read(game, id)
	if err != nil {
		return Entry{}, false, err
	}
	var held []Entry
	for _, e := range p.Entries {
		if slices.ContainsFunc(e.Mods, func(m Component) bool {
			return slices.ContainsFunc(found, func(f manifest.Mod) bool { return mod.Equal(f.ModID(), m.ID) })
		}) {
			held = append(held, e)
		}
	}
	if len(held) != 1 {
		return Entry{}, false, nil
	}
	return held[0], true, nil
}

// scanItem lays out a store item and scans it for mods; an item with none is a NoModError. done removes any temp
// layout and is always safe to defer.
func (s *Store) scanItem(game, id, key string, choices map[string]map[string][]string) (src string, found []manifest.Mod, done func(), err error) {
	src, tmp, err := s.layoutItem(game, id, key, choices)
	done = func() {
		if tmp != "" {
			_ = fsx.RemoveAll(tmp)
		}
	}
	if err != nil {
		return "", nil, done, err
	}
	if found, err = manifest.Scan(src); err != nil {
		return "", nil, done, err
	}
	if len(found) == 0 {
		return "", nil, done, &NoModError{Key: key}
	}
	return src, found, done, nil
}

func cloneFomod(in map[string]map[string][]string) map[string]map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string]map[string][]string, len(in))
	for k, g := range in {
		ng := make(map[string][]string, len(g))
		for gk, v := range g {
			ng[gk] = append([]string(nil), v...)
		}
		out[k] = ng
	}
	return out
}

// FomodPreview is the wizard at the given choices, for the store item key.
func (s *Store) FomodPreview(game, id, key string, choices map[string]map[string][]string) (FomodAsk, error) {
	p, err := s.read(game, id)
	if err != nil {
		return FomodAsk{}, err
	}
	var src Source
	held := false
	for _, e := range p.Entries {
		if e.Key == key {
			src, held = e.Source, true
			if choices == nil {
				choices = e.Fomod
			}
			break
		}
	}
	if !held && len(choices) == 0 {
		if _, ask, need, err := s.installAsk(game, id, key, src); err != nil || need {
			return ask, err
		}
	}
	ask, _, err := s.fomodAsk(game, id, key, src, "", choices)
	if err != nil {
		return FomodAsk{}, err
	}
	if ask.Key == "" {
		cfg, ok, err := s.fomodOf(game, key)
		if err != nil || !ok {
			return FomodAsk{}, err
		}
		modsDir, err := s.ModsDir(game, id)
		if err != nil {
			return FomodAsk{}, err
		}
		ask = askFrom(cfg, key, src, "", choices, s.fomodEval(game, s.fileIndex(modsDir)))
	}
	return ask, nil
}

// InstallFomod adds or replaces the store item using FOMOD choices. An entry that already has this key is rewritten
// in place so Reinstall with options can run.
func (s *Store) InstallFomod(game, id, key string, source Source, choices map[string]map[string][]string) (InstallResult, error) {
	p, err := s.read(game, id)
	if err != nil {
		return InstallResult{}, err
	}
	src := source.WithFomod(choices)
	for _, e := range p.Entries {
		if e.Key == key {
			out, err := s.applyFomod(game, id, key, choices)
			if err != nil {
				return InstallResult{}, installError(err)
			}
			res := InstallResult{Profile: out, Added: addedNames(out, key), Updated: true}
			if err := s.RecordModsSnapshot(game, id); err != nil {
				return InstallResult{}, err
			}
			return res, nil
		}
	}
	return s.installKey(game, id, key, src)
}

func (s *Store) applyFomod(game, id, key string, choices map[string]map[string][]string) (Profile, error) {
	return s.updateMods(game, id, func(p *Profile, dir string) error {
		ei, err := requireEntry(p.Entries, key)
		if err != nil {
			return err
		}
		_, found, done, err := s.scanItem(game, id, key, choices)
		defer done()
		if err != nil {
			return err
		}
		e := p.Entries[ei]
		e.Fomod = cloneFomod(choices)
		e.Mods = entryMods(found)
		disabled := e.Disabled
		e.Disabled = []mod.ID{}
		for _, m := range e.Mods {
			if hasID(disabled, m.ID) {
				e.Disabled = append(e.Disabled, m.ID)
			}
		}
		modsDir := filepath.Join(dir, "mods")
		if err := removeEntryFolders(modsDir, key); err != nil {
			return err
		}
		if err := s.place(game, modsDir, e); err != nil {
			return err
		}
		p.Entries[ei] = e
		return nil
	})
}

// FomodImage returns a file from the store item; rel must stay inside it.
func (s *Store) FomodImage(game, key, rel string) ([]byte, error) {
	root, err := s.items.Path(game, key)
	if err != nil {
		return nil, err
	}
	rel = filepath.FromSlash(strings.ReplaceAll(rel, "\\", "/"))
	if rel == "" || !filepath.IsLocal(rel) {
		return nil, fmt.Errorf("image path leaves the item")
	}
	p := filepath.Join(root, rel)
	info, err := fsx.Stat(p)
	if err != nil || !info.Mode().IsRegular() {
		return nil, store.ErrNotFound
	}
	return fsx.ReadFile(p)
}
