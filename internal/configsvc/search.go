package configsvc

import (
	"cmp"
	"fmt"
	"strings"
)

// Hit is one setting a profile-wide search found: which mod's file holds it and where.
type Hit struct {
	ModID   string `json:"modId"`
	ModName string `json:"modName"`
	File    string `json:"file"`
	Section string `json:"section"`
	Key     string `json:"key"`
	Label   string `json:"label,omitempty"`
	Value   string `json:"value"`
}

// MaxHits bounds a search so a one-letter query does not return every setting of every mod.
const MaxHits = 200

// matches reports whether the query, already lower-cased, is in the entry's key, label or description, or in the name
// of its section.
func matches(q string, section Section, e Entry) bool {
	return strings.Contains(strings.ToLower(section.Name), q) ||
		strings.Contains(strings.ToLower(e.Key), q) ||
		strings.Contains(strings.ToLower(e.Label), q) ||
		strings.Contains(strings.ToLower(e.Description), q)
}

// hitsIn is the entries of one schema that match the query.
func hitsIn(q, modID, modName string, sc Schema) []Hit {
	var out []Hit
	for _, sec := range sc.Sections {
		for _, e := range sec.Entries {
			if matches(q, sec, e) {
				out = append(out, Hit{ModID: modID, ModName: modName, File: sc.File.Name, Section: sec.Name, Key: e.Key, Label: e.Label, Value: e.Value})
			}
		}
	}
	return out
}

// textHits is the lines of a text file, and its name when no line matches, that hold the query. A text file has no
// entries, so a hit names the line.
func textHits(q, modID, modName, file, body string) []Hit {
	var out []Hit
	for i, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(strings.ToLower(line), q) {
			out = append(out, Hit{ModID: modID, ModName: modName, File: file, Key: fmt.Sprintf("line %d", i+1), Label: truncate(line, 80), Value: line})
		}
	}
	if len(out) == 0 && strings.Contains(strings.ToLower(file), q) {
		out = append(out, Hit{ModID: modID, ModName: modName, File: file, Key: file})
	}
	return out
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// searchFile is the hits of one config file, by its format.
func (s *Service) searchFile(game, profileID, q, modID, modName string, f ConfigFile) []Hit {
	if f.Format == FormatText {
		body, err := s.Text(game, profileID, modID, f.Name)
		if err != nil {
			return nil
		}
		return textHits(q, modID, modName, f.Name, body)
	}
	sc, err := s.Schema(game, profileID, modID, f.Name)
	if err != nil {
		return nil
	}
	return hitsIn(q, modID, modName, sc)
}

// Search finds the settings, across every config file in the profile (each mod's, BepInEx.cfg and the files no mod
// owns), whose key, label, description or section holds the query; a text file is searched line by line and the
// in-game menu like any other file.
func (s *Service) Search(game, profileID, query string) ([]Hit, error) {
	q := strings.ToLower(strings.TrimSpace(query))
	out := []Hit{}
	if q == "" {
		return out, nil
	}
	list, err := s.Mods(game, profileID)
	if err != nil {
		return nil, err
	}
	add := func(modID, modName string, files []ModFile) bool {
		for _, f := range files {
			name := cmp.Or(modName, f.Name)
			out = append(out, s.searchFile(game, profileID, q, modID, name, ConfigFile{Name: f.Name, Format: f.Format})...)
			if len(out) >= MaxHits {
				out = out[:MaxHits]
				return true
			}
		}
		return false
	}
	for _, m := range list.Mods {
		if add(string(m.ID), m.Name, m.Files) {
			return out, nil
		}
	}
	add("", "", list.Other)
	return out, nil
}
