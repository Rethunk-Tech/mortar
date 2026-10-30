package configsvc

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-cfgedit"
	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// valueError is a value the editor will not write. Its English sentence is the error text; Detail is what the GUI words
// its own sentence from (usererr.Detailer).
type valueError struct {
	// Problem is one of line, bool, whole_number, number, enum, range, empty, list, toml and color.
	Problem string   `json:"problem"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Values  []string `json:"values,omitempty"`
	msg     string
}

func (e *valueError) Error() string { return e.msg }

func (e *valueError) Detail() any { return e }

// valueErr is err as the service reports it: a refused value is Invalid with its problem as detail, a missing setting
// NotFound.
func valueErr(err error) error {
	var ve *cfgedit.ValueError
	switch {
	case err == nil:
		return nil
	case errors.As(err, &ve):
		return usererr.Wrap(usererr.Invalid, &valueError{Problem: string(ve.Problem), Min: ve.Min, Max: ve.Max, Values: ve.Values, msg: ve.Error()})
	case errors.Is(err, cfgedit.ErrNoSetting):
		return usererr.Wrap(usererr.NotFound, err)
	}
	return err
}

// schemaOf groups entries read in file order by the section run they sit in, keeping those keep accepts.
func schemaOf(file ConfigFile, entries []cfgedit.Entry, keep func(section string) bool) Schema {
	s := Schema{File: file, Sections: []Section{}}
	for _, e := range entries {
		if keep(e.Section) {
			s.Sections = addToSection(s.Sections, e.Section, entryOf(e))
		}
	}
	return s
}

func entryOf(e cfgedit.Entry) Entry {
	return Entry{
		Key: e.Key, Type: string(e.Type), Value: e.Value, Default: e.Default, HasDefault: e.HasDefault,
		Description: e.Description, Min: e.Min, Max: e.Max, Values: e.Values, Flags: e.Flags,
	}
}

func cfgSchema(d *cfgedit.BepInEx, file ConfigFile) Schema {
	return schemaOf(file, d.Entries(), func(string) bool { return true })
}

// addToSection appends e to the last section when it is named section, else to a new one: entries read in file order
// group by the section run they sit in.
func addToSection(sections []Section, section string, e Entry) []Section {
	if n := len(sections); n == 0 || sections[n-1].Name != section {
		sections = append(sections, Section{Name: section})
	}
	last := &sections[len(sections)-1]
	last.Entries = append(last.Entries, e)
	return sections
}

// withSync marks the entries ConfigSync shares from the host, read from the sidecar beside the .cfg at path. A
// missing or malformed sidecar marks nothing.
func withSync(s Schema, path string) Schema {
	raw, err := safefs.ReadFile(strings.TrimSuffix(path, filepath.Ext(path)) + syncSidecar)
	if err != nil {
		return s
	}
	var side struct {
		Settings []struct {
			Section string `json:"section"`
			Key     string `json:"key"`
			Level   string `json:"level"`
		} `json:"settings"`
	}
	if json.Unmarshal(raw, &side) != nil {
		return s
	}
	for _, sh := range side.Settings {
		for i := range s.Sections {
			if s.Sections[i].Name != sh.Section {
				continue
			}
			for j := range s.Sections[i].Entries {
				if e := &s.Sections[i].Entries[j]; e.Key == sh.Key {
					e.Sync = sh.Level
				}
			}
		}
	}
	return s
}

// HostValue is the value a ConfigSync host sent for a shared setting. InUse is false while the player keeps their own
// value for a host recommendation.
type HostValue struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	InUse   bool   `json:"inUse"`
}

// askHostFor bounds the bridge query the config editor repeats while it is open.
const askHostFor = 2 * time.Second

// HostValues asks the running game which values the host sent for the settings the mod behind file shares through
// ConfigSync. There are none while the game is not running, before the bridge is up, or with a bridge older than
// the query.
func (s *Service) HostValues(game, profileID, file string) ([]HostValue, error) {
	none := []HostValue{}
	if s.Running == nil || !s.Running(game, profileID) || !strings.EqualFold(filepath.Ext(file), cfgExt) {
		return none, nil
	}
	dir, err := s.Profiles.ProfileDir(game, profileID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), askHostFor)
	defer cancel()
	raw, err := bridge.Query(ctx, filepath.Join(dir, filepath.FromSlash(bridge.BepInEx.StateFile)), "configsync")
	var rejected *bridge.RejectedError
	if errors.Is(err, bridge.ErrNotRunning) || errors.Is(err, bridge.ErrNotReady) || errors.As(err, &rejected) {
		return none, nil
	}
	if err != nil {
		return nil, err
	}
	return hostValuesOf(raw, strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)))
}

// hostValuesOf picks the bridge's rows for one plugin: a BepInEx config file is named after its plugin's GUID.
func hostValuesOf(raw json.RawMessage, guid string) ([]HostValue, error) {
	var rows []struct {
		Mod string `json:"mod"`
		HostValue
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := []HostValue{}
	for _, r := range rows {
		if r.Mod == guid {
			out = append(out, r.HostValue)
		}
	}
	return out, nil
}

// SaveHostValues keeps the host's current values of the settings the file's mod shares, to write into the player's own
// .cfg once the game is closed: Mortar writes no config while the game runs, which may rewrite it. It returns how many.
func (s *Service) SaveHostValues(game, profileID, modID, file string) (int, error) {
	values, err := s.HostValues(game, profileID, file)
	if err != nil || len(values) == 0 {
		return 0, err
	}
	edits := make([]edit, 0, len(values))
	for _, v := range values {
		edits = append(edits, edit{v.Section, v.Key, v.Value})
	}
	s.hostMu.Lock()
	defer s.hostMu.Unlock()
	s.loadKept()
	s.hostKept[hostKey{game, profileID, modID, file}] = edits
	s.saveKept()
	return len(edits), nil
}

type keptRecord struct {
	Game    string     `json:"game"`
	Profile string     `json:"profile"`
	Mod     string     `json:"mod"`
	File    string     `json:"file"`
	Edits   []keptEdit `json:"edits"`
}

type keptEdit struct {
	Section string `json:"section"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

const keptFile = "host-kept.json"

// loadKept reads the kept values the first time they are needed, so they survive a restart; callers hold hostMu.
func (s *Service) loadKept() {
	if s.hostKept != nil {
		return
	}
	s.hostKept = map[hostKey][]edit{}
	dir, err := datadir.Dir()
	if err != nil {
		return
	}
	var recs []keptRecord
	if _, err := datadir.ReadJSON(filepath.Join(dir, keptFile), &recs); err != nil {
		return
	}
	for _, r := range recs {
		edits := make([]edit, 0, len(r.Edits))
		for _, e := range r.Edits {
			edits = append(edits, edit{e.Section, e.Key, e.Value})
		}
		s.hostKept[hostKey{r.Game, r.Profile, r.Mod, r.File}] = edits
	}
}

// saveKept persists the kept values; callers hold hostMu.
func (s *Service) saveKept() {
	dir, err := datadir.Dir()
	if err != nil {
		return
	}
	recs := make([]keptRecord, 0, len(s.hostKept))
	for k, edits := range s.hostKept {
		r := keptRecord{Game: k.game, Profile: k.profile, Mod: k.mod, File: k.file}
		for _, e := range edits {
			r.Edits = append(r.Edits, keptEdit{e.section, e.key, e.value})
		}
		recs = append(recs, r)
	}
	slices.SortFunc(recs, func(a, b keptRecord) int {
		return strings.Compare(a.Game+"\x00"+a.Profile+"\x00"+a.Mod+"\x00"+a.File, b.Game+"\x00"+b.Profile+"\x00"+b.Mod+"\x00"+b.File)
	})
	if b, err := json.Marshal(recs); err == nil {
		_ = safefs.WriteFileAtomic(filepath.Join(dir, keptFile), b, 0o600)
	}
}

// FlushKept writes the kept host values of every profile whose game is no longer running; the app calls it when a
// profile unlocks, so the values land without the config page being opened.
func (s *Service) FlushKept() {
	s.hostMu.Lock()
	s.loadKept()
	keys := make([]hostKey, 0, len(s.hostKept))
	for k := range s.hostKept {
		keys = append(keys, k)
	}
	s.hostMu.Unlock()
	for _, k := range keys {
		if s.Running != nil && s.Running(k.game, k.profile) {
			continue
		}
		files, err := s.Files(k.game, k.profile, k.mod)
		if err != nil {
			log.Printf("config: kept values of %s: %v", k.file, err)
			continue
		}
		for _, f := range files {
			if f.Name != k.file {
				continue
			}
			if _, err := s.writeKept(k.game, k.profile, k.mod, f); err != nil {
				log.Printf("config: kept values of %s: %v", k.file, err)
			}
		}
	}
}

// writeKept writes the host's values kept for f, once the game is no longer running.
func (s *Service) writeKept(game, profileID, modID string, f ConfigFile) (bool, error) {
	if s.Running != nil && s.Running(game, profileID) {
		return false, nil
	}
	k := hostKey{game, profileID, modID, f.Name}
	s.hostMu.Lock()
	s.loadKept()
	edits, ok := s.hostKept[k]
	if ok {
		delete(s.hostKept, k)
		s.saveKept()
	}
	s.hostMu.Unlock()
	if !ok {
		return false, nil
	}
	return true, s.write(game, profileID, modID, f, edits)
}
