package profile

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fomod"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

// OverlayAsk turns a RemapAsk into the choice of where an optional file without a manifest goes inside its main
// mod: Tree (on the RemapAsk) is the optional file's folders, Targets the main entry's.
type OverlayAsk struct {
	BaseKey   string      `json:"baseKey"`
	BaseLabel string      `json:"baseLabel"`
	Targets   []RemapNode `json:"targets"`
}

// NoBaseError is a Nexus file without a manifest whose mod has no file with one in the profile.
type NoBaseError struct {
	Archive string
	ModName string
	ModID   int
}

func (e *NoBaseError) Error() string {
	name := cmp.Or(e.ModName, "this mod")
	return fmt.Sprintf("%s has no manifest. Install the main file of %s first, then this optional file goes on top of it.", e.Archive, name)
}

type overlayPlace struct {
	from, to string
	off      bool
}

// WithOverlay returns a copy that lays a manifest-less store item at to inside its main entry, taking its from
// folder, instead of working out where it goes.
func (s Source) WithOverlay(from, to string) Source {
	s.overlay = &overlayPlace{from: from, to: to}
	return s
}

// WithOverlayOff makes an optional file placed by WithOverlay start switched off.
func (s Source) WithOverlayOff(off bool) Source {
	if s.overlay != nil {
		s.overlay = &overlayPlace{from: s.overlay.from, to: s.overlay.to, off: off}
	}
	return s
}

// IsOverlay reports an optional file laid over another entry's folder.
func (e Entry) IsOverlay() bool { return e.OverlayOf != "" }

// overlaysOf lists the entries laid over baseKey, in profile order, which is the order they are applied in.
func overlaysOf(entries []Entry, baseKey string) []Entry {
	var out []Entry
	for _, e := range entries {
		if e.OverlayOf == baseKey && baseKey != "" {
			out = append(out, e)
		}
	}
	return out
}

func overlaysOn(entries []Entry, baseKey string) []Entry {
	return slices.DeleteFunc(overlaysOf(entries, baseKey), func(e Entry) bool { return e.OverlayOff })
}

// overlayBaseFor is the entry holding the manifest of Nexus mod modID, which its optional files lay over.
func overlayBaseFor(entries []Entry, modID int) (Entry, bool) {
	for _, e := range entries {
		if !e.IsOverlay() && e.Source.Kind == KindNexus && e.Source.ModID == modID && len(e.Mods) > 0 {
			return e, true
		}
	}
	return Entry{}, false
}

func hasManifestFile(root string) (bool, error) {
	found := false
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(d.Name(), manifest.FileName) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found, err
}

// isOverlayItem reports a Nexus store item with no manifest anywhere and no installer: an optional file that
// replaces files of its mod's main file.
func (s *Store) isOverlayItem(game, key string, source Source) (bool, error) {
	if source.Kind != KindNexus || source.ModID <= 0 {
		return false, nil
	}
	dir, err := s.items.Path(game, key)
	if err != nil {
		return false, err
	}
	// A recorded mod root means the item once held a manifest; its folder going missing asks again instead.
	if contentRootFile(dir) != "" {
		return false, nil
	}
	if _, _, ok, err := fomod.Open(dir); err != nil || ok {
		return false, err
	}
	has, err := hasManifestFile(dir)
	return !has, err
}

// overlayFiles lists the files under root by slash path, skipping dot-named ones and the store's own markers.
func overlayFiles(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == root {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || (filepath.Dir(p) == root && d.Name() == store.RootFile) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}

func overlayRel(rel string) (string, error) {
	rel = strings.Trim(filepath.ToSlash(rel), "/")
	if rel == "" || rel == "." {
		return "", nil
	}
	if !filepath.IsLocal(filepath.FromSlash(rel)) {
		return "", fmt.Errorf("overlay path %q leaves its folder", rel)
	}
	return path.Clean(rel), nil
}

// overlaySource is the folder of o's store item that is laid over its base, and the files in it.
func (s *Store) overlaySource(game string, o Entry) (string, []string, error) {
	dir, err := s.items.Path(game, o.Key)
	if err != nil {
		return "", nil, err
	}
	from, err := overlayRel(o.OverlayFrom)
	if err != nil {
		return "", nil, err
	}
	root := filepath.Join(dir, filepath.FromSlash(from))
	files, err := overlayFiles(root)
	return root, files, err
}

// mapOverlay works out which folder of an optional file (files, by slash path) goes where inside its base (the base
// entry's laid-out files, and the folders of its mods). It reports false when no layout fits well enough to choose
// without asking.
func mapOverlay(files, base []string, mods []Component) (from, to string, ok bool) {
	if len(files) == 0 {
		return "", "", false
	}
	have := map[string]bool{}
	for _, f := range base {
		have[strings.ToLower(f)] = true
	}
	tos := []string{""}
	for _, m := range mods {
		if m.Folder != "." && m.Folder != "" && !slices.Contains(tos, m.Folder) {
			tos = append(tos, m.Folder)
		}
	}
	wrapper := singleTop(files)
	if wrapper != "" {
		for _, t := range tos[1:] {
			if strings.EqualFold(path.Base(t), wrapper) {
				return wrapper, t, true
			}
		}
	}
	froms := []string{""}
	if wrapper != "" {
		froms = append(froms, wrapper)
	}
	best, bestFrom, bestTo, total := 0, "", "", 0
	for _, f := range froms {
		under := stripTop(files, f)
		for _, t := range tos {
			n := 0
			for _, rel := range under {
				if have[strings.ToLower(path.Join(t, rel))] {
					n++
				}
			}
			if n > best {
				best, bestFrom, bestTo, total = n, f, t, len(under)
			}
		}
	}
	if best*2 > total {
		return bestFrom, bestTo, true
	}
	if wrapper == "" {
		return "", "", false
	}
	tops := map[string]bool{}
	for _, rel := range base {
		for _, t := range tos {
			if rest, ok := strings.CutPrefix(rel, t+"/"); ok || t == "" {
				if t == "" {
					rest = rel
				}
				first, _, _ := strings.Cut(rest, "/")
				tops[strings.ToLower(t+"\x00"+first)] = true
			}
		}
	}
	for _, t := range tos {
		for _, rel := range stripTop(files, wrapper) {
			first, _, _ := strings.Cut(rel, "/")
			if tops[strings.ToLower(t+"\x00"+first)] {
				return wrapper, t, true
			}
		}
	}
	return "", "", false
}

// singleTop is the one folder every file sits under, or empty when the files start at more than one name.
func singleTop(files []string) string {
	top := ""
	for _, f := range files {
		first, _, nested := strings.Cut(f, "/")
		if !nested || (top != "" && first != top) {
			return ""
		}
		top = first
	}
	return top
}

func stripTop(files []string, top string) []string {
	if top == "" {
		return files
	}
	out := make([]string, 0, len(files))
	for _, f := range files {
		if rest, ok := strings.CutPrefix(f, top+"/"); ok {
			out = append(out, rest)
		}
	}
	return out
}

// livePath is rel inside an entry folder, following the dot names of switched-off mod folders on the way.
func livePath(entryDir, rel string) string {
	cur := entryDir
	parts := strings.Split(rel, "/")
	for i, part := range parts {
		next := filepath.Join(cur, part)
		if i < len(parts)-1 && !exists(next) && exists(filepath.Join(cur, "."+part)) {
			next = filepath.Join(cur, "."+part)
		}
		cur = next
	}
	return cur
}

// layOverlays rewrites the overlaid paths of base's folder at entryDir: every path an overlay in was or on covers
// gets the base's own file back (or goes, when the base has none), then each overlay in on is laid over it in
// order, so a later one wins. Replaced files are new names, so the shared store items are never written.
func (s *Store) layOverlays(game, id string, base Entry, entryDir string, was, on []Entry) error {
	if len(was) == 0 && len(on) == 0 {
		return nil
	}
	baseSrc, tmp, err := s.layoutItem(game, id, base.Key, base.Fomod)
	if tmp != "" {
		defer func() { _ = fsx.RemoveAll(tmp) }()
	}
	if err != nil {
		return err
	}
	type laid struct {
		root, to string
		files    []string
	}
	touched := map[string]bool{}
	var apply []laid
	for i, o := range slices.Concat(was, on) {
		root, files, err := s.overlaySource(game, o)
		if err != nil {
			return err
		}
		to, err := overlayRel(o.OverlayTo)
		if err != nil {
			return err
		}
		for _, f := range files {
			touched[path.Join(to, f)] = true
		}
		if i >= len(was) {
			apply = append(apply, laid{root: root, to: to, files: files})
		}
	}
	for rel := range touched {
		if err := restoreBaseFile(entryDir, baseSrc, rel); err != nil {
			return err
		}
	}
	for _, l := range apply {
		for _, f := range l.files {
			rel := path.Join(l.to, f)
			dst := livePath(entryDir, rel)
			if err := os.Remove(dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
				return err
			}
			if err := datadir.MaterializeFile(filepath.Join(l.root, filepath.FromSlash(f)), dst, rel); err != nil {
				return err
			}
		}
	}
	return nil
}

func restoreBaseFile(entryDir, baseSrc, rel string) error {
	dst := livePath(entryDir, rel)
	if err := os.Remove(dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	src := filepath.Join(baseSrc, filepath.FromSlash(rel))
	if regularFile(src) {
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return err
		}
		return datadir.MaterializeFile(src, dst, rel)
	}
	pruneEmptyDirs(entryDir, filepath.Dir(dst))
	return nil
}

// pruneEmptyDirs drops the folders only an overlay brought once they are empty, from d up to entryDir; os.Remove
// leaves a folder that still holds anything.
func pruneEmptyDirs(entryDir, d string) {
	for ; d != entryDir && strings.HasPrefix(d, entryDir+string(filepath.Separator)); d = filepath.Dir(d) {
		if os.Remove(d) != nil {
			return
		}
	}
}

func regularFile(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

// relayBase lays base's overlays in p over its folder again, after was were the ones laid before.
func (s *Store) relayBase(game string, p *Profile, dir, baseKey string, was []Entry) error {
	bi := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == baseKey && !e.IsOverlay() })
	if bi < 0 {
		return nil
	}
	base := p.Entries[bi]
	return s.layOverlays(game, p.ID, base, liveEntryDir(filepath.Join(dir, "mods"), base.Key), was, overlaysOn(p.Entries, base.Key))
}

func overlayAsk(key string, source Source, base Entry, itemDir, baseSrc string) (RemapAsk, error) {
	tree, err := remapTree(itemDir)
	if err != nil {
		return RemapAsk{}, err
	}
	targets, err := remapTree(baseSrc)
	if err != nil {
		return RemapAsk{}, err
	}
	return RemapAsk{Key: key, Source: source, Tree: tree, Overlay: &OverlayAsk{
		BaseKey: base.Key, BaseLabel: entryLabel(base), Targets: foldersOnly(targets),
	}}, nil
}

func foldersOnly(nodes []RemapNode) []RemapNode {
	out := []RemapNode{}
	for _, n := range nodes {
		if n.Dir {
			n.Children = foldersOnly(n.Children)
			out = append(out, n)
		}
	}
	return out
}

// placeOverlayLocked adds the manifest-less Nexus store item key to the profile as an optional file of the entry
// holding its mod's manifest, laid over that entry's folder. Where it goes comes from the source's overlay place, or
// from matching its files to the base's; when neither settles it, the user is asked.
func (s *Store) placeOverlayLocked(game, id, key string, source Source) (Profile, error) {
	cur, err := s.read(game, id)
	if err != nil {
		return Profile{}, err
	}
	itemDir, err := s.items.Path(game, key)
	if err != nil {
		return Profile{}, err
	}
	base, ok := overlayBaseFor(cur.Entries, source.ModID)
	if !ok {
		if raw, err := hasRawXNB(itemDir); err == nil && raw {
			return Profile{}, &rawXNBError{Key: key}
		}
		return Profile{}, &NoBaseError{Archive: source.Name, ModName: source.ModName, ModID: source.ModID}
	}
	for _, e := range cur.Entries {
		if e.Key == key {
			return Profile{}, &DuplicateError{Key: key, Label: entryLabel(e)}
		}
	}
	replaced := func(e Entry) bool {
		return source.replacing > 0 && e.OverlayOf == base.Key && e.Source.FileID == source.replacing
	}
	// A newer version of an installed optional file goes where the old one went when it still has that folder.
	if old := slices.IndexFunc(cur.Entries, replaced); old >= 0 && source.overlay == nil {
		o := cur.Entries[old]
		if info, err := os.Stat(filepath.Join(itemDir, filepath.FromSlash(o.OverlayFrom))); err == nil && info.IsDir() {
			source = source.WithOverlay(o.OverlayFrom, o.OverlayTo)
		}
	}
	from, to, err := s.overlayTarget(game, id, key, source, base, itemDir)
	if err != nil {
		return Profile{}, err
	}
	off := source.overlay != nil && source.overlay.off
	place := source
	place.fomod, place.disabled, place.overlay, place.replacing = nil, nil, nil, 0
	p, err := s.updateLocked(game, id, func(p *Profile, dir string) error {
		was := overlaysOf(p.Entries, base.Key)
		if i := slices.IndexFunc(p.Entries, replaced); i >= 0 {
			e := &p.Entries[i]
			e.Key, e.Source, e.OverlayFrom, e.OverlayTo = key, place, from, to
			return s.relayBase(game, p, dir, base.Key, was)
		}
		p.Entries = append(p.Entries, Entry{
			Key: key, Source: place, Mods: []Component{}, Disabled: []mod.ID{}, Added: time.Now().UTC(),
			OverlayOf: base.Key, OverlayFrom: from, OverlayTo: to, OverlayOff: off,
		})
		if !off {
			if err := s.offAlternatives(game, p, key); err != nil {
				return err
			}
		}
		return s.relayBase(game, p, dir, base.Key, was)
	})
	if err != nil {
		return Profile{}, err
	}
	return p, s.items.Touch(game, key)
}

func (s *Store) overlayTarget(game, id, key string, source Source, base Entry, itemDir string) (string, string, error) {
	if source.overlay != nil {
		from, err := overlayRel(source.overlay.from)
		if err != nil {
			return "", "", err
		}
		to, err := overlayRel(source.overlay.to)
		if err != nil {
			return "", "", err
		}
		if info, err := os.Stat(filepath.Join(itemDir, filepath.FromSlash(from))); err != nil || !info.IsDir() {
			return "", "", fmt.Errorf("%q is not a folder of this file", from)
		}
		return from, to, nil
	}
	files, err := overlayFiles(itemDir)
	if err != nil {
		return "", "", err
	}
	baseSrc, tmp, err := s.layoutItem(game, id, base.Key, base.Fomod)
	if tmp != "" {
		defer func() { _ = fsx.RemoveAll(tmp) }()
	}
	if err != nil {
		return "", "", err
	}
	baseFiles, err := overlayFiles(baseSrc)
	if err != nil {
		return "", "", err
	}
	if from, to, ok := mapOverlay(files, baseFiles, base.Mods); ok {
		return from, to, nil
	}
	ask, err := overlayAsk(key, source, base, itemDir, baseSrc)
	if err != nil {
		return "", "", err
	}
	return "", "", &NeedRootError{Ask: ask}
}

// InstallOverlay adds the store item key as an optional file of its main entry, taking its from folder and
// laying it at to inside the main entry's folder: the answer to an overlay RemapAsk.
func (s *Store) InstallOverlay(game, id, key, from, to string, source Source) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	return s.installKey(game, id, key, source.WithOverlay(from, to))
}

// SetOverlayEnabled switches an optional file on or off; off puts the main entry's own files back.
func (s *Store) SetOverlayEnabled(game, id, key string, enabled bool) (Profile, error) {
	return s.updateModsRecorded(game, id, func(p *Profile, dir string) error {
		return s.setOverlayLocked(game, p, dir, key, enabled)
	})
}

func (s *Store) setOverlayLocked(game string, p *Profile, dir, key string, enabled bool) error {
	i, err := requireEntry(p.Entries, key)
	if err != nil {
		return err
	}
	if !p.Entries[i].IsOverlay() {
		return fmt.Errorf("%q is not an optional file", key)
	}
	was := overlaysOf(p.Entries, p.Entries[i].OverlayOf)
	p.Entries[i].OverlayOff = !enabled
	if enabled {
		if err := s.offAlternatives(game, p, key); err != nil {
			return err
		}
	}
	return s.relayBase(game, p, dir, p.Entries[i].OverlayOf, was)
}

func (s *Service) InstallOverlay(game, id, key, from, to string, source Source) (InstallResult, error) {
	return s.store.InstallOverlay(game, id, key, from, to, source)
}

func (s *Service) SetOverlayEnabled(game, id, key string, enabled bool) (Profile, error) {
	return s.store.SetOverlayEnabled(game, id, key, enabled)
}

func isOverlayKey(p *Profile, key string) bool {
	return key != "" && slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == key && e.IsOverlay() })
}

// restoreOverlay puts a removed optional file back as it was, once its main entry is in the profile again.
func (s *Store) restoreOverlay(game string, p *Profile, dir string, want Entry) error {
	if !slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == want.OverlayOf && !e.IsOverlay() }) {
		return fmt.Errorf("%s needs its main file in the profile", entryLabel(want))
	}
	if _, err := s.items.Path(game, want.Key); err != nil {
		return err
	}
	was := overlaysOf(p.Entries, want.OverlayOf)
	p.Entries = append(p.Entries, want)
	return s.relayBase(game, p, dir, want.OverlayOf, was)
}

// basesFirst orders entries so each optional file comes after the entries it may be laid over.
func basesFirst(entries []Entry) []Entry {
	out := slices.Clone(entries)
	slices.SortStableFunc(out, func(a, b Entry) int {
		switch {
		case a.IsOverlay() == b.IsOverlay():
			return 0
		case a.IsOverlay():
			return 1
		default:
			return -1
		}
	})
	return out
}

// swapOverlaid is swapEntry for p.Entries[ei] with its optional files taken off first, so the update does not carry
// them over as the user's edits, and laid over the new version afterwards. They then name the new key.
func (s *Store) swapOverlaid(game string, p *Profile, dir string, ei int, newKey string, source *Source) (Entry, swapped, error) {
	e := p.Entries[ei]
	overs := overlaysOf(p.Entries, e.Key)
	modsDir := filepath.Join(dir, "mods")
	if len(overs) > 0 {
		if err := s.layOverlays(game, p.ID, e, liveEntryDir(modsDir, e.Key), overs, nil); err != nil {
			return Entry{}, swapped{}, err
		}
	}
	ne, w, err := s.swapEntry(game, p.ID, dir, e, newKey, source)
	if err != nil {
		if len(overs) > 0 && w.placed == "" {
			err = errors.Join(err, s.layOverlays(game, p.ID, e, liveEntryDir(modsDir, e.Key), nil, overlaysOn(p.Entries, e.Key)))
		}
		return ne, w, err
	}
	if len(overs) == 0 {
		return ne, w, nil
	}
	for i := range p.Entries {
		if p.Entries[i].OverlayOf == e.Key {
			p.Entries[i].OverlayOf = newKey
		}
	}
	return ne, w, s.layOverlays(game, p.ID, ne, liveEntryDir(modsDir, newKey), nil, overlaysOn(p.Entries, newKey))
}

// StoreKey is the store item the entry's files come from.
func (e Entry) StoreKey() string { return cmp.Or(e.Item, e.Key) }
