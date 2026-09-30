package profile

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
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

func putGameMod(t *testing.T, mods, rel, body string) {
	t.Helper()
	p := filepath.Join(mods, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := fsx.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestImportGameModsCopiesIntoAProfileAndLeavesTheGameFolderUnchanged(t *testing.T) {
	e := newEnv(t)
	gameDir := filepath.Join(t.TempDir(), "Stardew Valley")
	mods := filepath.Join(gameDir, "Mods")
	putGameMod(t, mods, "canary.txt", "do not touch")
	putGameMod(t, mods, "ConsoleCommands/manifest.json",
		`{"Name":"Console Commands","Version":"4.5.2","UniqueID":"SMAPI.ConsoleCommands"}`)
	putGameMod(t, mods, "SaveBackup/manifest.json",
		`{"Name":"Save Backup","Version":"4.5.2","UniqueID":"SMAPI.SaveBackup"}`)
	putGameMod(t, mods, "MortarSmapiBridge/manifest.json",
		`{"Name":"Bridge","Version":"1.0.1","UniqueID":"Rethunk.MortarSmapiBridge"}`)
	putGameMod(t, mods, "Pack/Alpha/manifest.json",
		`{"Name":"Alpha","Version":"2.0.0","UniqueID":"Me.Alpha","UpdateKeys":["Nexus:42"]}`)
	putGameMod(t, mods, "Pack/Beta/manifest.json",
		`{"Name":"Beta","Version":"2.0.0","UniqueID":"Me.Beta"}`)
	putGameMod(t, mods, ".Quiet/manifest.json",
		`{"Name":"Quiet","Version":"1.1.0","UniqueID":"Me.Quiet"}`)
	putGameMod(t, mods, ".Quiet/config.json", `{"volume":3}`)
	putGameMod(t, mods, "Loud/manifest.json",
		`{"Name":"Loud","Version":"3.0.0","UniqueID":"Me.Loud","UpdateKeys":["GitHub:me/loud"]}`)
	putGameMod(t, mods, "Loud/config.json", `{"on":true}`)
	putGameMod(t, mods, "Broken/manifest.json", `{not json`)
	before := dumpTree(t, gameDir)

	preview, err := e.PreviewGameMods(mods)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Mods) != 4 {
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
	if res.Profile.Name != importedProfileName || res.Imported != 3 || res.Skipped != 3 || res.Failed != 1 {
		t.Fatalf("result = %+v", res)
	}

	modsList, err := e.UserMods("stardew", res.Profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]Mod{}
	for _, m := range modsList {
		byID[m.UniqueID] = m
	}
	if len(byID) != 4 || byID["Me.Quiet"].Enabled || !byID["Me.Loud"].Enabled || !byID["Me.Alpha"].Enabled {
		t.Fatalf("mods = %+v", modsList)
	}
	var pack Source
	for _, ent := range res.Profile.Entries {
		for _, m := range ent.Mods {
			if m.UniqueID == "Me.Alpha" {
				pack = ent.Source
			}
		}
	}
	if pack.Kind != KindNexus || pack.ModID != 42 {
		t.Fatalf("pack source = %+v", pack)
	}
	loudCfg, err := e.ConfigPath("stardew", res.Profile.ID, byID["Me.Loud"].Key, "Me.Loud")
	if err != nil {
		t.Fatal(err)
	}
	if body, err := fsx.ReadFile(loudCfg); err != nil || string(body) != `{"on":true}` {
		t.Fatalf("loud config = %q %v", body, err)
	}
	quietCfg, err := e.ConfigPath("stardew", res.Profile.ID, byID["Me.Quiet"].Key, "Me.Quiet")
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

func TestInstallFolderUsesTheSameStorePathAsAnArchive(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	src := t.TempDir()
	putGameMod(t, src, "manifest.json", manifestJSON("Me.Folder"))
	res, err := e.InstallFolder("stardew", p.ID, src)
	if err != nil || len(res.Added) != 1 || res.Profile.Entries[0].Source.Kind != KindLocal {
		t.Fatalf("install folder: %+v %v", res, err)
	}
}
