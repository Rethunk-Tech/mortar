package profile

import (
	"strings"
	"testing"
)

func TestDuplicateRecordsCopyOrigin(t *testing.T) {
	e := newEnv(t)
	src := mustCreate(t, e, "Farm")
	dup, err := e.Duplicate("stardew", src.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dup.Origin != OriginCopy || dup.CopyOf != "Farm" {
		t.Fatalf("dup origin = %q %q", dup.Origin, dup.CopyOf)
	}
	fresh := mustCreate(t, e, "Blank")
	if fresh.Origin != "" || fresh.CopyOf != "" {
		t.Fatalf("new profile origin = %q %q", fresh.Origin, fresh.CopyOf)
	}
}

func TestImportGameModsRecordsOrigin(t *testing.T) {
	e := newEnv(t)
	mods := t.TempDir()
	writeFile(t, mods, "Loud/manifest.json",
		`{"Name":"Loud","Version":"3.0.0","UniqueID":"Me.Loud"}`)
	res, err := e.ImportGameMods("stardew", mods)
	if err != nil {
		t.Fatal(err)
	}
	if res.Profile.Origin != OriginGameMods || res.Profile.CopyOf != "" {
		t.Fatalf("import origin = %q %q", res.Profile.Origin, res.Profile.CopyOf)
	}
	if !strings.HasPrefix(res.Profile.Name, importedProfileName) {
		t.Fatalf("name = %q", res.Profile.Name)
	}
}
