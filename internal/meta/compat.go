package meta

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/jsonc"
)

// Wiki-backed compatibility list behind https://smapi.io/mods. SMAPI.Web (Pathoschild/SMAPI
// src/SMAPI.Web) loads Pathoschild/SmapiCompatibilityList rather than scraping the HTML table.
const defaultCompatURL = "https://raw.githubusercontent.com/Pathoschild/SmapiCompatibilityList/develop/data/mods.jsonc"

const (
	compatTTL       = 24 * time.Hour
	CompatCacheFile = "smapi-compat.json"
	maxCompat       = 16 << 20
)

// CompatEntry is one SMAPI compatibility-list row, keyed later by UniqueID and Nexus id.
type CompatEntry struct {
	Status        string `json:"status"`
	Summary       string `json:"summary"`
	BrokeIn       string `json:"brokeIn"`
	UnofficialURL string `json:"unofficialUrl"`
	Replacement   string `json:"replacement"`
}

// CompatIndex maps UniqueID (lowercased) and Nexus page id onto the same entry.
type CompatIndex struct {
	ByID    map[string]CompatEntry
	ByNexus map[int]CompatEntry
}

func (idx CompatIndex) Lookup(uniqueID string, nexusID int) (CompatEntry, bool) {
	if uniqueID != "" {
		if e, ok := idx.ByID[strings.ToLower(uniqueID)]; ok {
			return e, true
		}
	}
	if nexusID > 0 {
		if e, ok := idx.ByNexus[nexusID]; ok {
			return e, true
		}
	}
	return CompatEntry{}, false
}

// BrokenOn reports whether a broken entry applies to this Stardew Valley version. A mod that broke in a later game
// version (a beta the player does not run) is not broken for them; a break by anything else (SMAPI, Harmony), or an
// unknown version on either side, counts.
func (e CompatEntry) BrokenOn(gameVersion string) bool {
	if e.Status != StatusBroken {
		return false
	}
	v, ok := strings.CutPrefix(e.BrokeIn, "Stardew Valley ")
	if !ok || gameVersion == "" {
		return true
	}
	c, ok := CompareVersions(strings.TrimSuffix(strings.TrimSpace(v), "?"), gameVersion)
	return !ok || c <= 0
}

// CompatList fetches the SMAPI compatibility JSON (cached a day).
// compatMemo keeps the parsed cache file, so asking again does not re-parse megabytes of JSON while the file on
// disk is unchanged.
type compatMemo struct {
	mu    sync.Mutex
	stamp os.FileInfo
	index CompatIndex
}

func (c *Client) CompatList(ctx context.Context) (CompatIndex, error) {
	path, pathErr := c.cachePath(CompatCacheFile)
	stamp := func() os.FileInfo {
		if pathErr != nil {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil
		}
		return info
	}
	if now := stamp(); now != nil && c.now().Sub(now.ModTime()) < compatTTL {
		c.compat.mu.Lock()
		memo, index := c.compat.stamp, c.compat.index
		c.compat.mu.Unlock()
		if memo != nil && memo.Size() == now.Size() && memo.ModTime().Equal(now.ModTime()) {
			return index, nil
		}
	}
	index, err := Cached(c, CompatCacheFile, compatTTL, func() (CompatIndex, error) {
		return c.fetchCompat(ctx)
	})
	if err == nil {
		c.compat.mu.Lock()
		c.compat.stamp, c.compat.index = stamp(), index
		c.compat.mu.Unlock()
	}
	return index, err
}

func (c *Client) fetchCompat(ctx context.Context) (CompatIndex, error) {
	u := c.CompatURL
	if u == "" {
		u = defaultCompatURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return CompatIndex{}, err
	}
	b, err := c.do(req, maxCompat)
	if err != nil {
		return CompatIndex{}, err
	}
	return parseCompatJSON(b)
}

type rawCompatMod struct {
	ID               json.RawMessage `json:"id"`
	IDs              json.RawMessage `json:"ids"`
	Nexus            flexInt         `json:"nexus"`
	NexusID          flexInt         `json:"nexusID"`
	Compatibility    json.RawMessage `json:"compatibility"`
	UnofficialUpdate json.RawMessage `json:"unofficialUpdate"`
	Status           string          `json:"status"`
	Summary          string          `json:"summary"`
	BrokeIn          string          `json:"brokeIn"`
	UnofficialURL    string          `json:"unofficialUrl"`
	Replacement      json.RawMessage `json:"replacement"`
	Successor        json.RawMessage `json:"successor"`
}

type rawCompatBody struct {
	Status        string          `json:"status"`
	Summary       string          `json:"summary"`
	BrokeIn       string          `json:"brokeIn"`
	UnofficialURL string          `json:"unofficialUrl"`
	Replacement   json.RawMessage `json:"replacement"`
}

type flexInt int

func (n *flexInt) UnmarshalJSON(b []byte) error {
	var i int
	if json.Unmarshal(b, &i) == nil {
		*n = flexInt(i)
		return nil
	}
	var s string
	if json.Unmarshal(b, &s) == nil {
		i, err := strconv.Atoi(strings.TrimSpace(s))
		if err == nil {
			*n = flexInt(i)
		}
	}
	return nil
}

func parseCompatJSON(b []byte) (CompatIndex, error) {
	b = jsonc.Clean(b)
	mods, err := decodeCompatMods(b)
	if err != nil {
		return CompatIndex{}, err
	}
	idx := CompatIndex{
		ByID:    make(map[string]CompatEntry, len(mods)),
		ByNexus: make(map[int]CompatEntry, len(mods)),
	}
	for _, raw := range mods {
		e := entryFromRaw(raw)
		if e.Status == "" {
			continue
		}
		for _, id := range uniqueIDsOf(raw) {
			idx.ByID[strings.ToLower(id)] = e
		}
		nexus := int(raw.NexusID)
		if nexus == 0 {
			nexus = int(raw.Nexus)
		}
		if nexus > 0 {
			idx.ByNexus[nexus] = e
		}
	}
	return idx, nil
}

func decodeCompatMods(b []byte) ([]rawCompatMod, error) {
	var mods []rawCompatMod
	if json.Unmarshal(b, &mods) == nil && (len(mods) > 0 || bytes.Equal(bytes.TrimSpace(b), []byte("[]"))) {
		return mods, nil
	}
	var wrap struct {
		Mods []rawCompatMod `json:"mods"`
		Data []rawCompatMod `json:"data"`
	}
	if err := json.Unmarshal(b, &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Mods) > 0 {
		return wrap.Mods, nil
	}
	return wrap.Data, nil
}

func entryFromRaw(raw rawCompatMod) CompatEntry {
	e := CompatEntry{
		Status:        normalizeCompatStatus(raw.Status),
		Summary:       strings.TrimSpace(raw.Summary),
		BrokeIn:       strings.TrimSpace(raw.BrokeIn),
		UnofficialURL: strings.TrimSpace(raw.UnofficialURL),
		Replacement:   replacementOf(raw.Replacement),
	}
	if len(raw.Compatibility) > 0 {
		var nested rawCompatBody
		if json.Unmarshal(raw.Compatibility, &nested) == nil && (nested.Status != "" || nested.Summary != "") {
			if s := normalizeCompatStatus(nested.Status); s != "" {
				e.Status = s
			}
			if nested.Summary != "" {
				e.Summary = strings.TrimSpace(nested.Summary)
			}
			if nested.BrokeIn != "" {
				e.BrokeIn = strings.TrimSpace(nested.BrokeIn)
			}
			if nested.UnofficialURL != "" {
				e.UnofficialURL = strings.TrimSpace(nested.UnofficialURL)
			}
			if r := replacementOf(nested.Replacement); r != "" {
				e.Replacement = r
			}
		} else {
			var status string
			if json.Unmarshal(raw.Compatibility, &status) == nil {
				if s := normalizeCompatStatus(status); s != "" {
					e.Status = s
				}
			}
		}
	}
	if e.UnofficialURL == "" && len(raw.UnofficialUpdate) > 0 {
		var u struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(raw.UnofficialUpdate, &u) == nil {
			e.UnofficialURL = strings.TrimSpace(u.URL)
		}
	}
	if e.Replacement == "" {
		e.Replacement = replacementOf(raw.Successor)
	}
	// The list's schema leaves status out when it follows from the other fields.
	if e.Status == "" {
		switch {
		case e.UnofficialURL != "":
			e.Status = StatusUnofficial
		case e.BrokeIn != "":
			e.Status = StatusBroken
		default:
			e.Status = StatusOK
		}
	}
	return e
}

const (
	StatusOK         = "ok"
	StatusOptional   = "optional"
	StatusUnofficial = "unofficial"
	StatusBroken     = "broken"
	StatusObsolete   = "obsolete"
	StatusAbandoned  = "abandoned"
)

func normalizeCompatStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ok", "compatible":
		return StatusOK
	case "optional":
		return StatusOptional
	case "workaround":
		// Not compatible; the summary names the alternative.
		return StatusBroken
	case "unofficial":
		return StatusUnofficial
	case "broken":
		return StatusBroken
	case "obsolete":
		return StatusObsolete
	case "abandoned":
		return StatusAbandoned
	default:
		return ""
	}
}

func uniqueIDsOf(raw rawCompatMod) []string {
	ids := append(stringIDs(raw.ID), stringIDs(raw.IDs)...)
	out := ids[:0]
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[strings.ToLower(id)] {
			continue
		}
		seen[strings.ToLower(id)] = true
		out = append(out, id)
	}
	return out
}

func stringIDs(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return []string{one}
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		return many
	}
	return nil
}

func replacementOf(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	var obj struct {
		Name string `json:"name"`
		ID   string `json:"id"`
		URL  string `json:"url"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		for _, v := range []string{obj.Name, obj.ID, obj.URL} {
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil && len(many) > 0 {
		return strings.TrimSpace(many[0])
	}
	return ""
}
