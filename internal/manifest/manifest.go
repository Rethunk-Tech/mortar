// Package manifest reads SMAPI mod manifests the way SMAPI does: leniently.
package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// FileName is the manifest file SMAPI looks for.
const FileName = "manifest.json"

// Manifest holds the fields Mortar uses.
type Manifest struct {
	Name        string
	Author      string
	Version     string
	UniqueID    string
	Description string
	UpdateKeys  []string
	// ContentPackFor is the ContentPackFor framework UniqueID when the manifest declares one.
	ContentPackFor string
	// Dependencies lists Dependencies[] and, as a required entry, the ContentPackFor framework.
	Dependencies []Dependency
	// UpdateCautionMessage is shown at update time when Stardrop or Mortar cannot read the new version's manifest yet.
	UpdateCautionMessage string
	// DeleteOldVersion tells Mortar to carry over only user-written config (such as config.json), not other edited files.
	DeleteOldVersion bool
}

// Dependency is one mod another mod needs; Required defaults to true when the manifest omits IsRequired.
type Dependency struct {
	UniqueID       string
	MinimumVersion string
	Required       bool
}

// Parse reads a manifest tolerating a UTF-8 BOM, // and /* */ comments, trailing commas and any key casing.
// A manifest without a UniqueID is an error, since nothing can refer to that mod.
func Parse(b []byte) (Manifest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(dropTrailingCommas(dropComments(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")))), &raw); err != nil {
		return Manifest{}, fmt.Errorf("invalid manifest: %w", err)
	}
	m := Manifest{
		Name:                 text(raw, "name"),
		Author:               text(raw, "author"),
		Version:              version(field(raw, "version")),
		UniqueID:             text(raw, "uniqueid"),
		Description:          text(raw, "description"),
		UpdateKeys:           texts(field(raw, "updatekeys")),
		UpdateCautionMessage: text(raw, "updatecautionmessage"),
		DeleteOldVersion:     boolean(field(raw, "deleteoldversion")),
	}
	if d, ok := dependency(field(raw, "contentpackfor")); ok {
		m.ContentPackFor = d.UniqueID
	}
	m.Dependencies = dependencies(raw)
	if m.UniqueID == "" {
		return Manifest{}, errors.New("manifest has no UniqueID")
	}
	return m, nil
}

func texts(v json.RawMessage) []string {
	var items []json.RawMessage
	if json.Unmarshal(v, &items) != nil {
		return nil
	}
	var out []string
	for _, it := range items {
		var s string
		if json.Unmarshal(it, &s) == nil && strings.TrimSpace(s) != "" {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}

func dependency(v json.RawMessage) (Dependency, bool) {
	var obj map[string]json.RawMessage
	if json.Unmarshal(v, &obj) != nil {
		return Dependency{}, false
	}
	d := Dependency{UniqueID: text(obj, "uniqueid"), MinimumVersion: text(obj, "minimumversion"), Required: true}
	if b := field(obj, "isrequired"); b != nil {
		if required, ok := flexBool(b); ok {
			d.Required = required
		}
	}
	return d, d.UniqueID != ""
}

// dependencies reads Dependencies[] and ContentPackFor, skipping entries without a UniqueID.
func dependencies(raw map[string]json.RawMessage) []Dependency {
	var out []Dependency
	var items []json.RawMessage
	if json.Unmarshal(field(raw, "dependencies"), &items) == nil {
		for _, it := range items {
			if d, ok := dependency(it); ok {
				out = append(out, d)
			}
		}
	}
	if d, ok := dependency(field(raw, "contentpackfor")); ok {
		d.Required = true
		out = append(out, d)
	}
	return out
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

func boolean(v json.RawMessage) bool {
	b, _ := flexBool(v)
	return b
}

// flexBool reads a JSON boolean or the strings "true"/"false" in any case, as SMAPI's Json.NET reader does.
func flexBool(v json.RawMessage) (value, ok bool) {
	if json.Unmarshal(v, &value) == nil {
		return value, true
	}
	var s string
	if json.Unmarshal(v, &s) != nil {
		return false, false
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true":
		return true, true
	case "false":
		return false, true
	}
	return false, false
}

// version accepts a string or SMAPI's legacy {MajorVersion, MinorVersion, PatchVersion, Build} object.
func version(v json.RawMessage) string {
	var s string
	if json.Unmarshal(v, &s) == nil {
		return strings.TrimSpace(s)
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(v, &obj) != nil {
		return ""
	}
	num := func(name string) string {
		var n int
		_ = json.Unmarshal(field(obj, name), &n)
		return strconv.Itoa(n)
	}
	out := num("majorversion") + "." + num("minorversion") + "." + num("patchversion")
	if build := text(obj, "build"); build != "" {
		out += "-" + build
	}
	return out
}

// dropComments removes // and /* */ comments outside strings.
func dropComments(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		switch {
		case b[i] == '"':
			j := i + 1
			for j < len(b) && b[j] != '"' {
				if b[j] == '\\' {
					j++
				}
				j++
			}
			j = min(j, len(b)-1)
			out = append(out, b[i:j+1]...)
			i = j
		case b[i] == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
			out = append(out, '\n')
		case b[i] == '/' && i+1 < len(b) && b[i+1] == '*':
			end := bytes.Index(b[i+2:], []byte("*/"))
			if end < 0 {
				return out
			}
			i += end + 3
			out = append(out, ' ')
		default:
			out = append(out, b[i])
		}
	}
	return out
}

// dropTrailingCommas removes a comma whose next non-space byte closes an object or array.
func dropTrailingCommas(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '"':
			j := i + 1
			for j < len(b) && b[j] != '"' {
				if b[j] == '\\' {
					j++
				}
				j++
			}
			j = min(j, len(b)-1)
			out = append(out, b[i:j+1]...)
			i = j
		case ',':
			j := i + 1
			for j < len(b) && slices.Contains([]byte(" \t\r\n"), b[j]) {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				continue
			}
			out = append(out, ',')
		default:
			out = append(out, b[i])
		}
	}
	return out
}

// Mod is a manifest found by Scan.
type Mod struct {
	Manifest
	// Folder is the manifest's folder relative to the scanned root, slash-separated; "." for the root itself.
	Folder string
}

// Scan finds mods under root like SMAPI: it stops descending at a folder holding a manifest.json and skips
// subfolders whose names start with a dot. A manifest that does not parse is skipped, as SMAPI reports it
// as invalid rather than loading it.
func Scan(root string) ([]Mod, error) {
	base, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	var mods []Mod
	var walk func(dir, rel string) error
	walk = func(dir, rel string) error {
		b, err := fsx.ReadFile(filepath.Join(dir, FileName))
		if err == nil {
			if m, perr := Parse(b); perr == nil {
				mods = append(mods, Mod{Manifest: m, Folder: rel})
			}
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		items, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, it := range items {
			if strings.HasPrefix(it.Name(), ".") {
				continue
			}
			child := filepath.Join(dir, it.Name())
			if !datadir.RealDirUnder(base, child) {
				continue
			}
			next := it.Name()
			if rel != "." {
				next = rel + "/" + next
			}
			if err := walk(child, next); err != nil {
				return err
			}
		}
		return nil
	}
	return mods, walk(root, ".")
}

// LoaderManaged reports whether a mod is installed and kept current by SMAPI or Mortar itself, so a profile never
// lists, imports or downloads it as one of its own mods.
func LoaderManaged(uniqueID string) bool {
	switch strings.ToLower(uniqueID) {
	case "smapi.consolecommands", "smapi.savebackup", "rethunk.mortarsmapibridge":
		return true
	}
	return false
}
