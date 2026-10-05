package modconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

const (
	presetsDir     = "config-presets"
	maxPresetName  = 40
	maxPresetCount = 32
)

// ListPresets names saved configs for a mod, sorted.
func ListPresets(dataDir, game string, id mod.ID) ([]string, error) {
	dir, err := presetFolder(dataDir, game, id)
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(ents))
	for _, ent := range ents {
		n := ent.Name()
		if ent.IsDir() || !strings.HasSuffix(n, ".json") {
			continue
		}
		names = append(names, strings.TrimSuffix(n, ".json"))
	}
	slices.Sort(names)
	return names, nil
}

// SavePreset stores validated JSON as a named preset, at most maxPresetCount per mod.
func SavePreset(dataDir, game string, id mod.ID, name string, contents []byte) error {
	normalized, err := normalizePresetJSON(contents)
	if err != nil {
		return err
	}
	path, err := presetFile(dataDir, game, id, name)
	if err != nil {
		return err
	}
	names, err := ListPresets(dataDir, game, id)
	if err != nil {
		return err
	}
	if len(names) >= maxPresetCount && !slices.Contains(names, name) {
		return fmt.Errorf("at most %d presets per mod", maxPresetCount)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return datadir.WriteFile(path, normalized, 0o600)
}

// LoadPreset returns a saved preset's JSON.
func LoadPreset(dataDir, game string, id mod.ID, name string) ([]byte, error) {
	path, err := presetFile(dataDir, game, id, name)
	if err != nil {
		return nil, err
	}
	b, err := fsx.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("preset %q is not saved", name)
		}
		return nil, err
	}
	return b, nil
}

// DeletePreset removes a named preset.
func DeletePreset(dataDir, game string, id mod.ID, name string) error {
	path, err := presetFile(dataDir, game, id, name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("preset %q is not saved", name)
		}
		return err
	}
	return nil
}

func presetFolder(dataDir, game string, id mod.ID) (string, error) {
	g, err := pathSeg(game, "game")
	if err != nil {
		return "", err
	}
	seg, err := pathSeg(id.Local(), "mod")
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, presetsDir, g, seg), nil
}

func presetFile(dataDir, game string, id mod.ID, name string) (string, error) {
	if err := validatePresetName(name); err != nil {
		return "", err
	}
	dir, err := presetFolder(dataDir, game, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".json"), nil
}

func validatePresetName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("preset name is empty")
	}
	if utf8.RuneCountInString(name) > maxPresetName {
		return fmt.Errorf("preset name is longer than %d characters", maxPresetName)
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, string(filepath.Separator)) || name == "." || name == ".." {
		return fmt.Errorf("preset name %q is not allowed", name)
	}
	if filepath.Base(name) != name {
		return fmt.Errorf("preset name %q is not allowed", name)
	}
	return nil
}

func pathSeg(s, kind string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("%s id is empty", kind)
	}
	if strings.ContainsAny(s, `/\:`) || strings.Contains(s, string(filepath.Separator)) || s == "." || s == ".." {
		return "", fmt.Errorf("%s id %q is not allowed", kind, s)
	}
	if filepath.Base(s) != s {
		return "", fmt.Errorf("%s id %q is not allowed", kind, s)
	}
	return s, nil
}

func normalizePresetJSON(raw []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("preset is not valid JSON: %w", err)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
