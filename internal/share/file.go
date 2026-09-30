package share

import (
	"archive/zip"
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

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

// Caps on a .mortar file, enforced by counting bytes read rather than trusting declared sizes.
const (
	MaxFileBytes    = 32 << 20
	MaxConfigBytes  = 1 << 20
	MaxConfigTotal  = 32 << 20
	MaxConfigFiles  = 5000
	maxProfileBytes = 256 << 10
	maxUniqueID     = 100
	maxSegment      = 100
	maxRelPath      = 240
)

const profileFile = "profile.json"

var (
	uniqueID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	segment  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._()+~-]*$`)
	reserved = regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])(\..*)?$`)
)

// ErrBadFile means a .mortar file is not a valid share; the message names the offending entry.
var ErrBadFile = errors.New("not a valid .mortar file")

// Config is one config file to write into an installed mod's folder: Path is relative to the mod folder, slash
// separated, ends in .json and stays inside it.
type Config struct {
	UniqueID string `json:"uniqueId"`
	Path     string `json:"path"`
	Data     []byte `json:"data"`
}

// Preview is what reading a .mortar file yields. The apply step writes Configs only into installed mod folders.
type Preview struct {
	Shared
	Notes     string
	UniqueIDs []string
	Configs   []Config
}

type fileDoc struct {
	Version   int             `json:"version"`
	Name      string          `json:"name"`
	Notes     string          `json:"notes"`
	Entries   json.RawMessage `json:"entries"`
	UniqueIDs []string        `json:"uniqueIds"`
}

func validSegment(s string) bool {
	return len(s) <= maxSegment && segment.MatchString(s) && !strings.HasSuffix(s, ".") &&
		!strings.HasSuffix(s, " ") && !reserved.MatchString(s)
}

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

func validUniqueID(id string) bool {
	return len(id) <= maxUniqueID && uniqueID.MatchString(id) && !reserved.MatchString(id)
}

// Write writes the profile as a .mortar zip: profile.json with the link's entries plus name and notes, and the
// .json files of each enabled mod's folder under configs/<UniqueID>/. modsDir is the profile's mods/ folder.
// Config files that are over the caps or have unusual names are skipped and returned as paths.
func Write(w io.Writer, p profile.Profile, modsDir string) (skipped []string, err error) {
	s, _, _ := Collect(p)
	zw := zip.NewWriter(w)
	entries, err := json.Marshal(s.Entries)
	if err != nil {
		return nil, err
	}
	doc := fileDoc{Version: FormatVersion, Name: s.Name, Notes: p.Notes, Entries: entries, UniqueIDs: []string{}}
	if err := checkShared(s); err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(p.Notes) > profile.MaxNotes {
		return nil, fmt.Errorf("%w: notes are too long", ErrBadFile)
	}
	var configs []Config
	for _, e := range p.Entries {
		if bundled(e) || !enabled(e) {
			continue
		}
		for _, m := range e.Mods {
			if containsFold(e.Disabled, m.UniqueID) || !validUniqueID(m.UniqueID) {
				continue
			}
			doc.UniqueIDs = append(doc.UniqueIDs, m.UniqueID)
			found, skip, err := readConfigs(modsDir, e.Key, m)
			if err != nil {
				return nil, err
			}
			configs = append(configs, found...)
			skipped = append(skipped, skip...)
		}
	}
	var total int64
	kept := configs[:0]
	for _, c := range configs {
		total += int64(len(c.Data))
		if total > MaxConfigTotal || len(kept) >= MaxConfigFiles {
			skipped = append(skipped, c.UniqueID+"/"+c.Path)
			continue
		}
		kept = append(kept, c)
	}
	head, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	if err := putFile(zw, profileFile, head); err != nil {
		return nil, err
	}
	for _, c := range kept {
		if err := putFile(zw, "configs/"+c.UniqueID+"/"+c.Path, c.Data); err != nil {
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

// readConfigs collects the .json files under one enabled mod's folder, except its manifest.
func readConfigs(modsDir, key string, m profile.EntryMod) (found []Config, skipped []string, err error) {
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
		if strings.EqualFold(rel, "manifest.json") {
			return nil
		}
		data, err := readCapped(p)
		if errors.Is(err, errOverCap) || !validConfigPath(rel) {
			skipped = append(skipped, m.UniqueID+"/"+rel)
			return nil
		}
		if err != nil {
			return err
		}
		found = append(found, Config{UniqueID: m.UniqueID, Path: rel, Data: data})
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
// config for a UniqueID the profile does not list fails the whole file.
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
	known := map[string]string{}
	for _, id := range pv.UniqueIDs {
		known[strings.ToLower(id)] = id
	}
	seen := map[string]bool{profileFile: true}
	var total int64
	for _, f := range zr.File {
		if f.Name == profileFile || strings.HasSuffix(f.Name, "/") && f.UncompressedSize64 == 0 {
			continue
		}
		rest, ok := strings.CutPrefix(f.Name, "configs/")
		id, rel, ok2 := strings.Cut(rest, "/")
		if !ok || !ok2 {
			return Preview{}, fmt.Errorf("%w: unexpected entry %q", ErrBadFile, f.Name)
		}
		canon, ok := known[strings.ToLower(id)]
		if !ok {
			return Preview{}, fmt.Errorf("%w: %q is for a mod the profile does not list", ErrBadFile, f.Name)
		}
		if path.Clean(rel) != rel || !validConfigPath(rel) {
			return Preview{}, fmt.Errorf("%w: unsafe path %q", ErrBadFile, f.Name)
		}
		key := strings.ToLower(canon + "/" + rel)
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
		pv.Configs = append(pv.Configs, Config{UniqueID: canon, Path: rel, Data: data})
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
	entries, err := parseEntries(d.Entries)
	if err != nil {
		return Preview{}, err
	}
	pv := Preview{Name: d.Name, Entries: entries, Notes: d.Notes, UniqueIDs: d.UniqueIDs}
	if err := checkShared(pv.Shared); err != nil {
		return Preview{}, err
	}
	if utf8.RuneCountInString(d.Notes) > profile.MaxNotes {
		return Preview{}, fmt.Errorf("%w: notes are too long", ErrBadFile)
	}
	if len(d.UniqueIDs) > MaxConfigFiles {
		return Preview{}, fmt.Errorf("%w: too many mods", ErrBadFile)
	}
	for _, id := range d.UniqueIDs {
		if !validUniqueID(id) {
			return Preview{}, fmt.Errorf("%w: bad UniqueID %q", ErrBadFile, id)
		}
	}
	return pv, nil
}
