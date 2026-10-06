package profile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

const historySettings = "settings"

// ProfileState is what a history event keeps of the profile beside its entries: the settings that change how the
// profile loads and launches. Reverting to the event restores them; cosmetic fields (name, notes, cover, colour,
// order) are not history.
type ProfileState struct {
	Groups              []Group           `json:"groups,omitempty"`
	Loader              string            `json:"loader,omitempty"`
	Install             string            `json:"install,omitempty"`
	SeparateSaves       bool              `json:"separateSaves,omitempty"`
	LaunchOptions       string            `json:"launchOptions,omitempty"`
	LaunchPrefix        string            `json:"launchPrefix,omitempty"`
	LaunchEnv           string            `json:"launchEnv,omitempty"`
	LaunchPresets       []LaunchPreset    `json:"launchPresets,omitempty"`
	DefaultLaunchPreset string            `json:"defaultLaunchPreset,omitempty"`
	Overrides           map[string]string `json:"overrides,omitempty"`
	Origin              string            `json:"origin,omitempty"`
	Collection          *CollectionRef    `json:"collection,omitempty"`
}

func stateOf(p Profile) ProfileState {
	st := ProfileState{
		Loader: p.Loader, Install: p.Install, SeparateSaves: p.SeparateSaves,
		LaunchOptions: p.LaunchOptions, LaunchPrefix: p.LaunchPrefix, LaunchEnv: p.LaunchEnv,
		DefaultLaunchPreset: p.DefaultLaunchPreset, Origin: p.Origin,
		LaunchPresets: slices.Clone(p.LaunchPresets),
	}
	if p.Collection != nil {
		c := *p.Collection
		st.Collection = &c
	}
	for _, g := range p.Groups {
		st.Groups = append(st.Groups, Group{Name: g.Name, Keys: slices.Clone(g.Keys)})
	}
	st.Overrides = maps.Clone(p.Overrides)
	return st
}

func (st ProfileState) applyTo(p *Profile) {
	p.Groups, p.Loader, p.Install, p.SeparateSaves = st.Groups, st.Loader, st.Install, st.SeparateSaves
	p.LaunchOptions, p.LaunchPrefix, p.LaunchEnv = st.LaunchOptions, st.LaunchPrefix, st.LaunchEnv
	p.LaunchPresets, p.DefaultLaunchPreset, p.Overrides = st.LaunchPresets, st.DefaultLaunchPreset, st.Overrides
	p.Origin, p.Collection = st.Origin, st.Collection
}

// stateChange names what differs between two states, or "" when nothing does. Origin and Collection are set once
// when a profile is made, so they are restored by a revert but never start an event of their own.
func stateChange(a, b ProfileState) HistoryChange {
	same := func(x, y any) bool {
		xb, _ := json.Marshal(x)
		yb, _ := json.Marshal(y)
		return bytes.Equal(xb, yb)
	}
	switch {
	case !same(a.Groups, b.Groups):
		return ChangeGroups
	case a.Loader != b.Loader:
		return ChangeLoader
	case a.Install != b.Install:
		return ChangeInstall
	case a.SeparateSaves != b.SeparateSaves:
		return ChangeSaves
	case a.LaunchOptions != b.LaunchOptions, a.LaunchPrefix != b.LaunchPrefix, a.LaunchEnv != b.LaunchEnv,
		a.DefaultLaunchPreset != b.DefaultLaunchPreset, !same(a.LaunchPresets, b.LaunchPresets):
		return ChangeLaunch
	case !same(a.Overrides, b.Overrides):
		return ChangeSettings
	}
	return ""
}

// configHash is the sha256 of a captured or live config file, "" when there is none.
func configHash(body []byte, ok bool) string {
	if !ok {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// capturedConfig is the hash history holds of the entry's config.json at snapshotID, "" when none was captured.
func capturedConfig(dir, snapshotID, key string) string {
	idx, err := readSnapshotIndex(dir, snapshotID)
	if err != nil {
		return ""
	}
	return idx[key][configFile]
}

func liveConfigHash(dir, key string) string {
	body, err := fsx.ReadFile(filepath.Join(liveEntryDir(filepath.Join(dir, "mods"), key), configFile))
	return configHash(body, err == nil)
}

// restoreConfigsTo puts back the config.json history captured at targetID for each entry, but only where the live file
// still is what the newest event captured: a file edited since (by the game, say) is not Mortar's to roll back. It
// returns the keys of the entries whose config it changed.
func restoreConfigsTo(dir string, entries []Entry, targetID, headID string) []string {
	var changed []string
	for _, e := range entries {
		live := liveConfigHash(dir, e.Key)
		if e.IsOverlay() || live != capturedConfig(dir, headID, e.Key) {
			continue
		}
		want := capturedConfig(dir, targetID, e.Key)
		if want == live {
			continue
		}
		if want == "" {
			if os.Remove(filepath.Join(liveEntryDir(filepath.Join(dir, "mods"), e.Key), configFile)) == nil {
				changed = append(changed, e.Key)
			}
			continue
		}
		if restoreHistoryConfig(dir, targetID, e.Key, configFile) == nil {
			changed = append(changed, e.Key)
		}
	}
	if restoreGmcmPending(dir, targetID, headID) {
		changed = append(changed, gmcmPendingKey)
	}
	return changed
}
