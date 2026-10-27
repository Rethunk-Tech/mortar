package profile

import (
	"cmp"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

// What a BepInEx package ships, as ModFact.Kind says it.
const (
	KindPlugin  = "plugin"
	KindPatcher = "patcher"
	KindBoth    = "both"
	// KindLibrary is a package whose DLLs declare no plugin and are no patcher: a library other plugins load.
	KindLibrary = "library"
	// KindAssets is a package with no DLL.
	KindAssets = "assets"
)

// ModFact is what the Mods list shows of a mod beyond its manifest. Fields stay empty where the mod's loader or source
// does not say.
type ModFact struct {
	// IDs are the mod's own ids in its loader's terms (loader.ModIDNamer): a SMAPI mod's UniqueID, the GUIDs of the
	// BepInPlugin attributes in a BepInEx package's DLLs, a folder game's modinfo.json modid.
	IDs []string `json:"ids,omitempty"`
	// PluginVersions are the versions those plugins declare, in the order of IDs; only a BepInEx package has them.
	PluginVersions []string `json:"pluginVersions,omitempty"`
	// Kind says what a BepInEx package ships: KindPlugin, KindPatcher, KindBoth, KindLibrary or KindAssets.
	Kind string `json:"kind"`
	// HardDeps are the GUIDs the package's plugins declare a hard BepInDependency on.
	HardDeps []string `json:"hardDeps,omitempty"`
	// Released is when the installed version was published (RFC 3339), Latest the newest version its source offers and
	// LatestReleased its date.
	Released       string `json:"released"`
	Latest         string `json:"latest"`
	LatestReleased string `json:"latestReleased"`
	// Updated, Rating, Downloads and Categories are the source's own figures for the package (Thunderstore's
	// date_updated, rating_score, the downloads of all its versions, and its categories; GitHub fills only Downloads).
	Updated    string   `json:"updated"`
	Rating     int      `json:"rating"`
	Downloads  int      `json:"downloads"`
	Categories []string `json:"categories,omitempty"`
}

// packageScan is what the DLLs of one store item declare.
type packageScan struct {
	ids, versions, hardDeps []string
	kind                    string
}

// scanCacheMax bounds the cached scans: a store item never changes under its key, so an entry is never stale, only
// unused, and a start that scans a few hundred packages again costs a second at most.
const scanCacheMax = 512

type scanCache struct {
	mu sync.Mutex
	m  map[string]packageScan
}

func (c *scanCache) get(key string) (packageScan, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.m[key]
	return v, ok
}

func (c *scanCache) put(key string, v packageScan) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = map[string]packageScan{}
	}
	if _, ok := c.m[key]; !ok && len(c.m) >= scanCacheMax {
		for k := range c.m {
			delete(c.m, k)
			break
		}
	}
	c.m[key] = v
}

// scanPackage reads the plugins, hard dependencies and kind of the package entry e from its stored DLLs, once per store
// key. ok is false for an entry that is no package or whose item is gone.
func (s *Store) scanPackage(game string, e Entry) (packageScan, bool) {
	if !e.Package || s.items == nil {
		return packageScan{}, false
	}
	key := game + "\x00" + e.StoreKey()
	if v, ok := s.scans.get(key); ok {
		return v, true
	}
	dir, err := s.items.Path(game, e.StoreKey())
	if err != nil {
		return packageScan{}, false
	}
	g := installerGame(game)
	bepin := slices.Contains(g.Loaders, bepinex5.ID)
	pkg := cmp.Or(e.Source.Name, e.Key)
	var sc packageScan
	var patcher, plugin, dll bool
	for _, path := range dotnet.DLLs(dir) {
		dll = true
		d, _ := dotnet.Scan(path)
		isPatcher := false
		if rel, err := filepath.Rel(dir, path); bepin && err == nil {
			isPatcher = strings.HasPrefix(bepinex5.Route(filepath.ToSlash(rel), pkg, g.Routes), "BepInEx/patchers/")
		}
		patcher = patcher || isPatcher
		plugin = plugin || !isPatcher && len(d.Plugins) > 0
		for _, pl := range d.Plugins {
			sc.ids, sc.versions = append(sc.ids, pl.GUID), append(sc.versions, pl.Version)
		}
		for _, r := range d.Relations {
			if r.Kind == dotnet.HardDependency && !slices.Contains(sc.hardDeps, r.GUID) {
				sc.hardDeps = append(sc.hardDeps, r.GUID)
			}
		}
	}
	if bepin {
		switch {
		case patcher && plugin:
			sc.kind = KindBoth
		case patcher:
			sc.kind = KindPatcher
		case plugin:
			sc.kind = KindPlugin
		case dll:
			sc.kind = KindLibrary
		default:
			sc.kind = KindAssets
		}
	}
	s.scans.put(key, sc)
	return sc, true
}

// factRef is how a source's cache names the package of e: a Thunderstore package's lower-case Namespace-Name, a GitHub
// repository's lower-case owner/repo. It is empty for a source that has no cached facts.
func factRef(e Entry) string {
	switch e.Source.Kind {
	case KindThunderstore:
		return strings.ToLower(e.Source.Name)
	case KindGitHub:
		return strings.ToLower(e.Source.Repo)
	}
	return ""
}

// ModFacts is the facts of each of the profile's mods, keyed by the mod's id as Mods lists it. A package's source facts
// come from its source's cache (SourceFacts) and are left empty when it holds none.
// lastBuilt is when a mod installed from a watched folder (a project's build output) last changed: its newest file's
// time, RFC 3339; empty for any other entry.
func lastBuilt(e Entry) string {
	if e.Source.Kind != KindLocal || e.Source.Folder == "" {
		return ""
	}
	var newest time.Time
	_ = filepath.WalkDir(e.Source.Folder, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			// An unreadable folder adds nothing; the rest are still read.
			return fs.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil && info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	if newest.IsZero() {
		return ""
	}
	return newest.UTC().Format(time.RFC3339)
}

func (s *Store) ModFacts(game, id string) (map[string]ModFact, error) {
	p, err := s.read(game, id)
	if err != nil {
		return nil, err
	}
	out := map[string]ModFact{}
	installed := map[string]map[string]string{}
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		sc, scanned := s.scanPackage(game, e)
		ref := factRef(e)
		built := lastBuilt(e)
		for _, c := range e.Mods {
			if _, dup := out[string(c.ID)]; dup {
				continue
			}
			var f ModFact
			switch {
			case c.ModID != "":
				f.IDs = []string{c.ModID}
			case scanned:
				f.IDs, f.PluginVersions = slices.Clone(sc.ids), slices.Clone(sc.versions)
			case c.ID.Format() == mod.FormatSMAPI:
				f.IDs = []string{c.ID.Local()}
			}
			if scanned {
				f.Kind, f.HardDeps = sc.kind, slices.Clone(sc.hardDeps)
			}
			f.Updated = built
			out[string(c.ID)] = f
			if ref != "" {
				if installed[e.Source.Kind] == nil {
					installed[e.Source.Kind] = map[string]string{}
				}
				installed[e.Source.Kind][ref] = cmp.Or(e.Source.Version, c.Version)
			}
		}
	}
	if s.SourceFacts == nil {
		return out, nil
	}
	facts := map[string]map[string]source.Facts{}
	for kind, refs := range installed {
		facts[kind] = s.SourceFacts(game, kind, refs)
	}
	for _, e := range p.Entries {
		f, ok := facts[e.Source.Kind][factRef(e)]
		if !ok {
			continue
		}
		for _, c := range e.Mods {
			m := out[string(c.ID)]
			m.Released, m.Latest, m.LatestReleased = f.Released, f.Latest, f.LatestReleased
			m.Updated, m.Rating, m.Downloads, m.Categories = f.Updated, f.Rating, f.Downloads, f.Categories
			out[string(c.ID)] = m
		}
	}
	return out, nil
}
