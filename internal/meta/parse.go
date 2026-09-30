package meta

import (
	"encoding/json"
	"errors"
	"strings"
)

// text accepts a string, a number (kept as written) or anything else (empty).
type text string

func (t *text) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*t = text(strings.TrimSpace(s))
		return nil
	}
	var n json.Number
	if json.Unmarshal(b, &n) == nil {
		*t = text(n)
	}
	return nil
}

// list keeps the string entries of an array and ignores a value of any other shape.
type list []text

func (l *list) UnmarshalJSON(b []byte) error {
	var items []text
	if json.Unmarshal(b, &items) == nil {
		*l = items
	}
	return nil
}

type rawDependency struct {
	UniqueID       text  `json:"UniqueID"`
	MinimumVersion text  `json:"MinimumVersion"`
	IsRequired     *bool `json:"IsRequired"`
}

type rawManifest struct {
	Name           text            `json:"Name"`
	Version        text            `json:"Version"`
	UniqueID       text            `json:"UniqueID"`
	UpdateKeys     list            `json:"UpdateKeys"`
	Dependencies   []rawDependency `json:"Dependencies"`
	ContentPackFor *rawDependency  `json:"ContentPackFor"`
}

type rawPage struct {
	ID        int64 `json:"ID"`
	Name      text  `json:"Name"`
	Author    text  `json:"Author"`
	PageURL   text  `json:"PageURL"`
	Version   text  `json:"Version"`
	Downloads []struct {
		ID          int64 `json:"ID"`
		Type        text  `json:"Type"`
		Version     text  `json:"Version"`
		FileName    text  `json:"FileName"`
		SizeInBytes int64 `json:"SizeInBytes"`
		Mods        []struct {
			Manifest *rawManifest `json:"Manifest"`
		} `json:"Mods"`
	} `json:"Downloads"`
}

// parsePage reads a dataset page entry, ignoring unknown fields and tolerating fields of the wrong type
// (which json reports as an error after decoding the rest, so the partial result is kept).
func parsePage(b []byte) (Page, error) {
	var r rawPage
	if err := json.Unmarshal(b, &r); err != nil {
		if _, ok := errors.AsType[*json.UnmarshalTypeError](err); !ok {
			return Page{}, err
		}
	}
	p := Page{ID: int(r.ID), Name: string(r.Name), Author: string(r.Author), PageURL: string(r.PageURL), Version: string(r.Version)}
	for _, d := range r.Downloads {
		f := File{ID: d.ID, Type: string(d.Type), Version: string(d.Version), FileName: string(d.FileName), SizeInBytes: d.SizeInBytes}
		for _, m := range d.Mods {
			if m.Manifest == nil || m.Manifest.UniqueID == "" {
				continue
			}
			f.Mods = append(f.Mods, m.Manifest.mod())
		}
		p.Downloads = append(p.Downloads, f)
	}
	return p, nil
}

func (m *rawManifest) mod() Mod {
	out := Mod{UniqueID: string(m.UniqueID), Name: string(m.Name), Version: string(m.Version)}
	for _, k := range m.UpdateKeys {
		out.UpdateKeys = append(out.UpdateKeys, string(k))
	}
	for _, d := range m.Dependencies {
		if d.UniqueID != "" {
			out.Dependencies = append(out.Dependencies, Dependency{string(d.UniqueID), string(d.MinimumVersion), d.IsRequired == nil || *d.IsRequired})
		}
	}
	if cp := m.ContentPackFor; cp != nil && cp.UniqueID != "" {
		out.Dependencies = append(out.Dependencies, Dependency{string(cp.UniqueID), string(cp.MinimumVersion), true})
	}
	return out
}
