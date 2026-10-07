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

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/manifest"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

// sameJobScore is how much of the larger footprint, by member weight, two mods must share to be flagged. They must
// also share a member written by at most sameJobDistinctDF mods: a member most mods write, such as the open menu,
// says nothing about the job even when it is all two small footprints hold.
const (
	sameJobScore      = 0.45
	sameJobDistinctDF = 12
)

// Below sameJobSmall code mods the member weights are too noisy to score, so a pair is flagged only when one
// footprint sits inside the other and every shared member is written by at most sameJobRare mods.
const (
	sameJobSmall = 20
	sameJobRare  = 3
)

// A smaller mod overlaps a larger one when the larger changes sameJobCovered of its footprint by weight. Large
// mods write many common members and so cover small ones by accident: the smaller footprint must weigh at least
// sameJobCoveredWeight, which rules out one or two common members, and every shared member must be written by at most
// sameJobCoveredDF mods.
const (
	sameJobCovered       = 0.8
	sameJobCoveredWeight = 10
	sameJobCoveredDF     = 12
)

// sameJobRows lists the enabled C# mods that do the same job as another, from their assemblies and the last run.
func (s *Service) sameJobRows(gameID, id string, mods []framework.Mod) []framework.Redundant {
	dir, err := s.profiles.ProfileDir(gameID, id)
	if err != nil {
		return nil
	}
	fp := footprints(mods, launchsvc.LatestReplaces(dir))
	if pkgs, _ := s.profiles.EnabledPackages(gameID, id); len(pkgs) > 0 {
		var more map[string]map[string]bool
		more, mods = packageFootprints(pkgs, mods)
		maps.Copy(fp, more)
	}
	return sameJob(fp, mods)
}

// withSameJob adds the "sameJob" rows, skipping mods another check already lists under Redundant.
func withSameJob(r Result, rows []framework.Redundant) Result {
	listed := map[string]bool{}
	for _, x := range r.Redundant {
		listed[x.Key] = true
	}
	for _, x := range rows {
		if !listed[x.Key] {
			r.Redundant = append(r.Redundant, x)
		}
	}
	return r
}

// inputMembers are game members every mod that reads or fakes input assigns: they say how a mod drives the game,
// not what job it does, so a pair sharing only these is not doing the same thing.
var inputMembers = map[string]bool{"mouseClickPolling": true, "lastClick": true, "FacingDirection": true}

func withoutInputMembers(fp map[string]map[string]bool) map[string]map[string]bool {
	out := make(map[string]map[string]bool, len(fp))
	for id, set := range fp {
		out[id] = maps.Clone(set)
		maps.DeleteFunc(out[id], func(member string, _ bool) bool {
			_, name, _ := strings.Cut(member, "::")
			return inputMembers[name]
		})
	}
	return out
}

// footprints maps each enabled C# mod, by folded mod id, to the game members it changes: those its assembly
// assigns, and the methods the bridge saw it replace through Harmony as "harmony:Type::Method". replaces is keyed by
// Harmony ID, which mods set to their SMAPI id by convention.
func footprints(mods []framework.Mod, replaces map[string][]string) map[string]map[string]bool {
	byID := map[string][]string{}
	for owner, methods := range replaces {
		byID[mod.SMAPI(owner).Fold()] = methods
	}
	var ids, dlls []string
	for _, m := range mods {
		if m.Enabled && m.Folder != "" && m.EntryDll != "" && filepath.IsLocal(m.EntryDll) {
			ids = append(ids, m.ModID().Fold())
			dlls = append(dlls, filepath.Join(m.Folder, m.EntryDll))
		}
	}
	out := map[string]map[string]bool{}
	stardew := func(dll string) []string { w, _ := dotnet.Writes(dll, "StardewValley", "Netcode"); return w }
	for i, writes := range assemblyMembers(dlls, stardew) {
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

// sameJob flags pairs of enabled mods whose footprints overlap, weighting each member by how few mods change it, and
// smaller mods whose footprint overlaps a larger one's (listed on the smaller mod only). A footprint sees only what a
// mod assigns in the game, not what it draws or adds to the UI, so an overlap is a hint and never proof of redundancy.
// Pairs that are meant to run together are left out: one depends on the other, they share an author or download,
// or either is something another enabled mod builds on, since a framework writes what its users write.
func sameJob(fp map[string]map[string]bool, mods []framework.Mod) []framework.Redundant {
	fp = withoutInputMembers(fp)
	builtOn := map[string]bool{}
	var code []framework.Mod
	seen := map[string]bool{}
	for _, m := range mods {
		if !m.Enabled {
			continue
		}
		for _, d := range m.Dependencies {
			builtOn[d.ModID().Fold()] = true
		}
		id := m.ModID().Fold()
		if len(fp[id]) > 0 && !seen[id] {
			seen[id] = true
			code = append(code, m)
		}
	}
	df := map[string]int{}
	for _, m := range code {
		for member := range fp[m.ModID().Fold()] {
			df[member]++
		}
	}
	n := len(code)
	weight := func(member string) float64 { return math.Log(float64(n) / float64(df[member])) }
	total := map[string]float64{}
	for _, m := range code {
		id := m.ModID().Fold()
		for member := range fp[id] {
			total[id] += weight(member)
		}
	}
	by := map[string][]framework.ModRef{}
	shared := map[string]map[string]bool{}
	coveredBy := map[string][]framework.ModRef{}
	coveredShared := map[string]map[string]bool{}
	note := func(into map[string]map[string]bool, id string, members []string) {
		if into[id] == nil {
			into[id] = map[string]bool{}
		}
		for _, member := range members {
			into[id][member] = true
		}
	}
	for i, a := range code {
		ida := a.ModID().Fold()
		for _, b := range code[i+1:] {
			idb := b.ModID().Fold()
			if !mayShareJob(a, b, builtOn) {
				continue
			}
			var both []string
			sum := 0.0
			common, rarest := 0, n
			for member := range fp[ida] {
				if fp[idb][member] {
					both = append(both, member)
					sum += weight(member)
					common = max(common, df[member])
					rarest = min(rarest, df[member])
				}
			}
			if len(both) == 0 {
				continue
			}
			var similar bool
			if n < sameJobSmall {
				similar = common <= sameJobRare && len(both) == min(len(fp[ida]), len(fp[idb]))
			} else {
				similar = sum > 0 && rarest <= sameJobDistinctDF && sum >= sameJobScore*max(total[ida], total[idb])
			}
			if similar {
				by[ida] = append(by[ida], framework.ModRef{Key: b.Key, Name: b.Name})
				by[idb] = append(by[idb], framework.ModRef{Key: a.Key, Name: a.Name})
				note(shared, ida, both)
				note(shared, idb, both)
				continue
			}
			small, large := a, b
			if total[idb] < total[ida] {
				small, large = b, a
			}
			ids := small.ModID().Fold()
			if n >= sameJobSmall && total[ids] >= sameJobCoveredWeight && common <= sameJobCoveredDF && sum >= sameJobCovered*total[ids] {
				coveredBy[ids] = append(coveredBy[ids], framework.ModRef{Key: large.Key, Name: large.Name})
				note(coveredShared, ids, both)
			}
		}
	}
	// A similar pair is listed once, on the earlier mod; the later one still appears in By, which is what groups them.
	position := map[string]int{}
	for i, m := range code {
		position[m.Key] = i
	}
	var out []framework.Redundant
	for i, m := range code {
		id := m.ModID().Fold()
		switch {
		case len(by[id]) > 0:
			later := slices.DeleteFunc(slices.Clone(by[id]), func(r framework.ModRef) bool { return position[r.Key] < i })
			if len(later) > 0 {
				out = append(out, framework.Redundant{Kind: "sameJob", Key: m.Key, ID: m.ModID(), Name: m.Name, By: later, Detail: shortMembers(shared[id], weight)})
			}
		case len(coveredBy[id]) > 0:
			out = append(out, framework.Redundant{Kind: "sameJob", Key: m.Key, ID: m.ModID(), Name: m.Name, By: coveredBy[id], Detail: shortMembers(coveredShared[id], weight), Covered: true})
		}
	}
	return out
}

func mayShareJob(a, b framework.Mod, builtOn map[string]bool) bool {
	ida, idb := a.ModID().Fold(), b.ModID().Fold()
	if a.Key == b.Key || builtOn[ida] || builtOn[idb] {
		return false
	}
	if author := strings.TrimSpace(a.Author); author != "" && strings.EqualFold(author, strings.TrimSpace(b.Author)) {
		return false
	}
	return !dependsOn(a, idb) && !dependsOn(b, ida)
}

func dependsOn(m framework.Mod, id string) bool {
	return slices.ContainsFunc(m.Dependencies, func(d manifest.Dependency) bool { return d.ModID().Fold() == id })
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

const assemblyCacheVersion = 3

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

// assemblyMembers reads the game members each assembly changes with read, reusing results for files whose size and
// modification time are unchanged. A DLL belongs to one game, so its path alone keys the cache.
func assemblyMembers(dlls []string, read func(dll string) []string) [][]string {
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
			e = assemblyEntry{Size: info.Size(), MTime: info.ModTime().UnixNano(), Writes: read(dll)}
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
