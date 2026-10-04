package profile

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
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

func overlayEntry(p Profile, key string) Entry {
	for _, en := range p.Entries {
		if en.Key == key {
			return en
		}
	}
	return Entry{}
}

func TestOverlayAlternativesSwitchEachOtherOff(t *testing.T) {
	e, p, baseKey, optKey := installOverlayPair(t)
	second := buildZip(t, "opt2.zip", map[string]string{overlayDir + "/assets/a.png": "A-two"})
	res, err := e.InstallNexus("stardew", p.ID, second, overlaySource(3, "opt2.zip"))
	if err != nil {
		t.Fatal(err)
	}
	optKey2 := store.NexusKey(overlayModID, 3)
	art, added := overlayDir+"/assets/a.png", overlayDir+"/assets/new.png"
	if !overlayEntry(res.Profile, optKey).OverlayOff || readLive(t, e, p.ID, baseKey, art) != "A-two" {
		t.Fatalf("adding an alternative should switch the other off: %+v", res.Profile.Entries)
	}
	if exists(filepath.Join(e.mods(p.ID), baseKey, filepath.FromSlash(added))) {
		t.Fatal("a file only the switched-off alternative brought stayed")
	}
	sets, err := e.OverlayFiles("stardew", p.ID, baseKey)
	if err != nil || len(sets) != 2 {
		t.Fatalf("sets = %+v, %v", sets, err)
	}
	if s := sets[0]; s.Key != optKey || !slices.Equal(s.Replaces, []string{art}) || !slices.Equal(s.Adds, []string{added}) ||
		!slices.Equal(s.Alternatives, []string{optKey2}) {
		t.Fatalf("first set = %+v", s)
	}
	got, err := e.SetOverlayEnabled("stardew", p.ID, optKey, true)
	if err != nil || !overlayEntry(got, optKey2).OverlayOff || readLive(t, e, p.ID, baseKey, art) != "A-opt" ||
		readLive(t, e, p.ID, baseKey, added) != "N-opt" {
		t.Fatalf("switching one on = %+v, %v", got.Entries, err)
	}
	if _, err := e.SetModEnabled("stardew", p.ID, optKey, optKey, false); err != nil {
		t.Fatal(err)
	}
	if got := readLive(t, e, p.ID, baseKey, art); got != "A-main" {
		t.Fatalf("after both off = %q", got)
	}
	if exists(filepath.Join(e.mods(p.ID), baseKey, filepath.FromSlash(added))) {
		t.Fatal("a file only the optional file brought stayed")
	}
	drift, err := e.ScanModsDrift("stardew", p.ID)
	if err != nil || len(drift) != 0 {
		t.Fatalf("drift = %+v, %v", drift, err)
	}
	if _, err := e.SetOverlayEnabled("stardew", p.ID, optKey2, true); err != nil {
		t.Fatal(err)
	}
	if got := readLive(t, e, p.ID, baseKey, art); got != "A-two" {
		t.Fatalf("back on = %q", got)
	}
	got, err = e.RemoveEntry("stardew", p.ID, optKey2)
	if err != nil || len(got.Entries) != 2 || readLive(t, e, p.ID, baseKey, art) != "A-main" {
		t.Fatalf("remove overlay = %+v, %v", got.Entries, err)
	}
	got, err = e.RemoveEntries("stardew", p.ID, []string{baseKey, optKey})
	if err != nil || len(got.Entries) != 0 {
		t.Fatalf("remove base = %+v, %v", got.Entries, err)
	}
}

func TestOverlayNewVersionTakesItsPlace(t *testing.T) {
	e, p, baseKey, optKey := installOverlayPair(t)
	if _, err := e.SetOverlayEnabled("stardew", p.ID, optKey, false); err != nil {
		t.Fatal(err)
	}
	if NewestFromPage(p, overlayModID, 2) != 2 || NewestFromPage(p, overlayModID, 0) != 1 {
		t.Fatal("NewestFromPage counts an optional file's own versions only for it")
	}
	v2 := buildZip(t, "opt-2.zip", map[string]string{overlayDir + "/assets/a.png": "A-opt2"})
	res, err := e.InstallNexus("stardew", p.ID, v2, overlaySource(5, "opt-2.zip").WithReplacing(2))
	newKey := store.NexusKey(overlayModID, 5)
	if err != nil || len(res.Profile.Entries) != 2 {
		t.Fatalf("new version = %+v, %v", res.Profile.Entries, err)
	}
	if o := res.Profile.Entries[1]; o.Key != newKey || !o.OverlayOff || o.OverlayOf != baseKey || o.OverlayTo != overlayDir {
		t.Fatalf("replaced entry = %+v", o)
	}
	if _, err := e.SetOverlayEnabled("stardew", p.ID, newKey, true); err != nil {
		t.Fatal(err)
	}
	if got := readLive(t, e, p.ID, baseKey, overlayDir+"/assets/a.png"); got != "A-opt2" {
		t.Fatalf("new version laid = %q", got)
	}
	if exists(filepath.Join(e.mods(p.ID), baseKey, overlayDir, "assets", "new.png")) {
		t.Fatal("a file only the old version brought stayed")
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

func TestOverlayExportRestoreRoundTrip(t *testing.T) {
	e, p, baseKey, optKey := installOverlayPair(t)
	second := buildZip(t, "opt2.zip", map[string]string{overlayDir + "/assets/a.png": "A-two"})
	if _, err := e.InstallNexus("stardew", p.ID, second, overlaySource(3, "opt2.zip")); err != nil {
		t.Fatal(err)
	}
	if _, err := e.SetOverlayEnabled("stardew", p.ID, optKey, true); err != nil {
		t.Fatal(err)
	}
	art := overlayDir + "/assets/a.png"
	zipPath := filepath.Join(t.TempDir(), "farm.zip")
	if err := e.ExportZip("stardew", p.ID, zipPath, "0.0.1"); err != nil {
		t.Fatal(err)
	}
	if got := readLive(t, e, p.ID, baseKey, art); got != "A-opt" {
		t.Fatalf("export changed the live folder: %q", got)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	inZip := map[string]string{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		_ = rc.Close()
		inZip[f.Name] = string(b)
	}
	_ = zr.Close()
	if inZip["mods/"+baseKey+"/"+art] != "A-main" || inZip["overlays/"+optKey+"/"+art] != "A-opt" {
		t.Fatalf("zip does not keep the main folder and its optional files apart: %v", inZip)
	}
	if _, ok := inZip["mods/"+baseKey+"/"+overlayDir+"/assets/new.png"]; ok {
		t.Fatal("a file only an optional file brings was exported in the main folder")
	}
	got, err := e.RestoreZip("stardew", zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 3 {
		t.Fatalf("restored entries = %+v", got.Entries)
	}
	base, one, two := got.Entries[0], got.Entries[1], got.Entries[2]
	if one.OverlayOf != base.Key || two.OverlayOf != base.Key || one.OverlayOff || !two.OverlayOff ||
		one.OverlayTo != overlayDir || one.Source.FileID != 2 {
		t.Fatalf("restored optional files = %+v", got.Entries)
	}
	if readLive(t, e, got.ID, base.Key, art) != "A-opt" || readLive(t, e, got.ID, base.Key, overlayDir+"/assets/new.png") != "N-opt" {
		t.Fatal("restored optional file not laid over its main file")
	}
	drift, err := e.ScanModsDrift("stardew", got.ID)
	if err != nil || len(drift) != 0 {
		t.Fatalf("drift = %+v, %v", drift, err)
	}
}
