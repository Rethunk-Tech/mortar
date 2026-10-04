// Package profile stores each profile as <datadir>/profiles/<game>/<id>/profile.json beside its mods/ folder.
package profile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/Rethunk-AI/mortar/internal/fsx"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/store"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

const (
	fileName = "profile.json"
	maxName  = 60
)

var idPattern = regexp.MustCompile(`^[0-9a-f]{16}$`)

// What a Source's Kind can be, besides SourceSMAPI and SourceMortar for the bundled entries.
const (
	KindLocal  = "local"
	KindNexus  = "nexus"
	KindGitHub = "github"
)

// Source says where an entry came from. Kind is KindLocal for an archive the user picked, Name its file name;
// KindNexus for a Nexus file, which also names the mod, the file and its version, and holds the mod's picture and
// endorsement count as fetched at install; or KindGitHub for a release asset, named by Repo ("owner/repo"), Tag
// and Asset.
type Source struct {
	Kind             string `json:"kind"`
	Name             string `json:"name"`
	ModID            int    `json:"modId,omitempty"`
	FileID           int    `json:"fileId,omitempty"`
	Version          string `json:"version,omitempty"`
	Picture          string `json:"picture,omitempty"`
	EndorsementCount int    `json:"endorsementCount,omitempty"`
	Repo             string `json:"repo,omitempty"`
	Tag              string `json:"tag,omitempty"`
	Asset            string `json:"asset,omitempty"`
	fomod            *fomodChoices
	disabled         *disabledMods
}

type fomodChoices struct {
	m map[string]map[string][]string
}

type disabledMods struct {
	ids []string
}

// WithFomod returns a copy that carries FOMOD plugin choices into InstallStaged.
func (s Source) WithFomod(choices map[string]map[string][]string) Source {
	s.fomod = &fomodChoices{m: choices}
	return s
}

// WithDisabled returns a copy that switches the named mods off when installed.
func (s Source) WithDisabled(uniqueIDs []string) Source {
	s.disabled = &disabledMods{ids: slices.Clone(uniqueIDs)}
	return s
}

func (s Source) fomodMap() map[string]map[string][]string {
	if s.fomod == nil {
		return nil
	}
	return s.fomod.m
}

// EntryMod is one mod inside an entry. Folder holds its manifest.json, relative to mods/<key>/, in its
// enabled (not dot-prefixed) form; "." is the entry's own folder.
type EntryMod struct {
	UniqueID string   `json:"uniqueId"`
	Version  string   `json:"version"`
	Name     string   `json:"name"`
	Author   string   `json:"author"`
	Folder   string   `json:"folder"`
	Needs    []string `json:"needs,omitempty"`
	// Optional is the UniqueIDs in Needs whose manifest listed IsRequired as false.
	Optional []string `json:"optional,omitempty"`
	// ContentPackFor is the manifest ContentPackFor UniqueID when the mod is a content pack.
	ContentPackFor string `json:"contentPackFor,omitempty"`
}

// Entry is one mod archive in a profile. Disabled lists the UniqueIDs switched off.
type Entry struct {
	IgnoreUpdates bool   `json:"ignoreUpdates,omitempty"`
	Key           string `json:"key"`
	PreviousKey   string `json:"previousKey"`
	Source        Source `json:"source"`
	// PreviousSource is where the PreviousKey version came from, so a roll back restores it with the files.
	PreviousSource *Source    `json:"previousSource,omitempty"`
	Mods           []EntryMod `json:"mods"`
	Disabled       []string   `json:"disabled"`
	// LoadAfter is UniqueIDs this entry should load after, recorded when the user makes it win an edit conflict.
	LoadAfter []string `json:"loadAfter,omitempty"`
	// Added is when this entry was put in the profile; zero for entries written before the field existed.
	Added time.Time `json:"added,omitzero"`
	// Pinned keeps this entry on its current version; Mortar offers no update while it is true.
	Pinned bool `json:"pinned,omitempty"`
	// PinReason is an optional note for why this version is pinned.
	PinReason string `json:"pinReason,omitempty"`
	// SkipVersion is one newer version to hide; a later version is offered again.
	SkipVersion string `json:"skipVersion,omitempty"`
	// SkipSources hides updates reported by these sources.
	SkipSources []string `json:"skipSources,omitempty"`
	// UpdateChannel is main (empty), optional, or beta: which Nexus files count as updates.
	UpdateChannel string `json:"updateChannel,omitempty"`
	// Note is a per-entry remark in this profile, at most MaxEntryNote characters.
	Note string `json:"note,omitempty"`
	// Tags are per-entry labels in this profile, at most MaxEntryTags of MaxEntryTag characters each.
	Tags []string `json:"tags,omitempty"`
	// CategoryOverride is the primary category for grouping: a custom category id or a Nexus category name.
	CategoryOverride string `json:"categoryOverride,omitempty"`
	// Fomod is the chosen plugin names per install step and group, replayed on update and roll back.
	Fomod map[string]map[string][]string `json:"fomod,omitempty"`
	// ExtraStoreKeys are further store items from the same Nexus page, each copied to mods/<key>/<extra>/.
	ExtraStoreKeys []string `json:"extraStoreKeys,omitempty"`
	// PreviousExtraStoreKeys parallels ExtraStoreKeys after an update or roll back, for restoring each extra file.
	PreviousExtraStoreKeys []string `json:"previousExtraStoreKeys,omitempty"`
}

// CollectionRef is the Nexus collection a profile was imported from.
type CollectionRef struct {
	Domain   string `json:"domain"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Revision int    `json:"revision"`
}

// Profile is the on-disk shape of profile.json.
type Profile struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Notes   string    `json:"notes"`
	Cover   string    `json:"cover"`
	Order   int       `json:"order"`
	Hidden  bool      `json:"hidden"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Entries []Entry   `json:"entries"`
	// Groups are named sets of entry keys that toggle together.
	Groups []Group `json:"groups,omitempty"`
	// Origin is how the profile was created when that is known: OriginLink, OriginMortar,
	// OriginGameMods, OriginCopy, or OriginCollection. Empty for a profile made with New profile.
	Origin string `json:"origin,omitempty"`
	// Collection is the Nexus collection this profile was imported from, when Origin is OriginCollection.
	Collection *CollectionRef `json:"collection,omitempty"`
	// CopyOf is the source profile's name when Origin is OriginCopy.
	CopyOf string `json:"copyOf,omitempty"`
	// Color is a palette token from ProfileColors, or empty for the default sidebar mark.
	Color string `json:"color,omitempty"`
	// Icon is a Lucide name from ProfileIcons, or empty for the default profile mark.
	Icon string `json:"icon,omitempty"`
	// Description is a short blurb shown on the Profiles list and in the hero; at most MaxDescription runes.
	Description string `json:"description,omitempty"`
	// LaunchOptions is extra SMAPI arguments for this profile, as the user typed them.
	LaunchOptions string `json:"launchOptions,omitempty"`
	// LaunchPrefix is a shell-style command prefix for direct launches.
	LaunchPrefix string `json:"launchPrefix,omitempty"`
	// LaunchEnv contains one NAME=value environment variable per line for direct launches.
	LaunchEnv string `json:"launchEnv,omitempty"`
	// Overrides are profile values for settings.ProfileOverridable keys.
	Overrides map[string]string `json:"overrides,omitempty"`
	// Error is set on a list item whose profile.json could not be read.
	Error string `json:"error,omitempty"`
	// RepairError is why Repair is unavailable on a damaged list item.
	RepairError string `json:"repairError,omitempty"`
}

// Store reads and writes profiles under one root folder.
type Store struct {
	root     string
	trash    string
	items    *store.Store
	home     string
	settings *settings.Store
	mu       sync.Mutex
	// Bundled returns the store items every profile of a game holds: the loader's own mods and the console
	// bridge, whichever are installed.
	Bundled func(game string) []Bundle
	// Created is called after Create has added a profile; nil means nothing.
	Created func(game string)
	// ShortcutRenamed updates an existing launcher after a profile is renamed; nil means nothing.
	ShortcutRenamed func(game, id, profileName, gameName string) error
	// ShortcutRemoved removes launchers after a profile is deleted; nil means nothing.
	ShortcutRemoved func(game, id string) error
	// Running reports whether the game is running this profile; nil means never.
	Running func(game, id string) bool
	// BackupsKept returns how many save backups to retain; nil means backup.DefaultKeep.
	BackupsKept func() int
	// NewModsEnabled reports whether newly installed entries start enabled; nil means enabled.
	NewModsEnabled func() bool
	// OldFilesMode returns the game's oldFilesOnUpdate setting; nil means ask.
	OldFilesMode    func(game string) string
	historyKind     string
	historyLabel    string
	historyQuietIDs map[string]int
	historyBatches  map[string]historyBatch
	changesCache    map[string]changesSinceCache
}

func (s *Store) historyKeep() int {
	if s.settings != nil {
		if n := s.settings.Get().HistoryEventsKept; n > 0 {
			return n
		}
	}
	return maxHistory
}

// RunningError is returned by operations that would change the mods/ folder of a profile its game is running.
type RunningError struct{ Game string }

func (e *RunningError) Error() string {
	name := e.Game
	if g := game.Find(e.Game); g != nil {
		name = g.Name()
	}
	return name + " is running this profile: stop the game first"
}

// unlocked returns a *RunningError while the game is running the profile.
func (s *Store) unlocked(game, id string) error {
	if s.Running != nil && s.Running(game, id) {
		return &RunningError{Game: game}
	}
	return nil
}

// AnyRunning reports whether the game is running any of its profiles.
func (s *Store) AnyRunning(game string) bool {
	all, err := s.listOK(game)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(all, func(p Profile) bool { return s.unlocked(game, p.ID) != nil })
}

// ModsDir returns the absolute path of the profile's mods/ folder, the one passed to the game as its mods path.
func (s *Store) ModsDir(game, id string) (string, error) {
	dir, err := s.profileDir(game, id)
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(dir, "mods"))
}

// InMods runs fn on the profile and its absolute mods/ folder under the store's lock, so files fn writes there cannot
// race a mod being switched on or off, updated or removed.
func (s *Store) InMods(game, id string, fn func(p Profile, modsDir string) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.read(game, id)
	if err != nil {
		return err
	}
	modsDir, err := s.ModsDir(game, id)
	if err != nil {
		return err
	}
	return fn(p, modsDir)
}

// Open returns a store rooted at <datadir>/profiles, with deleted profiles in <datadir>/trash.
func Open(items *store.Store) (*Store, error) {
	dir, err := datadir.Dir()
	if err != nil {
		return nil, err
	}
	return &Store{
		root: filepath.Join(dir, "profiles"), trash: filepath.Join(dir, "trash"), items: items,
		historyQuietIDs: map[string]int{}, historyBatches: map[string]historyBatch{},
	}, nil
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("profile name is empty")
	}
	if utf8.RuneCountInString(name) > maxName {
		return "", fmt.Errorf("profile name is longer than %d characters", maxName)
	}
	return name, nil
}

// UniqueName returns name when no taken name equals it (ignoring case), else "name (2)", "name (3)" and so on with the
// first number that is free. A long name is cut so the result still fits maxName.
func UniqueName(taken []string, name string) string {
	name = strings.TrimSpace(name)
	used := func(n string) bool {
		return slices.ContainsFunc(taken, func(t string) bool { return strings.EqualFold(strings.TrimSpace(t), n) })
	}
	if !used(name) {
		return name
	}
	for n := 2; ; n++ {
		suffix := fmt.Sprintf(" (%d)", n)
		runes := []rune(name)
		candidate := string(runes[:min(len(runes), maxName-len(suffix))]) + suffix
		if !used(candidate) {
			return candidate
		}
	}
}

func (s *Store) gameDir(id string) (string, error) {
	if !game.Valid(id) {
		return "", fmt.Errorf("unknown game %q", id)
	}
	return filepath.Join(s.root, id), nil
}

func (s *Store) profileDir(game, id string) (string, error) {
	dir, err := s.gameDir(game)
	if err != nil {
		return "", err
	}
	if !idPattern.MatchString(id) {
		return "", usererr.Wrap(usererr.Invalid, fmt.Errorf("invalid profile id %q", id))
	}
	return filepath.Join(dir, id), nil
}

// List returns the game's profiles ordered by their order field.
func (s *Store) List(game string) ([]Profile, error) {
	dir, err := s.gameDir(game)
	if err != nil {
		return nil, err
	}
	dirs, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []Profile{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Profile{}
	for _, d := range dirs {
		if !d.IsDir() || !idPattern.MatchString(d.Name()) {
			continue
		}
		p, err := s.read(game, d.Name())
		if err != nil {
			row := Profile{ID: d.Name(), Error: usererr.Wrap(usererr.Damaged, err).Error()}
			if _, ok := latestSnapshotAt(filepath.Join(dir, d.Name())); !ok {
				row.RepairError = errNoSnapshot.Error()
			}
			out = append(out, row)
			continue
		}
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b Profile) int {
		if a.Order != b.Order {
			return a.Order - b.Order
		}
		return a.Created.Compare(b.Created)
	})
	return out, nil
}

func usable(p Profile) bool { return p.Error == "" }

func skipDamaged(all []Profile) []Profile {
	out := make([]Profile, 0, len(all))
	for _, p := range all {
		if usable(p) {
			out = append(out, p)
		}
	}
	return out
}

func (s *Store) listOK(game string) ([]Profile, error) {
	all, err := s.List(game)
	if err != nil {
		return nil, err
	}
	return skipDamaged(all), nil
}

// ListDamaged returns profiles whose profile.json could not be read.
func (s *Store) ListDamaged(game string) ([]Profile, error) {
	all, err := s.List(game)
	if err != nil {
		return nil, err
	}
	out := []Profile{}
	for _, p := range all {
		if !usable(p) {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Store) read(game, id string) (Profile, error) {
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	return readAt(dir, id)
}

func readAt(dir, id string) (Profile, error) {
	b, err := fsx.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Profile{}, usererr.Wrap(usererr.NotFound, err)
		}
		return Profile{}, err
	}
	var p Profile
	if err := json.Unmarshal(b, &p); err != nil {
		return Profile{}, usererr.Wrap(usererr.Damaged, fmt.Errorf("read profile %s: %w", id, err))
	}
	if p.Entries == nil {
		p.Entries = []Entry{}
	}
	for i := range p.Entries {
		if p.Entries[i].Mods == nil {
			p.Entries[i].Mods = []EntryMod{}
		}
		if p.Entries[i].Disabled == nil {
			p.Entries[i].Disabled = []string{}
		}
	}
	sanitizeAppearance(&p)
	return p, nil
}

// Create adds a profile holding only the bundled mods, if any are installed, after the existing ones.
func (s *Store) Create(game, name string) (Profile, error) {
	p, err := s.createBundled(game, name)
	if err == nil && s.Created != nil {
		s.Created(game)
	}
	return p, err
}

func (s *Store) createBundled(game, name string) (Profile, error) {
	p, err := s.create(game, name)
	if err != nil || s.Bundled == nil {
		return p, err
	}
	out := p
	for _, b := range s.Bundled(game) {
		withBundled, err := s.AddEntry(game, p.ID, b.Key, b.Source)
		if errors.Is(err, store.ErrNotFound) {
			// The item was collected while no profile used it; the next sync adds it again.
			continue
		}
		if err != nil {
			dir, _ := s.profileDir(game, p.ID)
			return Profile{}, errors.Join(err, os.RemoveAll(dir))
		}
		out = withBundled
	}
	return out, nil
}

func (s *Store) create(game, name string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	name, err := cleanName(name)
	if err != nil {
		return Profile{}, err
	}
	existing, err := s.listOK(game)
	if err != nil {
		return Profile{}, err
	}
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return Profile{}, err
	}
	id := hex.EncodeToString(raw[:])
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	if err := os.MkdirAll(filepath.Join(dir, "mods"), 0o700); err != nil {
		return Profile{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	p := Profile{ID: id, Name: name, Order: len(existing), Created: now, Updated: now, Entries: []Entry{}}
	if len(existing) > 0 {
		p.Order = existing[len(existing)-1].Order + 1
	}
	if err := datadir.WriteJSON(filepath.Join(dir, fileName), p); err != nil {
		return Profile{}, errors.Join(err, os.RemoveAll(dir))
	}
	return p, nil
}

// Rename changes a profile's name and touches its updated time.
func (s *Store) Rename(gameID, id, name string) (Profile, error) {
	name, err := cleanName(name)
	if err != nil {
		return Profile{}, err
	}
	p, err := s.update(gameID, id, func(p *Profile, _ string) error {
		p.Name = name
		return nil
	})
	if err != nil || s.ShortcutRenamed == nil {
		return p, err
	}
	gameName := gameID
	if g := game.Find(gameID); g != nil {
		gameName = g.Name()
	}
	// The rename has happened; a shortcut keeping the old name is logged, not a failed rename.
	if err := s.ShortcutRenamed(gameID, id, p.Name, gameName); err != nil {
		log.Printf("profile %s/%s: rename its shortcuts: %v", gameID, id, err)
	}
	return p, nil
}

// MaxNotes caps a profile's notes, in characters.
const MaxNotes = 20000

// SetNotes replaces the profile's notes. Notes never touch mods/, so a running game does not block it.
func (s *Store) SetNotes(game, id, notes string) (Profile, error) {
	if n := utf8.RuneCountInString(notes); n > MaxNotes {
		return Profile{}, fmt.Errorf("notes are too long: %d characters, the limit is %d", n, MaxNotes)
	}
	return s.update(game, id, func(p *Profile, _ string) error {
		p.Notes = notes
		return nil
	})
}

// SetSkipPlayCheck records whether the pre-Play problems dialog is skipped for this profile.
func (s *Store) SetSkipPlayCheck(game, id string, on bool) (Profile, error) {
	return s.SetOverride(game, id, overrideSkipPlayCheck, strconv.FormatBool(on))
}

// update reads the profile under the lock, applies fn (given the profile folder), then writes it with a new updated time.
func (s *Store) update(game, id string, fn func(p *Profile, dir string) error) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateLocked(game, id, fn)
}

func (s *Store) updateLocked(game, id string, fn func(p *Profile, dir string) error) (Profile, error) {
	p, err := s.read(game, id)
	if err != nil {
		return Profile{}, err
	}
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	before := cloneEntries(p.Entries)
	if err := fn(&p, dir); err != nil {
		s.historyKind, s.historyLabel = "", ""
		return Profile{}, err
	}
	p.Updated = time.Now().UTC().Truncate(time.Second)
	if err := datadir.WriteJSON(filepath.Join(dir, fileName), p); err != nil {
		s.historyKind, s.historyLabel = "", ""
		restoreModsOld(dir)
		return Profile{}, err
	}
	_ = os.RemoveAll(filepath.Join(dir, "mods") + ".old")
	kind, label := s.historyKind, s.historyLabel
	s.historyKind, s.historyLabel = "", ""
	if s.historyQuietIDs[id] == 0 {
		key := historyBatchKey(game, id)
		if batch, ok := s.historyBatches[key]; ok {
			if err := s.recordHistoryBatch(dir, &batch, p.Entries); err != nil {
				log.Printf("profile %s/%s: record history: %v", game, id, err)
			}
			s.historyBatches[key] = batch
		} else if err := recordHistory(dir, before, p.Entries, kind, label, s.historyKeep()); err != nil {
			log.Printf("profile %s/%s: record history: %v", game, id, err)
		}
	}
	return p, nil
}

// SourceOf is the source a profile of game recorded for store key, current or rolled-back, or zero when no profile
// holds key.
func (s *Store) SourceOf(game, key string) Source {
	return s.SourcesOf(game)[key]
}

// SourcesOf maps every store key the game's profiles hold, current or rolled-back, to the source recorded for it,
// reading each profile once. A key two profiles hold keeps the first profile's source.
func (s *Store) SourcesOf(game string) map[string]Source {
	out := map[string]Source{}
	all, err := s.listOK(game)
	if err != nil {
		return out
	}
	for _, p := range all {
		for _, e := range p.Entries {
			if _, ok := out[e.Key]; !ok {
				out[e.Key] = e.Source
			}
			if e.PreviousKey != "" && e.PreviousSource != nil {
				if _, ok := out[e.PreviousKey]; !ok {
					out[e.PreviousKey] = *e.PreviousSource
				}
			}
		}
	}
	return out
}
