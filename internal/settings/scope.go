package settings

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
)

// Scope names where a setting is read: the most specific part that is set wins, then the broader ones, then the
// key's default. Install and Loader are part of the tuple for keys that will live there; no key does yet.
type Scope struct {
	Source, Game, Install, Loader, Profile string
}

// The scope a registry key lives at; a profile may override a key marked ProfileOverridable.
const (
	ScopeSource  = "source"
	ScopeInstall = "install"
)

// sourceFields lists, per source id, the settings.json names of the keys that belong to that mod source.
var sourceFields = map[string][]string{
	"nexus": {
		"nexusUserId", "nexusName", "nexusPremium", "nexusPreferredDownloadServer", "nexusSeenDownloadServers",
		"nxmHandled", "nxmPreviousHandlers", "nxmAsked", "nxmPreviousName", "nxmRedirectOtherGames",
		"autoTrackNexus", "verifyNexusMD5",
	},
}

// gamesField is the per-game block's name, in settings.json and in exports.
const gamesField = "games"

// scoped is a settings object split by scope, the layout of settings.json and of an export.
type scoped struct {
	Global  map[string]json.RawMessage            `json:"global"`
	Sources map[string]map[string]json.RawMessage `json:"sources"`
	Games   json.RawMessage                       `json:"games,omitempty"`
}

// split sorts a flat settings object into its scopes.
func split(flat map[string]json.RawMessage) scoped {
	out := scoped{Global: map[string]json.RawMessage{}, Sources: map[string]map[string]json.RawMessage{}}
	owner := map[string]string{}
	for src, names := range sourceFields {
		for _, n := range names {
			owner[n] = src
		}
	}
	for k, v := range flat {
		switch src, ok := owner[k]; {
		case k == gamesField:
			out.Games = v
		case ok:
			if out.Sources[src] == nil {
				out.Sources[src] = map[string]json.RawMessage{}
			}
			out.Sources[src][k] = v
		default:
			out.Global[k] = v
		}
	}
	return out
}

// flatten is split's inverse.
func (sc scoped) flatten() map[string]json.RawMessage {
	out := make(map[string]json.RawMessage, len(sc.Global))
	maps.Copy(out, sc.Global)
	for src, fields := range sc.Sources {
		if _, known := sourceFields[src]; !known {
			continue
		}
		for k, v := range fields {
			if slices.Contains(sourceFields[src], k) {
				out[k] = v
			}
		}
	}
	if len(sc.Games) > 0 {
		out[gamesField] = sc.Games
	}
	return out
}

type diskFile struct {
	FormatVersion int `json:"formatVersion,omitempty"`
	scoped
}

// encodeFile is the settings.json document of s.
func encodeFile(s Settings) (diskFile, error) {
	flat, err := asObject(s)
	if err != nil {
		return diskFile{}, err
	}
	delete(flat, "formatVersion")
	return diskFile{FormatVersion: s.FormatVersion, scoped: split(flat)}, nil
}

// decodeFile reads a settings.json document.
func decodeFile(b []byte) (Settings, error) {
	var d diskFile
	if err := json.Unmarshal(b, &d); err != nil {
		return Settings{}, err
	}
	flat := d.flatten()
	if d.FormatVersion != 0 {
		flat["formatVersion"] = json.RawMessage(fmt.Sprint(d.FormatVersion))
	}
	merged, err := json.Marshal(flat)
	if err != nil {
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(merged, &s); err != nil {
		return Settings{}, err
	}
	return s, nil
}
