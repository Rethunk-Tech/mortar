package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/Rethunk-Tech/mortar/internal/winname"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

const (
	modsSnapshotFile = "mods-snapshot.json"
	modsHoldDir      = "mods-hold"
)

// DriftKind is one outside-edit finding in a profile's mods folder.
type DriftKind string

const (
	DriftUnknown DriftKind = "unknown"
	DriftDeleted DriftKind = "deleted"
	DriftChanged DriftKind = "changed"
)

// FolderStat is the cheap fingerprint of one top-level mods folder, ignoring user-written files.
type FolderStat struct {
	Files  int   `json:"files"`
	Size   int64 `json:"size"`
	Newest int64 `json:"newest"`
}

// ModsSnapshot is stored beside profile.json after a matching install or an explicit keep.
type ModsSnapshot struct {
	Folders map[string]FolderStat `json:"folders"`
}

// Drift is one unknown, deleted, or changed top-level mods folder.
type Drift struct {
	Kind   DriftKind `json:"kind"`
	Folder string    `json:"folder"`
	Key    string    `json:"key"`
}

// DriftBaseline identifies the recorded mods snapshot that drift is measured against; it changes whenever Mortar
// records a new one, so a caller can tell a remembered drift scan is out of date.
func (s *Store) DriftBaseline(game, id string) string {
	dir, err := s.ProfileDir(game, id)
	if err != nil {
		return ""
	}
	info, err := os.Stat(snapshotPath(dir))
	if err != nil {
		return ""
	}
	return strconv.FormatInt(info.ModTime().UnixNano(), 10) + "/" + strconv.FormatInt(info.Size(), 10)
}

func snapshotPath(profileDir string) string {
	return filepath.Join(profileDir, modsSnapshotFile)
}

func holdDir(profileDir string) string {
	return filepath.Join(profileDir, modsHoldDir)
}

func loadSnapshot(profileDir string) (ModsSnapshot, error) {
	var snap ModsSnapshot
	found, err := datadir.ReadJSON(snapshotPath(profileDir), &snap)
	if err != nil {
		return ModsSnapshot{}, err
	}
	if !found {
		return ModsSnapshot{Folders: map[string]FolderStat{}}, nil
	}
	if snap.Folders == nil {
		snap.Folders = map[string]FolderStat{}
	}
	return snap, nil
}

func writeSnapshot(profileDir string, snap ModsSnapshot) error {
	if snap.Folders == nil {
		snap.Folders = map[string]FolderStat{}
	}
	return datadir.WriteJSON(snapshotPath(profileDir), snap)
}

func entryFolderName(name string) string {
	return strings.TrimPrefix(name, ".")
}

func isUserWritten(rel string) bool {
	return datadir.WritableRel(rel)
}

// undotDirs drops the leading dot Mortar adds to a switched-off mod's folder, so its files still match the
// store copy, which keeps the plain name.
func undotDirs(rel string) string {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for i := range len(parts) - 1 {
		parts[i] = strings.TrimPrefix(parts[i], ".")
	}
	return filepath.FromSlash(strings.Join(parts, "/"))
}

func walkFolderStat(root, peer string) (FolderStat, error) {
	var inPeer map[string]struct{}
	if peer != "" && peer != root {
		var err error
		if inPeer, err = storeFiles(peer); err != nil {
			return FolderStat{}, err
		}
	}
	prefix := root + string(filepath.Separator)
	var st FolderStat
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := strings.TrimPrefix(path, prefix)
		if isUserWritten(rel) {
			return nil
		}
		if inPeer != nil {
			if _, ok := inPeer[rel]; !ok {
				if _, ok := inPeer[undotDirs(rel)]; !ok {
					return nil
				}
			}
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		st.Files++
		st.Size += info.Size()
		st.Newest = max(st.Newest, info.ModTime().UnixNano())
		return nil
	})
	return st, err
}

// storeListing is a store item's file list and stats. Store items do not change once installed, so
// they are listed once per process (again only if the item's folder is replaced) instead of every scan.
type storeListing struct {
	modTime int64
	files   map[string]struct{}
	stat    FolderStat
}

var storeListings = struct {
	sync.Mutex
	byPath map[string]storeListing
}{byPath: map[string]storeListing{}}

func storeListingFor(peer string) (storeListing, error) {
	info, err := os.Stat(peer)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return storeListing{files: map[string]struct{}{}}, nil
		}
		return storeListing{}, err
	}
	stamp := info.ModTime().UnixNano()
	storeListings.Lock()
	cached, ok := storeListings.byPath[peer]
	storeListings.Unlock()
	if ok && cached.modTime == stamp {
		return cached, nil
	}
	files, err := relFiles(peer)
	if err != nil {
		return storeListing{}, err
	}
	st, err := walkFolderStat(peer, peer)
	if err != nil {
		return storeListing{}, err
	}
	listing := storeListing{modTime: stamp, files: files, stat: st}
	storeListings.Lock()
	storeListings.byPath[peer] = listing
	storeListings.Unlock()
	return listing, nil
}

func storeFiles(peer string) (map[string]struct{}, error) {
	listing, err := storeListingFor(peer)
	return listing.files, err
}

// relFiles lists the files under root by path relative to it, from directory entries alone.
func relFiles(root string) (map[string]struct{}, error) {
	prefix := root + string(filepath.Separator)
	out := map[string]struct{}{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path == root {
				return filepath.SkipAll
			}
			return err
		}
		if !d.IsDir() {
			out[strings.TrimPrefix(path, prefix)] = struct{}{}
		}
		return nil
	})
	return out, err
}

func liveFolders(modsDir, heldDir string) (map[string]string, error) {
	names := map[string]string{}
	for _, root := range []string{modsDir, heldDir} {
		ents, err := os.ReadDir(root)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasPrefix(name, tempPrefix) {
				continue
			}
			key := entryFolderName(name)
			if _, ok := names[key]; !ok {
				names[key] = name
			}
		}
	}
	return names, nil
}

func folderStatAt(modsDir, heldDir, name, peer string) (FolderStat, error) {
	for _, root := range []string{modsDir, heldDir} {
		p := filepath.Join(root, name)
		if exists(p) {
			return walkFolderStat(p, peer)
		}
	}
	return FolderStat{}, nil
}

func storePeer(s *Store, game, key string) string {
	if s == nil || s.items == nil || key == "" {
		return ""
	}
	p, err := s.items.Path(game, key)
	if err != nil {
		return ""
	}
	return p
}

func scanDrift(names map[string]string, stats map[string]FolderStat, keys []string, snap ModsSnapshot) []Drift {
	known := map[string]struct{}{}
	for _, k := range keys {
		if k != "" {
			known[k] = struct{}{}
		}
	}
	var out []Drift
	for key, folder := range names {
		if _, ok := known[key]; !ok {
			out = append(out, Drift{Kind: DriftUnknown, Folder: folder, Key: key})
			continue
		}
		prev, ok := snap.Folders[key]
		if !ok {
			continue
		}
		cur := stats[key]
		if cur.Files != prev.Files || cur.Size != prev.Size || cur.Newest != prev.Newest {
			out = append(out, Drift{Kind: DriftChanged, Folder: folder, Key: key})
		}
	}
	for key := range known {
		if _, ok := names[key]; !ok {
			out = append(out, Drift{Kind: DriftDeleted, Folder: key, Key: key})
		}
	}
	slices.SortFunc(out, func(a, b Drift) int {
		if a.Kind != b.Kind {
			return strings.Compare(string(a.Kind), string(b.Kind))
		}
		return strings.Compare(a.Key, b.Key)
	})
	return out
}

func entryKeys(p Profile) []string {
	keys := make([]string, 0, len(p.Entries))
	for _, e := range p.Entries {
		if e.Key != "" && !e.IsOverlay() {
			keys = append(keys, e.Key)
		}
	}
	return keys
}

func safeFolder(name string) (string, error) {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." {
		return "", errors.New("invalid folder")
	}
	if !winname.Valid(name) {
		return "", errors.New("invalid folder")
	}
	return name, nil
}

func (s *Store) missingEntryFolders(game, id string) ([]string, error) {
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return nil, err
	}
	modsDir := filepath.Join(dir, "mods")
	var missing []string
	for _, key := range entryKeys(p) {
		if exists(filepath.Join(modsDir, key)) || exists(filepath.Join(modsDir, "."+key)) {
			continue
		}
		missing = append(missing, key)
	}
	return missing, nil
}

func (s *Store) parkUnknownModsLocked(dir string, p Profile) error {
	known := map[string]struct{}{}
	for _, key := range entryKeys(p) {
		known[key] = struct{}{}
	}
	modsDir := filepath.Join(dir, "mods")
	ents, err := os.ReadDir(modsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	hold := holdDir(dir)
	for _, e := range ents {
		if !e.IsDir() || strings.HasPrefix(e.Name(), tempPrefix) {
			continue
		}
		if _, ok := known[entryFolderName(e.Name())]; ok {
			continue
		}
		if err := os.MkdirAll(hold, 0o700); err != nil {
			return err
		}
		dst := uniqueHoldPath(hold, e.Name())
		if err := fsx.Rename(filepath.Join(modsDir, e.Name()), dst); err != nil {
			return err
		}
	}
	return nil
}

func uniqueHoldPath(dir, name string) string {
	dst := filepath.Join(dir, name)
	if !exists(dst) {
		return dst
	}
	for n := 2; ; n++ {
		dst = filepath.Join(dir, fmt.Sprintf("%s-%d", name, n))
		if !exists(dst) {
			return dst
		}
	}
}

func (s *Store) unplaceKeys(game, id string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return err
	}
	modsDir := filepath.Join(dir, "mods")
	for _, key := range keys {
		if err := removeEntryFolders(modsDir, key); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) liveDriftState(game, id string) (Profile, string, map[string]string, map[string]FolderStat, error) {
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return Profile{}, "", nil, nil, err
	}
	modsDir := filepath.Join(dir, "mods")
	held := holdDir(dir)
	names, err := liveFolders(modsDir, held)
	if err != nil {
		return Profile{}, "", nil, nil, err
	}
	stats := map[string]FolderStat{}
	for key, folder := range names {
		st, err := folderStatAt(modsDir, held, folder, storePeer(s, game, key))
		if err != nil {
			return Profile{}, "", nil, nil, err
		}
		stats[key] = st
	}
	return p, dir, names, stats, nil
}

func reconcileSnapshot(snap ModsSnapshot, keys []string, stats, storeStats map[string]FolderStat) ModsSnapshot {
	if snap.Folders == nil {
		snap.Folders = map[string]FolderStat{}
	}
	if len(snap.Folders) == 0 {
		for _, key := range keys {
			if st, ok := stats[key]; ok {
				snap.Folders[key] = st
			}
		}
		return snap
	}
	for _, key := range keys {
		st, ok := stats[key]
		if !ok {
			continue
		}
		if _, have := snap.Folders[key]; !have {
			snap.Folders[key] = st
			continue
		}
		ss, ok := storeStats[key]
		if ok && st == ss {
			snap.Folders[key] = st
		}
	}
	return snap
}

// RecordModsSnapshot writes the cheap mods-folder fingerprint beside profile.json.
func (s *Store) RecordModsSnapshot(game, id string) error {
	p, dir, _, stats, err := s.liveDriftState(game, id)
	if err != nil {
		return err
	}
	folders := map[string]FolderStat{}
	for _, key := range entryKeys(p) {
		if st, ok := stats[key]; ok {
			folders[key] = st
		}
	}
	return writeSnapshot(dir, ModsSnapshot{Folders: folders})
}

// ScanModsDrift compares the profile's mods folder with the last recorded snapshot.
func (s *Store) ScanModsDrift(game, id string) ([]Drift, error) {
	p, dir, names, stats, err := s.liveDriftState(game, id)
	if err != nil {
		return nil, err
	}
	keys := entryKeys(p)
	snap, err := loadSnapshot(dir)
	if err != nil {
		return nil, err
	}
	storeStats := map[string]FolderStat{}
	for _, key := range keys {
		if peer := storePeer(s, game, key); peer != "" {
			listing, err := storeListingFor(peer)
			if err != nil {
				return nil, err
			}
			storeStats[key] = listing.stat
		}
	}
	before := maps.Clone(snap.Folders)
	snap = reconcileSnapshot(snap, keys, stats, storeStats)
	if !maps.Equal(before, snap.Folders) {
		if err := writeSnapshot(dir, snap); err != nil {
			return nil, err
		}
	}
	return scanDrift(names, stats, keys, snap), nil
}

func (s *Store) refreshSnapshotKey(game, id, key string) error {
	if key == "" {
		return errors.New("empty key")
	}
	_, dir, _, stats, err := s.liveDriftState(game, id)
	if err != nil {
		return err
	}
	snap, err := loadSnapshot(dir)
	if err != nil {
		return err
	}
	if st, ok := stats[key]; ok {
		snap.Folders[key] = st
	} else {
		delete(snap.Folders, key)
	}
	return writeSnapshot(dir, snap)
}

func (s *Store) restoreDriftEntry(game, id, key string) (Profile, error) {
	return s.updateMods(game, id, func(p *Profile, dir string) error {
		i, err := requireEntry(p.Entries, key)
		if err != nil {
			return err
		}
		modsDir := filepath.Join(dir, "mods")
		if err := removeEntryFolders(modsDir, key); err != nil {
			return err
		}
		if err := s.place(game, modsDir, p.Entries[i]); err != nil {
			return err
		}
		return s.relayBase(game, p, dir, key, nil)
	})
}

func (s *Store) revertDriftEntry(game, id, key string) (Profile, error) {
	return s.updateMods(game, id, func(p *Profile, dir string) error {
		i, err := requireEntry(p.Entries, key)
		if err != nil {
			return err
		}
		e := p.Entries[i]
		if s.items == nil {
			return errors.New("no store")
		}
		src, err := s.items.Path(game, key)
		if err != nil {
			return err
		}
		modsDir := filepath.Join(dir, "mods")
		scratch, err := os.MkdirTemp(modsDir, tempPrefix)
		if err != nil {
			return err
		}
		if err := datadir.MaterializeTree(src, scratch); err != nil {
			return errors.Join(err, fsx.RemoveAll(scratch))
		}
		none := filepath.Join(scratch, ".no-old")
		for _, m := range e.Mods {
			plain, dotted, err := ModPaths(modsDir, e.Key, m.Folder)
			if err != nil {
				return errors.Join(err, fsx.RemoveAll(scratch))
			}
			cur := plain
			if !exists(cur) {
				cur = dotted
			}
			if !exists(cur) {
				continue
			}
			if err := carryOver(cur, none, filepath.Join(scratch, filepath.FromSlash(m.Folder))); err != nil {
				return errors.Join(err, fsx.RemoveAll(scratch))
			}
		}
		final, err := materialize(scratch, e)
		if err != nil {
			return errors.Join(err, fsx.RemoveAll(scratch))
		}
		w, err := replaceFolder(modsDir, e.Key, scratch, final)
		if err != nil {
			return errors.Join(err, fsx.RemoveAll(scratch))
		}
		w.commit()
		return s.relayBase(game, p, dir, key, overlaysOf(p.Entries, key))
	})
}

func (s *Store) adoptDriftFolder(game, id, folder string) (Profile, error) {
	folder, err := safeFolder(folder)
	if err != nil {
		return Profile{}, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	src := filepath.Join(dir, "mods", folder)
	if !exists(src) {
		src = filepath.Join(holdDir(dir), folder)
	}
	if !exists(src) {
		return Profile{}, os.ErrNotExist
	}
	res, err := s.InstallFolder(game, id, src)
	if err != nil {
		return Profile{}, err
	}
	_ = fsx.RemoveAll(src)
	if err := s.RecordModsSnapshot(game, id); err != nil {
		return Profile{}, err
	}
	return res.Profile, nil
}

// removedModsDir keeps folders taken out of a profile's mods/, apart from trashed profiles, which sit under
// trash/<game>/<profile id>.
func (s *Store) removedModsDir(game, id string) string {
	trash := s.trash
	if trash == "" {
		trash = filepath.Join(filepath.Dir(s.root), "trash")
	}
	return filepath.Join(trash, game, "removed-mods", id)
}

// trashModsFolder moves an untracked mods folder aside and returns the token RestoreModsFolder takes.
func (s *Store) trashModsFolder(game, id, folder string) (string, error) {
	folder, err := safeFolder(folder)
	if err != nil {
		return "", err
	}
	if err := s.unlocked(game, id); err != nil {
		return "", err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return "", err
	}
	src := filepath.Join(dir, "mods", folder)
	if !exists(src) {
		src = filepath.Join(holdDir(dir), folder)
	}
	if !exists(src) {
		return "", os.ErrNotExist
	}
	token := strconv.FormatInt(time.Now().UnixNano(), 10) + "/" + folder
	dst := filepath.Join(s.removedModsDir(game, id), filepath.FromSlash(token))
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return "", err
	}
	if err := fsx.Rename(src, dst); err != nil {
		return "", renameBusy(err)
	}
	return token, s.RecordModsSnapshot(game, id)
}

// RestoreModsFolder moves a folder trashModsFolder set aside back into mods/.
func (s *Store) RestoreModsFolder(game, id, token string) error {
	stamp, folder, ok := strings.Cut(token, "/")
	if _, err := strconv.ParseInt(stamp, 10, 64); !ok || err != nil {
		return errors.New("invalid restore token")
	}
	if folder, err := safeFolder(folder); err != nil || folder != strings.TrimSpace(folder) {
		return errors.New("invalid restore token")
	}
	if err := s.unlocked(game, id); err != nil {
		return err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return err
	}
	src := filepath.Join(s.removedModsDir(game, id), stamp, folder)
	if !exists(src) {
		return os.ErrNotExist
	}
	dst := filepath.Join(dir, "mods", folder)
	if exists(dst) {
		return fmt.Errorf("mods already has a folder named %q", folder)
	}
	if err := fsx.Rename(src, dst); err != nil {
		return renameBusy(err)
	}
	_ = os.Remove(filepath.Dir(src))
	return s.RecordModsSnapshot(game, id)
}

// renameBusy words a failed move of a mods folder: the trash lives in the data folder, so the usual cause is a file in
// the folder being open in another program.
func renameBusy(err error) error {
	return usererr.Wrap(usererr.Busy, fmt.Errorf("a file in that folder is in use by another program; close it and try again (%w)", err))
}

//wails:ignore
func (s *Service) ScanModsDrift(game, id string) ([]Drift, error) {
	return s.store.ScanModsDrift(game, id)
}

func (s *Service) KeepDriftChanges(game, id, key string) error {
	return s.store.refreshSnapshotKey(game, id, key)
}

func (s *Service) ForgetDriftEntry(game, id, key string) (Profile, error) {
	p, err := s.store.RemoveEntry(game, id, key)
	if err != nil {
		return Profile{}, err
	}
	_ = s.store.refreshSnapshotKey(game, id, key)
	return p, nil
}

func (s *Service) RestoreDriftEntry(game, id, key string) (Profile, error) {
	p, err := s.store.restoreDriftEntry(game, id, key)
	if err != nil {
		return Profile{}, err
	}
	if err := s.store.RecordModsSnapshot(game, id); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (s *Service) RevertDriftEntry(game, id, key string) (Profile, error) {
	p, err := s.store.revertDriftEntry(game, id, key)
	if err != nil {
		return Profile{}, err
	}
	if err := s.store.RecordModsSnapshot(game, id); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (s *Service) AdoptDriftFolder(game, id, folder string) (Profile, error) {
	return s.store.adoptDriftFolder(game, id, folder)
}

// RemoveDriftFolder sets an untracked folder aside and returns the token RestoreDriftFolder takes to undo it.
func (s *Service) RemoveDriftFolder(game, id, folder string) (string, error) {
	return s.store.trashModsFolder(game, id, folder)
}

// RestoreDriftFolder puts a folder RemoveDriftFolder set aside back into mods/.
func (s *Service) RestoreDriftFolder(game, id, token string) error {
	return s.store.RestoreModsFolder(game, id, token)
}
