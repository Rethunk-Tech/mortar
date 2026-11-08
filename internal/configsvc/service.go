package configsvc

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-cfgedit"
	"github.com/Rethunk-Tech/go-lenientjson"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/bytefmt"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	gamepkg "github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/modconfig"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/usererr"

	"go.yaml.in/yaml/v3"
)

const (
	jsonName = "config.json"
	cfgExt   = ".cfg"
	jsonExt  = ".json"
)

// Profiles is the part of the profile store configsvc reads through.
type Profiles interface {
	ProfileDir(game, id string) (string, error)
	ModFolder(game, id, key string, uniqueID mod.ID) (string, error)
	ReadConfig(game, id, key string, uniqueID mod.ID) (string, error)
	ShippedConfig(game, id, key string, uniqueID mod.ID) (string, bool)
	ContentSchema(game, id, key string, uniqueID mod.ID) (modconfig.Schema, error)
	PluginGUIDs(game, id string, uniqueID mod.ID) []string
	PluginDLLs(game, id string) map[string]profile.PluginDLL
	ConfigView(game, id string) (profile.ConfigView, error)
	SeedConfigs(game, id string) error
	UserMods(game, id string) ([]profile.Mod, error)
}

// Service edits the config files a profile holds. A modID is a SMAPI UniqueID, a package's id (whose plugins' GUIDs
// name its files), or a BepInEx plugin GUID (or its .cfg file name without the extension); an empty modID lists every
// .cfg of the profile.
type Service struct {
	Profiles Profiles
	// SetJSON writes one config.json value through the profile, which keeps its key order and checks the game is
	// not running.
	SetJSON func(game, profile string, uniqueID mod.ID, sets []profile.ConfigSet) error
	// SetGmcm sets one option of a mod's in-game menu for the game's next start.
	SetGmcm func(game, profile string, uniqueID mod.ID, page string, index int, value string) error
	// Running reports whether the profile's game is launching or running.
	Running func(game, profile string) bool
	// Starting reports whether a launch of the profile is placing its files or waiting for the game to come up; edits
	// are refused then.
	Starting func(game, profile string) bool

	changedMu sync.Mutex
	changed   map[changedKey]bool

	// parsed keeps each .cfg file's parse while its size and time are unchanged: a search and every page of the config
	// view read every file of the profile again.
	parsedMu sync.Mutex
	// parsedSweep is the size of parsed after its last sweep of deleted files.
	parsedSweep sweepMark
	parsed      map[string]parsedCfg

	// hostKept is the host's values a player chose to keep, written into the .cfg once the game is closed.
	hostMu   sync.Mutex
	hostKept map[hostKey][]edit

	dlls dllCache
}

type hostKey struct{ game, profile, mod, file string }

type parsedCfg struct {
	size int64
	at   time.Time
	doc  *cfgedit.BepInEx
}

// parseCached is the parse of the .cfg file at path, read again only when it changed since the last.
func (s *Service) parseCached(path string, info fs.FileInfo) (*cfgedit.BepInEx, error) {
	s.parsedMu.Lock()
	hit, ok := s.parsed[path]
	s.parsedMu.Unlock()
	if ok && hit.size == info.Size() && hit.at.Equal(info.ModTime()) {
		return hit.doc, nil
	}
	raw, err := safefs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	doc := cfgedit.ParseBepInEx(string(raw))
	s.parsedMu.Lock()
	if s.parsed == nil {
		s.parsed = map[string]parsedCfg{}
	}
	dropGone(s.parsed, &s.parsedSweep)
	s.parsed[path] = parsedCfg{size: info.Size(), at: info.ModTime(), doc: doc}
	s.parsedMu.Unlock()
	return doc, nil
}

// cacheLimit is how many entries a path-keyed cache holds before it drops those whose file is gone.
const cacheLimit = 256

// sweepGrowth is how many entries a cache must gain after a sweep before the next one, so a cache of live files is
// not statted on every insert.
const sweepGrowth = 64

// sweepMark is the size a cache had after its last sweep.
type sweepMark struct{ size int }

// dropGone removes the entries of m whose file no longer exists, once m has grown past cacheLimit and by sweepGrowth
// since the last sweep; it reports whether it swept.
func dropGone[V any](m map[string]V, mark *sweepMark) bool {
	if len(m) < cacheLimit || len(m) < mark.size+sweepGrowth {
		return false
	}
	defer func() { mark.size = len(m) }()
	for p := range m {
		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			delete(m, p)
		}
	}
	return true
}

func (s *Service) cfgDir(game, id string) (string, error) {
	dir, err := s.Profiles.ProfileDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "BepInEx", "config"), nil
}

// modConfigDir is the profile's own copy of the game's mod config folder (the catalog's modConfig path), "" for a game
// whose mods keep their settings elsewhere.
func (s *Service) modConfigDir(game, id string) (string, error) {
	if !gamepkg.HasPath(game, profile.ModConfigRole) {
		return "", nil
	}
	dir, err := s.Profiles.ProfileDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, profile.ModConfigFolder), nil
}

// jsonDocs lists the JSON files of the profile's mod config folder; none for a game without one.
func (s *Service) jsonDocs(game, profileID string) ([]cfgOnDisk, error) {
	dir, err := s.modConfigDir(game, profileID)
	if err != nil || dir == "" {
		return nil, err
	}
	ents, _, err := safefs.ReadDirIfExists(dir)
	if err != nil {
		return nil, err
	}
	var out []cfgOnDisk
	for _, e := range ents {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), jsonExt) {
			out = append(out, cfgOnDisk{name: e.Name(), stem: strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())), json: true})
		}
	}
	return out, nil
}

// jsonPath is a file of the profile's mod config folder; name must be a bare .json file name.
func (s *Service) jsonPath(game, id, name string) (string, error) {
	if name != filepath.Base(name) || !strings.EqualFold(filepath.Ext(name), jsonExt) {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a config file name", name))
	}
	dir, err := s.modConfigDir(game, id)
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", errors.New("this game keeps no mod config folder")
	}
	return filepath.Join(dir, name), nil
}

// cfgPath is a .cfg inside the profile's BepInEx config folder, or the copy set aside while its mod is switched off;
// name must be a bare file name.
func (s *Service) cfgPath(game, id, name string) (string, error) {
	if name != filepath.Base(name) || !strings.EqualFold(filepath.Ext(name), cfgExt) {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a config file name", name))
	}
	dir, err := s.cfgDir(game, id)
	if err != nil {
		return "", err
	}
	live := filepath.Join(dir, name)
	if _, err := os.Stat(live); err != nil {
		aside := filepath.Join(dir, "..", "..", profile.ConfigAsideDir, name)
		if _, err := os.Stat(aside); err == nil {
			return aside, nil
		}
	}
	return live, nil
}

// Files lists the mod's config files in the profile.
func (s *Service) Files(game, profileID, modID string) ([]ConfigFile, error) {
	out := []ConfigFile{}
	names := []string{modID}
	if modID != "" {
		names = append(names, s.Profiles.PluginGUIDs(game, profileID, mod.ID(modID))...)
		if folder, err := s.Profiles.ModFolder(game, profileID, "", mod.ID(modID)); err == nil {
			if _, err := os.Stat(filepath.Join(folder, jsonName)); err == nil {
				out = append(out, ConfigFile{Name: jsonName, Format: FormatSMAPI, Label: jsonName})
			}
		}
		if dir, err := s.Profiles.ProfileDir(game, profileID); err == nil {
			if _, err := os.Stat(gmcm.CapturePath(dir, mod.ID(modID))); err == nil {
				out = append(out, ConfigFile{Name: gmcmFileName, Format: FormatGMCM, Label: gmcmFileName})
			}
		}
	}
	docs, err := s.allDocs(game, profileID)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if modID != "" && !d.ownedBy(names) {
			continue
		}
		out = append(out, d.file())
	}
	if modID != "" {
		out = append(out, s.unwritten(game, profileID, names[1:], docs)...)
	}
	return append(out, s.melonFile(game, profileID, modID, names)...), nil
}

// cfgOnDisk is one .cfg of the profile's BepInEx config folder, read once, one JSON file of its mod config folder, or
// one other text file of the BepInEx config folder (text), whose name is its slash path there.
type cfgOnDisk struct {
	name string
	stem string
	doc  *cfgedit.BepInEx
	json bool
	text bool
	// folder is the subfolder a text file sits in, "" at the top: mods put their files in one named for the package.
	folder string
}

// minConfigName is the shortest mod id a JSON file name is matched against by containment; a shorter one would
// claim unrelated files.
const minConfigName = 3

// packageNames is the package ("Namespace-Name") and its name of a "source:Namespace-Name" id; a plugin GUID, which
// names no source, has neither.
func packageNames(n string) (pkg, name string) {
	id := mod.ID(n)
	if id.Format() == "" {
		return "", ""
	}
	pkg = id.Local()
	_, name, _ = strings.Cut(pkg, "-")
	return pkg, name
}

func (d cfgOnDisk) ownedBy(names []string) bool {
	if d.text {
		folder, stem := mod.LooseName(d.folder), mod.LooseName(d.stem)
		return slices.ContainsFunc(names, func(n string) bool {
			// A Thunderstore id is source:Namespace-Name; the package's folder is Namespace-Name, its file the Name.
			pkg, name := packageNames(n)
			for _, k := range []string{mod.LooseName(n), mod.LooseName(pkg), mod.LooseName(name)} {
				if len(k) >= minConfigName && (folder == k || strings.Contains(stem, k)) {
					return true
				}
			}
			return false
		})
	}
	if d.json {
		// Mods name their file after their mod id in their own spelling ("CarryOnConfig.json" for carryon).
		stem := mod.LooseName(d.stem)
		return slices.ContainsFunc(names, func(n string) bool {
			n = mod.LooseName(n)
			return len(n) >= minConfigName && strings.Contains(stem, n)
		})
	}
	return slices.ContainsFunc(names, func(n string) bool {
		if strings.EqualFold(d.stem, n) || strings.EqualFold(d.doc.GUID(), n) {
			return true
		}
		// A ConfigFile a plugin builds itself has no GUID header and is often named after the package
		// (LethalLevelLoader.cfg for thunderstore:IAmBatby-LethalLevelLoader).
		_, name := packageNames(n)
		return d.doc.GUID() == "" && name != "" && strings.EqualFold(d.stem, name)
	})
}

func (d cfgOnDisk) file() ConfigFile {
	if d.text {
		return ConfigFile{Name: d.name, Format: FormatText, Label: filepath.Base(filepath.FromSlash(d.name))}
	}
	if d.json {
		return ConfigFile{Name: d.name, Format: FormatJSON, Label: d.name}
	}
	label := d.doc.Plugin()
	label = cmp.Or(label, d.name)
	return ConfigFile{Name: d.name, Format: FormatBepInEx, Label: label}
}

// docChanged reports whether a .cfg differs from its defaults, as its plugin's DLL binds them;
// a mod's JSON file names none.
func (s *Service) docChanged(game, profileID string, d cfgOnDisk) bool {
	if d.json || d.text {
		return false
	}
	return schemaChanged(s.withBound(game, profileID, cfgSchema(d.doc, d.file()), guidOf(d.doc, d.name)))
}

// allDocs is the profile's .cfg files and the JSON files of its mod config folder.
func (s *Service) allDocs(game, profileID string) ([]cfgOnDisk, error) {
	docs, err := s.cfgDocs(game, profileID)
	if err != nil {
		return nil, err
	}
	more, err := s.jsonDocs(game, profileID)
	if err != nil {
		return nil, err
	}
	text, err := s.textDocs(game, profileID)
	return append(append(docs, more...), text...), err
}

// textExts are the file types listed as text files: what mods write beside their .cfg in BepInEx's config folder.
var textExts = []string{".json", ".yaml", ".yml", ".toml", ".xml", ".ini", ".txt"}

// syncSidecar ends the file ConfigSync writes beside a mod's .cfg to say which settings the host shares: Mortar's to
// read, never a setting to edit.
const syncSidecar = ".sync.json"

// maxTextFile bounds a text file opened for editing; a bigger one is data the mod generates, not settings.
const maxTextFile = 1 << 20

// textDocs lists the text files of the profile's BepInEx config folder and of the folders one level below it.
func (s *Service) textDocs(game, profileID string) ([]cfgOnDisk, error) {
	dir, err := s.cfgDir(game, profileID)
	if err != nil {
		return nil, err
	}
	var out []cfgOnDisk
	add := func(folder string) error {
		ents, found, err := safefs.ReadDirIfExists(filepath.Join(dir, folder))
		if err != nil {
			return err
		}
		if !found {
			return nil
		}
		for _, e := range ents {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if e.IsDir() || !slices.Contains(textExts, ext) || strings.HasSuffix(strings.ToLower(e.Name()), syncSidecar) {
				continue
			}
			name := e.Name()
			if folder != "" {
				name = folder + "/" + name
			}
			out = append(out, cfgOnDisk{name: name, stem: strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())), text: true, folder: folder})
		}
		return nil
	}
	if err := add(""); err != nil {
		return nil, err
	}
	ents, _, err := safefs.ReadDirIfExists(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range ents {
		if e.IsDir() {
			if err := add(e.Name()); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// textPath is a text file of the profile's BepInEx config folder, at most one folder down; name is its slash path.
func (s *Service) textPath(game, id, name string) (string, error) {
	rel := filepath.FromSlash(name)
	if !filepath.IsLocal(rel) || strings.Count(name, "/") > 1 || !slices.Contains(textExts, strings.ToLower(filepath.Ext(name))) {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%q is not a config file name", name))
	}
	dir, err := s.cfgDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, rel), nil
}

// Text reads one of a mod's text config files whole.
func (s *Service) Text(game, profileID, modID, file string) (string, error) {
	p, err := s.textFile(game, profileID, modID, file)
	if err != nil {
		return "", err
	}
	if st, err := os.Stat(p); err != nil {
		return "", err
	} else if st.Size() > maxTextFile {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%s is %s, too big to edit here; open its folder instead", file, bytefmt.Format(st.Size())))
	}
	raw, err := safefs.ReadFile(p)
	return string(mod.StripBOM(raw)), err
}

// SetText replaces one of a mod's text config files, atomically.
func (s *Service) SetText(game, profileID, modID, file, text string) error {
	if err := s.refuseEdit(game, profileID); err != nil {
		return err
	}
	p, err := s.textFile(game, profileID, modID, file)
	if err != nil {
		return err
	}
	if err := checkSyntax(file, text); err != nil {
		return usererr.Wrap(usererr.Invalid, err)
	}
	out := []byte(text)
	if old, err := safefs.ReadFile(p); err == nil && bytes.HasPrefix(old, []byte(mod.BOM)) {
		out = append([]byte(mod.BOM), out...)
	}
	return safefs.WriteFileAtomic(p, out, 0o600)
}

// OpenText opens one of a mod's text config files in the system's editor for it.
func (s *Service) OpenText(game, profileID, modID, file string) error {
	p, err := s.textFile(game, profileID, modID, file)
	if err != nil {
		return err
	}
	return datadir.Open(p)
}

// checkSyntax refuses JSON (as Newtonsoft reads it, comments and trailing commas allowed), YAML and XML that does not
// parse, naming the line; other text is the mod's to read.
func checkSyntax(file, text string) error {
	switch strings.ToLower(filepath.Ext(file)) {
	case jsonExt:
		clean := lenientjson.Clean([]byte(text))
		var v any
		if err := json.Unmarshal(clean, &v); err != nil {
			if se, ok := errors.AsType[*json.SyntaxError](err); ok {
				return usererr.Wrap(usererr.Invalid, fmt.Errorf("%s is not valid JSON at line %d: %w", file, 1+bytes.Count(clean[:se.Offset], []byte("\n")), err))
			}
			return usererr.Wrap(usererr.Invalid, fmt.Errorf("%s is not valid JSON: %w", file, err))
		}
	case ".yaml", ".yml":
		var v any
		if err := yaml.Unmarshal([]byte(text), &v); err != nil {
			return usererr.Wrap(usererr.Invalid, fmt.Errorf("%s is not valid YAML: %w", file, err))
		}
	case ".xml":
		d := xml.NewDecoder(strings.NewReader(text))
		for {
			_, err := d.Token()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return usererr.Wrap(usererr.Invalid, fmt.Errorf("%s is not valid XML: %w", file, err))
			}
		}
	}
	return nil
}

func (s *Service) textFile(game, profileID, modID, file string) (string, error) {
	f, err := s.fileOf(game, profileID, modID, file)
	if err != nil {
		return "", err
	}
	if f.Format != FormatText {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("%s is not edited as text", file))
	}
	return s.textPath(game, profileID, f.Name)
}

// cfgDocs reads every .cfg of the profile's BepInEx config folder, after seeding the shipped ones.
func (s *Service) cfgDocs(game, profileID string) ([]cfgOnDisk, error) {
	dir, err := s.cfgDir(game, profileID)
	if err != nil {
		return nil, err
	}
	if err := s.Profiles.SeedConfigs(game, profileID); err != nil {
		return nil, err
	}
	ents, _, err := safefs.ReadDirIfExists(dir)
	if err != nil {
		return nil, err
	}
	// A switched-off mod's settings wait in config-aside, so they stay editable.
	aside, _ := os.ReadDir(filepath.Join(dir, "..", "..", profile.ConfigAsideDir))
	var out []cfgOnDisk
	seen := map[string]bool{}
	for _, e := range append(ents, aside...) {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), cfgExt) || seen[e.Name()] {
			continue
		}
		seen[e.Name()] = true
		info, err := e.Info()
		if err != nil {
			continue
		}
		path, err := s.cfgPath(game, profileID, e.Name())
		if err != nil {
			continue
		}
		doc, err := s.parseCached(path, info)
		if err != nil {
			continue
		}
		out = append(out, cfgOnDisk{name: e.Name(), stem: strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())), doc: doc})
	}
	return out, nil
}

func (s *Service) fileOf(game, profileID, modID, name string) (ConfigFile, error) {
	files, err := s.Files(game, profileID, modID)
	if err != nil {
		return ConfigFile{}, err
	}
	for _, f := range files {
		if f.Name == name {
			return f, nil
		}
	}
	return ConfigFile{}, usererr.Wrap(usererr.NotFound, fmt.Errorf("%s has no config file %q in this profile", modID, name))
}

// Schema reads one file as typed sections and entries.
func (s *Service) Schema(game, profileID, modID, file string) (Schema, error) {
	f, err := s.fileOf(game, profileID, modID, file)
	if err != nil {
		return Schema{}, err
	}
	return s.schemaOf(game, profileID, modID, f)
}

// schemaOf reads a file the caller already found among the profile's config files.
func (s *Service) schemaOf(game, profileID, modID string, f ConfigFile) (Schema, error) {
	if f.Format == FormatGMCM {
		return s.gmcmSchema(game, profileID, mod.ID(modID), f)
	}
	if f.Format == FormatText {
		return Schema{File: f, Sections: []Section{}}, nil
	}
	if f.Format == FormatMelon {
		return s.melonSchema(game, profileID, modID, f)
	}
	if f.Format == FormatJSON {
		path, err := s.jsonPath(game, profileID, f.Name)
		if err != nil {
			return Schema{}, err
		}
		raw, err := safefs.ReadFile(path)
		if err != nil {
			return Schema{}, err
		}
		return jsonSchema(f, string(raw), "", nil, nil)
	}
	if f.Format == FormatBepInEx {
		return s.bepinexSchema(game, profileID, modID, f)
	}
	cur, err := s.Profiles.ReadConfig(game, profileID, "", mod.ID(modID))
	if err != nil {
		return Schema{}, err
	}
	shipped, _ := s.Profiles.ShippedConfig(game, profileID, "", mod.ID(modID))
	var capture *gmcm.Capture
	if dir, err := s.Profiles.ProfileDir(game, profileID); err == nil {
		if c, err := gmcm.ReadCapture(dir, mod.ID(modID)); err == nil {
			capture = &c
		}
	}
	cp, err := s.Profiles.ContentSchema(game, profileID, "", mod.ID(modID))
	if err != nil {
		return Schema{}, err
	}
	return jsonSchema(f, cur, shipped, capture, cp)
}

// bepinexSchema reads a BepInEx config file, or the defaults its plugin binds when the game has not written it yet.
func (s *Service) bepinexSchema(game, profileID, modID string, f ConfigFile) (Schema, error) {
	path, err := s.cfgPath(game, profileID, f.Name)
	if err != nil {
		return Schema{}, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		// A plugin the game has not run yet: its settings as its DLL binds them, at their defaults.
		if b, ok := s.boundFor(game, profileID, strings.TrimSuffix(f.Name, filepath.Ext(f.Name))); ok {
			return boundSync(cfgSchema(cfgedit.ParseBepInEx(b.render()), f), b), nil
		}
	}
	if err != nil {
		return Schema{}, err
	}
	if kept, err := s.writeKept(game, profileID, modID, f); err != nil {
		return Schema{}, err
	} else if kept {
		if info, err = os.Stat(path); err != nil {
			return Schema{}, err
		}
	}
	doc, err := s.parseCached(path, info)
	if err != nil {
		return Schema{}, err
	}
	return s.withBound(game, profileID, withSync(cfgSchema(doc, f), path), guidOf(doc, f.Name)), nil
}

func (s *Service) refuseRunning(game, profileID string) error {
	if s.Running != nil && s.Running(game, profileID) {
		return &profile.RunningError{Game: game}
	}
	return nil
}

// refuseEdit lets a running game's config files change: the mod reads them again at its next start, or sooner if it
// watches them. A launch still placing files refuses, and so does a running game whose mod config folder is swapped
// in, since a copied swap goes back over the profile's folder when the game exits and would drop the edit.
func (s *Service) refuseEdit(game, profileID string) error {
	if s.Starting != nil && s.Starting(game, profileID) {
		return &profile.RunningError{Game: game}
	}
	if gamepkg.HasPath(game, profile.ModConfigRole) {
		return s.refuseRunning(game, profileID)
	}
	return nil
}

// Set changes one value. The file is written atomically with every other line, comment and key as it was.
func (s *Service) Set(ctx context.Context, game, profileID, modID, file, section, key, value string) error {
	if err := s.refuseEdit(game, profileID); err != nil {
		return err
	}
	f, err := s.fileOf(game, profileID, modID, file)
	if err != nil {
		return err
	}
	if err := s.write(game, profileID, modID, f, []edit{{section, key, value}}); err != nil {
		return err
	}
	s.reloadInGame(ctx, game, profileID, f)
	return nil
}

// Reset puts one setting back to its default.
func (s *Service) Reset(ctx context.Context, game, profileID, modID, file, section, key string) error {
	skipped, err := s.reset(ctx, game, profileID, modID, file, func(sec, k string) bool { return sec == section && k == key })
	if err == nil && skipped > 0 {
		return fmt.Errorf("%s has no default to go back to", key)
	}
	return err
}

// ResetAll puts every setting that has a default back to it and leaves the others as they are; it returns how many
// settings have no default.
func (s *Service) ResetAll(ctx context.Context, game, profileID, modID, file string) (int, error) {
	return s.reset(ctx, game, profileID, modID, file, func(string, string) bool { return true })
}

func (s *Service) reset(ctx context.Context, game, profileID, modID, file string, pick func(section, key string) bool) (int, error) {
	if err := s.refuseEdit(game, profileID); err != nil {
		return 0, err
	}
	schema, err := s.Schema(game, profileID, modID, file)
	if err != nil {
		return 0, err
	}
	var edits []edit
	skipped := 0
	for _, sec := range schema.Sections {
		for _, e := range sec.Entries {
			if !pick(sec.Name, e.Key) {
				continue
			}
			if !e.HasDefault {
				skipped++
				continue
			}
			if e.Value != e.Default {
				edits = append(edits, edit{sec.Name, e.Key, e.Default})
			}
		}
	}
	if len(edits) == 0 {
		return skipped, nil
	}
	if err := s.write(game, profileID, modID, schema.File, edits); err != nil {
		return skipped, err
	}
	s.reloadInGame(ctx, game, profileID, schema.File)
	return skipped, nil
}

type edit struct{ section, key, value string }

func (s *Service) write(game, profileID, modID string, f ConfigFile, edits []edit) error {
	if f.Format == FormatGMCM {
		return s.writeGmcm(game, profileID, mod.ID(modID), edits)
	}
	if f.Format == FormatSMAPI {
		if s.SetJSON == nil {
			return errors.New("config.json edits are not wired")
		}
		sets := make([]profile.ConfigSet, len(edits))
		for i, e := range edits {
			sets[i] = profile.ConfigSet{Field: path(e.section, e.key), Value: e.value}
		}
		return s.SetJSON(game, profileID, mod.ID(modID), sets)
	}
	if f.Format == FormatJSON {
		return s.writeJSON(game, profileID, f.Name, edits)
	}
	if f.Format == FormatMelon {
		return s.writeMelon(game, profileID, edits)
	}
	p, err := s.cfgPath(game, profileID, f.Name)
	if err != nil {
		return err
	}
	b, hasBound := s.boundFor(game, profileID, strings.TrimSuffix(f.Name, filepath.Ext(f.Name)))
	raw, err := safefs.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) && hasBound {
		// The first edit of a plugin the game has not run yet writes the file BepInEx would, which it then keeps.
		raw, err = []byte(b.render()), nil
	}
	if err != nil {
		return err
	}
	text, err := applyBepInEx(string(raw), edits, b, hasBound)
	if err != nil {
		return err
	}
	return safefs.WriteFileAtomic(p, []byte(text), 0o600)
}

// applyBepInEx applies edits to a .cfg's text with one parse. A Set changes exactly one line of the parsed document, so
// each edit's line is copied from that result into the running lines; only a setting the file lacks (appended from the
// plugin's binding) shifts the lines and parses again.
func applyBepInEx(text string, edits []edit, b bound, hasBound bool) (string, error) {
	doc := cfgedit.ParseBepInEx(text)
	parsed := strings.Split(text, "\n")
	lines := slices.Clone(parsed)
	// touched is the line each setting's edit changed, so setting it back to its original value restores that line.
	touched := map[[2]string]int{}
	for _, e := range edits {
		next, err := doc.Set(e.section, e.key, e.value)
		if st, ok := b.setting(e.section, e.key); errors.Is(err, cfgedit.ErrNoSetting) && hasBound && ok {
			// A setting the plugin added since the game last wrote its file.
			text = strings.Join(lines, "\n") + fmt.Sprintf("\n[%s]\n", e.section) + entryText(st)
			doc = cfgedit.ParseBepInEx(text)
			parsed = strings.Split(text, "\n")
			lines = slices.Clone(parsed)
			touched = map[[2]string]int{}
			next, err = doc.Set(e.section, e.key, e.value)
		}
		if err != nil {
			return "", valueErr(err)
		}
		id := [2]string{e.section, e.key}
		changed := false
		for i, l := range strings.Split(next, "\n") {
			if l != parsed[i] {
				lines[i], touched[id], changed = l, i, true
				break
			}
		}
		if i, ok := touched[id]; ok && !changed {
			lines[i] = parsed[i]
		}
	}
	return strings.Join(lines, "\n"), nil
}

// writeJSON applies edits to one JSON file of the profile's mod config folder, keeping its key order.
func (s *Service) writeJSON(game, profileID, name string, edits []edit) error {
	p, err := s.jsonPath(game, profileID, name)
	if err != nil {
		return err
	}
	raw, err := safefs.ReadFile(p)
	if err != nil {
		return err
	}
	for _, e := range edits {
		if raw, err = profile.SetJSONValue(raw, path(e.section, e.key), e.value); err != nil {
			return err
		}
	}
	return safefs.WriteFileAtomic(p, raw, 0o600)
}
