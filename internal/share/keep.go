package share

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
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
