package installer

import (
	"encoding/xml"
	"path"
	"strings"
	"testing"
)

// FuzzFOMODLayout gives a FOMOD's required file or folder a fuzzed source and destination. Whatever the config says,
// every file must land inside the package's own folder of the mods target, under names Windows keeps as written.
func FuzzFOMODLayout(f *testing.F) {
	for _, s := range []struct {
		src, dest string
		folder    bool
	}{
		{"alpha", "alpha", true}, {"alpha/manifest.json", "", false}, {"..", "x", true}, {"alpha", "..", true},
		{"alpha", "../other-mod", true}, {`alpha\manifest.json`, `..\..\x`, false}, {".", "/abs", true},
		{"alpha", "a/./../../b", true}, {"beta/x.dll", "x.dll:stream", false}, {"alpha", "CON", true},
	} {
		f.Add(s.src, s.dest, s.folder)
	}
	f.Fuzz(func(t *testing.T, src, dest string, folder bool) {
		kind := "file"
		if folder {
			kind = "folder"
		}
		var attrs strings.Builder
		for _, a := range [][2]string{{"source", src}, {"destination", dest}} {
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
