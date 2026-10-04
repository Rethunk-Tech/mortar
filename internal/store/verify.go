package store

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

// VerifyEvery is how long a verified item is left alone by the background pass.
const VerifyEvery = 7 * 24 * time.Hour

const (
	verifyFile   = "verify.json"
	manifestsDir = ".manifests"
)

type fileSum struct {
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// hashRecord is the hash of every file of an item as it was when stored. It lives beside the store, not inside the
// item, so it is never copied into a profile.
type hashRecord struct {
	FormatVersion int                `json:"formatVersion"`
	Files         map[string]fileSum `json:"files"`
}

// Damage is how an item differs from what was recorded when it was stored.
type Damage struct {
	Missing []string `json:"missing"`
	Changed []string `json:"changed"`
	Extra   []string `json:"extra"`
}

// Empty reports that the item matches what was recorded.
func (d Damage) Empty() bool { return len(d.Missing)+len(d.Changed)+len(d.Extra) == 0 }

// Count is the number of files that differ.
func (d Damage) Count() int { return len(d.Missing) + len(d.Changed) + len(d.Extra) }

type verifyEntry struct {
	Verified time.Time `json:"verified"`
	Damage   *Damage   `json:"damage,omitempty"`
}

// verifyState maps game -> key -> the last verification.
type verifyState struct {
	FormatVersion int                               `json:"formatVersion"`
	Items         map[string]map[string]verifyEntry `json:"items"`
}

func (s *Store) manifestPath(game, key string) string {
	return filepath.Join(s.root, manifestsDir, game, key+".json")
}

func (s *Store) loadVerify() verifyState {
	st := verifyState{}
	if _, err := datadir.ReadJSON(filepath.Join(s.root, verifyFile), &st); err != nil {
		log.Printf("store verify state unreadable, starting over: %v", err)
		st = verifyState{}
	}
	if st.Items == nil {
		st.Items = map[string]map[string]verifyEntry{}
	}
	return st
}

func (s *Store) saveVerify(st verifyState) error {
	st.FormatVersion = datadir.FormatVersion
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return err
	}
	return datadir.WriteVersioned(filepath.Join(s.root, verifyFile), st)
}

// sumTree hashes every regular file under dir except the completion marker, keyed by slash-separated path.
func sumTree(ctx context.Context, dir string) (map[string]fileSum, error) {
	out := map[string]fileSum{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == completeMarker || !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		sum, err := fsx.SHA256(p)
		if err != nil {
			return err
		}
		out[rel] = fileSum{Size: info.Size(), SHA256: sum}
		return nil
	})
	return out, err
}

func diffSums(want, got map[string]fileSum) Damage {
	d := Damage{Missing: []string{}, Changed: []string{}, Extra: []string{}}
	for rel, w := range want {
		g, ok := got[rel]
		switch {
		case !ok:
			d.Missing = append(d.Missing, rel)
		case g != w:
			d.Changed = append(d.Changed, rel)
		}
	}
	for rel := range got {
		if _, ok := want[rel]; !ok {
			d.Extra = append(d.Extra, rel)
		}
	}
	slices.Sort(d.Missing)
	slices.Sort(d.Changed)
	slices.Sort(d.Extra)
	return d
}

func (s *Store) writeManifest(game, key string, files map[string]fileSum) error {
	path := s.manifestPath(game, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return datadir.WriteVersioned(path, hashRecord{FormatVersion: datadir.FormatVersion, Files: files})
}

// recordInstalled records the hashes of a freshly installed item and clears any earlier verdict on the key. A
// failure only means the item is baselined by its first verification instead.
func (s *Store) recordInstalled(game, key, dir string) {
	files, err := sumTree(context.Background(), dir)
	if err == nil {
		err = s.writeManifest(game, key, files)
	}
	if err != nil {
		log.Printf("store %s/%s: record hashes: %v", game, key, err)
		_ = os.Remove(s.manifestPath(game, key))
	}
	st := s.loadVerify()
	if st.Items[game] != nil {
		delete(st.Items[game], key)
		if err := s.saveVerify(st); err != nil {
			log.Printf("store %s/%s: clear verification: %v", game, key, err)
		}
	}
}

// forget drops what is recorded about removed items.
func (s *Store) forget(game string, keys ...string) {
	for _, key := range keys {
		_ = os.Remove(s.manifestPath(game, key))
	}
	st := s.loadVerify()
	changed := false
	for _, key := range keys {
		if _, ok := st.Items[game][key]; ok {
			delete(st.Items[game], key)
			changed = true
		}
	}
	if changed {
		if err := s.saveVerify(st); err != nil {
			log.Printf("store verify state: %v", err)
		}
	}
}

// Verify hashes the item's files and compares them with the hashes recorded when it was stored, reporting
// missing, changed and extra files. An item stored before hashes were recorded is baselined by its first
// verification, which cannot find earlier damage. The result is remembered for Damaged and VerifyDue.
func (s *Store) Verify(ctx context.Context, game, key string) (Damage, error) {
	s.mu.Lock()
	dir, err := s.folder(game, key)
	s.mu.Unlock()
	if err != nil {
		return Damage{}, err
	}
	got, err := sumTree(ctx, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return Damage{}, usererr.Wrap(usererr.NotFound, &Error{Game: game, Key: key, Err: ErrNotFound})
	}
	if err != nil {
		return Damage{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var want hashRecord
	found, rerr := datadir.ReadJSON(s.manifestPath(game, key), &want)
	if rerr != nil {
		log.Printf("store %s/%s: hash record unreadable, recording again: %v", game, key, rerr)
		found = false
	}
	d := Damage{}
	if found {
		d = diffSums(want.Files, got)
	} else if err := s.writeManifest(game, key, got); err != nil {
		return Damage{}, err
	}
	st := s.loadVerify()
	if st.Items[game] == nil {
		st.Items[game] = map[string]verifyEntry{}
	}
	entry := verifyEntry{Verified: time.Now().UTC()}
	if !d.Empty() {
		entry.Damage = &d
	}
	st.Items[game][key] = entry
	return d, s.saveVerify(st)
}

// Damaged lists the game's items whose last verification found differences.
func (s *Store) Damaged(game string) map[string]Damage {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]Damage{}
	for key, e := range s.loadVerify().Items[game] {
		if e.Damage != nil {
			out[key] = *e.Damage
		}
	}
	return out
}

// Refs lists the complete store items, least recently verified first. With due set it leaves out those verified
// within VerifyEvery.
func (s *Store) Refs(due bool, now time.Time) ([]Ref, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	games, err := s.games()
	if err != nil {
		return nil, err
	}
	st := s.loadVerify()
	type aged struct {
		ref Ref
		at  time.Time
	}
	var all []aged
	for _, g := range games {
		for _, key := range g.keys {
			dir := filepath.Join(s.root, g.name, key)
			if !completeItem(dir) {
				continue
			}
			at := st.Items[g.name][key].Verified
			if due && now.Sub(at) < VerifyEvery {
				continue
			}
			all = append(all, aged{Ref{Game: g.name, Key: key}, at})
		}
	}
	slices.SortFunc(all, func(a, b aged) int { return a.at.Compare(b.at) })
	out := make([]Ref, len(all))
	for i, a := range all {
		out[i] = a.ref
	}
	return out, nil
}

// Quarantine sets a damaged item aside so a fresh copy can be stored under its key; restore puts it back when
// that fails. A copy that is not restored is a leftover temp folder, which Cleanup deletes.
func (s *Store) Quarantine(game, key string) (restore func() error, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := s.itemDir(game, key)
	if err != nil {
		return nil, err
	}
	aside := filepath.Join(filepath.Dir(dir), tempPrefix+"damaged-"+key)
	if err := os.RemoveAll(aside); err != nil {
		return nil, err
	}
	if err := fsx.Rename(dir, aside); err != nil {
		return nil, &Error{Game: game, Key: key, Err: err}
	}
	return func() error {
		s.mu.Lock()
		defer s.mu.Unlock()
		if exists(dir) {
			return nil
		}
		return fsx.Rename(aside, dir)
	}, nil
}
