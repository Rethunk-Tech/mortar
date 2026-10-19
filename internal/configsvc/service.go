package configsvc

import (
	"bytes"
	"cmp"
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
	"unicode"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/jsonc"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/modconfig"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/usererr"

	"gopkg.in/yaml.v3"
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
	InstalledPluginGUIDs(game, id string) []string
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
	SetJSON func(game, profile string, uniqueID mod.ID, field, value string) error
	// SetGmcm sets one option of a mod's in-game menu for the game's next start.
	SetGmcm func(game, profile string, uniqueID mod.ID, page string, index int, value string) error
	// Running reports whether the profile's game is running; writes to its files are refused then.
	Running func(game, profile string) bool

	changedMu sync.Mutex
	changed   map[changedKey]bool
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
	info, ok := components.Game(game)
	if _, has := info.Paths[profile.ModConfigRole]; !ok || !has {
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
	ents, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
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
		return "", fmt.Errorf("%q is not a config file name", name)
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

// cfgPath is a .cfg inside the profile's BepInEx config folder; name must be a bare file name.
func (s *Service) cfgPath(game, id, name string) (string, error) {
	if name != filepath.Base(name) || !strings.EqualFold(filepath.Ext(name), cfgExt) {
		return "", fmt.Errorf("%q is not a config file name", name)
	}
	dir, err := s.cfgDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
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
	return append(out, s.melonFile(game, profileID, modID, names)...), nil
}

// cfgOnDisk is one .cfg of the profile's BepInEx config folder, read once, one JSON file of its mod config folder, or
// one other text file of the BepInEx config folder (text), whose name is its slash path there.
type cfgOnDisk struct {
	name string
	stem string
	doc  cfgDoc
	json bool
	text bool
	// folder is the subfolder a text file sits in, "" at the top: mods put their files in one named for the package.
	folder string
}

// minConfigName is the shortest mod id a JSON file name is matched against by containment; a shorter one would
// claim unrelated files.
const minConfigName = 3

func (d cfgOnDisk) ownedBy(names []string) bool {
	if d.text {
		return slices.ContainsFunc(names, func(n string) bool {
			// A Thunderstore id is source:Namespace-Name; the package's folder is Namespace-Name, its file the Name.
			_, pkg, _ := strings.Cut(n, ":")
			_, name, _ := strings.Cut(pkg, "-")
			for _, k := range []string{squash(n), squash(pkg), squash(name)} {
				if len(k) >= minConfigName && (squash(d.folder) == k || strings.Contains(squash(d.stem), k)) {
					return true
				}
			}
			return false
		})
	}
	if d.json {
		// Mods name their file after their mod id in their own spelling ("CarryOnConfig.json" for carryon).
		stem := squash(d.stem)
		return slices.ContainsFunc(names, func(n string) bool {
			n = squash(n)
			return len(n) >= minConfigName && strings.Contains(stem, n)
		})
	}
	return slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(d.stem, n) || strings.EqualFold(d.doc.guid, n) })
}

// squash lower-cases s and drops everything but letters and digits.
func squash(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

func (d cfgOnDisk) file() ConfigFile {
	if d.text {
		return ConfigFile{Name: d.name, Format: FormatText, Label: filepath.Base(filepath.FromSlash(d.name))}
	}
	if d.json {
		return ConfigFile{Name: d.name, Format: FormatJSON, Label: d.name}
	}
	label := d.doc.plugin
	label = cmp.Or(label, d.name)
	return ConfigFile{Name: d.name, Format: FormatBepInEx, Label: label}
}

// changed reports whether a .cfg differs from its defaults; a mod's JSON file names none.
func (d cfgOnDisk) changed() bool {
	return !d.json && !d.text && schemaChanged(d.doc.schema(d.file()))
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
		ents, err := os.ReadDir(filepath.Join(dir, folder))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
		for _, e := range ents {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if e.IsDir() || !slices.Contains(textExts, ext) {
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
	ents, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
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
		return "", fmt.Errorf("%q is not a config file name", name)
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
		return "", fmt.Errorf("%s is %d KB, too big to edit here; open its folder instead", file, st.Size()>>10)
	}
	raw, err := fsx.ReadFile(p)
	return string(bytes.TrimPrefix(raw, utf8BOM)), err
}

// utf8BOM starts many mods' XML and JSON files; Text leaves it out, so an edit at the top cannot push it inside the
// text, and SetText puts it back.
var utf8BOM = []byte("\xef\xbb\xbf")

// SetText replaces one of a mod's text config files, atomically.
func (s *Service) SetText(game, profileID, modID, file, text string) error {
	if err := s.refuseRunning(game, profileID); err != nil {
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
	if old, err := fsx.ReadFile(p); err == nil && bytes.HasPrefix(old, utf8BOM) {
		out = append(slices.Clone(utf8BOM), out...)
	}
	return datadir.WriteFile(p, out, 0o600)
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
		clean := jsonc.Clean([]byte(text))
		var v any
		if err := json.Unmarshal(clean, &v); err != nil {
			if se, ok := errors.AsType[*json.SyntaxError](err); ok {
				return fmt.Errorf("%s is not valid JSON at line %d: %w", file, 1+bytes.Count(clean[:se.Offset], []byte("\n")), err)
			}
			return fmt.Errorf("%s is not valid JSON: %w", file, err)
		}
	case ".yaml", ".yml":
		var v any
		if err := yaml.Unmarshal([]byte(text), &v); err != nil {
			return fmt.Errorf("%s is not valid YAML: %w", file, err)
		}
	case ".xml":
		d := xml.NewDecoder(strings.NewReader(text))
		for {
			_, err := d.Token()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return fmt.Errorf("%s is not valid XML: %w", file, err)
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
		return "", fmt.Errorf("%s is not edited as text", file)
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
	ents, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	var out []cfgOnDisk
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), cfgExt) {
			continue
		}
		raw, err := fsx.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		out = append(out, cfgOnDisk{name: e.Name(), stem: strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())), doc: parseCfg(string(raw))})
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
	return ConfigFile{}, fmt.Errorf("%s has no config file %q in this profile", modID, name)
}

// Schema reads one file as typed sections and entries.
func (s *Service) Schema(game, profileID, modID, file string) (Schema, error) {
	f, err := s.fileOf(game, profileID, modID, file)
	if err != nil {
		return Schema{}, err
	}
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
		raw, err := fsx.ReadFile(path)
		if err != nil {
			return Schema{}, err
		}
		return jsonSchema(f, string(raw), "", nil, nil)
	}
	if f.Format == FormatBepInEx {
		path, err := s.cfgPath(game, profileID, f.Name)
		if err != nil {
			return Schema{}, err
		}
		raw, err := fsx.ReadFile(path)
		if err != nil {
			return Schema{}, err
		}
		return parseCfg(string(raw)).schema(f), nil
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

func (s *Service) refuseRunning(game, profileID string) error {
	if s.Running != nil && s.Running(game, profileID) {
		return &profile.RunningError{Game: game}
	}
	return nil
}

// Set changes one value. The file is written atomically with every other line, comment and key as it was.
func (s *Service) Set(game, profileID, modID, file, section, key, value string) error {
	if err := s.refuseRunning(game, profileID); err != nil {
		return err
	}
	f, err := s.fileOf(game, profileID, modID, file)
	if err != nil {
		return err
	}
	return s.write(game, profileID, modID, f, []edit{{section, key, value}})
}

// Reset puts one setting back to its default.
func (s *Service) Reset(game, profileID, modID, file, section, key string) error {
	skipped, err := s.reset(game, profileID, modID, file, func(sec, k string) bool { return sec == section && k == key })
	if err == nil && skipped > 0 {
		return fmt.Errorf("%s has no default to go back to", key)
	}
	return err
}

// ResetAll puts every setting that has a default back to it and leaves the others as they are; it returns how many
// settings have no default.
func (s *Service) ResetAll(game, profileID, modID, file string) (int, error) {
	return s.reset(game, profileID, modID, file, func(string, string) bool { return true })
}

func (s *Service) reset(game, profileID, modID, file string, pick func(section, key string) bool) (int, error) {
	if err := s.refuseRunning(game, profileID); err != nil {
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
	return skipped, s.write(game, profileID, modID, schema.File, edits)
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
		for _, e := range edits {
			if err := s.SetJSON(game, profileID, mod.ID(modID), path(e.section, e.key), e.value); err != nil {
				return err
			}
		}
		return nil
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
	raw, err := fsx.ReadFile(p)
	if err != nil {
		return err
	}
	text := string(raw)
	for _, e := range edits {
		if text, err = parseCfg(text).set(e.section, e.key, e.value); err != nil {
			return err
		}
	}
	return datadir.WriteFile(p, []byte(text), 0o600)
}

// writeJSON applies edits to one JSON file of the profile's mod config folder, keeping its key order.
func (s *Service) writeJSON(game, profileID, name string, edits []edit) error {
	p, err := s.jsonPath(game, profileID, name)
	if err != nil {
		return err
	}
	raw, err := fsx.ReadFile(p)
	if err != nil {
		return err
	}
	for _, e := range edits {
		if raw, err = profile.SetJSONValue(raw, path(e.section, e.key), e.value); err != nil {
			return err
		}
	}
	return datadir.WriteFile(p, raw, 0o600)
}
