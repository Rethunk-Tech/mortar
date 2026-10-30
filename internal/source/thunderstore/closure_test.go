package thunderstore

import (
	"strings"
	"testing"

	tstore "github.com/Rethunk-Tech/go-thunderstore"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/source"
)

func custom(owner, name string, vs ...ver) map[string]any {
	l := listing(owner, name, name, 1, false, false)
	l["versions"] = vs
	return l
}

func v(number string, deps ...string) ver { return ver{VersionNumber: number, Dependencies: deps} }

func TestClosure(t *testing.T) {
	f := newFake(t)
	f.chunk0 = append(f.chunk0,
		custom("BepInEx", "BepInExPack", v("6.0.0"), v("5.4.2305"), v("5.4.2100")),
		custom("Ns", "A", v("1.0.0", "Ns-B-1.0.0", "BepInEx-BepInExPack-5.4.2100")),
		custom("Ns", "B", v("2.0.0", "Ns-D-1.0.0"), v("1.0.0")),
		custom("Ns", "C", v("1.0.0", "Ns-B-2.0.0")),
		custom("Ns", "D", v("1.0.0")),
		custom("Ns", "Ahead", v("1.0.0", "Ns-D-9.0.0")),
		custom("Ns", "Old", v("1.0.0", "BepInEx-BepInExPack-6.0.0")),
		custom("Ns", "Lost", v("1.0.0", "Ghost-Pkg-1.0.0")),
		custom("Ns", "X", v("1.0.0", "Ns-Y-1.0.0")),
		custom("Ns", "Y", v("1.0.0", "Ns-X-1.0.0")),
		custom("BepInEx", "BepInExPack_IL2CPP", v("6.0.788"), v("6.0.700")),
		custom("Ns", "Cat", v("1.0.0", "BepInEx-BepInExPack_IL2CPP-6.0.700")),
		custom("LavaGang", "MelonLoader", v("0.7.1"), v("0.6.0")),
		custom("Ns", "Mel", v("1.0.0", "LavaGang-MelonLoader-0.6.0")),
		custom("ebkr", "r2modman", v("3.0.0")),
		custom("Ns", "OldPin", v("1.0.0", "BepInEx-BepInExPack-5.3.1")),
		custom("Ns", "Managed", v("1.0.0", "ebkr-r2modman-3.0.0", "Ns-D-1.0.0")),
	)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	cases := []struct {
		name    string
		roots   []tstore.Ref
		want    string // "Name Version" in install order
		wantErr string
	}{
		{
			"highest version wins and brings its own dependencies",
			[]tstore.Ref{{Namespace: "Ns", Name: "A", Version: "1.0.0"}, {Namespace: "Ns", Name: "C", Version: ""}},
			"D 1.0.0, B 2.0.0, BepInExPack 5.4.2305, A 1.0.0, C 1.0.0", "",
		},
		{"a root without a version takes the latest", []tstore.Ref{{Namespace: "Ns", Name: "B", Version: ""}}, "D 1.0.0, B 2.0.0", ""},
		{"a dependency's pin is a minimum, so the newest installs", []tstore.Ref{{Namespace: "Ns", Name: "A", Version: "1.0.0"}}, "D 1.0.0, B 2.0.0, BepInExPack 5.4.2305, A 1.0.0", ""},
		{"a root named at an exact version keeps it", []tstore.Ref{{Namespace: "Ns", Name: "A", Version: "1.0.0"}, {Namespace: "Ns", Name: "B", Version: "1.0.0"}}, "B 1.0.0, BepInExPack 5.4.2305, A 1.0.0", ""},
		{"a pin newer than the index is an error", []tstore.Ref{{Namespace: "Ns", Name: "Ahead", Version: ""}}, "", "ns-d 9.0.0 is not in the index"},
		{"another loader major is refused", []tstore.Ref{{Namespace: "Ns", Name: "Old", Version: ""}}, "", "ns-old 1.0.0 -> bepinex-bepinexpack"},
		{"an IL2CPP pack's dependents take its newest BepInEx 6", []tstore.Ref{{Namespace: "Ns", Name: "Cat", Version: ""}}, "BepInExPack_IL2CPP 6.0.788, Cat 1.0.0", ""},
		{"another loader's pack is not held to BepInEx's series", []tstore.Ref{{Namespace: "Ns", Name: "Mel", Version: ""}}, "MelonLoader 0.7.1, Mel 1.0.0", ""},
		{"a missing dependency is left out", []tstore.Ref{{Namespace: "Ns", Name: "Lost", Version: ""}}, "Lost 1.0.0", ""},
		{"a missing root names the chain", []tstore.Ref{{Namespace: "Ghost", Name: "Pkg", Version: ""}}, "", "ghost-pkg is not in the index (requested by ghost-pkg)"},
		{"packages that list each other install together", []tstore.Ref{{Namespace: "Ns", Name: "X", Version: ""}}, "Y 1.0.0, X 1.0.0", ""},
		{"an older pack version of the same major is served by the newest", []tstore.Ref{{Namespace: "Ns", Name: "OldPin", Version: ""}}, "BepInExPack 5.4.2305, OldPin 1.0.0", ""},
		{"a mod manager listed as a dependency is left out", []tstore.Ref{{Namespace: "Ns", Name: "Managed", Version: ""}}, "D 1.0.0, Managed 1.0.0", ""},
		{"a mod manager asked for by name is refused", []tstore.Ref{{Namespace: "ebkr", Name: "r2modman", Version: ""}}, "", "ebkr-r2modman is a mod manager, not a mod"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := d.Closure(t.Context(), "lethal-company", c.roots, "1.2.3", DepsNewest)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("err %v, want it to contain %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, r := range got {
				names = append(names, r.Name+" "+r.Version)
			}
			if strings.Join(names, ", ") != c.want {
				t.Fatalf("got %v, want %s", names, c.want)
			}
		})
	}
}

func TestClosureDependencyVersions(t *testing.T) {
	f := newFake(t)
	f.chunk0 = append(f.chunk0,
		custom("Ns", "A", v("1.0.0", "Ns-B-1.0.0")),
		custom("Ns", "C", v("1.0.0", "Ns-B-2.0.0")),
		custom("Ns", "B", v("3.0.0"), v("2.0.0"), v("1.0.0")),
	)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	for name, c := range map[string]struct {
		deps  Deps
		roots []tstore.Ref
		want  string
	}{
		"newest takes the latest":                     {DepsNewest, []tstore.Ref{{Namespace: "Ns", Name: "A", Version: ""}}, "B 3.0.0, A 1.0.0"},
		"exact takes the pin":                         {DepsPinned, []tstore.Ref{{Namespace: "Ns", Name: "A", Version: ""}}, "B 1.0.0, A 1.0.0"},
		"exact with two pins takes the higher":        {DepsPinned, []tstore.Ref{{Namespace: "Ns", Name: "A", Version: ""}, {Namespace: "Ns", Name: "C", Version: ""}}, "B 2.0.0, A 1.0.0, C 1.0.0"},
		"exact keeps a root named at its own version": {DepsPinned, []tstore.Ref{{Namespace: "Ns", Name: "A", Version: ""}, {Namespace: "Ns", Name: "B", Version: "3.0.0"}}, "B 3.0.0, A 1.0.0"},
		"as-is adds no dependency":                    {DepsNone, []tstore.Ref{{Namespace: "Ns", Name: "A", Version: ""}}, "A 1.0.0"},
		"as-is keeps a root another root needs newer": {DepsNone, []tstore.Ref{{Namespace: "Ns", Name: "C", Version: ""}, {Namespace: "Ns", Name: "B", Version: "1.0.0"}}, "C 1.0.0, B 1.0.0"},
	} {
		got, err := d.Closure(t.Context(), "lethal-company", c.roots, "1.2.3", c.deps)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var names []string
		for _, r := range got {
			names = append(names, r.Name+" "+r.Version)
		}
		if strings.Join(names, ", ") != c.want {
			t.Errorf("%s: got %v, want %s", name, names, c.want)
		}
	}
}

func TestPublisherNeedsOneMatch(t *testing.T) {
	f := newFake(t)
	f.chunk0 = append(f.chunk0, custom("Ns", "Solo", v("1.0.0")), custom("A", "Twin", v("1.0.0")), custom("B", "Twin", v("1.0.0")))
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	for name, c := range map[string]struct {
		pkg, ver, want string
	}{"unique": {"Solo", "1.0.0", "Ns"}, "ambiguous": {"Twin", "1.0.0", ""}, "other version": {"Solo", "2.0.0", ""}} {
		got, ok, err := d.Publisher(t.Context(), "lethal-company", c.pkg, c.ver, "1.2.3")
		if err != nil || got != c.want || ok != (c.want != "") {
			t.Errorf("%s: %q %v %v", name, got, ok, err)
		}
	}
}

func TestSearchLeavesOutModManagers(t *testing.T) {
	f := newFake(t)
	f.chunk0 = append(f.chunk0, listing("Kesomannen", "GaleModManager", "a mod manager", 5, false, false))
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	page, err := d.Search(t.Context(), source.Query{Game: "lethal-company", Key: "lethal-company", Page: 1, Version: "1.2.3"})
	if err != nil || len(page.Items) == 0 {
		t.Fatalf("search: %d items, %v", len(page.Items), err)
	}
	for _, it := range page.Items {
		if it.Name == "GaleModManager" {
			t.Fatalf("search offered a mod manager: %+v", it)
		}
	}
}

// Thunderstore lets a deprecated package keep a differently-cased twin of a live one (recorded from lethal-company:
// MelanieMelicious_2_sToRy_ShIp... 2.2.14 beside a deprecated ..._2_Story_Ship... 0.2.3).
func TestClosureCaseTwinKeepsTheLivePackage(t *testing.T) {
	for _, liveFirst := range []bool{true, false} {
		f := newFake(t)
		live := custom("Mel", "Two_sToRy_Ship", v("2.2.14"), v("2.2.13"))
		twin := custom("Mel", "Two_Story_Ship", v("0.2.3"))
		twin["is_deprecated"] = true
		if liveFirst {
			f.chunk0, f.chunk1 = append(f.chunk0, live), append(f.chunk1, twin)
		} else {
			f.chunk0, f.chunk1 = append(f.chunk0, twin), append(f.chunk1, live)
		}
		d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
		got, err := d.Closure(t.Context(), "lethal-company", []tstore.Ref{{Namespace: "Mel", Name: "Two_sToRy_Ship", Version: "2.2.14"}}, "1.2.3", DepsNewest)
		if err != nil || len(got) != 1 || got[0].Version != "2.2.14" {
			t.Fatalf("liveFirst=%v: %v %v", liveFirst, got, err)
		}
	}
}

func TestClosureSwapsADependencyForItsDropIn(t *testing.T) {
	f := newFake(t)
	f.chunk0 = append(f.chunk0,
		custom("Old", "Loader", v("1.0.0")),
		custom("Mortar", "Loader", v("0.1.0")),
		custom("Ns", "Pack", v("1.0.0", "Old-Loader-1.0.0")),
	)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir(), DropIns: map[string]string{"old-loader": "Mortar-Loader"}}
	for roots, want := range map[tstore.Ref]string{
		{Namespace: "Ns", Name: "Pack", Version: ""}:    "Loader 0.1.0, Pack 1.0.0",
		{Namespace: "Old", Name: "Loader", Version: ""}: "Loader 1.0.0",
	} {
		got, err := d.Closure(t.Context(), "lethal-company", []tstore.Ref{roots}, "1.2.3", DepsNewest)
		if err != nil {
			t.Fatal(err)
		}
		names := make([]string, 0, len(got))
		for _, r := range got {
			names = append(names, r.Name+" "+r.Version)
		}
		if strings.Join(names, ", ") != want {
			t.Errorf("%v: %s, want %s", roots, strings.Join(names, ", "), want)
		}
	}
}

func TestClosureLoaderPackUnderAnotherNamespace(t *testing.T) {
	f := newFake(t)
	f.chunk0 = append(f.chunk0,
		custom("bbepis", "BepInExPack", v("5.4.2122", "Ns-Back-1.0.0")),
		custom("Ns", "Back", v("1.0.0", "bbepis-BepInExPack-5.4.2122")),
		custom("Ns", "Mod", v("1.0.0", "BepInEx-BepInExPack-5.4.2100")),
	)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	got, err := d.Closure(t.Context(), "lethal-company", []tstore.Ref{{Namespace: "Ns", Name: "Mod", Version: ""}}, "1.2.3", DepsNewest)
	if err != nil || len(got) != 1 || got[0].Name != "Mod" {
		t.Fatalf("got %v, %v; want Mod alone: the pack is the loader's, not the mod's to install", got, err)
	}
}

func TestClosureNamesTheMissingDependency(t *testing.T) {
	f := newFake(t)
	f.chunk0 = append(f.chunk0, custom("Ns", "Lost", v("1.0.0", "Ghost-Pkg-1.0.0")))
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	got, err := d.Closure(t.Context(), "lethal-company", []tstore.Ref{{Namespace: "Ns", Name: "Lost", Version: ""}}, "1.2.3", DepsNewest)
	if err != nil || len(got) != 1 || len(got[0].Missing) != 1 || got[0].Missing[0] != "Ghost-Pkg" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestParseVersionPrereleaseIsOlder(t *testing.T) {
	if meta.Newer("1.0.0-beta", "1.0.0") || !meta.Newer("1.0.0", "1.0.0-beta") {
		t.Fatal("a prerelease should rank below its release")
	}
}
