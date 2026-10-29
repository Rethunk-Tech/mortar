package packsvc

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/archive"
	"github.com/Rethunk-Tech/mortar/internal/i18n"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/scratch"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

const (
	backupDoc     = "backup.json"
	backupVersion = 1
	backupPrefix  = "files/"
	// maxBackupBytes caps what a restore holds in memory: backup.json and the files under files/, counted as read.
	maxBackupBytes = 256 << 20
	// maxBackupDisk caps what a restore unpacks to disk: the store items and saves, counted as read.
	maxBackupDisk = 16 << 30
)

type backupJSON struct {
	Version int             `json:"version"`
	Game    string          `json:"game"`
	Profile profile.Profile `json:"profile"`
	// Store is the content hash (treeHash) of each store item the backup carries, by key.
	Store map[string]string `json:"store,omitempty"`
}

// RestoreResult is the profile a backup made and what it queued. Unavailable names the mods neither a source nor the
// backup itself could bring back, which the profile keeps but cannot use until they are installed again.
type RestoreResult struct {
	Game        string   `json:"game"`
	Profile     string   `json:"profile"`
	Name        string   `json:"name"`
	Queued      int      `json:"queued"`
	Unavailable []string `json:"unavailable"`
}

// carried is the entries whose store items a backup holds: all of them with mods, otherwise only those whose files
// no source downloads again.
func carried(game string, mods bool) func(profile.Entry) bool {
	return func(e profile.Entry) bool {
		if mods {
			return true
		}
		_, ok := restoreRequest(game, "", e)
		return !ok
	}
}

// BackupSize is how many bytes a backup of the profile holds before compression, so the player sees it before saving.
func (s *Service) BackupSize(gameID, profileID string, mods bool) (int64, error) {
	p, files, err := s.Profiles.Backup(gameID, profileID)
	if err != nil {
		return 0, err
	}
	dirs, err := s.Profiles.BackupDirs(gameID, p, carried(gameID, mods))
	if err != nil {
		return 0, err
	}
	var n int64
	for _, data := range files {
		n += int64(len(data))
	}
	for prefix, dir := range dirs {
		err := walkFiles(dir, strings.HasPrefix(prefix, "store/"), func(_, path string, size int64) error {
			n += size
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	return n, nil
}

// Backup writes the profile to dest as one zip: backup.json (the format version, the game, every setting and entry
// of the profile, and the hash of each store item it carries), its history, config files and cover under files/, the
// store item of each mod under store/<key>/ (with mods; otherwise only the mods no source can download again, and a
// restore downloads the rest), and the profile's own saves under saves/.
func (s *Service) Backup(gameID, profileID, dest string, mods bool) (err error) {
	p, files, err := s.Profiles.Backup(gameID, profileID)
	if err != nil {
		return err
	}
	dirs, err := s.Profiles.BackupDirs(gameID, p, carried(gameID, mods))
	if err != nil {
		return err
	}
	return safefs.WriteStreamAtomic(dest, 0o600, func(out io.Writer) error {
		zw := zip.NewWriter(out)
		doc := backupJSON{Version: backupVersion, Game: gameID, Profile: p, Store: map[string]string{}}
		for prefix, dir := range dirs {
			h := sha256.New()
			err := walkFiles(dir, strings.HasPrefix(prefix, "store/"), func(rel, path string, _ int64) error {
				w, err := zw.Create(prefix + rel)
				if err != nil {
					return err
				}
				src, err := safefs.Open(path)
				if err != nil {
					return err
				}
				defer func() { _ = src.Close() }()
				sum := sha256.New()
				if _, err := io.Copy(io.MultiWriter(w, sum), src); err != nil {
					return err
				}
				fmt.Fprintf(h, "%s\x00%x\n", rel, sum.Sum(nil))
				return nil
			})
			if err != nil {
				return err
			}
			if key, ok := storeKey(prefix); ok {
				doc.Store[key] = hex.EncodeToString(h.Sum(nil))
			}
		}
		for rel, data := range files {
			w, err := zw.Create(backupPrefix + rel)
			if err != nil {
				return err
			}
			if _, err := w.Write(data); err != nil {
				return err
			}
		}
		raw, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return err
		}
		w, err := zw.Create(backupDoc)
		if err != nil {
			return err
		}
		if _, err := w.Write(raw); err != nil {
			return err
		}
		return zw.Close()
	})
}

const savesPrefix = "saves/"

func storeKey(prefix string) (string, bool) {
	key, ok := strings.CutPrefix(strings.TrimSuffix(prefix, "/"), "store/")
	return key, ok && key != "" && !strings.Contains(key, "/")
}

// walkFiles calls fn with the slash path, full path and size of every regular file below root, in walk order. A store
// item's completion marker is the store's own and is skipped when item is set.
func walkFiles(root string, item bool, fn func(rel, path string, size int64) error) error {
	return safefs.WalkFiles(root, func(rel, path string, size int64) error {
		if item && rel == store.CompleteMarker {
			return nil
		}
		return fn(rel, path, size)
	})
}

// treeHash is the hash Backup records for a store item, read back from its unpacked folder.
func treeHash(root string) (string, error) {
	h := sha256.New()
	err := walkFiles(root, true, func(rel, path string, _ int64) error {
		sum, err := safefs.HashFile(path, sha256.New())
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%s\n", rel, sum)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil)), err
}

// BackupDialog asks where to save, then backs up as Backup does; an empty path means the player cancelled.
func (s *Service) BackupDialog(gameID, profileID string, mods bool) (string, error) {
	dest, err := s.askSave(gameID, profileID, func(p profile.Profile) winhost.Dialog {
		return winhost.Dialog{Filename: packageName(p.Name) + ".mortar-backup.zip", Filters: []winhost.Filter{{Name: i18n.T("Mortar profile backup (zip)"), Pattern: "*.zip"}}}
	})
	if err != nil || dest == "" {
		return "", err
	}
	return dest, s.Backup(gameID, profileID, dest, mods)
}

// Restore makes a new profile from the backup at path, with or without mod files: the store items it carries go into
// the store once their hashes match, its saves become the profile's, and the mods this computer still lacks are queued
// from their sources. gameID may be empty; when given it must be the backup's game.
func (s *Service) Restore(ctx context.Context, path, gameID string) (RestoreResult, error) {
	tmp, err := scratch.Dir("mortar-restore-")
	if err != nil {
		return RestoreResult{}, err
	}
	defer func() { _ = safefs.RemoveAll(tmp) }()
	doc, files, err := readBackup(path, tmp)
	if err != nil {
		return RestoreResult{}, err
	}
	if gameID != "" && gameID != doc.Game {
		return RestoreResult{}, usererr.Wrap(usererr.OtherGame, fmt.Errorf("this backup is of a %s profile, not %s", doc.Game, gameID))
	}
	dirs := map[string]string{}
	if dir := filepath.Join(tmp, "saves"); exists(dir) {
		dirs[savesPrefix] = dir
	}
	if dir := filepath.Join(tmp, "changed"); exists(dir) {
		dirs[profile.ChangedPrefix] = dir
	}
	for key, want := range doc.Store {
		dir := filepath.Join(tmp, "store", key)
		if !filepath.IsLocal(key) || !exists(dir) {
			continue
		}
		// A store item that does not hash as it did when backed up is left out, so its mod shows as unavailable.
		if got, err := treeHash(dir); err == nil && got == want {
			dirs["store/"+key+"/"] = dir
		}
	}
	p, missing, err := s.Profiles.RestoreBackup(ctx, doc.Game, doc.Profile, files, dirs)
	if err != nil {
		return RestoreResult{}, err
	}
	res := RestoreResult{Game: doc.Game, Profile: p.ID, Name: p.Name, Unavailable: []string{}}
	var reqs []queue.Request
	for _, e := range missing {
		if r, ok := restoreRequest(doc.Game, p.ID, e); ok {
			reqs = append(reqs, r)
		} else {
			res.Unavailable = append(res.Unavailable, entryName(e))
		}
	}
	if len(reqs) == 0 {
		return res, nil
	}
	items, err := s.Queue.Add(ctx, reqs)
	res.Queued = len(items)
	return res, err
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// RestoreDialog asks for a backup file, then restores it as Restore does; an empty Profile means the player cancelled.
func (s *Service) RestoreDialog(ctx context.Context, gameID string) (RestoreResult, error) {
	if s.App == nil {
		return RestoreResult{}, errors.New("no window to ask for the file")
	}
	path, err := s.App.OpenFile(winhost.Dialog{Filters: []winhost.Filter{{Name: i18n.T("Mortar profile backup"), Pattern: "*.zip"}}})
	if err != nil || path == "" {
		return RestoreResult{Unavailable: []string{}}, err
	}
	return s.Restore(ctx, path, gameID)
}

// restoreRequest is the download that puts the entry's store item back: the same file from the same source, which
// lands under the same store key, so the entry the restore already wrote picks it up.
func restoreRequest(game, profileID string, e profile.Entry) (queue.Request, bool) {
	src := e.Source
	r := queue.Request{Kind: queue.KindInstall, Game: game, Profile: profileID, Name: src.Name, Version: src.Version, Disabled: e.Disabled, Fomod: e.Fomod}
	switch src.Kind {
	case profile.KindThunderstore:
		r.Package = src.Name
	case profile.KindModrinth, profile.KindCurseForge, profile.KindItch, profile.KindVSModDB, profile.KindParadox:
		r.Package, r.Source, r.PackageFile = src.Name, src.Kind, src.FileID
	case profile.KindGitHub:
		r.Repo, r.Tag, r.Asset, r.FileName = src.Repo, src.Tag, src.Asset, src.Asset
	case profile.KindNexus:
		r.ModID, r.FileID, r.FileName = src.ModID, src.FileID, src.Name
		if e.IsOverlay() {
			r.Overlay = &queue.OverlayPlace{From: e.OverlayFrom, To: e.OverlayTo, Off: e.OverlayOff}
		}
	default:
		return queue.Request{}, false
	}
	return r, true
}

// readBackup reads backup.json and the files under files/ into memory and unpacks store/ and saves/ below tmp.
func readBackup(path, tmp string) (backupJSON, map[string][]byte, error) {
	src, err := safefs.Open(path)
	if err != nil {
		return backupJSON{}, nil, usererr.Wrap(usererr.Invalid, fmt.Errorf("not a Mortar profile backup: %w", err))
	}
	defer func() { _ = src.Close() }()
	info, err := src.Stat()
	if err != nil {
		return backupJSON{}, nil, err
	}
	root, err := os.OpenRoot(tmp)
	if err != nil {
		return backupJSON{}, nil, err
	}
	defer func() { _ = root.Close() }()
	files := map[string][]byte{}
	var doc []byte
	folded := map[string]string{}
	var inMemory, onDisk int64
	// The caps below are the module's backstop: the counters in the loop answer first and name the damage. A backup
	// holds names straight from disk, so the module admits every segment and keeps case-only twins apart; unpack
	// refuses them for the entries that reach this computer's disk.
	opts := archive.Options{
		MaxEntries:    math.MaxInt32,
		MaxEntryBytes: maxBackupDisk,
		MaxTotalBytes: maxBackupDisk + maxBackupBytes,
		RatioFloor:    maxBackupDisk + maxBackupBytes,
		CaseSensitive: true,
	}
	err = archive.ZipFunc(src, info.Size(), opts, func(name string, isDir bool, r io.Reader) error {
		if isDir {
			return nil
		}
		rel, inFiles := strings.CutPrefix(name, backupPrefix)
		if name != backupDoc && !inFiles {
			return unpack(name, r, root, folded, &onDisk)
		}
		data, err := safefs.ReadCapped(r, maxBackupBytes-inMemory)
		if errors.Is(err, safefs.ErrTooLarge) {
			return usererr.New(usererr.Damaged, "the backup unpacks to too much")
		}
		if err != nil {
			return err
		}
		inMemory += int64(len(data))
		if inFiles {
			files[rel] = data
		} else {
			doc = data
		}
		return nil
	})
	if err != nil {
		return backupJSON{}, nil, backupReadError(err)
	}
	var b backupJSON
	if doc == nil {
		return backupJSON{}, nil, usererr.New(usererr.Invalid, "not a Mortar profile backup: no "+backupDoc)
	}
	if err := json.Unmarshal(doc, &b); err != nil {
		return backupJSON{}, nil, usererr.Wrap(usererr.Damaged, fmt.Errorf("%s: %w", backupDoc, err))
	}
	// A 0.3.8 to 0.3.13 backup keeps the profile's short description, which the profile type no longer has.
	var old struct {
		Profile struct {
			Description string `json:"description"`
		} `json:"profile"`
	}
	if json.Unmarshal(doc, &old) == nil {
		b.Profile.Notes = profile.FoldDescription(old.Profile.Description, b.Profile.Notes)
	}
	if b.Version != backupVersion {
		kind := usererr.Invalid
		if b.Version > backupVersion {
			kind = usererr.Outdated
		}
		return backupJSON{}, nil, usererr.Wrap(kind, fmt.Errorf("this backup is format %d; this Mortar reads format %d", b.Version, backupVersion))
	}
	if b.Game == "" {
		return backupJSON{}, nil, usererr.New(usererr.Invalid, backupDoc+" names no game")
	}
	return b, files, nil
}

// caseInsensitive is whether this computer's file systems fold case by default; a backup made on Linux can hold two
// entries that differ only by case, which only a case-sensitive file system can restore.
var caseInsensitive = runtime.GOOS == "windows" || runtime.GOOS == "darwin"

// backupReadError types a module failure for the player; an error fn already typed passes through.
func backupReadError(err error) error {
	if _, typed := errors.AsType[*usererr.Error](err); typed {
		return err
	}
	switch {
	case errors.Is(err, archive.ErrUnsupportedFormat), errors.Is(err, archive.ErrMalformed):
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("not a Mortar profile backup: %w", err))
	case errors.Is(err, archive.ErrCaseCollision):
		return usererr.Wrap(usererr.Damaged, fmt.Errorf("the backup holds an entry twice, or two this computer's file system cannot keep apart: %w", err))
	case errors.Is(err, archive.ErrTraversal), errors.Is(err, archive.ErrUnsafeName), errors.Is(err, archive.ErrLink), errors.Is(err, archive.ErrSpecialFile):
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("the backup holds an entry Mortar does not write: %w", err))
	case errors.Is(err, archive.ErrEntryTooLarge), errors.Is(err, archive.ErrArchiveTooLarge), errors.Is(err, archive.ErrTooManyEntries), errors.Is(err, archive.ErrRatio):
		return usererr.Wrap(usererr.Damaged, fmt.Errorf("the backup unpacks to too much: %w", err))
	}
	return err
}

// noteFolded remembers name by its lower-cased form and, on a host that folds case, refuses a second name that folds
// to the same one, since the two could not both be written.
func noteFolded(folded map[string]string, name string, fold bool) error {
	key := strings.ToLower(name)
	if other, ok := folded[key]; ok && other != name && fold {
		return usererr.Wrap(usererr.Damaged, fmt.Errorf("the backup holds %q and %q, which this computer's file system cannot keep apart", other, name))
	}
	folded[key] = name
	return nil
}

// unpack writes one store/, saves/ or changed/ entry of a backup below root, adding its size to total.
func unpack(name string, r io.Reader, root *os.Root, folded map[string]string, total *int64) error {
	_, inStore := storeKeyOf(name)
	if (!inStore && !strings.HasPrefix(name, savesPrefix) && !strings.HasPrefix(name, profile.ChangedPrefix)) || !filepath.IsLocal(filepath.FromSlash(name)) {
		return usererr.Wrap(usererr.Invalid, fmt.Errorf("the backup holds %q, which Mortar does not write", name))
	}
	if err := noteFolded(folded, name, caseInsensitive); err != nil {
		return err
	}
	to := filepath.FromSlash(name)
	if err := root.MkdirAll(filepath.Dir(to), 0o700); err != nil {
		return err
	}
	out, err := root.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return usererr.Wrap(usererr.Damaged, fmt.Errorf("the backup holds %q twice", name))
	}
	if err != nil {
		return err
	}
	n, err := io.Copy(out, safefs.CapReader(r, maxBackupDisk-*total))
	closeErr := out.Close()
	if errors.Is(err, safefs.ErrTooLarge) {
		return usererr.New(usererr.Damaged, "the backup unpacks to too much")
	}
	*total += n
	return errors.Join(err, closeErr)
}

// storeKeyOf is the key of a store/<key>/<file> entry name.
func storeKeyOf(name string) (string, bool) {
	rest, ok := strings.CutPrefix(name, "store/")
	key, file, found := strings.Cut(rest, "/")
	return key, ok && found && key != "" && file != ""
}
