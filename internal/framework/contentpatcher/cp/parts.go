package cp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
)

// A check after a change recomputes only what the change touches. Each conflict target's outcome is kept under a key
// made of the packs that touch the target, and each pack's compatibility settings under a key of the pack and the mods
// its conditions name. A pack's share of a key is its file fingerprint, its place in the profile and which of the mods
// its conditions name are enabled: PresentFor holds the checks to those mods, so nothing else in the profile can change
// a target's or a pack's outcome.

// partsCacheVersion changes whenever what a part records changes, so an older file is ignored.
const partsCacheVersion = 6

// partsKeep is how many checks may go by without using a part before it is dropped: enough for a few profiles' checks
// to take turns without evicting each other.
const partsKeep = 8

type PartEntry struct {
	// Files are the stamps, by absolute path, of what computing the part read outside its packs' own files.
	Files    []FileStamp `json:"files,omitempty"`
	Conflict *Conflict   `json:"conflict,omitempty"`
	Settings []Hint      `json:"settings,omitempty"`
	Check    int         `json:"check"`
}

type diskParts struct {
	Version int                  `json:"version"`
	Check   int                  `json:"check"`
	Entries map[string]PartEntry `json:"entries"`
}

var parts = struct {
	sync.Mutex
	loaded  bool
	dirty   bool
	check   int
	entries map[string]PartEntry
}{entries: map[string]PartEntry{}}

func partsCachePath() (string, error) {
	if cacheRoot == "" {
		return "", errNoCacheRoot
	}
	return filepath.Join(cacheRoot, "cache", "problems-content-patcher-parts.json.gz"), nil
}

func loadParts() {
	parts.Lock()
	defer parts.Unlock()
	if parts.loaded {
		return
	}
	parts.loaded = true
	path, err := partsCachePath()
	if err != nil {
		return
	}
	raw, err := safefs.ReadFile(path)
	if err != nil {
		return
	}
	payload, ok := unzipPackCache(raw)
	if !ok {
		return
	}
	var saved diskParts
	if json.Unmarshal(payload, &saved) == nil && saved.Version == partsCacheVersion && saved.Entries != nil {
		parts.entries, parts.check = saved.Entries, saved.Check
	}
}

func FlushParts() {
	parts.Lock()
	defer parts.Unlock()
	if !parts.dirty {
		return
	}
	path, err := partsCachePath()
	if err != nil {
		return
	}
	if writePackedCache(path, diskParts{Version: partsCacheVersion, Check: parts.check, Entries: parts.entries}) {
		parts.dirty = false
	}
}

func DropParts() {
	parts.Lock()
	parts.entries, parts.loaded, parts.dirty, parts.check = map[string]PartEntry{}, false, false, 0
	parts.Unlock()
}

// PartsRun is one check's use of the parts: what it found kept and what it computed. Nothing it computed is kept
// unless the whole check could stamp what it read (commit).
type PartsRun struct {
	Used  map[string]bool
	Fresh map[string]PartEntry
}

func NewPartsRun() *PartsRun {
	loadParts()
	return &PartsRun{Used: map[string]bool{}, Fresh: map[string]PartEntry{}}
}

// Part serves key's entry, or computes it; stable is false when a pack behind key changed too recently for its
// stamps to be trusted. A nil run always computes.
func (r *PartsRun) Part(key string, stable bool, compute func() PartEntry) PartEntry {
	if r == nil {
		return compute()
	}
	parts.Lock()
	e, ok := parts.entries[key]
	parts.Unlock()
	if ok && stampsValid(e.Files) {
		r.Used[key] = true
		return e
	}
	mark := markReads()
	e = compute()
	files, complete := readsSince(mark)
	if !stable || !complete {
		return e
	}
	cutoff := time.Now().Add(-racyStampWindow).UnixNano()
	seen := map[string]bool{}
	for _, f := range files {
		if f.ModTime > cutoff {
			return e
		}
		if !seen[f.Path] {
			seen[f.Path] = true
			e.Files = append(e.Files, f)
		}
	}
	r.Fresh[key] = e
	return e
}

func (r *PartsRun) Commit() {
	if r == nil {
		return
	}
	parts.Lock()
	defer parts.Unlock()
	parts.check++
	for key := range r.Used {
		if e, ok := parts.entries[key]; ok {
			e.Check = parts.check
			parts.entries[key] = e
		}
	}
	for key, e := range r.Fresh {
		e.Check = parts.check
		parts.entries[key] = e
	}
	pruned := false
	for key, e := range parts.entries {
		if e.Check <= parts.check-partsKeep {
			delete(parts.entries, key)
			pruned = true
		}
	}
	// A check that only reused parts leaves the file as it is; their use is saved with the next change.
	if len(r.Fresh) > 0 || pruned {
		parts.dirty = true
	}
}

// PresentFor narrows the enabled mods to those pack's conditions name, the only ones a check of the pack looks up.
func PresentFor(pack Pack, present map[string]bool) map[string]bool {
	out := map[string]bool{}
	for id := range pack.Mentions {
		if present[id] {
			out[id] = true
		}
	}
	return out
}

// PackSig is pack's share of a part's key; stable is false when the pack has no trusted stamps.
func PackSig(im Mod, pack Pack, present map[string]bool) (sig string, stable bool) {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%q|%q|%q|%q|%q|%q|", pack.Fingerprint, im.UniqueID, im.Name, im.Key, im.Folder, im.Author)
	for _, d := range im.Dependencies {
		_, _ = fmt.Fprintf(&b, "d%q", d.UniqueID)
	}
	for _, id := range im.LoadAfter {
		_, _ = fmt.Fprintf(&b, "a%q", id)
	}
	for _, id := range slices.Sorted(maps.Keys(present)) {
		_, _ = fmt.Fprintf(&b, "p%q", id)
	}
	cutoff := time.Now().Add(-racyStampWindow).UnixNano()
	stable = pack.Fingerprint != "" && !slices.ContainsFunc(pack.Files, func(f FileStamp) bool { return f.ModTime > cutoff })
	return b.String(), stable
}

// PartKey also covers what every part reads besides its packs: the build and the scan depth.
func PartKey(skip bool, fields ...string) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%d|%t", buildStamp(), contentPackParserVersion, skip)
	for _, f := range fields {
		_, _ = io.WriteString(h, "\x00")
		_, _ = io.WriteString(h, f)
	}
	return hex.EncodeToString(h.Sum(nil))
}
