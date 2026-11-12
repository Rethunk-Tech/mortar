package launchsvc

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/meta"

	"github.com/Rethunk-Tech/go-smapiweb"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/migrate"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testcatalog"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func sweepEnv(t *testing.T) (*Service, *profile.Store) {
	t.Helper()
	datadirtest.Use(t, t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	svc := NewService(t.TempDir(), set, profiles)
	svc.SweepVersions = func(string) (string, string, error) { return "1.6.15", "4.1.10", nil }
	svc.SweepCompat = func(context.Context) (smapiweb.CompatIndex, error) {
		return smapiweb.CompatIndex{ByID: map[string]smapiweb.CompatEntry{}}, nil
	}
	svc.SweepHasUpdate = func(mod.ID, int) bool { return false }
	return svc, profiles
}

func addSweepMod(t *testing.T, profiles *profile.Store, profileID, uniqueID, name string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	body := `{"Name":"` + name + `","Author":"A","Version":"1.0.0","UniqueID":"` + uniqueID + `"}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := profiles.ImportExternalMods(t.Context(), "stardew", profileID, []migrate.ModPreview{
		{SourcePath: dir, ID: mod.SMAPI(uniqueID), Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSweepNoChangeSkips(t *testing.T) {
	svc, _ := sweepEnv(t)
	if err := svc.settings.RecordLastSweep("stardew", "1.6.15", "smapi", "4.1.10"); err != nil {
		t.Fatal(err)
	}
	rep, err := svc.Sweep(t.Context(), "stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Triggered {
		t.Fatalf("unchanged versions triggered: %#v", rep)
	}
	if len(rep.Profiles) != 0 {
		t.Fatalf("no-change sweep listed profiles: %#v", rep.Profiles)
	}
}

func TestSweepBrokenWithAndWithoutFix(t *testing.T) {
	svc, profiles := sweepEnv(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	addSweepMod(t, profiles, p.ID, "A.Fixable", "Fixable")
	addSweepMod(t, profiles, p.ID, "A.Stuck", "Stuck")
	svc.SweepVersions = func(string) (string, string, error) { return "1.6.16", "4.2.0", nil }
	svc.SweepCompat = func(context.Context) (smapiweb.CompatIndex, error) {
		return smapiweb.CompatIndex{ByID: map[string]smapiweb.CompatEntry{
			"a.fixable": {Status: smapiweb.StatusBroken, Summary: "Use latest version"},
			"a.stuck":   {Status: smapiweb.StatusBroken, Summary: "Broken in 1.6.16"},
		}}, nil
	}
	svc.SweepHasUpdate = func(uniqueID mod.ID, _ int) bool {
		return uniqueID == "smapi:A.Fixable"
	}
	rep, err := svc.Sweep(t.Context(), "stardew", "")
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Triggered {
		t.Fatal("version change did not trigger")
	}
	if len(rep.Profiles) != 1 {
		t.Fatalf("profiles = %#v", rep.Profiles)
	}
	byID := map[string]SweepMod{}
	for _, m := range rep.Profiles[0].Broken {
		byID[m.ID.Local()] = m
	}
	if byID["A.Fixable"].Fix != fixUpdate {
		t.Fatalf("fixable = %#v", byID["A.Fixable"])
	}
	if byID["A.Stuck"].Fix != fixNone {
		t.Fatalf("stuck = %#v", byID["A.Stuck"])
	}
}

func TestSweepVersionIsPerGame(t *testing.T) {
	svc, _ := sweepEnv(t)
	if err := svc.settings.RecordLastSweep("lethal-company", "v73", "bepinex5", "5.4.2100"); err != nil {
		t.Fatal(err)
	}
	if svc.sweepVersionsChanged("lethal-company", "v73", "5.4.2100") {
		t.Fatal("lethal company's recorded sweep reads as changed")
	}
	if !svc.sweepVersionsChanged("valheim", "", "5.4.2100") {
		t.Fatal("valheim took lethal company's bepinex sweep version")
	}
}

// A loader that reads no game version is swept when the Steam build of the install changes: the first look is only
// the baseline, and the next build triggers.
func TestSweepNoticesASteamBuildForALoaderWithoutAGameVersion(t *testing.T) {
	const id = "build-game"
	testcatalog.AddGames(t, components.GameInfo{
		ID: id, Name: "Build Game", Enabled: true, Marker: "G.dll", Deploy: "profile",
		Targets: []components.TargetDef{{ID: "mods", Root: "{profile}/Mods", MaxDepth: map[string]int{"pkg": 1}}},
		Stores:  components.GameStores{Steam: &components.SteamStore{AppID: "7"}},
		Loaders: []components.GameLoader{{ID: "folder", Name: "Mod folder"}},
	})

	svc, _ := sweepEnv(t)
	svc.SweepVersions = nil
	lib := t.TempDir()
	folder := filepath.Join(lib, "steamapps", "common", "Build Game")
	if err := os.MkdirAll(folder, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "G.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	build := func(n string) {
		manifest := "\"AppState\"\n{\n\t\"buildid\"\t\t\"" + n + "\"\n}\n"
		if err := os.WriteFile(filepath.Join(lib, "steamapps", "appmanifest_7.acf"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.settings.Update(func(v *settings.Settings) { v.GameFolders[id] = folder }); err != nil {
		t.Fatal(err)
	}
	sweep := func() SweepReport {
		rep, err := svc.Sweep(t.Context(), id, "")
		if err != nil {
			t.Fatal(err)
		}
		return rep
	}
	build("100")
	if rep := sweep(); rep.Triggered || rep.GameVersion != "build 100" {
		t.Fatalf("the first look must only record the build: %+v", rep)
	}
	if rep := sweep(); rep.Triggered {
		t.Fatalf("the same build triggered: %+v", rep)
	}
	build("101")
	if rep := sweep(); !rep.Triggered || rep.GameVersion != "build 101" {
		t.Fatalf("a patch went unnoticed: %+v", rep)
	}
	if rep := sweep(); rep.Triggered {
		t.Fatalf("a swept build triggered again: %+v", rep)
	}
}

func TestSweepKnownBrokenListsWhatSMAPIsListDidNot(t *testing.T) {
	list := components.BrokenList{Entries: []components.BrokenEntry{
		{IDs: components.BrokenIDs{Thunderstore: "Ns-Fixable"}, Through: "1.0.0", Reason: "r"},
		{IDs: components.BrokenIDs{Thunderstore: "Ns-Stuck"}, Through: "1.0.0", Reason: "r"},
		{IDs: components.BrokenIDs{Thunderstore: "Ns-Old"}, Through: "1.0.0", Reason: "r", Kind: components.BrokenKindSuperseded},
		{IDs: components.BrokenIDs{Thunderstore: "Ns-Off"}, Through: "1.0.0", Reason: "r"},
		{IDs: components.BrokenIDs{Thunderstore: "Ns-Named"}, Through: "1.0.0", Reason: "r"},
	}}
	mods := []profile.Mod{
		{Key: "fixable", ID: "thunderstore:Ns-Fixable", Version: "1.0.0", Enabled: true},
		{Key: "stuck", ID: "thunderstore:Ns-Stuck", Version: "1.0.0", Enabled: true},
		{Key: "old", ID: "thunderstore:Ns-Old", Version: "1.0.0", Enabled: true},
		{Key: "off", ID: "thunderstore:Ns-Off", Version: "1.0.0"},
		{Key: "named", ID: "thunderstore:Ns-Named", Version: "1.0.0", Enabled: true},
	}
	has := func(id mod.ID, _ int) bool { return id == "thunderstore:Ns-Fixable" }
	got := sweepKnownBroken(mods, []SweepMod{{Key: "named"}}, list, "", has)
	fixes := map[string]string{}
	for _, m := range got {
		fixes[m.Key] = m.Fix
	}
	want := map[string]string{"fixable": fixUpdate, "stuck": fixNone, "old": fixOff}
	if !maps.Equal(fixes, want) {
		t.Fatalf("fixes = %v, want %v", fixes, want)
	}
}

func TestSweepCompatAsksThroughTheAppsMetaClient(t *testing.T) {
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		asked.Add(1)
		_, _ = w.Write([]byte(`[{"name":"Bus","id":"hootless.BusLocations","compatibility":{"status":"broken"}}]`))
	}))
	t.Cleanup(srv.Close)
	svc, _ := sweepEnv(t)
	svc.SweepCompat = nil
	svc.Meta = &meta.Client{HTTP: srv.Client(), CacheDir: t.TempDir(), CompatURL: srv.URL}
	idx, err := svc.sweepCompat(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.Lookup("hootless.BusLocations", 0); !ok || asked.Load() != 1 {
		t.Fatalf("asked %d times through the app's client, found %v", asked.Load(), ok)
	}
}
