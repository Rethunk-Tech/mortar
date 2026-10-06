package sharesvc

import (
	"strconv"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

// FuzzParseModRoute feeds the mortar://<game>/mod/<id> launch argument.
func FuzzParseModRoute(f *testing.F) {
	for _, s := range []string{"mortar://stardew/mod/2400", "mortar://stardew/mod/02400", "mortar://u@stardew/mod/1", "mortar://stardew/mod/1?x", "mortar://../mod/1", "mortar://stardew/mod/99999999999999999999"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		r, ok := parseModRoute(s)
		if ok && (!game.Valid(r.game) || r.modID < 1 || !strings.EqualFold(strings.TrimRight(s, "?#"), "mortar://"+r.game+"/mod/"+strconv.Itoa(r.modID))) {
			t.Fatalf("parseModRoute(%q) = %+v", s, r)
		}
	})
}

// FuzzParseCollectionURL feeds a pasted Nexus collection page link.
func FuzzParseCollectionURL(f *testing.F) {
	for _, s := range []string{
		"https://www.nexusmods.com/games/stardewvalley/collections/tckf0m", "https://next.nexusmods.com/games/stardewvalley/collections/tckf0m/revisions/3",
		"https://nexusmods.com.evil/games/a/collections/b", "http://nexusmods.com/games/a/collections/b", "https://nexusmods.com/games/a/collections/b/revisions/0",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		domain, slug, rev, ok := parseCollectionURL(s)
		if ok && (domain == "" || slug == "" || rev < 0) {
			t.Fatalf("parseCollectionURL(%q) = %q %q %d", s, domain, slug, rev)
		}
	})
}

// FuzzCollectionArchive feeds the curator's collection 7z, which a Premium import downloads and reads.
func FuzzCollectionArchive(f *testing.F) {
	for _, p := range []string{"testdata/collection-archive.7z", "../archive/testdata/valid.7z", "../archive/testdata/traversal.7z"} {
		b, err := fsx.ReadFile(p)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		d, err := readCollectionArchive(raw)
		if err == nil {
			checkDetails(t, d)
		}
	})
}

// FuzzCollectionManifest feeds collection.json, whose FOMOD choices are applied at install.
func FuzzCollectionManifest(f *testing.F) {
	f.Add([]byte(`{"info":{"name":"C"},"mods":[{"name":"M","source":{"type":"nexus","modId":100,"fileId":11},"choices":{"type":"fomod","options":[{"name":"Install Type","groups":[{"name":"Options","choices":[{"name":"Full","idx":0}]}]}]}}]}`))
	f.Add([]byte(`{"mods":[{"source":{"type":"nexus","modId":1,"fileId":1},"choices":{"type":"fomod","options":[{"name":"","groups":null}]}}]}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		fomod, err := parseChoices(raw)
		if err != nil {
			return
		}
		for mf, c := range fomod {
			if mf.modID == 0 || mf.fileID == 0 || !share.ValidFomod(c) {
				t.Fatalf("kept choices %+v for %+v", c, mf)
			}
		}
	})
}

// FuzzBundleConfigs feeds the files of one bundled mod folder: only its config files, at paths Apply would write,
// come out.
func FuzzBundleConfigs(f *testing.F) {
	f.Add("Tweaks/manifest.json", `{"UniqueID":"Pat.Tweaks"}`, "Tweaks/config.json", "Tweaks/config/../../../x.json")
	f.Add("manifest.json", `{"UniqueID":"../x"}`, "config.json", "config/CON.json")
	f.Add("a/manifest.json", `{"UniqueID":"A"}`, "b/config.json", "a/config/a b.json")
	f.Fuzz(func(t *testing.T, manifestPath, manifest, a, b string) {
		bd := bundle{files: map[string][]byte{manifestPath: []byte(manifest), a: []byte("{}"), b: []byte("{}")}}
		checkDetails(t, collectionDetails{Configs: bd.configs()})
	})
}

func checkDetails(t *testing.T, d collectionDetails) {
	t.Helper()
	for _, c := range d.Configs {
		if !share.ValidConfigPath(c.Path) || c.ID == "" {
			t.Fatalf("config %q for %q", c.Path, c.ID)
		}
	}
	for mf, c := range d.Fomod {
		if mf.modID == 0 || mf.fileID == 0 || !share.ValidFomod(c) {
			t.Fatalf("kept choices %+v for %+v", c, mf)
		}
	}
}
