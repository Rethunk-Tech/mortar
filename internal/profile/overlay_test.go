package profile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/store"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

const (
	overlayModID = 27345
	overlayDir   = "[CP] No Sell Effects"
)

func overlaySource(fileID int, name string) Source {
	return Source{Kind: KindNexus, Name: name, ModID: overlayModID, FileID: fileID, Version: "1.0.0", ModName: "No Sell Effects"}
}

func mainZip(t *testing.T, version, art string) string {
	t.Helper()
	return buildZip(t, "main-"+version+".zip", map[string]string{
		overlayDir + "/manifest.json": `{"Name":"No Sell Effects","Author":"me","Version":"` + version + `","UniqueID":"x.nosell"}`,
		overlayDir + "/content.json":  "content " + version,
		overlayDir + "/assets/a.png":  art,
	})
}

func readLive(t *testing.T, e env, profileID, key, rel string) string {
	t.Helper()
	b, err := fsx.ReadFile(filepath.Join(e.mods(profileID), key, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func installOverlayPair(t *testing.T) (env, Profile, string, string) {
	t.Helper()
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	if _, err := e.InstallNexus("stardew", p.ID, mainZip(t, "1.0.0", "A-main"), overlaySource(1, "main.zip")); err != nil {
		t.Fatal(err)
	}
	opt := buildZip(t, "opt.zip", map[string]string{overlayDir + "/assets/a.png": "A-opt", overlayDir + "/assets/new.png": "N-opt"})
	res, err := e.InstallNexus("stardew", p.ID, opt, overlaySource(2, "opt.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Remap != nil || len(res.Added) != 1 || res.Added[0] != "opt.zip" {
		t.Fatalf("overlay install = %+v", res)
	}
	return e, res.Profile, store.NexusKey(overlayModID, 1), store.NexusKey(overlayModID, 2)
}

func TestOverlayInstallsOverItsMainFile(t *testing.T) {
	e, p, baseKey, optKey := installOverlayPair(t)
	if len(p.Entries) != 2 || p.Entries[1].OverlayOf != baseKey || p.Entries[1].OverlayFrom != overlayDir || p.Entries[1].OverlayTo != overlayDir || len(p.Entries[1].Mods) != 0 {
		t.Fatalf("entries = %+v", p.Entries)
	}
	if got := readLive(t, e, p.ID, baseKey, overlayDir+"/assets/a.png"); got != "A-opt" {
		t.Fatalf("overlaid file = %q", got)
	}
	storeDir, _ := e.items.Path("stardew", baseKey)
	if b, _ := fsx.ReadFile(filepath.Join(storeDir, overlayDir, "assets", "a.png")); string(b) != "A-main" {
		t.Fatalf("store item written: %q", b)
	}
	if exists(filepath.Join(e.mods(p.ID), optKey)) {
		t.Fatal("an optional file got a folder of its own")
	}
	mods, err := e.UserMods("stardew", p.ID)
	if err != nil || len(mods) != 1 {
		t.Fatalf("mods = %+v, %v", mods, err)
	}
	drift, err := e.ScanModsDrift("stardew", p.ID)
	if err != nil || len(drift) != 0 {
		t.Fatalf("drift = %+v, %v", drift, err)
	}
}

func TestOverlayWithoutMainFileIsRefused(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	opt := buildZip(t, "opt.zip", map[string]string{overlayDir + "/assets/a.png": "A-opt"})
	_, err := e.InstallNexus("stardew", p.ID, opt, overlaySource(2, "opt.zip"))
	var ie *InstallError
	want := "opt.zip has no manifest. Install the main file of No Sell Effects first, then this optional file goes on top of it."
	if !errors.As(err, &ie) || ie.Msg != want || !errors.As(err, new(*NoBaseError)) || usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("err = %v", err)
	}
	local := buildZip(t, "loose.zip", map[string]string{"readme.txt": "x"})
	if _, err := e.InstallArchive("stardew", p.ID, local); !errors.As(err, new(*NoModError)) {
		t.Fatalf("local without a manifest = %v", err)
	}
}

func TestMapOverlay(t *testing.T) {
	base := []string{"Mod/manifest.json", "Mod/content.json", "Mod/assets/a.png", "Mod/i18n/default.json"}
	mods := []EntryMod{{UniqueID: "x", Folder: "Mod"}}
	cases := []struct {
		name     string
		files    []string
		from, to string
		ok       bool
	}{
		{"wrapper named like the mod", []string{"mod/assets/z.png"}, "mod", "Mod", true},
		{"wrapper whose children match", []string{"Optional Dark/assets/z.png"}, "Optional Dark", "Mod", true},
		{"root of the mod folder", []string{"content.json", "assets/a.png", "extra.txt"}, "", "Mod", true},
		{"same layout as the entry", []string{"Mod/content.json", "Mod/assets/a.png", "Other/x.png"}, "", "", true},
		{"nothing fits", []string{"readme.txt", "pics/b.png"}, "", "", false},
	}
	for _, c := range cases {
		from, to, ok := mapOverlay(c.files, base, mods)
		if ok != c.ok || from != c.from || to != c.to {
			t.Errorf("%s: got (%q, %q, %v), want (%q, %q, %v)", c.name, from, to, ok, c.from, c.to, c.ok)
		}
	}
}

func TestOverlayAmbiguousAsksForTarget(t *testing.T) {
	e := newEnv(t)
	p, _ := e.Create("stardew", "P")
	if _, err := e.InstallNexus("stardew", p.ID, mainZip(t, "1.0.0", "A-main"), overlaySource(1, "main.zip")); err != nil {
		t.Fatal(err)
	}
	opt := buildZip(t, "opt.zip", map[string]string{"Pics/b.png": "B", "readme.txt": "r"})
	res, err := e.InstallNexus("stardew", p.ID, opt, overlaySource(2, "opt.zip"))
	if err != nil || res.Remap == nil || res.Remap.Overlay == nil || len(res.Remap.Overlay.Targets) != 1 || len(res.Profile.Entries) != 1 {
		t.Fatalf("ask = %+v, %v", res, err)
	}
	res, err = e.InstallOverlay("stardew", p.ID, res.Remap.Key, "Pics", overlayDir+"/assets", res.Remap.Source)
	if err != nil || len(res.Profile.Entries) != 2 {
		t.Fatalf("answer = %+v, %v", res, err)
	}
	if got := readLive(t, e, p.ID, store.NexusKey(overlayModID, 1), overlayDir+"/assets/b.png"); got != "B" {
		t.Fatalf("placed = %q", got)
	}
}

func TestOverlaysApplyInOrderAndRestoreWhenOff(t *testing.T) {
	e, p, baseKey, optKey := installOverlayPair(t)
	second := buildZip(t, "opt2.zip", map[string]string{overlayDir + "/assets/a.png": "A-two"})
	res, err := e.InstallNexus("stardew", p.ID, second, overlaySource(3, "opt2.zip"))
	if err != nil {
		t.Fatal(err)
	}
	optKey2 := store.NexusKey(overlayModID, 3)
	art := overlayDir + "/assets/a.png"
	if got := readLive(t, e, res.Profile.ID, baseKey, art); got != "A-two" {
		t.Fatalf("later overlay should win, got %q", got)
	}
	if _, err := e.SetOverlayEnabled("stardew", p.ID, optKey2, false); err != nil {
		t.Fatal(err)
	}
	if got := readLive(t, e, p.ID, baseKey, art); got != "A-opt" {
		t.Fatalf("after second off = %q", got)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, optKey, optKey, false); err != nil {
		t.Fatal(err)
	}
	if got := readLive(t, e, p.ID, baseKey, art); got != "A-main" {
		t.Fatalf("after both off = %q", got)
	}
	if exists(filepath.Join(e.mods(p.ID), baseKey, overlayDir, "assets", "new.png")) {
		t.Fatal("a file only the optional file brought stayed")
	}
	drift, err := e.ScanModsDrift("stardew", p.ID)
	if err != nil || len(drift) != 0 {
		t.Fatalf("drift = %+v, %v", drift, err)
	}
	if _, err := e.SetOverlayEnabled("stardew", p.ID, optKey, true); err != nil {
		t.Fatal(err)
	}
	if got := readLive(t, e, p.ID, baseKey, art); got != "A-opt" {
		t.Fatalf("back on = %q", got)
	}
	got, err := e.RemoveEntry("stardew", p.ID, optKey)
	if err != nil || len(got.Entries) != 2 || readLive(t, e, p.ID, baseKey, art) != "A-main" {
		t.Fatalf("remove overlay = %+v, %v", got.Entries, err)
	}
	got, err = e.RemoveEntries("stardew", p.ID, []string{baseKey, optKey2})
	if err != nil || len(got.Entries) != 0 {
		t.Fatalf("remove base = %+v, %v", got.Entries, err)
	}
}

func TestOverlayFollowsMainFileUpdate(t *testing.T) {
	e, p, _, optKey := installOverlayPair(t)
	res, err := e.InstallNexus("stardew", p.ID, mainZip(t, "1.1.0", "A-main2"), overlaySource(4, "main-1.1.zip"))
	if err != nil || !res.Updated {
		t.Fatalf("update = %+v, %v", res, err)
	}
	newKey := store.NexusKey(overlayModID, 4)
	var over Entry
	for _, en := range res.Profile.Entries {
		if en.Key == optKey {
			over = en
		}
	}
	if over.OverlayOf != newKey {
		t.Fatalf("overlay still names %q", over.OverlayOf)
	}
	if got := readLive(t, e, p.ID, newKey, overlayDir+"/assets/a.png"); got != "A-opt" {
		t.Fatalf("overlay after update = %q", got)
	}
	if got := readLive(t, e, p.ID, newKey, overlayDir+"/content.json"); got != "content 1.1.0" {
		t.Fatalf("main file after update = %q", got)
	}
	if exists(filepath.Join(e.mods(p.ID), newKey, overlayDir, "assets", "a.png"+conflictSuffix)) {
		t.Fatal("the overlaid file was carried over as a user edit")
	}
	drift, err := e.ScanModsDrift("stardew", p.ID)
	if err != nil || len(drift) != 0 {
		t.Fatalf("drift = %+v, %v", drift, err)
	}
	back, err := e.RollBack("stardew", p.ID, newKey)
	if err != nil {
		t.Fatal(err)
	}
	if got := readLive(t, e, p.ID, back.Entries[0].Key, overlayDir+"/assets/a.png"); got != "A-opt" {
		t.Fatalf("overlay after roll back = %q", got)
	}
}

func TestOverlayRevertAndRebuild(t *testing.T) {
	e, p, baseKey, _ := installOverlayPair(t)
	art := overlayDir + "/assets/a.png"
	events, err := e.History("stardew", p.ID)
	if err != nil || len(events) < 2 {
		t.Fatalf("history = %+v, %v", events, err)
	}
	if err := os.RemoveAll(filepath.Join(e.mods(p.ID), baseKey)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.UserMods("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	if got := readLive(t, e, p.ID, baseKey, art); got != "A-opt" {
		t.Fatalf("rebuilt = %q", got)
	}
	reverted, err := e.Revert("stardew", p.ID, events[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reverted.Entries) != 1 || readLive(t, e, p.ID, baseKey, art) != "A-main" {
		t.Fatalf("revert = %+v", reverted.Entries)
	}
}
