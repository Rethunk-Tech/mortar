// Package profile stores each profile as <datadir>/profiles/<game>/<id>/profile.json beside its mods/ folder.
package profile

import (
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

	"github.com/Rethunk-Tech/mortar/internal/ids"

	"github.com/Rethunk-Tech/mortar/internal/fsx"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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
	// KindThunderstore is a Thunderstore package: Name is "Namespace-Name" and Version the package version.
	KindThunderstore = "thunderstore"
	// KindModrinth and KindItch are projects from those sites: Name is the project or game id.
	KindModrinth = "modrinth"
	KindItch     = "itch"
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
	// ModName is the Nexus mod page's name, and Category the file's category on that page (MAIN, OPTIONAL, ...).
	ModName  string `json:"modName,omitempty"`
	Category string `json:"category,omitempty"`
	fomod    *fomodChoices
	disabled *disabledMods
	overlay  *overlayPlace
	// replacing is the Nexus file id of the file this one is a newer version of.
	replacing int
}

type fomodChoices struct {
	m map[string]map[string][]string
}

type disabledMods struct {
	ids []mod.ID
}

// WithFomod returns a copy that carries FOMOD plugin choices into InstallStaged.
func (s Source) WithFomod(choices map[string]map[string][]string) Source {
	s.fomod = &fomodChoices{m: choices}
	return s
}

// WithDisabled returns a copy that switches the named mods off when installed.
func (s Source) WithDisabled(ids []mod.ID) Source {
	s.disabled = &disabledMods{ids: slices.Clone(ids)}
	return s
}

func (s Source) fomodMap() map[string]map[string][]string {
	if s.fomod == nil {
		return nil
	}
	return s.fomod.m
}

// Component is one mod inside a package. Folder holds its manifest.json, relative to mods/<key>/, in its
// enabled (not dot-prefixed) form; "." is the package's own folder.
type Component struct {
	ID      mod.ID   `json:"id"`
	Version string   `json:"version"`
	Name    string   `json:"name"`
	Author  string   `json:"author"`
	Folder  string   `json:"folder"`
	Needs   []mod.ID `json:"needs,omitempty"`
	// Optional is the ids in Needs whose manifest listed IsRequired as false.
	Optional []mod.ID `json:"optional,omitempty"`
	// ContentPackFor is the id of the framework when the mod is a content pack.
	ContentPackFor mod.ID `json:"contentPackFor,omitempty"`
}

// Entry is one package in a profile: a download unit with its source, holding one or more components. Disabled
// lists the component ids switched off.
type Entry struct {
	IgnoreUpdates bool   `json:"ignoreUpdates,omitempty"`
	Key           string `json:"key"`
	PreviousKey   string `json:"previousKey"`
	Source        Source `json:"source"`
	// Package marks an entry that has no folder in the profile's mods folder: its files are laid out and deployed
	// into the game when it launches (a Thunderstore package).
	Package bool `json:"package,omitempty"`
	// PreviousSource is where the PreviousKey version came from, so a roll back restores it with the files.
	PreviousSource *Source     `json:"previousSource,omitempty"`
	Mods           []Component `json:"mods"`
	Disabled       []mod.ID    `json:"disabled"`
	// LoadAfter is ids this entry should load after, recorded when the user makes it win an edit conflict.
	LoadAfter []mod.ID `json:"loadAfter,omitempty"`
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
	// OverlayOf is the key of the entry whose folder this optional file overlays; such an entry has no mods and no
	// folder of its own. OverlayFrom is the folder of its store item that is laid over, OverlayTo where inside the
	// base entry's folder it goes, and OverlayOff switches it off.
	OverlayOf   string `json:"overlayOf,omitempty"`
	OverlayFrom string `json:"overlayFrom,omitempty"`
	OverlayTo   string `json:"overlayTo,omitempty"`
	OverlayOff  bool   `json:"overlayOff,omitempty"`
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
	FormatVersion int       `json:"formatVersion,omitempty"`
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Notes         string    `json:"notes"`
	Cover         string    `json:"cover"`
	Order         int       `json:"order"`
	Hidden        bool      `json:"hidden"`
	Created       time.Time `json:"created"`
	Updated       time.Time `json:"updated"`
	Entries       []Entry   `json:"entries"`
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
	// LaunchPresets are named launch configurations beside the profile's own settings above.
	LaunchPresets []LaunchPreset `json:"launchPresets,omitempty"`
	// DefaultLaunchPreset is the id of the preset Play uses; empty means the profile's own settings.
	DefaultLaunchPreset string `json:"defaultLaunchPreset,omitempty"`
	// Install is the id of the game install this profile launches and reads saves from; empty means the game's
	// selected install.
	Install string `json:"install,omitempty"`
	// SeparateSaves gives the profile its own saves folder, which replaces the game's shared one while it runs.
	SeparateSaves bool `json:"separateSaves,omitempty"`
	// Loader is the id of the game loader this profile runs; empty means the game's primary loader.
	Loader string `json:"loader,omitempty"`
	// Overrides are profile values for settings.ProfileOverridable keys.
	Overrides map[string]string `json:"overrides,omitempty"`
	// Error is set on a list item whose profile.json could not be read.
	Error string `json:"error,omitempty"`
	// RepairError is why Repair is unavailable on a damaged list item.
	RepairError string `json:"repairError,omitempty"`
}

// Store reads and writes profiles under one root folder.
type Store struct {
	root  string
	trash string
	// dataDir is Mortar's data folder: custom categories, config presets and the default save backups live in it.
	dataDir string
	// coverAfterWrite runs after SetCover wrote the image, before the profile is saved; tests use it to break the save.
	coverAfterWrite func(dir string)
	items           *store.Store
	home            string
	settings        *settings.Store
	mu              sync.Mutex
	// Bundled returns the store items every profile of a game holds: the loader's own mods and the console
	// bridge, whichever are installed.
	Bundled func(game string) []Bundle
	// Created is called after Create has added a profile; nil means nothing.
	Created func(game string)
	// Tidied is told about each repair rebuild makes to a profile's mods folder (what was done, the profile's name, the
	// folder); nil means nothing.
	Tidied func(what, profileName, folder string)
	// Publisher is the Thunderstore namespace of the one package in the game's community index with this name and
	// version; ok is false when none or several match. Nil means no index.
	Publisher func(game, name, version string) (namespace string, ok bool)
	// ShortcutRenamed updates an existing launcher after a profile is renamed; nil means nothing.
	ShortcutRenamed func(game, id, profileName, gameName string) error
	// ShortcutRemoved removes launchers after a profile is deleted; nil means nothing.
	ShortcutRemoved func(game, id string) error
	// Running reports whether the game is running this profile; nil means never.
	Running func(game, id string) bool
	// GameRunning reports whether the game process is up at all, however it was started; nil means never.
	GameRunning func(game string) bool
	// NewModsEnabled reports whether newly installed entries start enabled; nil means enabled.
	NewModsEnabled func() bool
	// OldFilesMode returns the game's oldFilesOnUpdate setting; nil means ask.
	OldFilesMode func(game string) string
	historyKind  string
	historyLabel string
	// historyConfigs are the entries whose config.json the pending event's change rewrote.
	historyConfigs  []string
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

// HasStoreItem reports whether the mod store holds key for the game.
func (s *Store) HasStoreItem(game, key string) bool {
	_, err := s.items.Path(game, key)
	return err == nil
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
	return OpenIn(dir, items), nil
}

// OpenIn is Open for the data folder dir.
func OpenIn(dir string, items *store.Store) *Store {
	return &Store{
		root: filepath.Join(dir, "profiles"), trash: filepath.Join(dir, "trash"), dataDir: dir, items: items,
		historyQuietIDs: map[string]int{}, historyBatches: map[string]historyBatch{},
	}
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
	p, _, err := s.readDir(game, id)
	return p, err
}

// readDir is read that also returns the profile's folder.
func (s *Store) readDir(game, id string) (Profile, string, error) {
	dir, err := s.profileDir(game, id)
	if err != nil {
		return Profile{}, "", err
	}
	p, err := readAt(dir, id)
	return p, dir, err
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
			p.Entries[i].Mods = []Component{}
		}
		if p.Entries[i].Disabled == nil {
			p.Entries[i].Disabled = []mod.ID{}
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
			return Profile{}, errors.Join(err, fsx.RemoveAll(dir))
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
	id := ids.New()
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
	if err := writeProfile(dir, p); err != nil {
		return Profile{}, errors.Join(err, fsx.RemoveAll(dir))
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
	p, dir, err := s.readDir(game, id)
	if err != nil {
		return Profile{}, err
	}
	before := cloneEntries(p.Entries)
	stateBefore := stateOf(p)
	if err := fn(&p, dir); err != nil {
		s.historyKind, s.historyLabel, s.historyConfigs = "", "", nil
		return Profile{}, err
	}
	p.Updated = time.Now().UTC().Truncate(time.Second)
	if err := writeProfile(dir, p); err != nil {
		s.historyKind, s.historyLabel, s.historyConfigs = "", "", nil
		restoreModsOld(dir)
		return Profile{}, err
	}
	_ = fsx.RemoveAll(filepath.Join(dir, "mods") + ".old")
	kind, label, configs := s.historyKind, s.historyLabel, s.historyConfigs
	s.historyKind, s.historyLabel, s.historyConfigs = "", "", nil
	if s.historyQuietIDs[id] == 0 {
		key := historyBatchKey(game, id)
		if batch, ok := s.historyBatches[key]; ok {
			if err := s.recordHistoryBatch(dir, &batch, p.Entries); err != nil {
				log.Printf("profile %s/%s: record history: %v", game, id, err)
			}
			s.historyBatches[key] = batch
		} else if err := recordHistory(dir, before, p.Entries, stateChange(stateBefore, stateOf(p)), kind, label, configs, s.historyKeep()); err != nil {
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
