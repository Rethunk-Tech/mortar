// Package smapimanifest reads SMAPI mod manifests the way SMAPI does: leniently.
package smapimanifest

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
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/ghrepo"
	"github.com/Rethunk-Tech/mortar/internal/jsonc"
	"github.com/Rethunk-Tech/mortar/internal/mod"
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
	// EntryDll is the C# mod's assembly file, relative to the mod folder; empty for content packs.
	EntryDll string
	// ContentPackFor is the ContentPackFor framework UniqueID when the manifest declares one.
	ContentPackFor string
	// Dependencies lists Dependencies[] and, as a required entry, the ContentPackFor framework.
	Dependencies []mod.Dependency
	// UpdateCautionMessage is shown at update time when Stardrop or Mortar cannot read the new version's manifest yet.
	UpdateCautionMessage string
	// MinimumGameVersion is the oldest game version the mod runs on, when it declares one.
	MinimumGameVersion string
	// DeleteOldVersion tells Mortar to carry over only user-written config (such as config.json), not other edited files.
	DeleteOldVersion bool
	// Format is the mod.ID format UniqueID belongs to; empty is SMAPI, the only format a manifest.json is read in.
	Format string
}

// parsed holds manifests already parsed, by their exact bytes, so the same file read again (a start reads each
// installed manifest several times) is not parsed again. It is emptied when full rather than tracking age.
var parsed struct {
	sync.Mutex
	m map[string]Manifest
}

const maxParsed = 8192

// Parse reads a manifest tolerating a UTF-8 BOM, // and /* */ comments, trailing commas and any key casing.
// A manifest without a UniqueID is an error, since nothing can refer to that mod.
func Parse(b []byte) (Manifest, error) {
	parsed.Lock()
	m, ok := parsed.m[string(b)]
	parsed.Unlock()
	if ok {
		return m.clone(), nil
	}
	m, err := parse(b)
	if err != nil {
		return Manifest{}, err
	}
	parsed.Lock()
	if parsed.m == nil || len(parsed.m) >= maxParsed {
		parsed.m = map[string]Manifest{}
	}
	parsed.m[string(b)] = m.clone()
	parsed.Unlock()
	return m, nil
}

func (m Manifest) clone() Manifest {
	m.UpdateKeys = slices.Clone(m.UpdateKeys)
	m.Dependencies = slices.Clone(m.Dependencies)
	return m
}

func parse(b []byte) (Manifest, error) {
	raw, err := readObject(b)
	if err != nil {
		return Manifest{}, err
	}
	m := Manifest{
		Name:                 text(raw, "name"),
		Author:               text(raw, "author"),
		Version:              version(field(raw, "version")),
		UniqueID:             text(raw, "uniqueid"),
		Description:          text(raw, "description"),
		EntryDll:             text(raw, "entrydll"),
		UpdateKeys:           texts(field(raw, "updatekeys")),
		UpdateCautionMessage: text(raw, "updatecautionmessage"),
		DeleteOldVersion:     boolean(field(raw, "deleteoldversion")),
		MinimumGameVersion:   text(raw, "minimumgameversion"),
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

// ModInfo is what a folder game's mod-root file says of its mod (Vintage Story's modinfo.json).
type ModInfo struct {
	Name, ModID, Version string
	Authors              []string
	// Dependencies are the mod ids the file requires, lower-cased and sorted; the values (versions) are not read.
	Dependencies []string
	// GameVersion is the game version the mod requires, its "game" dependency, when it names one.
	GameVersion string
}

// ParseModInfo reads a mod-root file as leniently as a SMAPI manifest, its keys in any case.
func ParseModInfo(b []byte) (ModInfo, error) {
	raw, err := readObject(b)
	if err != nil {
		return ModInfo{}, err
	}
	info := ModInfo{
		Name: text(raw, "name"), ModID: text(raw, "modid"), Version: version(field(raw, "version")),
		Authors: texts(field(raw, "authors")),
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

// curlyQuotes turns typographic quotes into straight ones, the second try SMAPI makes on a manifest that fails to parse.
var curlyQuotes = strings.NewReplacer("\u201c", `"`, "\u201d", `"`)

// readObject reads b as the JSON object Newtonsoft would, or says where it stops being one.
func readObject(b []byte) (map[string]json.RawMessage, error) {
	clean := jsonc.Clean(b)
	var raw map[string]json.RawMessage
	err := json.Unmarshal(clean, &raw)
	if err != nil && bytes.ContainsAny(b, "\u201c\u201d") {
		retry := jsonc.Clean([]byte(curlyQuotes.Replace(string(b))))
		if json.Unmarshal(retry, &raw) == nil {
			return raw, nil
		}
	}
	var syntax *json.SyntaxError
	var typed *json.UnmarshalTypeError
	switch {
	case err == nil:
		if raw == nil {
			return nil, errors.New("manifest is not a JSON object")
		}
		return raw, nil
	case errors.As(err, &syntax):
		return nil, fmt.Errorf("invalid JSON at line %d: %s", 1+bytes.Count(clean[:min(int(syntax.Offset), len(clean))], []byte("\n")), syntax.Error())
	case errors.As(err, &typed):
		return nil, errors.New("manifest is not a JSON object")
	}
	return nil, err
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

func dependency(v json.RawMessage) (mod.Dependency, bool) {
	var obj map[string]json.RawMessage
	if json.Unmarshal(v, &obj) != nil {
		return mod.Dependency{}, false
	}
	d := mod.Dependency{UniqueID: text(obj, "uniqueid"), MinimumVersion: text(obj, "minimumversion"), Required: true}
	if b := field(obj, "isrequired"); b != nil {
		if required, ok := flexBool(b); ok {
			d.Required = required
		}
	}
	return d, d.UniqueID != ""
}

// dependencies reads Dependencies[] and ContentPackFor, skipping entries without a UniqueID.
func dependencies(raw map[string]json.RawMessage) []mod.Dependency {
	var out []mod.Dependency
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
		// Packs often list their framework in Dependencies too; it is one dependency, so it is checked and shown once.
		i := slices.IndexFunc(out, func(o mod.Dependency) bool { return mod.Equal(o.ModID(), d.ModID()) })
		switch {
		case i < 0:
			out = append(out, d)
		case out[i].MinimumVersion == "":
			out[i].Required, out[i].MinimumVersion = true, d.MinimumVersion
		default:
			out[i].Required = true
		}
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

// ReadFile reads dir's manifest.json. Mod managers that deploy by symlink (Vortex, on Linux and on Windows when set
// to) leave it as a link into their own staging folder, so the link is resolved first and the target read through its
// own folder; a dangling link reads as not existing.
func ReadFile(dir string) ([]byte, error) {
	resolved, err := fsx.EvalSymlinks(filepath.Join(dir, FileName))
	if err != nil {
		return nil, err
	}
	return fsx.ReadFile(resolved)
}

// Mod is a manifest found by Scan.
type Mod struct {
	Manifest
	// Folder is the manifest's folder relative to the scanned root, slash-separated; "." for the root itself.
	Folder string
}

// Invalid is a manifest.json that exists but cannot be read as a SMAPI manifest.
type Invalid struct {
	// Folder is the manifest's folder relative to the scanned root, slash-separated; "." for the root itself.
	Folder string
	Err    error
}

// Error names the manifest by its path under the scanned root and says why it was refused.
func (i Invalid) Error() string {
	name := FileName
	if i.Folder != "." {
		name = i.Folder + "/" + FileName
	}
	return name + ": " + i.Err.Error()
}

// Scan finds mods under root like SMAPI: it stops descending at a folder holding a manifest.json and skips
// subfolders whose names start with a dot. A manifest that does not parse is skipped, as SMAPI reports it
// as invalid rather than loading it. Unlike SMAPI it also descends into a folder that holds other files (a stray
// config.json or readme.pdf beside a mod folder): every caller installs each mod found as its own folder, so the
// mod loads once Mortar deploys it, where SMAPI would have reported the outer folder as one broken mod.
func Scan(root string) ([]Mod, error) {
	mods, _, err := ScanInvalid(root)
	return mods, err
}

// ScanInvalid is Scan that also returns the first manifest it skipped for not parsing, nil when there was none.
func ScanInvalid(root string) ([]Mod, *Invalid, error) {
	base, err := fsx.EvalSymlinks(root)
	if err != nil {
		return nil, nil, err
	}
	var mods []Mod
	var invalid *Invalid
	var walk func(dir, rel string) error
	walk = func(dir, rel string) error {
		b, err := ReadFile(dir)
		if err == nil {
			if m, perr := Parse(b); perr == nil {
				mods = append(mods, Mod{Manifest: m, Folder: rel})
			} else if invalid == nil {
				invalid = &Invalid{Folder: rel, Err: perr}
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
	err = walk(root, ".")
	return mods, invalid, err
}

// LoaderManaged reports whether a mod is installed and kept current by a loader or Mortar itself, so a profile never
// lists, imports or downloads it as one of its own mods.
func LoaderManaged(id mod.ID) bool {
	switch id.Fold() {
	case "smapi:smapi.consolecommands", "smapi:smapi.savebackup", "smapi:rethunk.mortarsmapibridge", "bepinex:Rethunk.MortarBepInExBridge":
		return true
	}
	return false
}

// NexusUpdateKey returns the mod page of a "Nexus:1234" or "Nexus:1234@subkey" update key.
func NexusUpdateKey(key string) (int, bool) {
	site, rest, ok := strings.Cut(key, ":")
	if !ok || !strings.EqualFold(strings.TrimSpace(site), "nexus") {
		return 0, false
	}
	rest, _, _ = strings.Cut(rest, "@")
	n, err := strconv.Atoi(strings.TrimSpace(rest))
	return n, err == nil
}

// CurseForgeUpdateKey returns the project id of a "CurseForge:<id>" update key.
func CurseForgeUpdateKey(key string) (string, bool) {
	site, rest, ok := strings.Cut(key, ":")
	if !ok || !strings.EqualFold(strings.TrimSpace(site), "curseforge") {
		return "", false
	}
	rest, _, _ = strings.Cut(rest, "@")
	rest = strings.TrimSpace(rest)
	n, err := strconv.Atoi(rest)
	return rest, err == nil && n > 0
}

// GitHubUpdateKey returns the "owner/repo" of a "GitHub:owner/repo" update key.
func GitHubUpdateKey(key string) (string, bool) {
	site, rest, ok := strings.Cut(key, ":")
	rest = strings.TrimSpace(rest)
	return rest, ok && strings.EqualFold(strings.TrimSpace(site), "github") && ghrepo.Valid(rest)
}

// ModID is the manifest's mod as a mod.ID.
func (m Manifest) ModID() mod.ID {
	if m.Format != "" {
		return mod.NewID(m.Format, m.UniqueID)
	}
	return mod.SMAPI(m.UniqueID)
}

// ContentPackForID is the ContentPackFor framework as a mod.ID, empty when the manifest declares none.
func (m Manifest) ContentPackForID() mod.ID {
	if m.ContentPackFor == "" {
		return ""
	}
	return mod.SMAPI(m.ContentPackFor)
}

// WithModID returns the manifest naming id, for a manifest built from another source's data.
func (m Manifest) WithModID(id mod.ID) Manifest {
	m.UniqueID, m.Format = id.Local(), ""
	if f := id.Format(); f != mod.FormatSMAPI {
		m.Format = f
	}
	return m
}
