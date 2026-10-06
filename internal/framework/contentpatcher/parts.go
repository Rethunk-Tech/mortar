package contentpatcher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

// A check after a change recomputes only what the change touches. Each conflict target's outcome is kept under a key
// made of the packs that touch the target, and each pack's compatibility settings under a key of the pack and the mods
// its conditions name. A pack's share of a key is its file fingerprint, its place in the profile and which of the mods
// its conditions name are enabled: presentFor holds the checks to those mods, so nothing else in the profile can change
// a target's or a pack's outcome.

// partsCacheVersion changes whenever what a part records changes, so an older file is ignored.
const partsCacheVersion = 6

// partsKeep is how many checks may go by without using a part before it is dropped: enough for a few profiles' checks
// to take turns without evicting each other.
const partsKeep = 8

type partEntry struct {
	// Files are the stamps, by absolute path, of what computing the part read outside its packs' own files.
	Files    []packFileStamp          `json:"files,omitempty"`
	Conflict *framework.AssetConflict `json:"conflict,omitempty"`
	Settings []framework.SettingHint  `json:"settings,omitempty"`
	Check    int                      `json:"check"`
}

type diskParts struct {
	Version int                  `json:"version"`
	Check   int                  `json:"check"`
	Entries map[string]partEntry `json:"entries"`
}

var parts = struct {
	sync.Mutex
	loaded  bool
	dirty   bool
	check   int
	entries map[string]partEntry
}{entries: map[string]partEntry{}}

func partsCachePath() (string, error) {
	base, err := datadir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "cache", "problems-content-patcher-parts.json.gz"), nil
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
	raw, err := fsx.ReadFile(path)
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

func flushParts() {
	parts.Lock()
	defer parts.Unlock()
	if !parts.dirty {
		return
	}
	path, err := partsCachePath()
	if err != nil {
		return
	}
	raw, err := json.Marshal(diskParts{Version: partsCacheVersion, Check: parts.check, Entries: parts.entries})
	if err != nil {
		return
	}
	packed, err := zipPackCache(raw)
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o700) != nil || datadir.WriteFile(path, packed, 0o600) != nil {
		return
	}
	parts.dirty = false
}

func dropParts() {
	parts.Lock()
	parts.entries, parts.loaded, parts.dirty, parts.check = map[string]partEntry{}, false, false, 0
	parts.Unlock()
}

// partsRun is one check's use of the parts: what it found kept and what it computed. Nothing it computed is kept
// unless the whole check could stamp what it read (commit).
type partsRun struct {
	used  map[string]bool
	fresh map[string]partEntry
}

func newPartsRun() *partsRun {
	loadParts()
	return &partsRun{used: map[string]bool{}, fresh: map[string]partEntry{}}
}

// part serves key's entry, or computes it; stable is false when a pack behind key changed too recently for its
// stamps to be trusted. A nil run always computes.
func (r *partsRun) part(key string, stable bool, compute func() partEntry) partEntry {
	if r == nil {
		return compute()
	}
	parts.Lock()
	e, ok := parts.entries[key]
	parts.Unlock()
	if ok && stampsValid(e.Files) {
		r.used[key] = true
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
	r.fresh[key] = e
	return e
}

func (r *partsRun) commit() {
	if r == nil {
		return
	}
	parts.Lock()
	defer parts.Unlock()
	parts.check++
	for key := range r.used {
		if e, ok := parts.entries[key]; ok {
			e.Check = parts.check
			parts.entries[key] = e
		}
	}
	for key, e := range r.fresh {
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
	if len(r.fresh) > 0 || pruned {
		parts.dirty = true
	}
}

// presentFor narrows the enabled mods to those pack's conditions name, the only ones a check of the pack looks up.
func presentFor(pack cachedPack, present map[string]bool) map[string]bool {
	out := map[string]bool{}
	for id := range pack.mentions {
		if present[id] {
			out[id] = true
		}
	}
	return out
}

// packSig is pack's share of a part's key; stable is false when the pack has no trusted stamps.
func packSig(im framework.Mod, pack cachedPack, present map[string]bool) (sig string, stable bool) {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "%q|%q|%q|%q|%q|%q|", pack.fingerprint, im.ModID(), im.Name, im.Key, im.Folder, im.Author)
	for _, d := range im.Dependencies {
		_, _ = fmt.Fprintf(&b, "d%q", d.ModID())
	}
	for _, id := range im.LoadAfter {
		_, _ = fmt.Fprintf(&b, "a%q", id)
	}
	for _, id := range sortedKeys(present) {
		_, _ = fmt.Fprintf(&b, "p%q", id)
	}
	cutoff := time.Now().Add(-racyStampWindow).UnixNano()
	stable = pack.fingerprint != "" && !slices.ContainsFunc(pack.files, func(f packFileStamp) bool { return f.ModTime > cutoff })
	return b.String(), stable
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// partKey also covers what every part reads besides its packs: the build and the scan depth.
func partKey(fields ...string) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%d|%t", buildStamp(), contentPackParserVersion, SkipImageOverlap)
	for _, f := range fields {
		_, _ = io.WriteString(h, "\x00")
		_, _ = io.WriteString(h, f)
	}
	return hex.EncodeToString(h.Sum(nil))
}
