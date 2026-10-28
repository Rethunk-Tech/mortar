package share

import (
	"archive/zip"
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Pin returns the .mortar payload dst with the named mods' on/off state and the named config files taken from src.
// A file is named by a zip entry name or its trailing path ("com.x.cfg", "Mod.Id/config.json"); every entry of either
// payload that matches counts, and one src lacks is dropped from dst.
func Pin(dst, src []byte, mods []mod.ID, files []string) ([]byte, error) {
	dv, err := ReadBytes(dst)
	if err != nil {
		return nil, err
	}
	sv, err := ReadBytes(src)
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
	dz, err := zip.NewReader(bytes.NewReader(dst), int64(len(dst)))
	if err != nil {
		return nil, fmt.Errorf("%w: not a zip", ErrBadFile)
	}
	sz, err := zip.NewReader(bytes.NewReader(src), int64(len(src)))
	if err != nil {
		return nil, fmt.Errorf("%w: not a zip", ErrBadFile)
	}
	pinned := func(name string) bool {
		return slices.ContainsFunc(files, func(f string) bool { return name == f || strings.HasSuffix(name, "/"+f) })
	}
	srcFile := map[string]*zip.File{}
	for _, f := range sz.File {
		if f.Name != profileFile && pinned(f.Name) && !skippedChanged(f.Name) {
			srcFile[f.Name] = f
		}
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	put := func(from *zip.File, name string, edit func([]byte) ([]byte, error)) error {
		data, err := readBounded(from, MaxFileBytes)
		if err != nil {
			return err
		}
		if edit != nil {
			if data, err = edit(data); err != nil {
				return err
			}
		}
		return putFile(zw, name, data)
	}
	written := map[string]bool{}
	for _, f := range dz.File {
		switch {
		case skippedChanged(f.Name):
		case f.Name == profileFile:
			err = put(f, f.Name, func(d []byte) ([]byte, error) { return withEntries(d, entries) })
		case pinned(f.Name):
			// The pinned file keeps its place, with src's content; one src lacks is dropped.
			if from, ok := srcFile[f.Name]; ok {
				err = put(from, f.Name, nil)
				written[f.Name] = true
			}
		default:
			err = put(f, f.Name, nil)
		}
		if err != nil {
			return nil, err
		}
	}
	for _, f := range sz.File {
		if _, ok := srcFile[f.Name]; ok && !written[f.Name] {
			if err := put(f, f.Name, nil); err != nil {
				return nil, err
			}
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
