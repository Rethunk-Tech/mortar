package profile

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/manifest"
)

// tempPrefix holds a character no store key can contain, so a leftover temp folder never collides with an
// entry folder (a disabled root-manifest entry is ".<key>").
const tempPrefix = ".tmp_"

// Mod is one mod as the UI lists it, flattened across entries.
type Mod struct {
	Key      string   `json:"key"`
	UniqueID string   `json:"uniqueId"`
	Name     string   `json:"name"`
	Author   string   `json:"author"`
	Version  string   `json:"version"`
	Enabled  bool     `json:"enabled"`
	Siblings []string `json:"siblings"`
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func sameID(a, b string) bool { return strings.EqualFold(a, b) }

func hasID(ids []string, id string) bool {
	return slices.ContainsFunc(ids, func(x string) bool { return sameID(x, id) })
}

func writeProfile(dir string, p Profile) error {
	return datadir.WriteJSON(filepath.Join(dir, fileName), p)
}

// pair returns a mod folder's enabled and disabled paths under entryRoot. folder comes from profile.json, so it must stay inside.
func pair(modsDir, key, folder string) (plain, dotted string, err error) {
	if folder == "." {
		return filepath.Join(modsDir, key), filepath.Join(modsDir, "."+key), nil
	}
	if !filepath.IsLocal(filepath.FromSlash(folder)) {
		return "", "", fmt.Errorf("mod folder %q leaves its entry", folder)
	}
	root := filepath.Join(modsDir, key)
	return filepath.Join(root, filepath.FromSlash(folder)),
		filepath.Join(root, filepath.FromSlash(path.Dir(folder)), "."+path.Base(folder)), nil
}

// flip renames a folder between its enabled and disabled names; it does nothing when it already has the wanted one.
func flip(plain, dotted string, enabled bool) error {
	from, to := dotted, plain
	if !enabled {
		from, to = plain, dotted
	}
	if exists(to) {
		return nil
	}
	if !exists(from) {
		return fmt.Errorf("mod folder %s is missing", filepath.Base(plain))
	}
	return os.Rename(from, to)
}

func isDisabled(e Entry, m EntryMod) bool { return hasID(e.Disabled, m.UniqueID) }

// materialize prepares a fresh copy of an entry in tmp: disabled nested mods get their dot names.
// It returns the folder name the entry belongs under in mods/.
func materialize(tmp string, e Entry) (string, error) {
	final := e.Key
	for _, m := range e.Mods {
		if !isDisabled(e, m) {
			continue
		}
		if m.Folder == "." {
			final = "." + e.Key
			continue
		}
		// tmp stands in for mods/<key>, so pair resolves the folder inside it.
		plain, dotted, err := pair(filepath.Dir(tmp), filepath.Base(tmp), m.Folder)
		if err != nil {
			return "", err
		}
		if err := flip(plain, dotted, false); err != nil {
			return "", err
		}
	}
	return final, nil
}

// place copies the store item for e into mods/ through a temp sibling and renames it into position.
func (s *Store) place(game, modsDir string, e Entry) error {
	src, err := s.items.Path(game, e.Key)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(modsDir, tempPrefix)
	if err != nil {
		return err
	}
	final, err := func() (string, error) {
		if err := datadir.CopyTree(src, tmp); err != nil {
			return "", err
		}
		return materialize(tmp, e)
	}()
	if err == nil {
		err = os.Rename(tmp, filepath.Join(modsDir, final))
	}
	if err != nil {
		return errors.Join(err, os.RemoveAll(tmp))
	}
	return nil
}

func entryMods(found []manifest.Mod) []EntryMod {
	out := make([]EntryMod, len(found))
	for i, m := range found {
		out[i] = EntryMod{UniqueID: m.UniqueID, Version: m.Version, Name: m.Name, Author: m.Author, Folder: m.Folder}
	}
	return out
}

// DuplicateError reports an entry the profile already holds; Label names its mods.
type DuplicateError struct{ Key, Label string }

func (e *DuplicateError) Error() string {
	return fmt.Sprintf("%q is already in this profile as %s", e.Key, e.Label)
}

// NoModError reports a store item without a readable manifest.
type NoModError struct{ Key string }

func (e *NoModError) Error() string {
	return fmt.Sprintf("%q holds no mod: no readable %s", e.Key, manifest.FileName)
}

// SourceSMAPI marks the entry holding the loader's own mods.
const SourceSMAPI = "smapi"

// addTo copies the store item key into the profile's mods/ and records its entry, switching off the
// mods in disabled that it holds. placed is the new folder, for the caller to remove if a later step fails.
func (s *Store) addTo(game string, p *Profile, dir, key string, source Source, disabled []string) (placed string, err error) {
	for _, e := range p.Entries {
		if e.Key == key {
			return "", &DuplicateError{Key: key, Label: entryLabel(e)}
		}
	}
	src, err := s.items.Path(game, key)
	if err != nil {
		return "", err
	}
	found, err := manifest.Scan(src)
	if err != nil {
		return "", err
	}
	if len(found) == 0 {
		return "", &NoModError{Key: key}
	}
	e := Entry{Key: key, Source: source, Mods: entryMods(found), Disabled: []string{}}
	for _, m := range e.Mods {
		if hasID(disabled, m.UniqueID) {
			e.Disabled = append(e.Disabled, m.UniqueID)
		}
	}
	modsDir := filepath.Join(dir, "mods")
	if err := os.MkdirAll(modsDir, 0o700); err != nil {
		return "", err
	}
	if err := s.place(game, modsDir, e); err != nil {
		return "", err
	}
	p.Entries = append(p.Entries, e)
	return filepath.Join(modsDir, key), nil
}

// AddEntry copies the store item key into the profile and records the mods it holds.
func (s *Store) AddEntry(game, id, key string, source Source) (Profile, error) {
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var placed string
	p, err := s.updateLocked(game, id, func(p *Profile, dir string) (err error) {
		placed, err = s.addTo(game, p, dir, key, source, nil)
		return err
	})
	if err != nil {
		if placed != "" {
			err = errors.Join(err, os.RemoveAll(placed))
		}
		return Profile{}, err
	}
	return p, s.items.Touch(game, key)
}

// ApplyBundled makes key the loader's bundled-mods entry in every profile of the game, replacing older ones
// and keeping each profile's switched-off mods off. Profiles that fail, including one the game is running, do not stop
// the others.
func (s *Store) ApplyBundled(game, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.List(game)
	if err != nil {
		return err
	}
	var errs []error
	for _, prof := range all {
		if err := s.unlocked(game, prof.ID); err != nil {
			errs = append(errs, err)
			continue
		}
		var placed string
		_, err := s.updateLocked(game, prof.ID, func(p *Profile, dir string) (err error) {
			var disabled []string
			for _, e := range slices.Backward(slices.Clone(p.Entries)) {
				if e.Source.Kind != SourceSMAPI || e.Key == key {
					continue
				}
				disabled = append(disabled, e.Disabled...)
				if err := removeFrom(p, dir, e.Key); err != nil {
					return err
				}
			}
			if slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == key }) {
				return nil
			}
			placed, err = s.addTo(game, p, dir, key, Source{Kind: SourceSMAPI, Name: "SMAPI"}, disabled)
			return err
		})
		if err != nil && placed != "" {
			err = errors.Join(err, os.RemoveAll(placed))
		}
		errs = append(errs, err)
	}
	if err := s.items.Touch(game, key); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func entryLabel(e Entry) string {
	names := make([]string, len(e.Mods))
	for i, m := range e.Mods {
		names[i] = m.Name
	}
	if len(names) == 0 {
		return e.Key
	}
	return strings.Join(names, ", ")
}

// removeFrom deletes the entry's folder and drops it from the profile.
func removeFrom(p *Profile, dir, key string) error {
	i := slices.IndexFunc(p.Entries, func(e Entry) bool { return e.Key == key })
	if i < 0 {
		return fmt.Errorf("%q is not in this profile", key)
	}
	modsDir := filepath.Join(dir, "mods")
	for _, name := range []string{key, "." + key} {
		if err := os.RemoveAll(filepath.Join(modsDir, name)); err != nil {
			return err
		}
	}
	p.Entries = slices.Delete(p.Entries, i, i+1)
	return nil
}

// RemoveEntry deletes the entry's folder and drops it from the profile.
func (s *Store) RemoveEntry(game, id, key string) (Profile, error) {
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, dir string) error {
		if slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == key && e.Source.Kind == SourceSMAPI }) {
			return errors.New("SMAPI's bundled mods are needed by every profile and cannot be removed")
		}
		return removeFrom(p, dir, key)
	})
}

// SetModEnabled switches a mod on or off by renaming its folder with or without a leading dot. key names the
// entry holding it, which tells apart two copies of one UniqueID; an empty key means the first entry that has it.
func (s *Store) SetModEnabled(game, id, key, uniqueID string, enabled bool) (Profile, error) {
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	return s.update(game, id, func(p *Profile, dir string) error {
		for ei := range p.Entries {
			e := &p.Entries[ei]
			if key != "" && e.Key != key {
				continue
			}
			mi := slices.IndexFunc(e.Mods, func(m EntryMod) bool { return sameID(m.UniqueID, uniqueID) })
			if mi < 0 {
				continue
			}
			plain, dotted, err := pair(filepath.Join(dir, "mods"), e.Key, e.Mods[mi].Folder)
			if err != nil {
				return err
			}
			if err := flip(plain, dotted, enabled); err != nil {
				return err
			}
			e.Disabled = slices.DeleteFunc(e.Disabled, func(x string) bool { return sameID(x, uniqueID) })
			if !enabled {
				e.Disabled = append(e.Disabled, e.Mods[mi].UniqueID)
			}
			return nil
		}
		return fmt.Errorf("no mod %q in this profile", uniqueID)
	})
}

// SetHidden hides or shows a profile in the list.
func (s *Store) SetHidden(game, id string, hidden bool) (Profile, error) {
	return s.update(game, id, func(p *Profile, _ string) error {
		p.Hidden = hidden
		return nil
	})
}

// Reorder gives the listed profiles order 0..n-1 in the given sequence; unlisted ones follow in their current order.
func (s *Store) Reorder(game string, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.List(game)
	if err != nil {
		return err
	}
	return s.applyOrder(game, all, ids)
}

func (s *Store) applyOrder(game string, all []Profile, ids []string) error {
	byID := map[string]Profile{}
	for _, p := range all {
		byID[p.ID] = p
	}
	seq := slices.Clone(ids)
	seen := map[string]bool{}
	for _, id := range ids {
		if _, ok := byID[id]; !ok || seen[id] {
			return fmt.Errorf("cannot order profile %q", id)
		}
		seen[id] = true
	}
	for _, p := range all {
		if !seen[p.ID] {
			seq = append(seq, p.ID)
		}
	}
	var errs []error
	for i, id := range seq {
		p := byID[id]
		if p.Order == i {
			continue
		}
		p.Order = i
		dir, err := s.profileDir(game, id)
		if err == nil {
			err = writeProfile(dir, p)
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// Duplicate copies a profile, mods/ included exactly as it is, and places the copy right after the source.
func (s *Store) Duplicate(game, id string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	src, err := s.read(game, id)
	if err != nil {
		return Profile{}, err
	}
	srcDir, _ := s.profileDir(game, id)
	gdir, _ := s.gameDir(game)
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Profile{}, err
	}
	newID := hex.EncodeToString(raw[:])
	dstDir, _ := s.profileDir(game, newID)

	const suffix = " copy"
	name := string([]rune(src.Name)[:min(len([]rune(src.Name)), maxName-len(suffix))]) + suffix
	now := time.Now().UTC().Truncate(time.Second)
	dup := src
	dup.ID, dup.Name, dup.Created, dup.Updated = newID, name, now, now
	dup.Entries = slices.Clone(src.Entries)

	if err := os.MkdirAll(gdir, 0o700); err != nil {
		return Profile{}, err
	}
	tmp, err := os.MkdirTemp(gdir, tempPrefix)
	if err != nil {
		return Profile{}, err
	}
	err = os.MkdirAll(filepath.Join(tmp, "mods"), 0o700)
	if err == nil {
		err = datadir.CopyTree(filepath.Join(srcDir, "mods"), filepath.Join(tmp, "mods"))
	}
	if err == nil {
		err = writeProfile(tmp, dup)
	}
	if err == nil {
		err = os.Rename(tmp, dstDir)
	}
	if err != nil {
		return Profile{}, errors.Join(err, os.RemoveAll(tmp))
	}

	all, err := s.List(game)
	if err != nil {
		return Profile{}, err
	}
	ids := make([]string, 0, len(all))
	for _, p := range all {
		if p.ID != newID {
			ids = append(ids, p.ID)
		}
	}
	at := slices.Index(ids, id) + 1
	ids = slices.Insert(ids, at, newID)
	if err := s.applyOrder(game, all, ids); err != nil {
		return Profile{}, err
	}
	return s.read(game, newID)
}

// Mods returns the profile's mods, first rebuilding any mods/ folder content that is missing from the store.
func (s *Store) Mods(game, id string) ([]Mod, error) { return s.mods(game, id, true) }

// UserMods is Mods without SMAPI's bundled mods, which every profile has and users never manage.
func (s *Store) UserMods(game, id string) ([]Mod, error) { return s.mods(game, id, false) }

func (s *Store) mods(game, id string, bundled bool) ([]Mod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.read(game, id)
	if err != nil {
		return nil, err
	}
	dir, _ := s.profileDir(game, id)
	if err := s.rebuild(game, dir, p); err != nil {
		return nil, err
	}
	out := []Mod{}
	for _, e := range p.Entries {
		if !bundled && e.Source.Kind == SourceSMAPI {
			continue
		}
		for _, m := range e.Mods {
			sib := []string{}
			for _, o := range e.Mods {
				if o.UniqueID != m.UniqueID {
					sib = append(sib, o.UniqueID)
				}
			}
			out = append(out, Mod{
				Key: e.Key, UniqueID: m.UniqueID, Name: m.Name, Author: m.Author, Version: m.Version,
				Enabled: !isDisabled(e, m), Siblings: sib,
			})
		}
	}
	return out, nil
}

// rebuild recopies mods/ and any entry folder that is gone, and clears temp folders an interrupted run left.
func (s *Store) rebuild(game, dir string, p Profile) error {
	modsDir := filepath.Join(dir, "mods")
	if err := os.MkdirAll(modsDir, 0o700); err != nil {
		return err
	}
	items, err := os.ReadDir(modsDir)
	if err != nil {
		return err
	}
	for _, it := range items {
		if strings.HasPrefix(it.Name(), tempPrefix) {
			if err := os.RemoveAll(filepath.Join(modsDir, it.Name())); err != nil {
				return err
			}
		}
	}
	for _, e := range p.Entries {
		if _, err := os.Stat(filepath.Join(modsDir, e.Key)); err == nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(modsDir, "."+e.Key)); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := s.place(game, modsDir, e); err != nil {
			return fmt.Errorf("rebuild %s: %w", e.Key, err)
		}
	}
	return nil
}

// ModFolder returns the mod's folder inside the profile, under whichever name (plain or dot-prefixed) it has now.
// key names the entry holding it, which tells apart two copies of one UniqueID; an empty key means the first entry that has it.
func (s *Store) ModFolder(game, id, key, uniqueID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.read(game, id)
	if err != nil {
		return "", err
	}
	dir, _ := s.profileDir(game, id)
	for _, e := range p.Entries {
		if key != "" && e.Key != key {
			continue
		}
		for _, m := range e.Mods {
			if !sameID(m.UniqueID, uniqueID) {
				continue
			}
			plain, dotted, err := pair(filepath.Join(dir, "mods"), e.Key, m.Folder)
			if err != nil {
				return "", err
			}
			if exists(plain) {
				return plain, nil
			}
			if exists(dotted) {
				return dotted, nil
			}
			return "", fmt.Errorf("mod folder %s is missing", filepath.Base(plain))
		}
	}
	return "", fmt.Errorf("no mod %q in this profile", uniqueID)
}
