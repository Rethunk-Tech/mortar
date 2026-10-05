package profile

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func dumpTree(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			fmt.Fprintf(&b, "D %s\n", rel)
			return nil
		}
		body, err := fsx.ReadFile(p)
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "F %s %q\n", rel, body)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestImportGameModsCopiesIntoAProfileAndLeavesTheGameFolderUnchanged(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	gameDir := filepath.Join(t.TempDir(), "Stardew Valley")
	mods := filepath.Join(gameDir, "Mods")
	writeFile(t, mods, "canary.txt", "do not touch")
	writeFile(t, mods, "ConsoleCommands/manifest.json",
		`{"Name":"Console Commands","Version":"4.5.2","UniqueID":"SMAPI.ConsoleCommands"}`)
	writeFile(t, mods, "SaveBackup/manifest.json",
		`{"Name":"Save Backup","Version":"4.5.2","UniqueID":"SMAPI.SaveBackup"}`)
	writeFile(t, mods, "MortarSmapiBridge/manifest.json",
		`{"Name":"Bridge","Version":"1.0.1","UniqueID":"Rethunk.MortarSmapiBridge"}`)
	writeFile(t, mods, "Pack/Alpha/manifest.json",
		`{"Name":"Alpha","Version":"2.0.0","UniqueID":"Me.Alpha","UpdateKeys":["Nexus:42"]}`)
	writeFile(t, mods, "Pack/Beta/manifest.json",
		`{"Name":"Beta","Version":"2.0.0","UniqueID":"Me.Beta"}`)
	writeFile(t, mods, ".Quiet/manifest.json",
		`{"Name":"Quiet","Version":"1.1.0","UniqueID":"Me.Quiet"}`)
	writeFile(t, mods, ".Quiet/config.json", `{"volume":3}`)
	writeFile(t, mods, "Loud/manifest.json",
		`{"Name":"Loud","Version":"3.0.0","UniqueID":"Me.Loud","UpdateKeys":["GitHub:me/loud"]}`)
	writeFile(t, mods, "Loud/config.json", `{"on":true}`)
	writeFile(t, mods, "Broken/manifest.json", `{not json`)
	before := dumpTree(t, gameDir)

	preview, err := e.PreviewGameMods(mods)
	if err != nil {
		t.Fatal(err)
	}
	byPreview := map[string]GameModPreview{}
	for _, m := range preview.Mods {
		byPreview[m.Name] = m
	}
	if len(preview.Mods) != 5 || byPreview["Console Commands"].Name != "" || byPreview["Save Backup"].Name != "" ||
		byPreview["Bridge"].Name != "" || byPreview["Broken"].Status != outcomeFailed ||
		byPreview["Quiet"].Disabled != true || byPreview["Loud"].Status != outcomeImported {
		t.Fatalf("preview = %+v", preview.Mods)
	}

	res, err := e.ImportGameMods("stardew", mods)
	if err != nil {
		t.Fatal(err)
	}
	after := dumpTree(t, gameDir)
	if after != before {
		t.Fatalf("game folder changed:\n--- before ---\n%s--- after ---\n%s", before, after)
	}
	if res.Profile.Name != importedProfileName || res.Imported != 3 || res.Skipped != 0 || res.Failed != 1 {
		t.Fatalf("result = %+v", res)
	}

	modsList, err := e.UserMods("stardew", res.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Mod{}
	for _, m := range modsList {
		byID[m.ID.Local()] = m
	}
	if len(byID) != 4 || byID["Me.Quiet"].Enabled || !byID["Me.Loud"].Enabled || !byID["Me.Alpha"].Enabled {
		t.Fatalf("mods = %+v", modsList)
	}
	var pack Source
	for _, ent := range res.Profile.Entries {
		for _, m := range ent.Mods {
			if m.ID == "smapi:Me.Alpha" {
				pack = ent.Source
			}
		}
	}
	if pack.Kind != KindNexus || pack.ModID != 42 {
		t.Fatalf("pack source = %+v", pack)
	}
	loudCfg, err := e.ConfigPath("stardew", res.Profile.ID, byID["Me.Loud"].Key, "smapi:Me.Loud")
	if err != nil {
		t.Fatal(err)
	}
	if body, err := fsx.ReadFile(loudCfg); err != nil || string(body) != `{"on":true}` {
		t.Fatalf("loud config = %q %v", body, err)
	}
	quietCfg, err := e.ConfigPath("stardew", res.Profile.ID, byID["Me.Quiet"].Key, "smapi:Me.Quiet")
	if err != nil {
		t.Fatal(err)
	}
	if body, err := fsx.ReadFile(quietCfg); err != nil || string(body) != `{"volume":3}` {
		t.Fatalf("quiet config = %q %v", body, err)
	}

	again, err := e.ImportGameMods("stardew", mods)
	if err != nil {
		t.Fatal(err)
	}
	if again.Profile.Name != importedProfileName+" (2)" {
		t.Fatalf("second name = %q", again.Profile.Name)
	}
	if dumpTree(t, gameDir) != before {
		t.Fatal("second import changed the game folder")
	}
}

func TestImportGameModsPrefersEnabledDuplicateAndOmitsBundled(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	gameDir := filepath.Join(t.TempDir(), "Stardew Valley")
	mods := filepath.Join(gameDir, "Mods")
	writeFile(t, mods, "GenericModConfigMenu/manifest.json",
		`{"Name":"Generic Mod Config Menu","Version":"1.14.1","UniqueID":"spacechase0.GenericModConfigMenu"}`)
	writeFile(t, mods, "NPCMapLocations/manifest.json",
		`{"Name":"NPC Map Locations","Version":"3.3.0","UniqueID":"Bouhm.NPCMapLocations"}`)
	writeFile(t, mods, ".DisabledCopy/manifest.json",
		`{"Name":"NPC Map Locations","Version":"3.3.0","UniqueID":"Bouhm.NPCMapLocations"}`)
	writeFile(t, mods, "ConsoleCommands/manifest.json",
		`{"Name":"Console Commands","Version":"4.5.2","UniqueID":"SMAPI.ConsoleCommands"}`)
	writeFile(t, mods, "SaveBackup/manifest.json",
		`{"Name":"Save Backup","Version":"4.5.2","UniqueID":"SMAPI.SaveBackup"}`)
	before := dumpTree(t, gameDir)

	preview, err := e.PreviewGameMods(mods)
	if err != nil {
		t.Fatal(err)
	}
	var npcImport, npcSkip, bundled int
	for _, m := range preview.Mods {
		switch {
		case m.Name == "NPC Map Locations" && m.Status == outcomeImported && !m.Disabled:
			npcImport++
		case m.Name == "DisabledCopy" && m.Status == outcomeSkipped && m.Reason == "same mod as NPCMapLocations":
			npcSkip++
		case strings.Contains(strings.ToLower(m.Name), "console") || strings.Contains(strings.ToLower(m.Name), "backup"):
			bundled++
		}
	}
	if npcImport != 1 || npcSkip != 1 || bundled != 0 {
		t.Fatalf("preview = %+v", preview.Mods)
	}

	res, err := e.ImportGameMods("stardew", mods)
	if err != nil {
		t.Fatal(err)
	}
	if dumpTree(t, gameDir) != before {
		t.Fatal("import wrote the game folder")
	}
	if res.Imported != 2 || res.Skipped != 1 || res.Failed != 0 {
		t.Fatalf("result = imported %d skipped %d failed %d outcomes %+v", res.Imported, res.Skipped, res.Failed, res.Outcomes)
	}
	modsList, err := e.UserMods("stardew", res.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Mod{}
	for _, m := range modsList {
		byID[m.ID.Local()] = m
	}
	if len(byID) != 2 || !byID["Bouhm.NPCMapLocations"].Enabled || !byID["spacechase0.GenericModConfigMenu"].Enabled {
		t.Fatalf("mods = %+v", modsList)
	}
}

func TestImportGameModsPicksNewestWhenBothCopiesAreOff(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	gameDir := filepath.Join(t.TempDir(), "Stardew Valley")
	mods := filepath.Join(gameDir, "Mods")
	writeFile(t, mods, ".Old/manifest.json",
		`{"Name":"Map","Version":"1.0.0","UniqueID":"Me.Map"}`)
	writeFile(t, mods, ".New/manifest.json",
		`{"Name":"Map","Version":"2.0.0","UniqueID":"Me.Map"}`)
	before := dumpTree(t, gameDir)

	preview, err := e.PreviewGameMods(mods)
	if err != nil {
		t.Fatal(err)
	}
	var kept, skipped bool
	for _, m := range preview.Mods {
		if m.Status == outcomeImported && m.Disabled && m.Version == "2.0.0" {
			kept = true
		}
		if m.Status == outcomeSkipped && m.Reason == "same mod as New" {
			skipped = true
		}
	}
	if !kept || !skipped {
		t.Fatalf("preview = %+v", preview.Mods)
	}

	res, err := e.ImportGameMods("stardew", mods)
	if err != nil {
		t.Fatal(err)
	}
	if dumpTree(t, gameDir) != before {
		t.Fatal("import wrote the game folder")
	}
	modsList, err := e.UserMods("stardew", res.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(modsList) != 1 || modsList[0].Enabled || modsList[0].Version != "2.0.0" {
		t.Fatalf("mods = %+v", modsList)
	}
}

func TestInstallFolderUsesTheSameStorePathAsAnArchive(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	src := t.TempDir()
	writeFile(t, src, "manifest.json", manifestJSON("Me.Folder"))
	res, err := e.InstallFolder("stardew", p.ID, src)
	if err != nil || len(res.Added) != 1 || res.Profile.Entries[0].Source.Kind != KindLocal {
		t.Fatalf("install folder: %+v %v", res, err)
	}
}

func TestPreviewGameModsSkipsASymlinkDirectory(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	mods := filepath.Join(t.TempDir(), "Mods")
	writeFile(t, mods, "Loud/manifest.json", `{"Name":"Loud","Version":"1.0.0","UniqueID":"Me.Loud"}`)
	outside := t.TempDir()
	writeFile(t, outside, "manifest.json", `{"Name":"Docs","Version":"1.0.0","UniqueID":"Me.Docs"}`)
	if err := os.Symlink(outside, filepath.Join(mods, "Innocent")); err != nil {
		t.Skipf("cannot create a symlink (Windows needs the symlink privilege): %v", err)
	}
	preview, err := e.PreviewGameMods(mods)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Mods) != 1 || preview.Mods[0].Name != "Loud" {
		t.Fatalf("preview = %+v", preview.Mods)
	}
}
