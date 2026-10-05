package configsvc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/fsx"
	"github.com/Rethunk-Tech/mortar/internal/gmcm"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func TestTheInGameMenuIsAFileWhoseEditsWaitForTheNextStart(t *testing.T) {
	s, _ := newService(t)
	dir := s.Profiles.ProfileDir
	pdir, _ := dir("stardew", "p")
	capture, err := fsx.ReadFile(filepath.Join("..", "gmcm", "testdata", "gmcm-capture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(gmcm.CapturePath(pdir, "Author.Mod")), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := datadir.WriteFile(gmcm.CapturePath(pdir, "Author.Mod"), capture, 0o600); err != nil {
		t.Fatal(err)
	}
	type set struct {
		page  string
		index int
		value string
	}
	var sets []set
	s.SetGmcm = func(_, _ string, _ mod.ID, page string, index int, value string) error {
		sets = append(sets, set{page, index, value})
		return gmcm.WritePending(pdir, "Author.Mod", []gmcm.Edit{{Page: page, Index: index, Kind: "int", Name: "Count", Value: 5}})
	}

	files, err := s.Files("stardew", "p", "Author.Mod")
	if err != nil || len(files) != 2 || files[1].Format != FormatGMCM {
		t.Fatalf("files = %+v, %v", files, err)
	}
	schema, err := s.Schema("stardew", "p", "Author.Mod", files[1].Name)
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]Entry{}
	sections := []string{}
	for _, sec := range schema.Sections {
		sections = append(sections, sec.Name)
		for _, e := range sec.Entries {
			byKey[e.Key] = e
		}
	}
	if len(sections) != 2 || sections[0] != "" || sections[1] != "Section" {
		t.Fatalf("sections = %q", sections)
	}
	count := byKey["/2"]
	if count.Type != TypeInt || *count.Min != 1 || *count.Max != 5 || count.Value != "3" || count.Default != "3" || count.Pending {
		t.Fatalf("count = %+v", count)
	}
	pick := byKey["/3"]
	if pick.Type != TypeEnum || len(pick.Values) == 0 || len(pick.Labels) != len(pick.Values) {
		t.Fatalf("pick = %+v", pick)
	}
	if tint := byKey["/7"]; !tint.ReadOnly {
		t.Fatalf("a colour picker is the game's to change: %+v", tint)
	}

	if err := s.Set("stardew", "p", "Author.Mod", files[1].Name, "", "/2", "5"); err != nil {
		t.Fatal(err)
	}
	if len(sets) != 1 || sets[0] != (set{"", 2, "5"}) {
		t.Fatalf("sets = %+v", sets)
	}
	schema, _ = s.Schema("stardew", "p", "Author.Mod", files[1].Name)
	if e := schema.Sections[0].Entries[2]; e.Key != "/2" || e.Value != "5" || !e.Pending {
		t.Fatalf("pending count = %+v", e)
	}
	if err := s.Reset("stardew", "p", "Author.Mod", files[1].Name, "", "/2"); err != nil {
		t.Fatal(err)
	}
	if sets[len(sets)-1] != (set{"", 2, "3"}) {
		t.Fatalf("reset should set the captured value back: %+v", sets)
	}
}
