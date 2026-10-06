package bepinex5

import (
	"path"
	"strings"
	"testing"
)

// FuzzParseManifest feeds a Thunderstore package's manifest.json, which every BepInEx install reads for its
// dependencies.
func FuzzParseManifest(f *testing.F) {
	for _, s := range []string{
		"\xef\xbb\xbf" + `{"name":"Mod","version_number":"1.2.3","namespace":"Ns","author":"Ns","dependencies":["BepInEx-BepInExPack-5.4.2100","Ns-Lib-1.0.0"]}`,
		`{"dependencies":["a-b","-x-1","a--1","Ns-Lib-1.0.0-extra"]}`,
		`{"dependencies":"Ns-Lib-1.0.0"}`,
	} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		m, err := ParseManifest(b)
		if err != nil {
			return
		}
		for _, d := range m.Needs() {
			if pkg, ok := strings.CutPrefix(string(d.ModID()), "thunderstore:"); !ok || strings.Count(pkg, "-") != 1 {
				t.Fatalf("dependency %q from %q", d.ModID(), m.Dependencies)
			}
		}
	})
}

// FuzzRoute feeds package file paths and names: every routed file lands under BepInEx/ in the profile root.
func FuzzRoute(f *testing.F) {
	for _, s := range [][2]string{
		{"plugins/Mod.dll", "Ns-Mod"},
		{"BepInEx/config/a.cfg", "Ns-Mod"},
		{`..\..\x.dll`, "Ns-Mod"},
		{"Mod.mm.dll", "Ns-Mod"},
		{"BepInEx/../../x", "Ns-Mod"},
		{"config/../../x", "Ns-Mod"},
		{"a.dll", ".."},
		{"C:/x.dll", "Ns-Mod"},
		{"manifest.json", "Ns-Mod"},
	} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, rel, pkg string) {
		got := Route(rel, pkg)
		if got == "" {
			return
		}
		if path.Clean(got) != got || !strings.HasPrefix(got, "BepInEx/") || strings.Contains(got, ":") || strings.Contains(got, `\`) {
			t.Fatalf("Route(%q, %q) = %q", rel, pkg, got)
		}
		for s := range strings.SplitSeq(got, "/") {
			if s == ".." {
				t.Fatalf("Route(%q, %q) = %q escapes", rel, pkg, got)
			}
		}
	})
}
