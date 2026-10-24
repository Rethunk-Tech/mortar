package installer

import (
	"encoding/xml"
	"path"
	"strings"
	"testing"
)

// FuzzFOMODLayout gives a FOMOD's required file or folder a fuzzed source and destination, or none. Whatever
// the config says, every file must land inside the package's own folder of the mods target, under names Windows keeps as written.
func FuzzFOMODLayout(f *testing.F) {
	for _, s := range []struct {
		src, dest      string
		folder, noDest bool
	}{
		{"alpha", "alpha", true, false},
		{"alpha/manifest.json", "", false, false},
		{"alpha", "", true, true},
		{"beta/x.dll", "", false, true},
		{"..", "x", true, false},
		{"alpha", "..", true, false},
		{"alpha", "../other-mod", true, false},
		{`alpha\manifest.json`, `..\..\x`, false, false},
		{".", "/abs", true, false},
		{"alpha", "a/./../../b", true, false},
		{"beta/x.dll", "x.dll:stream", false, false},
		{"alpha", "CON", true, false},
	} {
		f.Add(s.src, s.dest, s.folder, s.noDest)
	}
	f.Fuzz(func(t *testing.T, src, dest string, folder, noDest bool) {
		kind := "file"
		if folder {
			kind = "folder"
		}
		pairs := [][2]string{{"source", src}, {"destination", dest}}
		if noDest {
			pairs = pairs[:1]
		}
		var attrs strings.Builder
		for _, a := range pairs {
			attrs.WriteString(" " + a[0] + `="`)
			if xml.EscapeText(&attrs, []byte(a[1])) != nil {
				return
			}
			attrs.WriteString(`"`)
		}
		cfg := `<config><moduleName>Fuzz</moduleName><requiredInstallFiles><` + kind + attrs.String() +
			` priority="0"/></requiredInstallFiles></config>`
		a := extracted(t, "nexus-5-6", map[string]string{
			"fomod/ModuleConfig.xml": cfg, "alpha/manifest.json": "a", "beta/x.dll": "b", "root.txt": "r",
		})
		game := Game{Loaders: []string{"smapi"}, Targets: []Target{{ID: TargetMods}}}
		l, err := fomodInstaller{}.Layout(a, game, nil)
		if err != nil {
			return
		}
		for _, file := range l.Files {
			if file.Target != TargetMods || !strings.HasPrefix(file.Rel, a.Key+"/") || path.Clean(file.Rel) != file.Rel {
				t.Fatalf("source %q destination %q put %q at %s:%q, outside the package's folder", src, dest, file.Src, file.Target, file.Rel)
			}
		}
	})
}
