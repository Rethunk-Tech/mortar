package problems

import (
	"cmp"
	"encoding/json"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/dotnet"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/manifest"
)

// sameJobScore is how much of the larger footprint, by member weight, two mods must share to be flagged.
const sameJobScore = 0.45

// Below sameJobSmall code mods the member weights are too noisy to score, so a pair is flagged only when one
// footprint sits inside the other and every shared member is written by at most sameJobRare mods.
const (
	sameJobSmall = 20
	sameJobRare  = 3
)

// withSameJob adds the "sameJob" rows, skipping mods another check already lists under Redundant.
func (s *Service) withSameJob(gameID, id string, r Result, mods []Installed) Result {
	dir, err := s.profiles.ProfileDir(gameID, id)
	if err != nil {
		return r
	}
	listed := map[string]bool{}
	for _, x := range r.Redundant {
		listed[x.Key] = true
	}
	for _, x := range sameJob(footprints(mods, launchsvc.LatestReplaces(dir)), mods) {
		if !listed[x.Key] {
			r.Redundant = append(r.Redundant, x)
		}
	}
	return r
}

// footprints maps each enabled C# mod, by lower-case UniqueID, to the game members it changes: those its assembly
// assigns, and the methods the bridge saw it replace through Harmony as "harmony:Type::Method". replaces is keyed by
// Harmony ID, which mods set to their UniqueID by convention.
func footprints(mods []Installed, replaces map[string][]string) map[string]map[string]bool {
	byID := map[string][]string{}
	for owner, methods := range replaces {
		byID[strings.ToLower(owner)] = methods
	}
	var ids, dlls []string
	for _, m := range mods {
		if m.Enabled && m.Folder != "" && m.EntryDll != "" && filepath.IsLocal(m.EntryDll) {
			ids = append(ids, strings.ToLower(m.UniqueID))
			dlls = append(dlls, filepath.Join(m.Folder, m.EntryDll))
		}
	}
	out := map[string]map[string]bool{}
	for i, writes := range assemblyWrites(dlls) {
		set := map[string]bool{}
		for _, member := range writes {
			set[member] = true
		}
		for _, method := range byID[ids[i]] {
			set["harmony:"+method] = true
		}
		if len(set) > 0 {
			out[ids[i]] = set
		}
	}
	return out
}

// sameJob flags pairs of enabled mods whose footprints overlap, weighting each member by how few mods change it.
// Pairs that are meant to run together are left out: one depends on the other, they share an author or download,
// or either is something another enabled mod builds on, since a framework writes what its users write.
func sameJob(fp map[string]map[string]bool, mods []Installed) []Redundant {
	builtOn := map[string]bool{}
	var code []Installed
	seen := map[string]bool{}
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		for _, d := range m.Dependencies {
			builtOn[strings.ToLower(d.UniqueID)] = true
		}
		id := strings.ToLower(m.UniqueID)
		if len(fp[id]) > 0 && !seen[id] {
			seen[id] = true
			code = append(code, m)
		}
	}
	df := map[string]int{}
	for _, m := range code {
		for member := range fp[strings.ToLower(m.UniqueID)] {
			df[member]++
		}
	}
	n := len(code)
	weight := func(member string) float64 { return math.Log(float64(n) / float64(df[member])) }
	total := map[string]float64{}
	for _, m := range code {
		id := strings.ToLower(m.UniqueID)
		for member := range fp[id] {
			total[id] += weight(member)
		}
	}
	by := map[string][]ModRef{}
	shared := map[string]map[string]bool{}
	for i, a := range code {
		ida := strings.ToLower(a.UniqueID)
		for _, b := range code[i+1:] {
			idb := strings.ToLower(b.UniqueID)
			if !mayShareJob(a, b, builtOn) {
				continue
			}
			var both []string
			sum := 0.0
			rare := true
			for member := range fp[ida] {
				if fp[idb][member] {
					both = append(both, member)
					sum += weight(member)
					rare = rare && df[member] <= sameJobRare
				}
			}
			if len(both) == 0 {
				continue
			}
			if n < sameJobSmall {
				if !rare || len(both) < min(len(fp[ida]), len(fp[idb])) {
					continue
				}
			} else if sum == 0 || sum < sameJobScore*max(total[ida], total[idb]) {
				continue
			}
			by[ida] = append(by[ida], ModRef{Key: b.Key, Name: b.Name})
			by[idb] = append(by[idb], ModRef{Key: a.Key, Name: a.Name})
			for _, id := range []string{ida, idb} {
				if shared[id] == nil {
					shared[id] = map[string]bool{}
				}
				for _, member := range both {
					shared[id][member] = true
				}
			}
		}
	}
	var out []Redundant
	for _, m := range code {
		if id := strings.ToLower(m.UniqueID); len(by[id]) > 0 {
			out = append(out, Redundant{Kind: "sameJob", Key: m.Key, UniqueID: m.UniqueID, Name: m.Name, By: by[id], Detail: shortMembers(shared[id], weight)})
		}
	}
	return out
}

func mayShareJob(a, b Installed, builtOn map[string]bool) bool {
	ida, idb := strings.ToLower(a.UniqueID), strings.ToLower(b.UniqueID)
	if a.Key == b.Key || builtOn[ida] || builtOn[idb] {
		return false
	}
	if author := strings.TrimSpace(a.Author); author != "" && strings.EqualFold(author, strings.TrimSpace(b.Author)) {
		return false
	}
	return !dependsOn(a, idb) && !dependsOn(b, ida)
}

func dependsOn(m Installed, id string) bool {
	return slices.ContainsFunc(m.Dependencies, func(d manifest.Dependency) bool { return strings.EqualFold(d.UniqueID, id) })
}

// shortMembers names up to three members, most distinctive first, as "Type.Member" from "Namespace.Type::Member".
// Members every mod writes say nothing, so they are named only when there is nothing else.
func shortMembers(members map[string]bool, weight func(string) float64) string {
	sorted := slices.Collect(maps.Keys(members))
	slices.SortFunc(sorted, func(a, b string) int {
		return cmp.Or(cmp.Compare(weight(b), weight(a)), strings.Compare(a, b))
	})
	var names []string
	for _, member := range sorted {
		if weight(member) == 0 && len(names) > 0 {
			break
		}
		typ, name, _ := strings.Cut(strings.TrimPrefix(member, "harmony:"), "::")
		if short := typ[strings.LastIndexAny(typ, ".+")+1:] + "." + name; !slices.Contains(names, short) {
			names = append(names, short)
		}
	}
	if len(names) > 3 {
		return strings.Join(names[:3], ", ") + ", …"
	}
	return strings.Join(names, ", ")
}

const assemblyCacheVersion = 1

type assemblyEntry struct {
	Size   int64    `json:"size"`
	MTime  int64    `json:"mtime"`
	Writes []string `json:"writes,omitempty"`
}

type assemblyCacheFile struct {
	Version int                      `json:"version"`
	DLLs    map[string]assemblyEntry `json:"dlls"`
}

var assemblyCache struct {
	sync.Mutex
	path    string
	entries map[string]assemblyEntry
}

// assemblyWrites reads what each assembly assigns in the game, reusing results for files whose size and modification
// time are unchanged. An unreadable assembly writes nothing.
func assemblyWrites(dlls []string) [][]string {
	assemblyCache.Lock()
	defer assemblyCache.Unlock()
	path := ""
	if base, err := datadir.Dir(); err == nil {
		path = filepath.Join(base, "cache", "problems-assemblies.json")
	}
	if assemblyCache.entries == nil || assemblyCache.path != path {
		assemblyCache.path = path
		var disk assemblyCacheFile
		if found, err := datadir.ReadJSON(path, &disk); path == "" || !found || err != nil || disk.Version != assemblyCacheVersion || disk.DLLs == nil {
			disk.DLLs = map[string]assemblyEntry{}
		}
		assemblyCache.entries = disk.DLLs
	}
	dirty := false
	out := make([][]string, len(dlls))
	for i, dll := range dlls {
		info, err := os.Stat(dll)
		if err != nil {
			continue
		}
		e, ok := assemblyCache.entries[dll]
		if !ok || e.Size != info.Size() || e.MTime != info.ModTime().UnixNano() {
			writes, _ := dotnet.Writes(dll, "StardewValley", "Netcode")
			e = assemblyEntry{Size: info.Size(), MTime: info.ModTime().UnixNano(), Writes: writes}
			assemblyCache.entries[dll] = e
			dirty = true
		}
		out[i] = e.Writes
	}
	if dirty && path != "" {
		for dll := range assemblyCache.entries {
			if _, err := os.Stat(dll); err != nil {
				delete(assemblyCache.entries, dll)
			}
		}
		if raw, err := json.Marshal(assemblyCacheFile{Version: assemblyCacheVersion, DLLs: assemblyCache.entries}); err == nil && os.MkdirAll(filepath.Dir(path), 0o700) == nil {
			_ = datadir.WriteFile(path, raw, 0o600)
		}
	}
	return out
}
