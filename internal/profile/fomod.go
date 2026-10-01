package profile

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fomod"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/store"
)

// FomodAsk is a FOMOD wizard the window shows before files land in the profile.
type FomodAsk struct {
	Key        string                         `json:"key"`
	Source     Source                         `json:"source"`
	OldKey     string                         `json:"oldKey,omitempty"`
	ModuleName string                         `json:"moduleName"`
	Steps      []FomodStep                    `json:"steps"`
	Choices    map[string]map[string][]string `json:"choices,omitempty"`
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
	g := game.Find(gameID)
	if g == nil {
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
	return g.LoaderStatus(dir, sett.Get().Loaders[gameID]).GameVersion
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
	if fomod.Match(cfg, choices, eval) {
		return FomodAsk{}, false, nil
	}
	return askFrom(cfg, key, source, oldKey, choices, eval), true, nil
}

func (s *Store) layoutItem(game, id, key string, choices map[string]map[string][]string) (src, tmp string, err error) {
	root, err := s.items.Path(game, key)
	if err != nil {
		return "", "", err
	}
	cfg, _, ok, err := fomod.Open(root)
	if err != nil || !ok {
		return root, "", err
	}
	modsDir, err := s.ModsDir(game, id)
	if err != nil {
		return "", "", err
	}
	eval := s.fomodEval(game, s.fileIndex(modsDir))
	if !fomod.Match(cfg, choices, eval) {
		return "", "", &NeedChoicesError{Ask: askFrom(cfg, key, Source{}, "", choices, eval)}
	}
	tmp, err = os.MkdirTemp("", "mortar-fomod-")
	if err != nil {
		return "", "", err
	}
	if err := fomod.Apply(root, tmp, fomod.Resolve(cfg, choices, eval)); err != nil {
		_ = os.RemoveAll(tmp)
		return "", "", err
	}
	return tmp, tmp, nil
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
	for _, e := range p.Entries {
		if e.Key == key {
			src = e.Source
			if choices == nil {
				choices = e.Fomod
			}
			break
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
			res := InstallResult{Profile: out, Added: []string{}, Updated: true}
			for _, e := range out.Entries {
				if e.Key == key {
					for _, m := range e.Mods {
						res.Added = append(res.Added, m.Name)
					}
				}
			}
			return res, nil
		}
	}
	return s.installKey(game, id, key, src)
}

func (s *Store) applyFomod(game, id, key string, choices map[string]map[string][]string) (Profile, error) {
	return s.updateMods(game, id, func(p *Profile, dir string) error {
		ei := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == key })
		if ei < 0 {
			return fmt.Errorf("%q is not in this profile", key)
		}
		src, tmp, err := s.layoutItem(game, id, key, choices)
		if tmp != "" {
			defer func() { _ = os.RemoveAll(tmp) }()
		}
		if err != nil {
			return err
		}
		found, err := manifest.Scan(src)
		if err != nil {
			return err
		}
		if len(found) == 0 {
			return &NoModError{Key: key}
		}
		e := p.Entries[ei]
		e.Fomod = cloneFomod(choices)
		e.Mods = entryMods(found)
		disabled := e.Disabled
		e.Disabled = []string{}
		for _, m := range e.Mods {
			if hasID(disabled, m.UniqueID) {
				e.Disabled = append(e.Disabled, m.UniqueID)
			}
		}
		modsDir := filepath.Join(dir, "mods")
		for _, name := range []string{key, "." + key} {
			if err := os.RemoveAll(filepath.Join(modsDir, name)); err != nil {
				return err
			}
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
