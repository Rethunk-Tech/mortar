package share

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/bytefmt"
	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/winname"
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

// manifestFile is a mod's own manifest, which a shared config never carries.
const manifestFile = "manifest.json"

var (
	uniqueID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	format   = regexp.MustCompile(`^[a-z][a-z0-9]*$`)
	segment  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._()+~-]*$`)
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
	Favourite     bool
	IDs           []mod.ID
	Configs       []Config
	LoaderConfigs []LoaderConfig
	ChangedFiles  []ChangedFile
	Groups        []FileGroup
	Choices       ProblemChoices
}

type fileDoc struct {
	Version    int               `json:"version"`
	Name       string            `json:"name"`
	Game       string            `json:"game"`
	SourceKeys map[string]string `json:"sourceKeys"`
	Notes      string            `json:"notes"`
	// Description is only read: a file may carry a profile's old short description, which joins the notes.
	Description string          `json:"description,omitempty"`
	Favourite   bool            `json:"favourite,omitempty"`
	Entries     json.RawMessage `json:"entries"`
	IDs         []mod.ID        `json:"ids"`
	Groups      []FileGroup     `json:"groups,omitempty"`
	Problems    *ProblemChoices `json:"problems,omitempty"`
}

func validSegment(s string) bool {
	return len(s) <= maxSegment && segment.MatchString(s) && !strings.HasSuffix(s, ".") &&
		!strings.HasSuffix(s, " ")
}

// ValidFomod reports whether recorded FOMOD choices are within the bounds a share link enforces.
func ValidFomod(f map[string]map[string][]string) bool { return validDetails(Ref{Fomod: f}) }

// winSafe reports whether every segment of the slash path p is a name Windows stores as written. It is checked apart
// from the shape rules so reading can skip such a file, which an older release wrote, instead of refusing the share.
func winSafe(p string) bool {
	for s := range strings.SplitSeq(p, "/") {
		if !winname.Valid(s) {
			return false
		}
	}
	return true
}

// ValidConfigPath reports whether p is a path Apply would write: a slash-separated relative .json path whose every
// segment is plain and valid on Windows.
func ValidConfigPath(p string) bool { return validConfigShape(p) && winSafe(p) }

func validConfigShape(p string) bool {
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

// validID accepts ids whose format and local part are safe to use as zip path segments. Only a device name is
// refused: 0.3.8 accepted an id ending in a dot or space, so share files holding one must still read.
func validID(id mod.ID) bool {
	local := id.Local()
	return format.MatchString(id.Format()) && len(local) <= maxIDLen && uniqueID.MatchString(local) && winname.Valid(strings.TrimRight(local, ". "))
}

// Write writes the profile as a .mortar zip: profile.json with the link's entries plus name and notes,
// the .json files of each enabled mod's folder under configs/<mod id>/, and the text files of the
// loader's config folders under loader/<path in the profile>. modsDir is the profile's mods/ folder, inside the
// profile's own. Config files that are over the caps or have unusual names are skipped and returned as paths.
func Write(w io.Writer, gameID string, p profile.Profile, modsDir string, include ...Include) (skipped []string, err error) {
	inc := DefaultInclude()
	if len(include) > 0 {
		inc = include[0]
	}
	p = p.WithoutDev()
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
		Version: FormatVersion, Name: s.Name, Game: s.Game, SourceKeys: s.SourceKeys, Notes: notes, Favourite: p.Favourite,
		Entries: entries, IDs: []mod.ID{}, Groups: collectFileGroups(p),
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
			found, skip, err := ReadConfigs(modsDir, e.Key, m)
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
	var changed []ChangedFile
	if inc.ConfigFiles && modsDir != "" {
		found, skip, err := readChangedFiles(filepath.Dir(modsDir), gameID)
		if err != nil {
			return nil, err
		}
		changed = found
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
	keptChanged := changed[:0]
	for _, c := range changed {
		total += int64(len(c.Data))
		if total > MaxConfigTotal || len(kept)+len(keptLoader)+len(keptChanged) >= MaxConfigFiles {
			skipped = append(skipped, c.Path)
			continue
		}
		keptChanged = append(keptChanged, c)
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
	for _, c := range keptChanged {
		if err := putFile(zw, changedPrefix+c.Path, c.Data); err != nil {
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
	folder := filepath.Join(modsDir, key)
	if m.Folder != "." {
		if !filepath.IsLocal(filepath.FromSlash(m.Folder)) {
			return nil, nil, fmt.Errorf("mod folder %q leaves its entry", m.Folder)
		}
		folder = filepath.Join(folder, filepath.FromSlash(m.Folder))
	}
	// Read refuses two entries differing only by case, so a case variant of an earlier name is left out.
	folded := map[string]bool{}
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
		if strings.EqualFold(rel, manifestFile) {
			return nil
		}
		data, err := readCapped(p)
		if errors.Is(err, safefs.ErrTooLarge) || !ValidConfigPath(rel) {
			skipped = append(skipped, m.ID.Local()+"/"+rel)
			return nil
		}
		if err != nil {
			return err
		}
		key := strings.ToLower(rel)
		if folded[key] {
			slog.Warn("share: skipped a config whose name differs from an earlier one only by case", "mod", m.ID, "path", rel)
			skipped = append(skipped, m.ID.Local()+"/"+rel)
			return nil
		}
		folded[key] = true
		found = append(found, Config{ID: m.ID, Path: rel, Data: data})
		return nil
	})
	return found, skipped, err
}

func readCapped(p string) ([]byte, error) {
	f, err := safefs.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return safefs.ReadCapped(f, MaxConfigBytes)
}

// Read opens a .mortar file and returns its preview. Any entry outside the layout, any unsafe path, and any
// config for a mod id the profile does not list fails the whole file.
func Read(file string) (Preview, error) {
	info, err := os.Stat(file)
	if err != nil {
		return Preview{}, err
	}
	if info.Size() > MaxFileBytes {
		return Preview{}, fmt.Errorf("%w: file is larger than %s", ErrBadFile, bytefmt.Format(MaxFileBytes))
	}
	f, err := safefs.Open(file)
	if err != nil {
		return Preview{}, err
	}
	defer func() { _ = f.Close() }()
	entries, err := readEntries(f, info.Size())
	if err != nil {
		return Preview{}, err
	}
	return readZip(entries)
}

// ReadBytes reads a .mortar zip from memory.
func ReadBytes(raw []byte) (Preview, error) {
	pv, _, err := readPayload(raw)
	return pv, err
}

// readPayload is ReadBytes that also returns the file's entries, for a caller that rewrites them.
func readPayload(raw []byte) (Preview, []entry, error) {
	if len(raw) > MaxFileBytes {
		return Preview{}, nil, fmt.Errorf("%w: file is larger than %s", ErrBadFile, bytefmt.Format(MaxFileBytes))
	}
	entries, err := readEntries(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return Preview{}, nil, err
	}
	pv, err := readZip(entries)
	return pv, entries, err
}

// entry is one file of a .mortar zip under its cleaned name. data is nil for a skippedEntry, which is never read.
type entry struct {
	name string
	data []byte
}

// readEntries reads a .mortar zip's files in archive order under go-archive's admission rules. Names are judged by
// the layout's own rules afterwards; case is too, since two folders differing only by case read in files earlier
// releases wrote.
func readEntries(src io.ReaderAt, size int64) ([]entry, error) {
	var out []entry
	opts := archive.Options{MaxEntries: MaxConfigFiles + 1, MaxTotalBytes: MaxConfigTotal + maxProfileBytes, CaseSensitive: true}
	err := archive.ZipFunc(src, size, opts, func(name string, isDir bool, r io.Reader) error {
		if isDir {
			return nil
		}
		e := entry{name: name}
		if !skippedEntry(name) {
			limit := int64(MaxConfigBytes)
			if name == profileFile {
				limit = maxProfileBytes
			}
			b, err := safefs.ReadCapped(r, limit)
			if err != nil {
				return fmt.Errorf("%w: %s: %w", ErrBadFile, name, err)
			}
			e.data = b
		}
		out = append(out, e)
		return nil
	})
	if ae, ok := errors.AsType[*archive.Error](err); ok {
		if errors.Is(ae, archive.ErrUnsupportedFormat) {
			return nil, fmt.Errorf("%w: not a zip", ErrBadFile)
		}
		return nil, fmt.Errorf("%w: %s: %w", ErrBadFile, ae.Entry, ae.Reason)
	}
	return out, err
}

func readZip(entries []entry) (Preview, error) {
	i := slices.IndexFunc(entries, func(e entry) bool { return e.name == profileFile })
	if i < 0 {
		return Preview{}, fmt.Errorf("%w: no %s", ErrBadFile, profileFile)
	}
	pv, err := parseFileDoc(entries[i].data)
	if err != nil {
		return Preview{}, err
	}
	known := map[string]mod.ID{}
	for _, id := range pv.IDs {
		known[id.Fold()] = id
	}
	seen := map[string]bool{profileFile: true}
	var total int64
	// read returns one entry's bytes, refusing a second entry under the same key and a file whose entries together
	// pass the size cap.
	read := func(f entry, key string) ([]byte, error) {
		if seen[key] {
			return nil, fmt.Errorf("%w: duplicate %q", ErrBadFile, f.name)
		}
		seen[key] = true
		if total += int64(len(f.data)); total > MaxConfigTotal {
			return nil, fmt.Errorf("%w: configs exceed the size cap", ErrBadFile)
		}
		return f.data, nil
	}
	for _, f := range entries {
		if f.name == profileFile {
			continue
		}
		if rel, ok := strings.CutPrefix(f.name, loaderPrefix); ok {
			if !validLoaderConfigShape(rel) {
				return Preview{}, fmt.Errorf("%w: unsafe path %q", ErrBadFile, f.name)
			}
			if !winSafe(rel) {
				slog.Warn("share: skipped a loader config whose name Windows cannot store", "path", f.name)
				continue
			}
			data, err := read(f, strings.ToLower(f.name))
			if err != nil {
				return Preview{}, err
			}
			pv.LoaderConfigs = append(pv.LoaderConfigs, LoaderConfig{Path: rel, Data: data})
			continue
		}
		if rel, ok := strings.CutPrefix(f.name, changedPrefix); ok {
			// On Windows IsLocal refuses a device name, which would fail the whole file; as on Linux the copy is skipped.
			if !validChangedLocation(pv.Game, rel) && (!hasDeviceSegment(rel) || !validChangedLocation(pv.Game, devicesRenamed(rel))) {
				return Preview{}, fmt.Errorf("%w: unsafe path %q", ErrBadFile, f.name)
			}
			// An older release let a copy of any non-program type travel; the file is still good without it.
			if skippedEntry(f.name) {
				slog.Warn("share: skipped a changed copy of a type no longer carried or a name Windows cannot store", "path", rel)
				continue
			}
			data, err := read(f, strings.ToLower(f.name))
			if err != nil {
				return Preview{}, err
			}
			pv.ChangedFiles = append(pv.ChangedFiles, ChangedFile{Path: rel, Data: data})
			continue
		}
		rest, ok := strings.CutPrefix(f.name, "configs/")
		fmtSeg, rest, ok2 := strings.Cut(rest, "/")
		local, rel, ok3 := strings.Cut(rest, "/")
		if !ok || !ok2 || !ok3 {
			return Preview{}, fmt.Errorf("%w: unexpected entry %q", ErrBadFile, f.name)
		}
		canon, ok := known[mod.NewID(fmtSeg, local).Fold()]
		if !ok {
			return Preview{}, fmt.Errorf("%w: %q is for a mod the profile does not list", ErrBadFile, f.name)
		}
		if path.Clean(rel) != rel || !validConfigShape(rel) {
			return Preview{}, fmt.Errorf("%w: unsafe path %q", ErrBadFile, f.name)
		}
		if !winSafe(rel) {
			slog.Warn("share: skipped a config whose name Windows cannot store", "path", f.name)
			continue
		}
		data, err := read(f, strings.ToLower(string(canon)+"/"+rel))
		if err != nil {
			return Preview{}, err
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
	pv := Preview{Name: d.Name, Game: d.Game, SourceKeys: d.SourceKeys, Entries: entries, Notes: d.Notes, Favourite: d.Favourite, IDs: d.IDs, Groups: d.Groups}
	if err := checkShared(pv.Shared); err != nil {
		return Preview{}, err
	}
	if utf8.RuneCountInString(d.Notes) > profile.MaxNotes {
		return Preview{}, fmt.Errorf("%w: notes are too long", ErrBadFile)
	}
	// The description joins the notes only where they all fit: the player's own notes are never cut for it.
	if folded := profile.FoldDescription(d.Description, pv.Notes); utf8.RuneCountInString(folded) <= profile.MaxNotes {
		pv.Notes = folded
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

// skippedEntry reports an entry reading leaves out, so every rewrite of the file must too: a changed/ entry of a type
// no longer carried, or any config, loader config or changed copy whose name Windows cannot store.
func skippedEntry(name string) bool {
	if rel, ok := strings.CutPrefix(name, changedPrefix); ok {
		return !profile.ChangedTypeAllowed(rel) || !winSafe(rel)
	}
	if rel, ok := strings.CutPrefix(name, loaderPrefix); ok {
		return !winSafe(rel)
	}
	if rest, ok := strings.CutPrefix(name, "configs/"); ok {
		parts := strings.SplitN(rest, "/", 3)
		return len(parts) == 3 && !winSafe(parts[2])
	}
	return false
}
