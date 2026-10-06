package configsvc

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

const (
	jsonName = "config.json"
	cfgExt   = ".cfg"
)

// Profiles is the part of the profile store configsvc reads through.
type Profiles interface {
	ProfileDir(game, id string) (string, error)
	ModFolder(game, id, key string, uniqueID mod.ID) (string, error)
	ReadConfig(game, id, key string, uniqueID mod.ID) (string, error)
	ShippedConfig(game, id, key string, uniqueID mod.ID) (string, bool)
	PluginGUIDs(game, id string, uniqueID mod.ID) []string
	SeedConfigs(game, id string) error
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
}

func (s *Service) cfgDir(game, id string) (string, error) {
	dir, err := s.Profiles.ProfileDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "BepInEx", "config"), nil
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
	for _, e := range ents {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), cfgExt) {
			continue
		}
		raw, err := fsx.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		doc := parseCfg(string(raw))
		stem := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if modID != "" && !slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(stem, n) || strings.EqualFold(doc.guid, n) }) {
			continue
		}
		label := doc.plugin
		if label == "" {
			label = e.Name()
		}
		out = append(out, ConfigFile{Name: e.Name(), Format: FormatBepInEx, Label: label})
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
	return jsonSchema(f, cur, shipped, capture)
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
