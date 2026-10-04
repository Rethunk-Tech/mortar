package sharesvc

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/nexus"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

func TestCollectionExportRoundTrip(t *testing.T) {
	modsDir := t.TempDir()
	folder := filepath.Join(modsDir, "k1", "Cozy")
	if err := os.MkdirAll(filepath.Join(folder, "config"), 0o750); err != nil {
		t.Fatal(err)
	}
	write := func(rel, data string) {
		if err := os.WriteFile(filepath.Join(folder, filepath.FromSlash(rel)), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("manifest.json", `{"Name":"Cozy","UniqueID":"A.Cozy","Version":"1.0"}`)
	write("config.json", `{"a":1}`)
	write("config/extra.json", `{"b":2}`)

	fomod := map[string]map[string][]string{"Step": {"Group": {"Plugin A", "Plugin B"}}}
	p := profile.Profile{Name: "Cozy Farm", Notes: "Read me first", Description: "desc", Entries: []profile.Entry{
		{
			Key: "k1", Source: profile.Source{Kind: profile.KindNexus, Name: "cozy.zip", ModID: 100, FileID: 7, Version: "1.0"},
			Mods: []profile.EntryMod{{UniqueID: "A.Cozy", Name: "Cozy", Folder: "Cozy"}}, Fomod: fomod, Note: "needs SMAPI",
		},
		{Key: "k2", Source: profile.Source{Kind: profile.KindNexus, ModID: 200, FileID: 9, Name: "opt.zip"}},
		{Key: "k3", Source: profile.Source{Kind: profile.KindLocal, Name: "mine.zip"}},
		{Key: "k4", Source: profile.Source{Kind: profile.KindGitHub, Repo: "o/r", Tag: "v1", Asset: "a.zip"}},
	}}
	facts := func(modID, fileID int) (nexus.File, bool) {
		if modID == 100 {
			return nexus.File{FileID: fileID, MD5: "abc", SizeKB: 2, Category: "MAIN"}, true
		}
		return nexus.File{FileID: fileID, Category: "OPTIONAL"}, true
	}
	doc, files, skipped, err := buildCollection(p, "stardewvalley", modsDir, facts)
	if err != nil || len(skipped) != 0 {
		t.Fatalf("build: %v %v", err, skipped)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	packed, err := zipCollection(raw, files)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(packed), int64(len(packed)))
	if err != nil {
		t.Fatal(err)
	}
	read := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		if read[f.Name], err = io.ReadAll(rc); err != nil {
			t.Fatal(err)
		}
		_ = rc.Close()
	}
	if !bytes.Equal(read[collectionManifestName], raw) || len(read) != len(files)+1 {
		t.Fatalf("zip entries = %d", len(read))
	}
	raw = read[collectionManifestName]
	files = files[:0]
	for name, data := range read {
		if name != collectionManifestName {
			files = append(files, bundledFile{name, data})
		}
	}

	got, err := parseChoices(raw)
	if err != nil || !reflect.DeepEqual(got[modFile{100, 7}], fomod) {
		t.Fatalf("choices = %v, %v", got, err)
	}
	var back collectionDoc
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.Info.InstallInstructions != "Read me first" || back.Info.DomainName != "stardewvalley" || len(back.Mods) != 4 {
		t.Fatalf("info = %+v, %d mods", back.Info, len(back.Mods))
	}
	if s := back.Mods[0].Source; s.ModID != 100 || s.FileID != 7 || s.MD5 != "abc" || s.FileSize != 2048 || back.Mods[0].Optional {
		t.Fatalf("source = %+v", s)
	}
	if !back.Mods[1].Optional || back.Mods[2].Source.Type != "manual" || !strings.HasSuffix(back.Mods[3].Source.URL, "/o/r/releases/download/v1/a.zip") {
		t.Fatalf("mods = %+v", back.Mods)
	}

	bundles := map[string]*bundle{}
	for _, f := range files {
		dir, rel, _ := strings.Cut(strings.TrimPrefix(f.Path, collectionBundleDir+"/"), "/")
		if bundles[dir] == nil {
			bundles[dir] = &bundle{files: map[string][]byte{}}
		}
		bundles[dir].files[path.Clean(rel)] = f.Data
	}
	var cfgs []string
	for _, b := range bundles {
		for _, c := range b.configs() {
			cfgs = append(cfgs, c.UniqueID+"/"+c.Path+"="+string(c.Data))
		}
	}
	slices.Sort(cfgs)
	want := []string{"A.Cozy/config.json={\"a\":1}", "A.Cozy/config/extra.json={\"b\":2}"}
	if !slices.Equal(cfgs, want) {
		t.Fatalf("configs = %v", cfgs)
	}
}
