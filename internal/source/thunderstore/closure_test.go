package thunderstore

import (
	"strings"
	"testing"

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
		custom("ebkr", "r2modman", v("3.0.0")),
		custom("Ns", "Managed", v("1.0.0", "ebkr-r2modman-3.0.0", "Ns-D-1.0.0")),
	)
	d := Driver{URL: f.srv.URL, CacheDir: t.TempDir()}
	cases := []struct {
		name    string
		roots   []Ref
		want    string // "Name Version" in install order
		wantErr string
	}{
		{
			"highest version wins and brings its own dependencies",
			[]Ref{{"Ns", "A", "1.0.0"}, {"Ns", "C", ""}},
			"D 1.0.0, B 2.0.0, BepInExPack 5.4.2305, A 1.0.0, C 1.0.0", "",
		},
		{"a root without a version takes the latest", []Ref{{"Ns", "B", ""}}, "D 1.0.0, B 2.0.0", ""},
		{"a dependency's pin is a minimum, so the newest installs", []Ref{{"Ns", "A", "1.0.0"}}, "D 1.0.0, B 2.0.0, BepInExPack 5.4.2305, A 1.0.0", ""},
		{"a root named at an exact version keeps it", []Ref{{"Ns", "A", "1.0.0"}, {"Ns", "B", "1.0.0"}}, "B 1.0.0, BepInExPack 5.4.2305, A 1.0.0", ""},
		{"a pin newer than the index is an error", []Ref{{"Ns", "Ahead", ""}}, "", "ns-d 9.0.0 is not in the index"},
		{"another loader major is refused", []Ref{{"Ns", "Old", ""}}, "", "ns-old 1.0.0 -> bepinex-bepinexpack"},
		{"a missing package names the chain", []Ref{{"Ns", "Lost", ""}}, "", "ghost-pkg is not in the index (requested by ns-lost 1.0.0 -> ghost-pkg)"},
		{"a cycle is an error", []Ref{{"Ns", "X", ""}}, "", "dependency cycle: ns-x 1.0.0 -> ns-y 1.0.0 -> ns-x"},
		{"a mod manager listed as a dependency is left out", []Ref{{"Ns", "Managed", ""}}, "D 1.0.0, Managed 1.0.0", ""},
		{"a mod manager asked for by name is refused", []Ref{{"ebkr", "r2modman", ""}}, "", "ebkr-r2modman is a mod manager, not a mod"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := d.Closure(t.Context(), "lethal-company", c.roots, "1.2.3")
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
