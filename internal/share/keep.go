package share

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// Identity names the file a ref installs, ignoring the per-entry options (disabled mods, notes, FOMOD picks).
func (r Ref) Identity() string {
	switch {
	case r.Package != "":
		return "t:" + strings.ToLower(r.Package) + "@" + r.Version
	case r.GitHub != "":
		return "g:" + strings.ToLower(r.GitHub)
	default:
		return fmt.Sprintf("n:%d:%d", r.ModID, r.FileID)
	}
}

// WithRefs returns the .mortar payload raw with add appended to its entries; everything else in it is kept.
func WithRefs(raw []byte, add []Ref) ([]byte, error) {
	pv, err := ReadBytes(raw)
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("%w: not a zip", ErrBadFile)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		data, err := readBounded(f, MaxFileBytes)
		if err != nil {
			return nil, err
		}
		if f.Name == profileFile {
			if data, err = withEntries(data, append(pv.Entries, add...)); err != nil {
				return nil, err
			}
		}
		if err := putFile(zw, f.Name, data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func withEntries(doc []byte, entries []Ref) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(doc, &fields); err != nil {
		return nil, fmt.Errorf("%w: bad %s", ErrBadFile, profileFile)
	}
	enc, err := json.Marshal(entries)
	if err != nil {
		return nil, err
	}
	fields["entries"] = enc
	return json.Marshal(fields)
}

// SameExceptArrival reports whether have differs from base only by files in missing not having arrived: the profile
// name, notes, description, every entry's options and order, and every config file that is there must match. Mod ids
// and config files base holds for a file that has not arrived are not in have and are not a difference.
func SameExceptArrival(base, have Preview, missing map[string]bool) bool {
	if base.Name != have.Name || base.Notes != have.Notes || base.Description != have.Description {
		return false
	}
	var want []string
	for _, r := range base.Entries {
		if id := r.Identity(); !missing[id] || slices.ContainsFunc(have.Entries, func(h Ref) bool { return h.Identity() == id }) {
			enc, _ := json.Marshal(r)
			want = append(want, string(enc))
		}
	}
	var got []string
	for _, r := range have.Entries {
		enc, _ := json.Marshal(r)
		got = append(got, string(enc))
	}
	if !slices.Equal(want, got) {
		return false
	}
	for _, id := range have.IDs {
		if !slices.Contains(base.IDs, id) {
			return false
		}
	}
	type cfg struct {
		id   mod.ID
		path string
	}
	held := map[cfg]string{}
	for _, c := range base.Configs {
		held[cfg{c.ID, c.Path}] = string(c.Data)
	}
	for _, c := range have.Configs {
		if data, ok := held[cfg{c.ID, c.Path}]; !ok || data != string(c.Data) {
			return false
		}
	}
	return true
}
