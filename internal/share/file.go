package share

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// Caps on a .mortar file, enforced by counting bytes read rather than trusting declared sizes.
const (
	MaxFileBytes    = 32 << 20
	MaxConfigBytes  = 1 << 20
	MaxConfigTotal  = 32 << 20
	MaxConfigFiles  = 5000
	maxProfileBytes = 512 << 10
	maxIDLen        = 100
	maxSegment      = 100
	maxRelPath      = 240
)

const profileFile = "profile.json"

var (
	uniqueID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	format   = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	segment  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._()+~-]*$`)
	reserved = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])(\..*)?$`)
)

// ErrBadFile means a .mortar file is not a valid share; the message names the offending entry.
var ErrBadFile = errors.New("not a valid .mortar file")

// Config is one config file to write into an installed mod's folder: Path is relative to the mod folder, slash
// separated, ends in .json and stays inside it.
type Config struct {
	ID   mod.ID `json:"id"`
	Path string `json:"path"`
	Data []byte `json:"data"`
}

// Preview is what reading a .mortar file yields. The apply step writes Configs only into installed mod folders, and
// LoaderConfigs only under the receiving profile's loader config folders.
type Preview struct {
	Shared
	Notes         string
	Description   string
	IDs           []mod.ID
	Configs       []Config
	LoaderConfigs []LoaderConfig
	Groups        []FileGroup
	Choices       ProblemChoices
}

type fileDoc struct {
	Version     int               `json:"version"`
	Name        string            `json:"name"`
	Game        string            `json:"game"`
	SourceKeys  map[string]string `json:"sourceKeys"`
	Notes       string            `json:"notes"`
	Description string            `json:"description,omitempty"`
	Entries     json.RawMessage   `json:"entries"`
	IDs         []mod.ID          `json:"ids"`
	Groups      []FileGroup       `json:"groups,omitempty"`
	Problems    *ProblemChoices   `json:"problems,omitempty"`
}

func validSegment(s string) bool {
	return len(s) <= maxSegment && segment.MatchString(s) && !strings.HasSuffix(s, ".") &&
		!strings.HasSuffix(s, " ") && !reserved.MatchString(s)
}

// ValidConfigPath reports whether p is a path Apply would write.
func ValidConfigPath(p string) bool { return validConfigPath(p) }

// ValidFomod reports whether recorded FOMOD choices are within the bounds a share link enforces.
func ValidFomod(f map[string]map[string][]string) bool { return validDetails(Ref{Fomod: f}) }

// validConfigPath accepts a slash-separated relative .json path whose every segment is plain.
func validConfigPath(p string) bool {
	if len(p) > maxRelPath || !strings.HasSuffix(strings.ToLower(p), ".json") {
		return false
	}
	for s := range strings.SplitSeq(p, "/") {
		if !validSegment(s) {
			return false
		}
	}
	return true
}

// validID accepts ids whose format and local part are safe to use as zip path segments.
func validID(id mod.ID) bool {
	local := id.Local()
	return format.MatchString(id.Format()) && len(local) <= maxIDLen && uniqueID.MatchString(local) && !reserved.MatchString(local)
}

// Write writes the profile as a .mortar zip: profile.json with the link's entries plus name, notes and
// description, the .json files of each enabled mod's folder under configs/<mod id>/, and the text files of the
// loader's config folders under loader/<path in the profile>. modsDir is the profile's mods/ folder, inside the
// profile's own. Config files that are over the caps or have unusual names are skipped and returned as paths.
func Write(w io.Writer, gameID string, p profile.Profile, modsDir string, include ...Include) (skipped []string, err error) {
	inc := DefaultInclude()
	if len(include) > 0 {
		inc = include[0]
	}
	s, _, _ := Collect(p, inc)
	s.Game, s.SourceKeys = gameID, SourceKeys(gameID)
	zw := zip.NewWriter(w)
	entries, err := json.Marshal(s.Entries)
	if err != nil {
		return nil, err
	}
	notes := p.Notes
	if !inc.Notes {
		notes = ""
	}
	doc := fileDoc{
		Version: FormatVersion, Name: s.Name, Game: s.Game, SourceKeys: s.SourceKeys, Notes: notes, Description: p.Description,
		Entries: entries, IDs: []mod.ID{},
	}
	if inc.Notes {
		doc.Groups = collectFileGroups(p)
	}
	if inc.ProblemChoices {
		doc.Problems = collectProblemChoices(p, inc.Dismissed, func(e profile.Entry) bool { return Enabled(e) || inc.DisabledMods })
	}
	if err := checkShared(s); err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(notes) > profile.MaxNotes {
		return nil, fmt.Errorf("%w: notes are too long", ErrBadFile)
	}
	if utf8.RuneCountInString(p.Description) > profile.MaxDescription {
		return nil, fmt.Errorf("%w: description is too long", ErrBadFile)
	}
	var configs []Config
	for _, e := range p.Entries {
		if !inc.ConfigFiles {
			break
		}
		if e.Source.Bundled() || (!Enabled(e) && !inc.DisabledMods) {
			continue
		}
		for _, m := range e.Mods {
			if !e.Enabled(m.ID) || !validID(m.ID) {
				continue
			}
			doc.IDs = append(doc.IDs, m.ID)
			found, skip, err := readConfigs(modsDir, e.Key, m)
			if err != nil {
				return nil, err
			}
			configs = append(configs, found...)
			skipped = append(skipped, skip...)
		}
	}
	var loaderConfigs []LoaderConfig
	if inc.ConfigFiles && modsDir != "" {
		found, skip, err := readLoaderConfigs(filepath.Dir(modsDir), gameID, p.Loader)
		if err != nil {
			return nil, err
		}
		loaderConfigs = found
		skipped = append(skipped, skip...)
	}
	var total int64
	kept := configs[:0]
	for _, c := range configs {
		total += int64(len(c.Data))
		if total > MaxConfigTotal || len(kept) >= MaxConfigFiles {
			skipped = append(skipped, c.ID.Local()+"/"+c.Path)
			continue
		}
		kept = append(kept, c)
	}
	keptLoader := loaderConfigs[:0]
	for _, c := range loaderConfigs {
		total += int64(len(c.Data))
		if total > MaxConfigTotal || len(kept)+len(keptLoader) >= MaxConfigFiles {
			skipped = append(skipped, c.Path)
			continue
		}
		keptLoader = append(keptLoader, c)
	}
	head, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	if err := putFile(zw, profileFile, head); err != nil {
		return nil, err
	}
	for _, c := range kept {
		if err := putFile(zw, "configs/"+c.ID.Format()+"/"+c.ID.Local()+"/"+c.Path, c.Data); err != nil {
			return nil, err
		}
	}
	for _, c := range keptLoader {
		if err := putFile(zw, loaderPrefix+c.Path, c.Data); err != nil {
			return nil, err
		}
	}
	return skipped, zw.Close()
}

func putFile(zw *zip.Writer, name string, data []byte) error {
	f, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

// ReadConfigs collects the .json files of one mod's folder, except its manifest; skipped are the ones left out for
// their size or name.
func ReadConfigs(modsDir, key string, m profile.Component) (found []Config, skipped []string, err error) {
	return readConfigs(modsDir, key, m)
}

// readConfigs collects the .json files under one enabled mod's folder, except its manifest.
func readConfigs(modsDir, key string, m profile.Component) (found []Config, skipped []string, err error) {
	folder := filepath.Join(modsDir, key)
	if m.Folder != "." {
		if !filepath.IsLocal(filepath.FromSlash(m.Folder)) {
			return nil, nil, fmt.Errorf("mod folder %q leaves its entry", m.Folder)
		}
		folder = filepath.Join(folder, filepath.FromSlash(m.Folder))
	}
	err = filepath.WalkDir(folder, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if errors.Is(walkErr, fs.ErrNotExist) {
				return nil
			}
			return walkErr
		}
		if !d.Type().IsRegular() || !strings.EqualFold(filepath.Ext(p), ".json") {
			return nil
		}
		rel, err := filepath.Rel(folder, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.EqualFold(rel, manifest.FileName) {
			return nil
		}
		data, err := readCapped(p)
		if errors.Is(err, errOverCap) || !validConfigPath(rel) {
			skipped = append(skipped, m.ID.Local()+"/"+rel)
			return nil
		}
		if err != nil {
			return err
		}
		found = append(found, Config{ID: m.ID, Path: rel, Data: data})
		return nil
	})
	return found, skipped, err
}

var errOverCap = errors.New("over cap")

func readCapped(p string) ([]byte, error) {
	f, err := fsx.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, MaxConfigBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxConfigBytes {
		return nil, errOverCap
	}
	return b, nil
}

// Read opens a .mortar file and returns its preview. Any entry outside the layout, any unsafe path, and any
// config for a mod id the profile does not list fails the whole file.
func Read(file string) (Preview, error) {
	info, err := os.Stat(file)
	if err != nil {
		return Preview{}, err
	}
	if info.Size() > MaxFileBytes {
		return Preview{}, fmt.Errorf("%w: file is larger than %d MiB", ErrBadFile, MaxFileBytes>>20)
	}
	f, err := fsx.Open(file)
	if err != nil {
		return Preview{}, err
	}
	defer func() { _ = f.Close() }()
	zr, err := zip.NewReader(f, info.Size())
	if err != nil {
		return Preview{}, fmt.Errorf("%w: not a zip", ErrBadFile)
	}
	return readZip(zr)
}

// ReadBytes reads a .mortar zip from memory.
func ReadBytes(raw []byte) (Preview, error) {
	if len(raw) > MaxFileBytes {
		return Preview{}, fmt.Errorf("%w: file is larger than %d MiB", ErrBadFile, MaxFileBytes>>20)
	}
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return Preview{}, fmt.Errorf("%w: not a zip", ErrBadFile)
	}
	return readZip(zr)
}

func readBounded(f *zip.File, limit int64) ([]byte, error) {
	if f.UncompressedSize64 > uint64(max(limit, 0)) {
		return nil, fmt.Errorf("%w: %s is over its size cap", ErrBadFile, f.Name)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrBadFile, f.Name, err)
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrBadFile, f.Name, err)
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("%w: %s is over its size cap", ErrBadFile, f.Name)
	}
	return b, nil
}

func readZip(zr *zip.Reader) (Preview, error) {
	if len(zr.File) > MaxConfigFiles+1 {
		return Preview{}, fmt.Errorf("%w: too many entries", ErrBadFile)
	}
	var head *zip.File
	for _, f := range zr.File {
		if f.Name == profileFile {
			head = f
		}
	}
	if head == nil {
		return Preview{}, fmt.Errorf("%w: no %s", ErrBadFile, profileFile)
	}
	raw, err := readBounded(head, maxProfileBytes)
	if err != nil {
		return Preview{}, err
	}
	pv, err := parseFileDoc(raw)
	if err != nil {
		return Preview{}, err
	}
	known := map[string]mod.ID{}
	for _, id := range pv.IDs {
		known[id.Fold()] = id
	}
	seen := map[string]bool{profileFile: true}
	var total int64
	for _, f := range zr.File {
		if f.Name == profileFile || strings.HasSuffix(f.Name, "/") && f.UncompressedSize64 == 0 {
			continue
		}
		if rel, ok := strings.CutPrefix(f.Name, loaderPrefix); ok {
			if !validLoaderConfigPath(rel) {
				return Preview{}, fmt.Errorf("%w: unsafe path %q", ErrBadFile, f.Name)
			}
			key := strings.ToLower(f.Name)
			if seen[key] {
				return Preview{}, fmt.Errorf("%w: duplicate %q", ErrBadFile, f.Name)
			}
			seen[key] = true
			data, err := readBounded(f, MaxConfigBytes)
			if err != nil {
				return Preview{}, err
			}
			if total += int64(len(data)); total > MaxConfigTotal {
				return Preview{}, fmt.Errorf("%w: configs exceed the size cap", ErrBadFile)
			}
			pv.LoaderConfigs = append(pv.LoaderConfigs, LoaderConfig{Path: rel, Data: data})
			continue
		}
		rest, ok := strings.CutPrefix(f.Name, "configs/")
		fmtSeg, rest, ok2 := strings.Cut(rest, "/")
		local, rel, ok3 := strings.Cut(rest, "/")
		if !ok || !ok2 || !ok3 {
			return Preview{}, fmt.Errorf("%w: unexpected entry %q", ErrBadFile, f.Name)
		}
		canon, ok := known[mod.NewID(fmtSeg, local).Fold()]
		if !ok {
			return Preview{}, fmt.Errorf("%w: %q is for a mod the profile does not list", ErrBadFile, f.Name)
		}
		if path.Clean(rel) != rel || !validConfigPath(rel) {
			return Preview{}, fmt.Errorf("%w: unsafe path %q", ErrBadFile, f.Name)
		}
		key := strings.ToLower(string(canon) + "/" + rel)
		if seen[key] {
			return Preview{}, fmt.Errorf("%w: duplicate %q", ErrBadFile, f.Name)
		}
		seen[key] = true
		data, err := readBounded(f, MaxConfigBytes)
		if err != nil {
			return Preview{}, err
		}
		if total += int64(len(data)); total > MaxConfigTotal {
			return Preview{}, fmt.Errorf("%w: configs exceed the size cap", ErrBadFile)
		}
		pv.Configs = append(pv.Configs, Config{ID: canon, Path: rel, Data: data})
	}
	return pv, nil
}

func parseFileDoc(raw []byte) (Preview, error) {
	var d fileDoc
	if err := json.Unmarshal(raw, &d); err != nil || d.Version < 1 {
		return Preview{}, fmt.Errorf("%w: bad %s", ErrBadFile, profileFile)
	}
	if d.Version > FormatVersion {
		return Preview{}, ErrNewerVersion
	}
	if d.Version != FormatVersion {
		return Preview{}, fmt.Errorf("%w: unsupported version %d", ErrBadFile, d.Version)
	}
	entries, err := parseEntries(d.Entries)
	if err != nil {
		return Preview{}, err
	}
	pv := Preview{Name: d.Name, Game: d.Game, SourceKeys: d.SourceKeys, Entries: entries, Notes: d.Notes, Description: d.Description, IDs: d.IDs, Groups: d.Groups}
	if err := checkShared(pv.Shared); err != nil {
		return Preview{}, err
	}
	if utf8.RuneCountInString(d.Notes) > profile.MaxNotes {
		return Preview{}, fmt.Errorf("%w: notes are too long", ErrBadFile)
	}
	if utf8.RuneCountInString(d.Description) > profile.MaxDescription {
		return Preview{}, fmt.Errorf("%w: description is too long", ErrBadFile)
	}
	if len(d.IDs) > MaxConfigFiles {
		return Preview{}, fmt.Errorf("%w: too many mods", ErrBadFile)
	}
	for _, id := range d.IDs {
		if !validID(id) {
			return Preview{}, fmt.Errorf("%w: bad mod id %q", ErrBadFile, id)
		}
	}
	if d.Problems != nil {
		if err := checkProblemChoices(*d.Problems); err != nil {
			return Preview{}, err
		}
		pv.Choices = *d.Problems
	}
	return pv, nil
}
