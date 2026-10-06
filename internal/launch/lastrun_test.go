package launch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAttributeLogMapsFixtureByNameAndUniqueID(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "errors.txt"))
	if err != nil {
		t.Fatal(err)
	}
	mods := []ModRef{
		{Name: "Content Patcher", ID: "smapi:Pathoschild.ContentPatcher"},
		{Name: "Lookup Anything", ID: "smapi:Pathoschild.LookupAnything"},
	}
	got := AttributeLog(string(body), mods, nil)
	if len(got) != 2 {
		t.Fatalf("got %#v", got)
	}
	if got[0].ID != "smapi:Pathoschild.ContentPatcher" || got[0].Errors != 2 || got[0].Warnings != 0 {
		t.Fatalf("Content Patcher = %#v", got[0])
	}
	if got[1].ID != "smapi:Pathoschild.LookupAnything" || got[1].Errors != 1 || got[1].Warnings != 0 {
		t.Fatalf("Lookup Anything = %#v", got[1])
	}
}

func TestAttributeLogMatchesUniqueIDColumn(t *testing.T) {
	log := "[19:43:50 ERROR Pathoschild.ContentPatcher] boom\n" +
		"[19:43:51 WARN  Pathoschild.ContentPatcher] maybe\n" +
		"[19:43:52 INFO  Pathoschild.ContentPatcher] fine\n"
	got := AttributeLog(log, []ModRef{
		{Name: "Content Patcher", ID: "smapi:Pathoschild.ContentPatcher"},
	}, nil)
	if len(got) != 1 || got[0].Errors != 1 || got[0].Warnings != 1 {
		t.Fatalf("got %#v", got)
	}
}

func TestAttributeLogIgnoresUnmappedModsAndContinuations(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "errors.txt"))
	if err != nil {
		t.Fatal(err)
	}
	got := AttributeLog(string(body), []ModRef{
		{Name: "Content Patcher", ID: "smapi:Pathoschild.ContentPatcher"},
	}, nil)
	if len(got) != 1 || got[0].Errors != 2 {
		t.Fatalf("unmapped SMAPI and Lookup Anything must not land; got %#v", got)
	}
}

func TestAttributeLogEmptyWhenNewerRunIsClean(t *testing.T) {
	log := "[19:43:50 INFO  Content Patcher] loaded\n"
	got := AttributeLog(log, []ModRef{
		{Name: "Content Patcher", ID: "smapi:Pathoschild.ContentPatcher"},
	}, nil)
	if len(got) != 0 {
		t.Fatalf("clean run must clear badges; got %#v", got)
	}
}

func TestMatchModColumnIsExact(t *testing.T) {
	mods := []ModRef{
		{Name: "SpaceCore", ID: "smapi:spacechase0.SpaceCore"},
		{Name: "SpaceCore Extra", ID: "smapi:Me.SpaceCoreExtra"},
	}
	if _, ok := MatchModColumn("Space", mods); ok {
		t.Fatal("substring must not match")
	}
	got, ok := MatchModColumn("SpaceCore", mods)
	if !ok || got.ID != "smapi:spacechase0.SpaceCore" {
		t.Fatalf("got %#v, %v", got, ok)
	}
	got, ok = MatchModColumn("spacechase0.SpaceCore", mods)
	if !ok || got.Name != "SpaceCore" {
		t.Fatalf("UniqueID match = %#v, %v", got, ok)
	}
}

func TestAttributeLogMapsAPluginAliasToItsPackage(t *testing.T) {
	pkg := ModRef{Name: "SoundAPI", ID: "thunderstore:loaforc-SoundAPI"}
	got := AttributeLog("[Error  :Me.Loaforc.SoundAPI] boom\n", []ModRef{pkg}, map[string]ModRef{"me.loaforc.soundapi": pkg})
	if len(got) != 1 || got[0].Name != "SoundAPI" || got[0].Errors != 1 {
		t.Fatalf("issues = %+v", got)
	}
}
