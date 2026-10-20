package settings

import (
	"bytes"
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
	"curseforge":   {"curseforgeHandleLinks=handleLinks"},
	"vsmoddb":      {"vsmoddbHandleLinks=handleLinks"},
	"itch":         {"itchName=name"},
	"nexus": {
		"nexusUserId=userId", "nexusName=name", "nexusPremium=premium",
		"nexusPreferredDownloadServer=preferredDownloadServer", "nexusSeenDownloadServers=seenDownloadServers",
		"nxmHandled", "nxmPreviousHandlers", "nxmAsked", "nxmPreviousName", "nxmRedirectOtherGames",
		"autoTrackNexus=autoTrack", "verifyNexusMD5=verifyMD5",
	},
}

// loaderFields is sourceFields for mod loaders: the keys that belong to one loader, whichever game uses it.
var loaderFields = map[string][]string{
	"smapi": {"smapiBuilds=builds", "showSmapiConsole=showConsole"},
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

// loaderPrefsField is the flat name of Settings.LoaderPrefs: its fields live in the loaders block under
// LoaderKey(game, loader), such as "valheim/bepinex5", beside the loader-id blocks loaderFields fills.
const loaderPrefsField = "loaderPrefs"

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
		case k == loaderPrefsField:
			var prefs map[string]map[string]json.RawMessage
			if json.Unmarshal(v, &prefs) == nil {
				for id, fields := range prefs {
					if out.Loaders[id] == nil {
						out.Loaders[id] = map[string]json.RawMessage{}
					}
					maps.Copy(out.Loaders[id], fields)
				}
			}
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
	prefs := map[string]map[string]json.RawMessage{}
	for id, fields := range sc.Loaders {
		known := map[string]bool{}
		for _, n := range loaderFields[id] {
			_, local := fieldNames(n)
			known[local] = true
		}
		for name, v := range fields {
			if known[name] {
				continue
			}
			if prefs[id] == nil {
				prefs[id] = map[string]json.RawMessage{}
			}
			prefs[id][name] = v
		}
	}
	if len(prefs) > 0 {
		if b, err := json.Marshal(prefs); err == nil {
			out[loaderPrefsField] = b
		}
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
	def, err := asObject(Defaults())
	if err != nil {
		return diskFile{}, err
	}
	// The shortcuts in s are the user's overrides, which the defaults never equal; none is the common case.
	def["shortcuts"] = json.RawMessage("{}")
	for k, v := range flat {
		if d, ok := def[k]; ok && bytes.Equal(v, d) {
			delete(flat, k)
		}
	}
	doc := diskFile{FormatVersion: s.FormatVersion, scoped: split(flat)}
	if len(doc.Games) > 0 {
		games, err := stripDefaultGames(doc.Games)
		if err != nil {
			return diskFile{}, err
		}
		doc.Games = games
	}
	return doc, nil
}

// stripDefaultGames drops, from each game's block, the prefs that equal that game's defaults, and the games left with
// none, so a default changed in code reaches a game that never set it.
func stripDefaultGames(raw json.RawMessage) (json.RawMessage, error) {
	var games map[string]json.RawMessage
	if err := json.Unmarshal(raw, &games); err != nil {
		return nil, err
	}
	for id, block := range games {
		def, err := asObject(defaultGameSettingsFor(id))
		if err != nil {
			return nil, err
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(block, &fields); err != nil {
			return nil, err
		}
		for k, v := range fields {
			if d, ok := def[k]; ok && bytes.Equal(v, d) {
				delete(fields, k)
			}
		}
		if len(fields) == 0 {
			delete(games, id)
			continue
		}
		if games[id], err = json.Marshal(fields); err != nil {
			return nil, err
		}
	}
	if len(games) == 0 {
		return nil, nil
	}
	return json.Marshal(games)
}

// decodeFile reads a settings.json document.
func decodeFile(b []byte) (Settings, error) {
	var d diskFile
	// Windows editors and PowerShell's utf8 encoding write a byte order mark that JSON does not allow.
	if err := json.Unmarshal(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")), &d); err != nil {
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
	// A key the file lacks keeps its default, so a default changed in code reaches a file that never set it. The
	// shortcuts are the exception: the file holds only the user's overrides, which normalizeShortcuts lays over the
	// defaults.
	s := Defaults()
	s.Shortcuts = nil
	if err := json.Unmarshal(merged, &s); err != nil {
		return Settings{}, err
	}
	for id, block := range d.gameBlocks() {
		g := defaultGameSettingsFor(id)
		if err := json.Unmarshal(block, &g); err != nil {
			return Settings{}, err
		}
		s.Games[id] = &g
	}
	return s, nil
}

// gameBlocks is the file's per-game blocks by game id.
func (d diskFile) gameBlocks() map[string]json.RawMessage {
	var games map[string]json.RawMessage
	if len(d.Games) > 0 && json.Unmarshal(d.Games, &games) == nil {
		return games
	}
	return nil
}
