package problems

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/nexus"
)

func TestAuthorMarkedMods(t *testing.T) {
	home := t.TempDir()
	cache := filepath.Join(home, "cache", "nexus")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	details := cachedNexusDetails{
		Page: nexus.Page{
			Name:    "Foo",
			Summary: "This mod is deprecated, use Bar (https://www.nexusmods.com/stardewvalley/mods/123)",
		},
		Files: []nexus.File{{FileID: 42, Name: "Foo 1.6", Description: "Deprecated - use the 1.6 version"}},
	}
	data, err := json.Marshal(details)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "details-v3-stardewvalley-10.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	fileOnly, err := json.Marshal(cachedNexusDetails{
		Page:  nexus.Page{Name: "File Only"},
		Files: []nexus.File{{FileID: 43, Name: "File Only 1.6", Description: "Deprecated - use the 1.6 version"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "details-v3-stardewvalley-11.json"), fileOnly, 0o600); err != nil {
		t.Fatal(err)
	}

	mods := []Installed{
		{Key: "local-obsolete", Enabled: true, Name: "[OBSOLETE] Foo", UniqueID: "foo"},
		{Key: "nexus-10-42", Enabled: true, Name: "Foo", UniqueID: "foo.file"},
		{Key: "nexus-11-43", Enabled: true, Name: "File Only", UniqueID: "file.only"},
		{Key: "local-replaces", Enabled: true, Name: "Replacer", UniqueID: "replacer", Description: "Replaces the obsolete vanilla menu"},
		{Key: "local-spelling", Enabled: true, Name: "Spelling", UniqueID: "spelling", Description: "This mod was depreciated by its author"},
	}
	got := authorMarkedMods(home, mods)
	if len(got) != 4 {
		t.Fatalf("got %d marked mods: %+v", len(got), got)
	}
	if got[0].Status != "obsolete" {
		t.Fatalf("name status = %q", got[0].Status)
	}
	if got[1].Replacement == nil || got[1].Replacement.PageID != 123 {
		t.Fatalf("replacement = %+v", got[1].Replacement)
	}
	if got[2].Status != "deprecated" || got[3].Status != "deprecated" {
		t.Fatalf("deprecated statuses = %q, %q", got[2].Status, got[3].Status)
	}
}

func TestAuthorMarkedManifestDescription(t *testing.T) {
	var mod Installed
	var fields map[string]string
	if err := json.Unmarshal([]byte(`{"Name":"Foo","UniqueID":"foo","Description":"This file is deprecated - use the 1.6 version"}`), &fields); err != nil {
		t.Fatal(err)
	}
	value := reflect.ValueOf(&mod).Elem().FieldByName("Manifest")
	for name, text := range fields {
		value.FieldByName(name).SetString(text)
	}
	got := authorMarkedMods(t.TempDir(), []Installed{mod})
	if len(got) != 1 || got[0].Status != "deprecated" {
		t.Fatalf("got %+v", got)
	}
}
