package share

import (
	"archive/zip"
	"bytes"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Pin returns the .mortar payload dst with the named mods' on/off state and the named config files taken from src.
// A file is named by a zip entry name or its trailing path ("com.x.cfg", "Mod.Id/config.json"); every entry of either
// payload that matches counts, and one src lacks is dropped from dst.
func Pin(dst, src []byte, mods []mod.ID, files []string) ([]byte, error) {
	dv, dz, err := readPayload(dst)
	if err != nil {
		return nil, err
	}
	sv, sz, err := readPayload(src)
	if err != nil {
		return nil, err
	}
	entries := make([]Ref, len(dv.Entries))
	for i, r := range dv.Entries {
		r.Disabled = slices.DeleteFunc(slices.Clone(r.Disabled), func(id mod.ID) bool { return slices.Contains(mods, id) })
		entries[i] = r
	}
	for _, id := range mods {
		for _, sr := range sv.Entries {
			if !slices.Contains(sr.Disabled, id) {
				continue
			}
			if i := slices.IndexFunc(entries, func(r Ref) bool { return r.Identity() == sr.Identity() }); i >= 0 {
				entries[i].Disabled = append(entries[i].Disabled, id)
			}
		}
	}
	pinned := func(name string) bool {
		return slices.ContainsFunc(files, func(f string) bool { return name == f || strings.HasSuffix(name, "/"+f) })
	}
	srcFile := map[string][]byte{}
	for _, e := range sz {
		if e.name != profileFile && pinned(e.name) && !skippedEntry(e.name) {
			srcFile[e.name] = e.data
		}
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	written := map[string]bool{}
	for _, e := range dz {
		switch {
		case skippedEntry(e.name):
		case e.name == profileFile:
			data, err := withEntries(e.data, entries)
			if err != nil {
				return nil, err
			}
			err = putFile(zw, e.name, data)
		case pinned(e.name):
			// The pinned file keeps its place, with src's content; one src lacks is dropped.
			if data, ok := srcFile[e.name]; ok {
				err = putFile(zw, e.name, data)
				written[e.name] = true
			}
		default:
			err = putFile(zw, e.name, e.data)
		}
		if err != nil {
			return nil, err
		}
	}
	for _, e := range sz {
		if data, ok := srcFile[e.name]; ok && !written[e.name] {
			if err := putFile(zw, e.name, data); err != nil {
				return nil, err
			}
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
