package profile

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/Rethunk-Tech/mortar/internal/dotnet"
	"github.com/Rethunk-Tech/mortar/internal/loader/bepinex5"
	"github.com/Rethunk-Tech/mortar/internal/mod"
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
	// date_updated, rating_score, the downloads of all its versions, and its categories).
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

// ModFacts is the facts of each of the profile's mods, keyed by the mod's id as Mods lists it. A Thunderstore package's
// source facts come from the cached community index (PackageFacts) and are left empty when it holds none.
func (s *Store) ModFacts(game, id string) (map[string]ModFact, error) {
	p, err := s.read(game, id)
	if err != nil {
		return nil, err
	}
	out := map[string]ModFact{}
	installed := map[string]string{}
	for _, e := range p.Entries {
		if e.Source.Bundled() {
			continue
		}
		sc, scanned := s.scanPackage(game, e)
		pkgName := strings.ToLower(e.Source.Name)
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
			out[string(c.ID)] = f
			if e.Source.Kind == KindThunderstore && pkgName != "" {
				installed[pkgName] = cmp.Or(e.Source.Version, c.Version)
			}
		}
	}
	if s.PackageFacts == nil || len(installed) == 0 {
		return out, nil
	}
	facts := s.PackageFacts(game, installed)
	for _, e := range p.Entries {
		f, ok := facts[strings.ToLower(e.Source.Name)]
		if !ok || e.Source.Kind != KindThunderstore {
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
