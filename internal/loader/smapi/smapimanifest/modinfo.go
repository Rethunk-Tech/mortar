package smapimanifest

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-lenientjson"
)

// ModInfo is what a folder game's mod-root file says of its mod (Vintage Story's modinfo.json).
type ModInfo struct {
	Name, ModID, Version string
	Authors              []string
	// Dependencies are the mod ids the file requires, lower-cased and sorted; the values (versions) are not read.
	Dependencies []string
	// GameVersion is the game version the mod requires, its "game" dependency, when it names one.
	GameVersion string
}

// ParseModInfo reads a mod-root file as leniently as Newtonsoft, which the game reads it with, its keys in any case.
func ParseModInfo(b []byte) (ModInfo, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(lenientjson.Clean(b), &raw); err != nil {
		return ModInfo{}, err
	}
	if raw == nil {
		return ModInfo{}, errors.New("mod info is not a JSON object")
	}
	info := ModInfo{Name: text(raw, "name"), ModID: text(raw, "modid"), Version: text(raw, "version")}
	var authors []string
	_ = json.Unmarshal(field(raw, "authors"), &authors)
	for _, a := range authors {
		if a = strings.TrimSpace(a); a != "" {
			info.Authors = append(info.Authors, a)
		}
	}
	var deps map[string]json.RawMessage
	if json.Unmarshal(field(raw, "dependencies"), &deps) == nil {
		for id, v := range deps {
			info.Dependencies = append(info.Dependencies, strings.ToLower(id))
			var ver string
			if strings.EqualFold(id, "game") && json.Unmarshal(v, &ver) == nil && ver != "*" {
				info.GameVersion = strings.TrimPrefix(strings.TrimSpace(ver), "v")
			}
		}
		slices.Sort(info.Dependencies)
	}
	return info, nil
}

func field(raw map[string]json.RawMessage, name string) json.RawMessage {
	for k, v := range raw {
		if strings.EqualFold(k, name) {
			return v
		}
	}
	return nil
}

func text(raw map[string]json.RawMessage, name string) string {
	var s string
	if json.Unmarshal(field(raw, name), &s) != nil {
		return ""
	}
	return strings.TrimSpace(s)
}
