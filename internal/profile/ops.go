package profile

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/ids"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
)

// tempPrefix holds a character no store key can contain, so a leftover temp folder never collides with an
// entry folder (a disabled root-manifest entry is ".<key>").
const tempPrefix = ".tmp_"

// Mod is one mod as the UI lists it, flattened across entries.
type Mod struct {
	Key     string `json:"key"`
	ID      mod.ID `json:"id"`
	Name    string `json:"name"`
	Author  string `json:"author"`
	Version string `json:"version"`
	Enabled bool   `json:"enabled"`
	// Picture is the Nexus page's picture or the Thunderstore package's icon; Endorsements come from the Nexus page.
	Picture        string   `json:"picture"`
	Endorsements   int      `json:"endorsements"`
	Needs          []mod.ID `json:"needs,omitempty"`
	Optional       []mod.ID `json:"optional,omitempty"`
	ContentPackFor mod.ID   `json:"contentPackFor,omitempty"`
}

// EnableRef names one mod to switch, matching SetModEnabled's key and mod id.
type EnableRef struct {
	Key string `json:"key"`
	ID  mod.ID `json:"id"`
}

// EnableResult is a profile after switching mods, plus required dependencies turned on with them.
type EnableResult struct {
	Profile     Profile  `json:"profile"`
	AlsoEnabled []string `json:"alsoEnabled"`
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

func hasID(ids []mod.ID, id mod.ID) bool {
	return slices.ContainsFunc(ids, func(x mod.ID) bool { return mod.Equal(x, id) })
}

func writeProfile(dir string, p Profile) error {
	p.FormatVersion = datadir.FormatVersion
	p.LastChange = ""
	path := filepath.Join(dir, fileName)
	forgetProfile(path)
	return datadir.WriteVersioned(path, p)
}

// ModPaths returns a mod folder's enabled and disabled paths under modsDir. folder comes from profile.json, so it must stay inside.
func ModPaths(modsDir, key, folder string) (plain, dotted string, err error) {
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

// removeEntryFolders deletes both the enabled and disabled folder of a mod entry.
func removeEntryFolders(modsDir, key string) error {
	for _, name := range []string{key, "." + key} {
		if err := fsx.RemoveAll(filepath.Join(modsDir, name)); err != nil {
			return err
		}
	}
	return nil
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
	return fsx.Rename(from, to)
}

// Enabled reports whether the entry's mod with this unique ID is switched on. A mod without an ID cannot be
// switched off on its own, so it always counts as on.
func (e Entry) Enabled(uniqueID mod.ID) bool {
	return uniqueID == "" || !hasID(e.Disabled, uniqueID)
}

// materialize prepares a fresh copy of an entry in tmp: disabled nested mods get their dot names.
// It returns the folder name the entry belongs under in mods/.
func materialize(tmp string, e Entry) (string, error) {
	if err := os.Remove(filepath.Join(tmp, ".complete")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	final := e.Key
	for _, m := range e.Mods {
		if e.Enabled(m.ID) {
			continue
		}
		if m.Folder == "." {
			final = "." + e.Key
			continue
		}
		// tmp stands in for mods/<key>, so pair resolves the folder inside it.
		plain, dotted, err := ModPaths(filepath.Dir(tmp), filepath.Base(tmp), m.Folder)
		if err != nil {
			return "", err
		}
		if err := flip(plain, dotted, false); err != nil {
			return "", err
		}
	}
	return final, nil
}

// place lays the store item for e out as mods/<key> through a temp sibling and renames it into position.
func (s *Store) place(game, modsDir string, e Entry) error {
	_, err := s.placeStaged(game, modsDir, e, func(final string) string { return final })
	return err
}

// pendingPrefix names a folder laid out in full but not yet renamed to its entry's name, because profile.json does not
// record the entry yet. Its suffix is the folder's final name, so rebuild can finish the rename when Mortar stopped
// after profile.json was written, and sweeps the folder (a temp prefix) when it was not.
const pendingPrefix = tempPrefix + "pending_"

// placeStaged is place with the final rename aimed at stage(final) in mods/. It returns final, the folder name e belongs
// under, or "" when e has no folder of its own.
func (s *Store) placeStaged(game, modsDir string, e Entry, stage func(final string) string) (string, error) {
	arch, l, driver, err := s.layoutOf(game, filepath.Base(filepath.Dir(modsDir)), e.StoreKey(), e.Fomod)
	if err != nil || driver == driverThunderstore {
		return "", err
	}
	scratch, err := os.MkdirTemp(modsDir, tempPrefix)
	if err != nil {
		return "", err
	}
	final, err := func() (string, error) {
		if err := writeLayout(arch, l, scratch); err != nil {
			return "", err
		}
		return materialize(scratch, e)
	}()
	if err == nil {
		err = fsx.Rename(scratch, filepath.Join(modsDir, stage(final)))
	}
	if err != nil {
		return "", errors.Join(err, fsx.RemoveAll(scratch))
	}
	return final, nil
}

func entryMods(found []manifest.Mod) []Component {
	out := make([]Component, len(found))
	for i, m := range found {
		needs := make([]mod.ID, 0, len(m.Dependencies))
		optional := make([]mod.ID, 0)
		for _, d := range m.Dependencies {
			if d.ModID() != "" {
				needs = append(needs, d.ModID())
				if !d.Required {
					optional = append(optional, d.ModID())
				}
			}
		}
		out[i] = Component{
			ID: m.ModID(), Version: m.Version, Name: m.Name, Author: m.Author, Folder: m.Folder,
			Needs: needs, Optional: optional, ContentPackFor: m.ContentPackForID(),
		}
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

// rawXNBError reports an archive that replaces game content rather than carrying a SMAPI mod.
type rawXNBError struct{ Key string }

func (e *rawXNBError) Error() string { return fmt.Sprintf("%q holds raw .xnb files", e.Key) }

// SourceSMAPI marks the entry holding the loader's own mods, and SourceMortar the one holding Mortar's console
// bridge. Both are in every profile and hidden from users.
const (
	SourceSMAPI  = "smapi"
	SourceMortar = "mortar"
)

// Bundle is a store item every profile of a game holds: the loader's own mods or Mortar's bridge.
type Bundle struct {
	Key    string
	Source Source
}

// Bundled reports whether the source is a loader or bridge entry every profile holds.
func (s Source) Bundled() bool { return s.Kind == SourceSMAPI || s.Kind == SourceMortar }

// addTo copies the store item key into the profile's mods/ and records its entry, switching off the
// mods in disabled that it holds. placed is the new folder, for the caller to remove if a later step fails.
func (s *Store) addTo(game string, p *Profile, dir, key string, source Source, disabled []mod.ID) (placed string, err error) {
	placed, _, err = s.addToStaged(game, p, dir, key, source, disabled, false)
	return placed, err
}

// addToStaged is addTo that, when staged, leaves the folder under its pending name and returns that as placed and the
// name to give it once profile.json records the entry as final. A folder that needs no staging has them equal.
func (s *Store) addToStaged(game string, p *Profile, dir, key string, source Source, disabled []mod.ID, staged bool) (placed, final string, err error) {
	for _, e := range p.Entries {
		if e.Key == key {
			return "", "", &DuplicateError{Key: key, Label: entryLabel(e)}
		}
	}
	mods, isPackage, err := s.packageMods(game, key)
	if err != nil {
		return "", "", err
	}
	if isPackage {
		mods[0].Version = cmp.Or(mods[0].Version, source.Version)
	} else {
		src, tmp, err := s.layoutItem(game, p.ID, key, source.fomodMap())
		if tmp != "" {
			defer func() { _ = fsx.RemoveAll(tmp) }()
		}
		if err != nil {
			return "", "", err
		}
		found, err := manifest.Scan(src)
		if err != nil {
			return "", "", err
		}
		if len(found) == 0 {
			return "", "", &NoModError{Key: key}
		}
		mods = entryMods(found)
	}
	e := Entry{Key: key, Source: source, Mods: mods, Disabled: []mod.ID{}, Added: time.Now().UTC(), Fomod: cloneFomod(source.fomodMap()), Package: isPackage}
	if source.disabled != nil {
		disabled = append(disabled, source.disabled.ids...)
	}
	startEnabled := true
	if s.NewModsEnabled != nil {
		startEnabled = s.NewModsEnabled()
	}
	for _, m := range e.Mods {
		if hasID(disabled, m.ID) || !startEnabled {
			e.Disabled = append(e.Disabled, m.ID)
		}
	}
	modsDir := filepath.Join(dir, "mods")
	if err := os.MkdirAll(modsDir, 0o700); err != nil {
		return "", "", err
	}
	stage := func(name string) string { return name }
	if staged {
		stage = func(name string) string { return pendingPrefix + name }
	}
	name, err := s.placeStaged(game, modsDir, e, stage)
	if err != nil {
		return "", "", err
	}
	p.Entries = append(p.Entries, e)
	name = cmp.Or(name, key)
	return filepath.Join(modsDir, stage(name)), filepath.Join(modsDir, name), nil
}

// commitPlaced gives a staged folder its entry's name once profile.json records the entry. A rename that fails is left
// for rebuild, which finishes it because profile.json now lists the entry.
func commitPlaced(placed, final string) {
	if placed == final {
		return
	}
	if err := fsx.Rename(placed, final); err != nil {
		log.Printf("profile: could not move %s into place; the next rebuild will: %v", filepath.Base(placed), err)
	}
}

// AddEntry copies the store item key into the profile and records the mods it holds.
func (s *Store) AddEntry(game, id, key string, source Source) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	return s.addEntryLocked(game, id, key, source)
}

func (s *Store) addEntryLocked(game, id, key string, source Source) (Profile, error) {
	if over, err := s.isOverlayItem(game, key, source); err != nil {
		return Profile{}, err
	} else if over {
		return s.placeOverlayLocked(game, id, key, source)
	}
	if mods, isPackage, err := s.packageMods(game, key); err != nil {
		return Profile{}, err
	} else if isPackage && slices.Contains(installerGame(game).Loaders, folderLoader) {
		p, _, _, err := s.placeFolderLocked(game, id, key, source, mods)
		return p, err
	}
	var placed, final string
	p, err := s.updateLockedCommit(game, id, func(p *Profile, dir string) (err error) {
		placed, final, err = s.addToStaged(game, p, dir, key, source, nil, true)
		return err
	}, func() { commitPlaced(placed, final) })
	if err != nil {
		if placed != "" {
			err = errors.Join(err, fsx.RemoveAll(placed))
		}
		return Profile{}, err
	}
	return p, s.items.Touch(game, key)
}

// ApplyBundled makes b the game's bundled entry of its source kind in every profile, replacing older ones
// and keeping each profile's switched-off mods off. Profiles that fail, including one the game is running, do not stop
// the others.
func (s *Store) ApplyBundled(game string, b Bundle) error {
	return s.applyBundled(game, b, false)
}

// ApplyBundledForStart is ApplyBundled during Play's loader install: the profile is already marked running, but
// bundled loader mods must still land. Other writes stay locked.
func (s *Store) ApplyBundledForStart(game string, b Bundle) error {
	return s.applyBundled(game, b, true)
}

func (s *Store) applyBundled(game string, b Bundle, duringStart bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.listOK(game)
	if err != nil {
		return err
	}
	var errs []error
	for _, prof := range all {
		// A profile already on this bundle is left unwritten: a rewrite would mark it updated at every start.
		if !slices.ContainsFunc(prof.Entries, func(e Entry) bool { return e.Source.Kind == b.Source.Kind && e.Key != b.Key }) &&
			slices.ContainsFunc(prof.Entries, func(e Entry) bool { return e.Key == b.Key }) {
			continue
		}
		if !duringStart {
			if err := s.unlocked(game, prof.ID); err != nil {
				errs = append(errs, err)
				continue
			}
		}
		var placed string
		if s.historyQuietIDs == nil {
			s.historyQuietIDs = map[string]int{}
		}
		s.historyQuietIDs[prof.ID]++
		_, err := s.updateLocked(game, prof.ID, func(p *Profile, dir string) (err error) {
			var disabled []mod.ID
			for _, e := range slices.Backward(slices.Clone(p.Entries)) {
				if e.Source.Kind != b.Source.Kind || e.Key == b.Key {
					continue
				}
				disabled = append(disabled, e.Disabled...)
				if err := s.removeFrom(game, p, dir, e.Key); err != nil {
					return err
				}
			}
			if slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == b.Key }) {
				return nil
			}
			placed, err = s.addTo(game, p, dir, b.Key, b.Source, disabled)
			return err
		})
		s.historyQuietIDs[prof.ID]--
		if s.historyQuietIDs[prof.ID] <= 0 {
			delete(s.historyQuietIDs, prof.ID)
		}
		if err != nil && placed != "" {
			err = errors.Join(err, fsx.RemoveAll(placed))
		}
		errs = append(errs, err)
	}
	if err := s.items.Touch(game, b.Key); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func entryLabel(e Entry) string {
	names := make([]string, len(e.Mods))
	for i, m := range e.Mods {
		names[i] = m.Name
	}
	if len(names) == 0 && e.Source.Name != "" {
		return e.Source.Name
	}
	if len(names) == 0 {
		return e.Key
	}
	return strings.Join(names, ", ")
}

// removeFrom deletes the entry's folder and drops it from the profile, with the optional files laid over it.
// Removing an optional file puts its main entry's own files back.
func (s *Store) removeFrom(game string, p *Profile, dir, key string) error {
	i, err := requireEntry(p.Entries, key)
	if err != nil {
		return err
	}
	if base := p.Entries[i].OverlayOf; base != "" {
		was := overlaysOf(p.Entries, base)
		p.Entries = slices.Delete(p.Entries, i, i+1)
		dropKeyFromGroups(p, key)
		return s.relayBase(game, p, dir, base, was)
	}
	modsDir := filepath.Join(dir, "mods")
	if err := removeEntryFolders(modsDir, key); err != nil {
		return err
	}
	tray := p.Entries[i].TrayFiles
	p.Entries = slices.Delete(p.Entries, i, i+1)
	s.releaseTray(game, p, tray)
	dropKeyFromGroups(p, key)
	for _, o := range overlaysOf(p.Entries, key) {
		p.Entries = slices.DeleteFunc(p.Entries, func(e Entry) bool { return e.Key == o.Key })
		dropKeyFromGroups(p, o.Key)
	}
	return nil
}

// RemoveEntries deletes each named entry's folder and drops it from the profile, one write.
func (s *Store) RemoveEntries(game, id string, keys []string) (Profile, error) {
	p, err := s.updateModsRecorded(game, id, func(p *Profile, dir string) error {
		seen := map[string]bool{}
		for _, e := range p.Entries {
			// Removing the main entry takes its optional files with it.
			if e.IsOverlay() && slices.Contains(keys, e.OverlayOf) {
				seen[e.Key] = true
			}
		}
		for _, key := range keys {
			if seen[key] {
				continue
			}
			seen[key] = true
			if slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == key && e.Source.Bundled() }) {
				return errors.New("the bundled mods are needed by every profile and cannot be removed")
			}
			if err := s.removeFrom(game, p, dir, key); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Profile{}, err
	}
	return p, nil
}

// updateModsRecorded is updateMods followed by a mods/ snapshot, for changes that write the profile's mods.
func (s *Store) updateModsRecorded(game, id string, fn func(p *Profile, dir string) error) (Profile, error) {
	p, err := s.updateMods(game, id, fn)
	if err != nil {
		return Profile{}, err
	}
	if err := s.RecordModsSnapshot(game, id); err != nil {
		return Profile{}, err
	}
	return p, nil
}

// updateMods is update for changes to the mods/ folder, refused while the game runs the profile. The check holds the
// lock so that a launch, which reads the profile under it, sees either the whole change or none of it.
func (s *Store) updateMods(game, id string, fn func(p *Profile, dir string) error) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.unlocked(game, id); err != nil {
		return Profile{}, err
	}
	return s.updateLocked(game, id, fn)
}

func applyRestoredMeta(p *Profile, want Entry) {
	for i := range p.Entries {
		if p.Entries[i].Key != want.Key {
			continue
		}
		p.Entries[i].Disabled = slices.Clone(want.Disabled)
		p.Entries[i].Pinned = want.Pinned
		p.Entries[i].SkipVersion = want.SkipVersion
		p.Entries[i].SkipSources = slices.Clone(want.SkipSources)
		p.Entries[i].UpdateChannel = want.UpdateChannel
		p.Entries[i].Note = want.Note
		p.Entries[i].Tags = slices.Clone(want.Tags)
		p.Entries[i].CategoryOverride = want.CategoryOverride
		if !want.Added.IsZero() {
			p.Entries[i].Added = want.Added
		}
		return
	}
}

// RestoreEntries copies each store item back into the profile and reapplies its previous enabled
// state, pins, tags and other entry fields.
func (s *Store) RestoreEntries(game, id string, entries []Entry) (Profile, error) {
	if len(entries) == 0 {
		p, err := s.read(game, id)
		return p, err
	}
	entries = basesFirst(entries)
	p, err := s.updateModsRecorded(game, id, func(p *Profile, dir string) error {
		for _, want := range entries {
			if slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == want.Key }) {
				continue
			}
			if want.IsOverlay() {
				if err := s.restoreOverlay(game, p, dir, want); err != nil {
					return err
				}
				continue
			}
			src := want.Source
			if len(want.Fomod) > 0 {
				src = src.WithFomod(want.Fomod)
			}
			if _, err := s.addTo(game, p, dir, want.Key, src, want.Disabled); err != nil {
				return err
			}
			for _, m := range p.Entries {
				if m.Key != want.Key {
					continue
				}
				for _, im := range m.Mods {
					if err := applyEnabled(p, dir, want.Key, im.ID, !hasID(want.Disabled, im.ID)); err != nil {
						return err
					}
				}
				break
			}
			applyRestoredMeta(p, want)
		}
		return nil
	})
	if err != nil {
		return Profile{}, err
	}
	return p, nil
}

// RemoveEntry deletes the entry's folder and drops it from the profile.
func (s *Store) RemoveEntry(game, id, key string) (Profile, error) {
	p, err := s.updateModsRecorded(game, id, func(p *Profile, dir string) error {
		if slices.ContainsFunc(p.Entries, func(e Entry) bool { return e.Key == key && e.Source.Bundled() }) {
			return errors.New("the bundled mods are needed by every profile and cannot be removed")
		}
		return s.removeFrom(game, p, dir, key)
	})
	if err != nil {
		return Profile{}, err
	}
	return p, nil
}

func applyEnabled(p *Profile, dir, key string, uniqueID mod.ID, enabled bool) error {
	for ei := range p.Entries {
		e := &p.Entries[ei]
		if key != "" && e.Key != key {
			continue
		}
		mi := slices.IndexFunc(e.Mods, func(m Component) bool { return mod.Equal(m.ID, uniqueID) })
		if mi < 0 {
			continue
		}
		if e.hasFolder() {
			plain, dotted, err := ModPaths(filepath.Join(dir, "mods"), e.Key, e.Mods[mi].Folder)
			if err != nil {
				return err
			}
			if err := flip(plain, dotted, enabled); err != nil {
				return err
			}
		}
		e.Disabled = slices.DeleteFunc(e.Disabled, func(x mod.ID) bool { return mod.Equal(x, uniqueID) })
		if !enabled {
			e.Disabled = append(e.Disabled, e.Mods[mi].ID)
		}
		return nil
	}
	return fmt.Errorf("no mod %q in this profile", uniqueID)
}

// SetModEnabled switches a mod on or off by renaming its folder with or without a leading dot. key names the
// entry holding it, which tells apart two copies of one mod id; an empty key means the first entry that has it.
func (s *Store) SetModEnabled(game, id, key string, uniqueID mod.ID, enabled bool) (Profile, error) {
	p, _, err := s.enableMod(game, id, key, uniqueID, enabled)
	return p, err
}

func (s *Store) enableMod(game, id, key string, uniqueID mod.ID, enabled bool) (Profile, []string, error) {
	var also []string
	overlay := false
	p, err := s.updateMods(game, id, func(p *Profile, dir string) error {
		if overlay = isOverlayKey(p, key); overlay {
			return s.setOverlayLocked(game, p, dir, key, enabled)
		}
		if err := applyEnabled(p, dir, key, uniqueID, enabled); err != nil {
			return err
		}
		if enabled && s.autoEnableRequirements(game, p.PrefOverrides()) {
			also = enableRequired(p, dir, uniqueID)
		}
		return nil
	})
	if err == nil && overlay {
		err = s.RecordModsSnapshot(game, id)
	}
	return p, also, err
}

// SetModsEnabled switches each named mod on or off in one profile write.
func (s *Store) SetModsEnabled(game, id string, mods []EnableRef, enabled bool) (Profile, error) {
	p, _, err := s.enableMods(game, id, mods, enabled)
	return p, err
}

func (s *Store) enableMods(game, id string, mods []EnableRef, enabled bool) (Profile, []string, error) {
	var also []string
	overlay := false
	p, err := s.updateMods(game, id, func(p *Profile, dir string) error {
		for _, m := range mods {
			if isOverlayKey(p, m.Key) {
				overlay = true
				if err := s.setOverlayLocked(game, p, dir, m.Key, enabled); err != nil {
					return err
				}
				continue
			}
			if err := applyEnabled(p, dir, m.Key, m.ID, enabled); err != nil {
				return err
			}
			if enabled && s.autoEnableRequirements(game, p.PrefOverrides()) {
				also = append(also, enableRequired(p, dir, m.ID)...)
			}
		}
		return nil
	})
	if err == nil && overlay {
		err = s.RecordModsSnapshot(game, id)
	}
	return p, also, err
}

// Reorder gives the listed profiles order 0..n-1 in the given sequence; unlisted ones follow in their current order.
func (s *Store) Reorder(game string, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.listOK(game)
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
	src, srcDir, err := s.readDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	gdir, _ := s.gameDir(game)
	all, err := s.listOK(game)
	if err != nil {
		return Profile{}, err
	}
	taken := make([]string, 0, len(all))
	for _, p := range all {
		taken = append(taken, p.Name)
	}
	newID := ids.New()
	dstDir, _ := s.profileDir(game, newID)

	const suffix = " copy"
	runes := []rune(src.Name)
	name := UniqueName(taken, string(runes[:min(len(runes), maxName-len(suffix))])+suffix)
	now := time.Now().UTC().Truncate(time.Second)
	dup := src
	dup.ID, dup.Name, dup.Created, dup.Updated = newID, name, now, now
	dup.Origin, dup.CopyOf = OriginCopy, src.Name
	dup.Entries = slices.Clone(src.Entries)
	if err := copyProfile(gdir, srcDir, dstDir, dup); err != nil {
		return Profile{}, err
	}

	all, err = s.listOK(game)
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

// copyProfile writes dup into dstDir with the mods/ folder and cover of the profile in srcDir, building it beside
// the profiles in gdir first so a failed copy leaves nothing a listing would show.
func copyProfile(gdir, srcDir, dstDir string, dup Profile) error {
	if err := os.MkdirAll(gdir, 0o700); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(gdir, tempPrefix)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Join(tmp, "mods"), 0o700)
	if err == nil {
		err = datadir.MaterializeTreeExclusive(filepath.Join(srcDir, "mods"), filepath.Join(tmp, "mods"))
	}
	if err == nil && coverType(dup.Cover) != "" {
		err = datadir.CopyFile(filepath.Join(srcDir, dup.Cover), filepath.Join(tmp, dup.Cover))
		if errors.Is(err, fs.ErrNotExist) {
			dup.Cover, err = "", nil
		}
	}
	if err == nil {
		err = writeProfile(tmp, dup)
	}
	if err == nil {
		err = fsx.Rename(tmp, dstDir)
	}
	if err != nil {
		return errors.Join(err, fsx.RemoveAll(tmp))
	}
	return nil
}

// Mods returns the profile's mods, first rebuilding any mods/ folder content that is missing from the store.
func (s *Store) Mods(game, id string) ([]Mod, error) { return s.mods(game, id, true) }

// Rebuild is Mods for a caller that needs the mods folder put right, under the profile's lock, and not the list.
func (s *Store) Rebuild(game, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return err
	}
	return s.rebuild(game, dir, p)
}

// UserMods is Mods without the bundled mods (SMAPI's and the console bridge), which every profile has and users never manage.
func (s *Store) UserMods(game, id string) ([]Mod, error) { return s.mods(game, id, false) }

func (s *Store) mods(game, id string, bundled bool) ([]Mod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return nil, err
	}
	if err := s.rebuild(game, dir, p); err != nil {
		return nil, err
	}
	out := []Mod{}
	for _, e := range p.Entries {
		if !bundled && e.Source.Bundled() {
			continue
		}
		for _, m := range e.Mods {
			out = append(out, Mod{
				Key: e.Key, ID: m.ID, Name: m.Name, Author: m.Author, Version: m.Version,
				Enabled: e.Enabled(m.ID),
				Picture: e.Source.Picture, Endorsements: e.Source.EndorsementCount,
				Needs: m.Needs, Optional: m.Optional, ContentPackFor: m.ContentPackFor,
			})
		}
	}
	return out, nil
}

// rebuild recopies mods/ and any entry folder that is gone, puts back an entry folder an interrupted update left
// aside, and clears temp folders and folders of entries profile.json does not record.
func (s *Store) rebuild(game, dir string, p Profile) error {
	modsDir := filepath.Join(dir, "mods")
	if err := os.MkdirAll(modsDir, 0o700); err != nil {
		return err
	}
	if err := s.parkUnknownModsLocked(dir, p); err != nil {
		return err
	}
	known := map[string]bool{}
	for _, e := range p.Entries {
		known[e.Key], known["."+e.Key] = true, true
		if exists(filepath.Join(modsDir, e.Key)) || exists(filepath.Join(modsDir, "."+e.Key)) {
			continue
		}
		if err := s.finishInterrupted(modsDir, p.Name, e.Key); err != nil {
			return err
		}
	}
	items, err := os.ReadDir(modsDir)
	if err != nil {
		return err
	}
	for _, it := range items {
		if strings.HasPrefix(it.Name(), tempPrefix) || (it.IsDir() && !known[it.Name()]) {
			if err := fsx.RemoveAll(filepath.Join(modsDir, it.Name())); err != nil {
				return err
			}
			s.tidied("Removed a leftover mod folder", p.Name, it.Name())
		}
	}
	for _, e := range p.Entries {
		if e.IsOverlay() || !e.hasFolder() {
			continue
		}
		if _, err := os.Stat(filepath.Join(modsDir, e.Key)); err == nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(modsDir, "."+e.Key)); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := s.placeEntry(game, dir, p, e); err != nil {
			return fmt.Errorf("rebuild %s: %w", e.Key, err)
		}
		s.tidied("Rebuilt a mod folder from the store", p.Name, e.Key)
	}
	return s.repairSnapshot(game, dir, p)
}

// finishInterrupted renames the folder an interrupted run left under a temp name back to the entry's own: an update's
// aside copy, or an install's pending folder, whose entry profile.json already lists.
func (s *Store) finishInterrupted(modsDir, profileName, key string) error {
	for _, name := range []string{key, "." + key} {
		for _, prefix := range []string{asidePrefix, pendingPrefix} {
			left := filepath.Join(modsDir, prefix+name)
			if !exists(left) {
				continue
			}
			if err := fsx.Rename(left, filepath.Join(modsDir, name)); err != nil {
				return err
			}
			s.tidied("Put back a mod folder an interrupted install or update left aside", profileName, name)
			return nil
		}
	}
	return nil
}

// placeEntry copies the entry's folder from the store into the profile at dir, with its overlays and any config files
// a restore kept for it.
func (s *Store) placeEntry(game, dir string, p Profile, e Entry) error {
	modsDir := filepath.Join(dir, "mods")
	if err := s.place(game, modsDir, e); err != nil {
		return err
	}
	if err := s.layOverlays(game, p.ID, e, liveEntryDir(modsDir, e.Key), nil, overlaysOn(p.Entries, e.Key)); err != nil {
		return err
	}
	return applyRestored(dir, modsDir, e.Key)
}

func (s *Store) tidied(what, profileName, folder string) {
	if s.Tidied != nil {
		s.Tidied(what, profileName, folder)
	}
}

// ErrNoModFolder is a mod whose entry has no folder of its own in the profile: a package, whose files the launch lays
// out by its loader's rules, with its settings in the loader's config folder.
var ErrNoModFolder = errors.New("this mod has no folder of its own in the profile")

// packageDir is the store item holding the package that has uniqueID.
func (s *Store) packageDir(game, id, key string, uniqueID mod.ID) (string, error) {
	p, err := s.read(game, id)
	if err != nil {
		return "", err
	}
	e, _, ok := p.FindMod(key, uniqueID)
	if !ok {
		return "", fmt.Errorf("no mod %q in this profile", uniqueID)
	}
	return s.items.Path(game, e.StoreKey())
}

// ModFolder returns the mod's folder inside the profile, under whichever name (plain or dot-prefixed) it has now.
// key names the entry holding it, which tells apart two copies of one mod id; an empty key means the first entry that has it.
// A package's entry is ErrNoModFolder.
func (s *Store) ModFolder(game, id, key string, uniqueID mod.ID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.modFolderLocked(game, id, key, uniqueID)
}

func (s *Store) modFolderLocked(game, id, key string, uniqueID mod.ID) (string, error) {
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return "", err
	}
	return modFolderIn(p, dir, key, uniqueID)
}

func modFolderIn(p Profile, dir, key string, uniqueID mod.ID) (string, error) {
	for _, e := range p.Entries {
		if key != "" && e.Key != key {
			continue
		}
		for _, m := range e.Mods {
			if !mod.Equal(m.ID, uniqueID) {
				continue
			}
			if !e.hasFolder() {
				return "", ErrNoModFolder
			}
			plain, dotted, err := ModPaths(filepath.Join(dir, "mods"), e.Key, m.Folder)
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
