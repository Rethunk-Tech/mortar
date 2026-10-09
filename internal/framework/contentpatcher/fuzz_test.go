package contentpatcher

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzContentPack feeds a pack's content.json and one file it can name (as an Include, a Load or EditData source, a
// map or an image), with a decoy pack beside it. Scanning must not panic, read the decoy, or stamp a file outside
// the pack, and what it finds must survive the disk cache.
func FuzzContentPack(f *testing.F) {
	for _, s := range [][2]string{
		{
			`{"Format":"2.0.0","ConfigSchema":{"Mode":{"AllowValues":"A, B","Default":"A"}},"DynamicTokens":[{"Name":"T","Value":"{{Mode}}","When":{"HasMod":"X.Y"}}],` +
				`"Changes":[{"Action":"Include","FromFile":"data.json"},{"Action":"Load","Target":"Maps/A","FromFile":"data.tmx","When":{"Mode":"A"}},` +
				`{"Action":"EditMap","Target":"Maps/B","FromFile":"data.tmj","FromArea":{"X":0,"Y":0,"Width":2,"Height":2},"ToArea":{"X":1,"Y":1,"Width":2,"Height":2}},` +
				`{"Action":"EditImage","Target":"TileSheets/x","FromFile":"data.png","PatchMode":"Overlay"},{"Action":"EditData","Target":"Data/Objects","Entries":{"1":"{{T}}"},"Priority":"Late"}]}`,
			`{"Changes":[{"Action":"EditData","Target":"Data/Crops","Fields":{"1":{"2":"x"}}}]}`,
		},
		{`{"Changes":[{"Action":"Include","FromFile":"../decoy/content.json"},{"Action":"Include","FromFile":"/etc/passwd"},{"Action":"Load","Target":"A, B","FromFile":"{{Target}}.json"}]}`, `{}`},
		{`{"Changes":[{"Action":"EditMap","Target":"Maps/T","FromFile":"data.tbin"}]}`, "tBIN10\x00\x00\x00\x00"},
		{`{"Changes":[{"Action":"EditImage","Target":"TileSheets/x","FromFile":"data.png","PatchMode":"Overlay"}]}`, "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x75\x30\x00\x00\x75\x30\x08\x02\x00\x00\x00\x00\x00\x00\x00"},
		{`{"Changes":[{"Action":"Load","Target":"Maps/T","FromFile":"data.tmx"}]}`, `<map><tileset><image source="../Maps/spring_town.png"/></tileset></map>`},
		{`// comment` + "\n" + `{"Changes":[{"Action":"Include","FromFile":"data.json, content.json"},],}`, `{"Changes":[{"Action":"Include","FromFile":"content.json"}]}`},
	} {
		f.Add([]byte(s[0]), []byte(s[1]))
	}
	f.Fuzz(func(t *testing.T, content, data []byte) {
		parent := t.TempDir()
		root, decoy := filepath.Join(parent, "pack"), filepath.Join(parent, "decoy")
		files := map[string][]byte{
			filepath.Join(root, "content.json"):  content,
			filepath.Join(decoy, "content.json"): []byte(`{"Changes":[{"Action":"Load","Target":"Decoy/Leak","FromFile":"x.json"}]}`),
		}
		for _, name := range []string{"data.json", "data.tmj", "data.tmx", "data.tbin", "data.png"} {
			files[filepath.Join(root, name)] = data
		}
		for p, b := range files {
			if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, b, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		pack := cachedPack{mentions: map[string]bool{}}
		pack.schema = readConfigSchema(root)
		pack.values = packConfigValues(pack.schema, root)
		scanContentFile(root, "content.json", map[string]bool{}, cpWhen{}, &pack)
		for _, p := range pack.patches {
			if strings.Contains(strings.ToLower(p.target), "decoy/leak") {
				t.Fatalf("scanned a file outside the pack: %+v", p)
			}
		}
		for _, s := range pack.files {
			if filepath.IsAbs(s.Path) || s.Path == ".." || strings.HasPrefix(filepath.ToSlash(s.Path), "../") {
				t.Fatalf("stamped %q outside the pack", s.Path)
			}
		}
		raw, err := encodeDiskPack(diskPackOf(pack))
		if err != nil {
			t.Fatalf("pack does not encode: %v", err)
		}
		if _, ok := decodeDiskPack(raw); !ok {
			t.Fatal("encoded pack does not decode")
		}
	})
}
