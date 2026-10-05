// Package syncsvc shares each profile's state (its .mortar export: entries, sources and configs, never mod files)
// between machines through a folder that something else, such as Syncthing, Dropbox or a NAS, carries around. Each
// profile has a payload and a small version file naming the revision every machine has written; a revision from another
// machine is offered, never merged, and a profile both sides changed is a conflict the user settles.
package syncsvc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// Events.
const (
	// OffersEvent is emitted with the []Offer waiting for an answer, when that list changes.
	OffersEvent = "sync:offers"
	// FolderEvent names the folder watcher's event for the sync folder's game folders.
	FolderEvent = "sync:folder"
)

// Ref names one local profile.
type Ref struct {
	Game, ID, Name string
	Updated        time.Time
}

// Diff is what applying another machine's state would change in a profile, as mod names.
type Diff struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
}

// Source is the profile store and the share format as the sync sees them.
type Source interface {
	Games() []string
	Profiles(game string) ([]Ref, error)
	Export(game, id string) ([]byte, error)
	// Create makes an empty profile and returns its id.
	Create(game, name string) (string, error)
	Preview(ctx context.Context, game, id string, payload []byte) (Diff, error)
	// Apply makes the profile match the payload; mods it names are fetched from their sources.
	Apply(ctx context.Context, game, id string, payload []byte) error
}

// Deps wires the service to the app.
type Deps struct {
	Source Source
	// Folder is the sync folder; empty turns sync off.
	Folder func() string
	// Dir holds this machine's sync state.
	Dir string
	// Machine is the name other machines see; empty uses the host name.
	Machine string
	// Quiet is how long a profile must stay unchanged before it is written out.
	Quiet time.Duration
	Emit  func(name string, data any)
}

// Vector maps a machine id to the newest revision of the profile that machine has written.
type Vector map[string]int

// newer reports whether v holds a revision w lacks.
func (v Vector) newer(w Vector) bool {
	for m, r := range v {
		if r > w[m] {
			return true
		}
	}
	return false
}

func (v Vector) merged(w Vector) Vector {
	out := Vector{}
	for _, x := range []Vector{v, w} {
		for m, r := range x {
			out[m] = max(out[m], r)
		}
	}
	return out
}

// shared is the version file next to a payload.
type shared struct {
	Name        string `json:"name"`
	Machine     string `json:"machine"`
	MachineName string `json:"machineName"`
	Vector      Vector `json:"vector"`
}

// tracked is what this machine knows of one local profile's sync.
type tracked struct {
	// Remote is the profile's id in the sync folder; it differs from the local id for a profile another machine made.
	Remote string `json:"remote"`
	// Synced is the vector this machine last wrote or applied.
	Synced Vector `json:"synced"`
	// Seen is the profile's updated time at that moment; a later one means local changes.
	Seen time.Time `json:"seen"`
}

type state struct {
	Machine  string             `json:"machine"`
	Profiles map[string]tracked `json:"profiles"`
}

// Offer is another machine's revision of a profile that waits for the user's answer.
type Offer struct {
	Game string `json:"game"`
	// Profile is the local profile's id; empty when the profile does not exist here yet.
	Profile string `json:"profile"`
	// Remote is the profile's id in the sync folder.
	Remote   string `json:"remote"`
	Name     string `json:"name"`
	Machine  string `json:"machine"`
	New      bool   `json:"new"`
	Conflict bool   `json:"conflict"`
	vector   string
}

// Service keeps profiles in step with the sync folder.
type Service struct {
	d  Deps
	mu sync.Mutex
	st state
	// announced is the last offer list emitted, so an unchanged one is not announced again.
	announced string
	offers    []Offer
	kick      chan struct{}
}

// New loads the machine's sync state.
func New(d Deps) (*Service, error) {
	s := &Service{d: d, kick: make(chan struct{}, 1)}
	if d.Machine == "" {
		s.d.Machine, _ = os.Hostname()
	}
	b, err := fsx.ReadFile(s.statePath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		if err := json.Unmarshal(b, &s.st); err != nil {
			return nil, fmt.Errorf("sync state: %w", err)
		}
	}
	if s.st.Machine == "" {
		id := make([]byte, 6)
		if _, err := rand.Read(id); err != nil {
			return nil, err
		}
		s.st.Machine = hex.EncodeToString(id)
	}
	if s.st.Profiles == nil {
		s.st.Profiles = map[string]tracked{}
	}
	return s, nil
}

func (s *Service) statePath() string { return filepath.Join(s.d.Dir, "sync-state.json") }

func (s *Service) saveState() error { return datadir.WriteJSON(s.statePath(), s.st) }

func key(game, id string) string { return game + "/" + id }

func (s *Service) payloadPath(folder, game, remote string) string {
	return filepath.Join(folder, game, remote+".mortar")
}

func (s *Service) sharedPath(folder, game, remote string) string {
	return filepath.Join(folder, game, remote+".sync.json")
}

func (s *Service) readShared(folder, game, remote string) (shared, bool) {
	b, err := fsx.ReadFile(s.sharedPath(folder, game, remote))
	var sh shared
	if err != nil || json.Unmarshal(b, &sh) != nil || sh.Vector == nil {
		return shared{}, false
	}
	return sh, true
}

// Kick asks Run for a scan soon; a folder change calls it.
//
//wails:ignore
func (s *Service) Kick() {
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

// Run scans at start, on every kick and every interval until ctx ends.
//
//wails:ignore
func (s *Service) Run(ctx context.Context, interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		_, _ = s.Scan(ctx)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		case <-s.kick:
		}
	}
}

// Offers are the revisions waiting for an answer as of the last scan.
func (s *Service) Offers() []Offer {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Offer{}, s.offers...)
}

// Scan writes out the profiles that changed here and returns the revisions from other machines waiting for an answer.
func (s *Service) Scan(ctx context.Context) ([]Offer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	folder := s.d.Folder()
	if folder == "" {
		s.offers = nil
		return []Offer{}, nil
	}
	var offers []Offer
	var errs []error
	known := map[string]bool{}
	for _, game := range s.d.Source.Games() {
		refs, err := s.d.Source.Profiles(game)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, ref := range refs {
			t := s.st.Profiles[key(game, ref.ID)]
			if t.Remote == "" {
				t.Remote = ref.ID
			}
			known[key(game, t.Remote)] = true
			offer, waiting, err := s.scanOne(folder, ref, t)
			if err != nil {
				errs = append(errs, err)
			}
			if waiting {
				offers = append(offers, offer)
			}
		}
		offers = append(offers, s.newOffers(folder, game, known)...)
	}
	s.offers = offers
	s.announce()
	return append([]Offer{}, offers...), errors.Join(errs...)
}

// scanOne writes ref out when it changed here and nothing newer is waiting, or returns the offer for what is.
func (s *Service) scanOne(folder string, ref Ref, t tracked) (Offer, bool, error) {
	remote, found := s.readShared(folder, ref.Game, t.Remote)
	dirty := t.Seen.IsZero() || !ref.Updated.Equal(t.Seen)
	if found && remote.Vector.newer(t.Synced) {
		return Offer{
			Game: ref.Game, Profile: ref.ID, Remote: t.Remote, Name: remote.Name, Machine: remote.MachineName,
			Conflict: dirty && !t.Seen.IsZero(), vector: fmt.Sprint(remote.Vector),
		}, true, nil
	}
	if !dirty || time.Since(ref.Updated) < s.d.Quiet {
		s.st.Profiles[key(ref.Game, ref.ID)] = t
		return Offer{}, false, nil
	}
	return Offer{}, false, s.push(folder, ref, t, t.Synced)
}

// newOffers are the profiles in the sync folder that no local profile is linked to.
func (s *Service) newOffers(folder, game string, known map[string]bool) []Offer {
	entries, _ := os.ReadDir(filepath.Join(folder, game))
	var out []Offer
	for _, e := range entries {
		remote, ok := trimSuffix(e.Name(), ".sync.json")
		if !ok || known[key(game, remote)] {
			continue
		}
		if sh, ok := s.readShared(folder, game, remote); ok {
			out = append(out, Offer{Game: game, Remote: remote, Name: sh.Name, Machine: sh.MachineName, New: true, vector: fmt.Sprint(sh.Vector)})
		}
	}
	return out
}

func trimSuffix(name, suffix string) (string, bool) {
	if len(name) > len(suffix) && name[len(name)-len(suffix):] == suffix {
		return name[:len(name)-len(suffix)], true
	}
	return "", false
}

// push writes ref's payload as the next revision on top of base.
func (s *Service) push(folder string, ref Ref, t tracked, base Vector) error {
	payload, err := s.d.Source.Export(ref.Game, ref.ID)
	if err != nil {
		return fmt.Errorf("export %s: %w", ref.Name, err)
	}
	vec := base.merged(nil)
	vec[s.st.Machine]++
	if err := os.MkdirAll(filepath.Join(folder, ref.Game), 0o750); err != nil {
		return err
	}
	// The version file goes last: another machine that sees it finds the payload it names.
	if err := datadir.WriteFile(s.payloadPath(folder, ref.Game, t.Remote), payload, 0o600); err != nil {
		return err
	}
	meta, err := json.Marshal(shared{Name: ref.Name, Machine: s.st.Machine, MachineName: s.d.Machine, Vector: vec})
	if err != nil {
		return err
	}
	if err := datadir.WriteFile(s.sharedPath(folder, ref.Game, t.Remote), meta, 0o600); err != nil {
		return err
	}
	t.Synced, t.Seen = vec, ref.Updated
	s.st.Profiles[key(ref.Game, ref.ID)] = t
	return s.saveState()
}

func (s *Service) announce() {
	var b strings.Builder
	fmt.Fprint(&b, len(s.offers))
	for _, o := range s.offers {
		fmt.Fprint(&b, "|", o.Game, o.Remote, o.vector, o.Conflict)
	}
	if sig := b.String(); sig == s.announced {
		return
	}
	s.announced = b.String()
	if s.d.Emit != nil {
		s.d.Emit(OffersEvent, append([]Offer{}, s.offers...))
	}
}

// Choices for Resolve.
const (
	// Theirs makes the profile match the other machine's revision.
	Theirs = "theirs"
	// Mine keeps this machine's profile and writes it out over the other revision.
	Mine = "mine"
)

func (s *Service) find(game, remote string) (Offer, bool) {
	for _, o := range s.offers {
		if o.Game == game && o.Remote == remote {
			return o, true
		}
	}
	return Offer{}, false
}

// Resolve answers the offer for the profile whose sync id is remote: Theirs applies the other revision, Mine keeps
// the local profile and writes it over it. Neither merges anything.
func (s *Service) Resolve(ctx context.Context, game, remote, choice string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	folder := s.d.Folder()
	o, ok := s.find(game, remote)
	if folder == "" || !ok {
		return errors.New("that change is no longer waiting")
	}
	sh, ok := s.readShared(folder, game, remote)
	if !ok {
		return errors.New("the synced profile is gone from the sync folder")
	}
	local := o.Profile
	switch choice {
	case Theirs:
		payload, err := fsx.ReadFile(s.payloadPath(folder, game, remote))
		if err != nil {
			return err
		}
		if o.New {
			if local, err = s.d.Source.Create(game, sh.Name); err != nil {
				return err
			}
		}
		if err := s.d.Source.Apply(ctx, game, local, payload); err != nil {
			return err
		}
		ref, err := s.ref(game, local)
		if err != nil {
			return err
		}
		s.st.Profiles[key(game, local)] = tracked{Remote: remote, Synced: sh.Vector, Seen: ref.Updated}
	case Mine:
		ref, err := s.ref(game, local)
		if err != nil {
			return err
		}
		t := s.st.Profiles[key(game, local)]
		t.Remote = remote
		if err := s.push(folder, ref, t, t.Synced.merged(sh.Vector)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown choice %q", choice)
	}
	if err := s.saveState(); err != nil {
		return err
	}
	s.offers = slicesWithout(s.offers, o)
	s.announce()
	return nil
}

func slicesWithout(offers []Offer, o Offer) []Offer {
	var out []Offer
	for _, x := range offers {
		if x.Game != o.Game || x.Remote != o.Remote {
			out = append(out, x)
		}
	}
	return out
}

func (s *Service) ref(game, id string) (Ref, error) {
	refs, err := s.d.Source.Profiles(game)
	if err != nil {
		return Ref{}, err
	}
	for _, r := range refs {
		if r.ID == id {
			return r, nil
		}
	}
	return Ref{}, fmt.Errorf("profile %q is gone", id)
}

// Diff is what applying the other machine's revision would change in the local profile.
func (s *Service) Diff(ctx context.Context, game, remote string) (Diff, error) {
	s.mu.Lock()
	o, ok := s.find(game, remote)
	folder := s.d.Folder()
	s.mu.Unlock()
	if !ok {
		return Diff{}, errors.New("that change is no longer waiting")
	}
	payload, err := fsx.ReadFile(s.payloadPath(folder, game, remote))
	if err != nil {
		return Diff{}, err
	}
	return s.d.Source.Preview(ctx, game, o.Profile, payload)
}
