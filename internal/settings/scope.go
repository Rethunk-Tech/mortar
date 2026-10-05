package settings

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
)

// Scope names where a setting is read: the most specific part that is set wins, then the broader ones, then the
// key's default. Install is part of the tuple for keys that will live there; no key does yet.
type Scope struct {
	Source, Game, Install, Loader, Profile string
}

// The scope a registry key lives at; a profile may override a key marked ProfileOverridable.
const (
	ScopeSource  = "source"
	ScopeLoader  = "loader"
	ScopeInstall = "install"
)

// sourceFields lists, per source id, the keys that belong to that mod source. An entry is the Settings field's JSON
// name, or "name=local" when the key is stored in the source's block under a shorter local name.
var sourceFields = map[string][]string{
	"thunderstore": {"thunderstoreHandleLinks=handleLinks"},
	"nexus": {
		"nexusUserId=userId", "nexusName=name", "nexusPremium=premium",
		"nexusPreferredDownloadServer=preferredDownloadServer", "nexusSeenDownloadServers=seenDownloadServers",
		"nxmHandled", "nxmPreviousHandlers", "nxmAsked", "nxmPreviousName", "nxmRedirectOtherGames",
		"autoTrackNexus=autoTrack", "verifyNexusMD5=verifyMD5",
	},
}

// loaderFields is sourceFields for mod loaders: the keys that belong to one loader, whichever game uses it.
var loaderFields = map[string][]string{
	"smapi": {"smapiBuilds=builds", "smapiPin=pin", "showSmapiConsole=showConsole", "tellWhenSmapiOut=tellWhenOut", "smapiToastAt=toastAt"},
}

// scopeTables names each keyed scope's block in settings.json and its field table.
var scopeTables = []struct {
	block  string
	fields map[string][]string
}{{"sources", sourceFields}, {"loaders", loaderFields}}

func fieldNames(entry string) (flat, local string) {
	flat, local, found := strings.Cut(entry, "=")
	if !found {
		local = flat
	}
	return flat, local
}

// gamesField is the per-game block's name, in settings.json and in exports.
const gamesField = "games"

// scoped is a settings object split by scope, the layout of settings.json and of an export.
type scoped struct {
	Global  map[string]json.RawMessage            `json:"global"`
	Sources map[string]map[string]json.RawMessage `json:"sources"`
	Loaders map[string]map[string]json.RawMessage `json:"loaders"`
	Games   json.RawMessage                       `json:"games,omitempty"`
}

// blocks is the keyed scopes of sc by block name.
func (sc scoped) blocks() map[string]map[string]map[string]json.RawMessage {
	return map[string]map[string]map[string]json.RawMessage{"sources": sc.Sources, "loaders": sc.Loaders}
}

// split sorts a flat settings object into its scopes.
func split(flat map[string]json.RawMessage) scoped {
	out := scoped{
		Global:  map[string]json.RawMessage{},
		Sources: map[string]map[string]json.RawMessage{},
		Loaders: map[string]map[string]json.RawMessage{},
	}
	type home struct{ block, id, local string }
	owner := map[string]home{}
	for _, tbl := range scopeTables {
		for id, names := range tbl.fields {
			for _, n := range names {
				flatName, local := fieldNames(n)
				owner[flatName] = home{tbl.block, id, local}
			}
		}
	}
	blocks := out.blocks()
	for k, v := range flat {
		switch h, ok := owner[k]; {
		case k == gamesField:
			out.Games = v
		case ok:
			block := blocks[h.block]
			if block[h.id] == nil {
				block[h.id] = map[string]json.RawMessage{}
			}
			block[h.id][h.local] = v
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
	blocks := sc.blocks()
	for _, tbl := range scopeTables {
		for id, fields := range blocks[tbl.block] {
			for _, n := range tbl.fields[id] {
				flatName, local := fieldNames(n)
				if v, ok := fields[local]; ok {
					out[flatName] = v
				}
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
