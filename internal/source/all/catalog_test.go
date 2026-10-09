package all_test

import (
	"regexp"
	"slices"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/metadata"
	"github.com/Rethunk-Tech/mortar/internal/source"
	_ "github.com/Rethunk-Tech/mortar/internal/source/all"
)

var (
	knownLoaders = []string{"smapi", "bepinex5"}
	knownRoles   = []string{"saves", "errorLogs", "startupPreferences", "unityLog"}
	knownTokens  = []string{"appData", "localAppData", "localLow", "documents", "xdgConfig", "xdgData", "home", "install"}
	tokenRE      = regexp.MustCompile(`\{([^{}]*)\}`)
)

// TestEveryCatalogReferenceResolves fails on a catalog edit that names a source, loader, metadata provider, store
// field or path role/token nothing in Mortar answers to.
func TestEveryCatalogReferenceResolves(t *testing.T) {
	m, err := components.BundledManifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Games) == 0 {
		t.Fatal("catalog lists no games")
	}
	ids := map[string]bool{}
	for _, g := range m.Games {
		if g.ID == "" || g.Name == "" || ids[g.ID] {
			t.Errorf("game %q: id and name must be set and the id unique", g.ID)
		}
		ids[g.ID] = true

		for _, s := range g.Sources {
			if _, ok := source.Get(s.ID); !ok {
				t.Errorf("%s: source %q has no registered driver", g.ID, s.ID)
			}
		}
		for _, l := range g.Loaders {
			if !slices.Contains(knownLoaders, l.ID) {
				t.Errorf("%s: loader %q is not a known loader", g.ID, l.ID)
			}
		}
		for _, id := range g.Metadata {
			if p := metadata.For(&meta.Client{}, []string{id}); p == (metadata.Providers{}) {
				t.Errorf("%s: metadata %q has no provider", g.ID, id)
			}
		}
		if s := g.Stores.Steam; s != nil && s.AppID == "" {
			t.Errorf("%s: steam store lacks an app id", g.ID)
		}
		if s := g.Stores.GOG; s != nil && (s.ProductID == "" || s.Folder == "") {
			t.Errorf("%s: gog store lacks a product id or folder", g.ID)
		}
		if s := g.Stores.Lutris; s != nil && (s.Slug == "" || s.Keyword == "") {
			t.Errorf("%s: lutris store lacks a slug or keyword", g.ID)
		}
		if s := g.Stores.EA; s != nil && s.Folder == "" {
			t.Errorf("%s: ea store lacks a folder", g.ID)
		}
		for role, tpl := range g.Paths {
			if !slices.Contains(knownRoles, role) {
				t.Errorf("%s: path role %q is unknown", g.ID, role)
			}
			if tpl == (components.PathTemplate{}) {
				t.Errorf("%s: path %q has no platform", g.ID, role)
			}
			for _, p := range []string{tpl.Windows, tpl.Linux, tpl.Darwin} {
				for _, tok := range tokenRE.FindAllStringSubmatch(p, -1) {
					if !slices.Contains(knownTokens, tok[1]) {
						t.Errorf("%s: path %q uses unknown token {%s}", g.ID, role, tok[1])
					}
				}
			}
		}
	}
}
