package launchsvc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-safefs/diskcache"
	"github.com/Rethunk-Tech/go-smapiweb"
	"github.com/Rethunk-Tech/mortar/internal/appversion"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/migrate"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

func TestSweepUpdateSetReturnsNilWhenHasUpdateIsInjected(t *testing.T) {
	svc, _ := sweepEnv(t)
	if got := svc.sweepUpdateSet(t.Context(), "stardew", "1.6.16", "4.2.0", nil); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestSweepUpdateSetSkipsErrorProfilesAndModsWithoutID(t *testing.T) {
	svc, profiles := sweepEnv(t)
	svc.SweepHasUpdate = nil
	good := testenv.Profile(t, profiles, "stardew", "Good")
	addSweepMod(t, profiles, good.ID, "A.Mod", "HasID")
	noid := testenv.Profile(t, profiles, "stardew", "NoID")
	addSweepModWithoutID(t, profiles, noid.ID)
	broken := testenv.Profile(t, profiles, "stardew", "Broken")
	corruptProfile(t, profiles, "stardew", broken.ID)

	const gameVer, smapiVer = "1.6.16", "4.2.0"
	data, err := datadir.Dir()
	if err != nil {
		t.Fatal(err)
	}
	cache := &diskcache.Cache{Dir: filepath.Join(data, "cache")}
	key := metaUpdateKey("A.Mod", "1.0.0", gameVer, smapiVer)
	store := map[string]diskcache.Entry[smapiweb.UpdateResult]{
		key: {
			Fetched: time.Now(),
			Stamp:   appversion.Build(),
			Value: smapiweb.UpdateResult{
				ID: "A.Mod", Known: true,
				Suggested: &smapiweb.Update{Version: "2.0.0"},
			},
		},
	}
	if err := diskcache.PutEntry(cache, "smapi-updates.json", diskcache.Entry[map[string]diskcache.Entry[smapiweb.UpdateResult]]{
		Fetched: time.Now(), Value: store, Stamp: appversion.Build(),
	}); err != nil {
		t.Fatal(err)
	}

	list, err := profiles.List("stardew")
	if err != nil {
		t.Fatal(err)
	}
	got := svc.sweepUpdateSet(t.Context(), "stardew", gameVer, smapiVer, list)
	want := mod.SMAPI("A.Mod").Fold()
	if !got[want] || len(got) != 1 {
		t.Fatalf("got %v, want only %s", got, want)
	}
}

func TestSweepReportNeedsAttention(t *testing.T) {
	if (SweepReport{}).NeedsAttention() {
		t.Fatal("empty report")
	}
	if !(SweepReport{Profiles: []ProfileSweep{{Broken: []SweepMod{{Key: "a"}}}}}).NeedsAttention() {
		t.Fatal("broken mods")
	}
	if !(SweepReport{Profiles: []ProfileSweep{{MissingDeps: 1}}}).NeedsAttention() {
		t.Fatal("missing deps")
	}
}

func TestMaybeSweepEmitsWhenAProfileNeedsAttention(t *testing.T) {
	svc, profiles := sweepEnv(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	addSweepMod(t, profiles, p.ID, "A.Stuck", "Stuck")
	if err := svc.settings.RecordLastSweep("stardew", "1.6.15", "smapi", "4.1.10"); err != nil {
		t.Fatal(err)
	}
	svc.SweepVersions = func(string) (string, string, error) { return "1.6.16", "4.2.0", nil }
	svc.SweepCompat = func(context.Context) (smapiweb.CompatIndex, error) {
		return smapiweb.CompatIndex{ByID: map[string]smapiweb.CompatEntry{
			"a.stuck": {Status: smapiweb.StatusBroken, Summary: "Broken in 1.6.16"},
		}}, nil
	}
	events := make(chan SweepReport, 1)
	svc.App = sweepHost{events: events}
	svc.MaybeSweep(t.Context(), "stardew")
	select {
	case got := <-events:
		if !got.Triggered || !got.NeedsAttention() {
			t.Fatalf("emitted %+v", got)
		}
	default:
		t.Fatal("MaybeSweep did not emit")
	}

	svc.App = sweepHost{events: events}
	svc.MaybeSweep(t.Context(), "stardew")
	select {
	case got := <-events:
		t.Fatalf("unchanged versions emitted %+v", got)
	default:
	}
}

func TestSweepOnStartEmitsInTheBackground(t *testing.T) {
	svc, profiles := sweepEnv(t)
	p := testenv.Profile(t, profiles, "stardew", "Farm")
	addSweepMod(t, profiles, p.ID, "A.Stuck", "Stuck")
	if err := svc.settings.RecordLastSweep("stardew", "1.6.15", "smapi", "4.1.10"); err != nil {
		t.Fatal(err)
	}
	svc.SweepVersions = func(string) (string, string, error) { return "1.6.16", "4.2.0", nil }
	svc.SweepCompat = func(context.Context) (smapiweb.CompatIndex, error) {
		return smapiweb.CompatIndex{ByID: map[string]smapiweb.CompatEntry{
			"a.stuck": {Status: smapiweb.StatusBroken},
		}}, nil
	}
	events := make(chan SweepReport, 1)
	svc.App = sweepHost{events: events}
	SweepOnStart(svc, "stardew")
	select {
	case <-events:
	case <-time.After(2 * time.Second):
		t.Fatal("SweepOnStart did not emit")
	}
}

type sweepHost struct {
	winhost.Host
	events chan SweepReport
}

func (h sweepHost) Emit(name string, data any) {
	if name == SweepEvent {
		if rep, ok := data.(SweepReport); ok {
			h.events <- rep
		}
	}
}

func addSweepModWithoutID(t *testing.T, profiles *profile.Store, profileID string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "NoID")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"Name":"NoID","Author":"A","Version":"1.0.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.ImportExternalMods(t.Context(), "stardew", profileID, []migrate.ModPreview{
		{SourcePath: dir, Enabled: true},
	}); err != nil {
		t.Fatal(err)
	}
}

func corruptProfile(t *testing.T, profiles *profile.Store, gameID, profileID string) {
	t.Helper()
	dir, err := profiles.ProfileDir(gameID, profileID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profile.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func metaUpdateKey(id, version, gameVer, smapiVer string) string {
	return strings.ToLower(id + "||" + version + "|" + gameVer + "|" + smapiVer + "|" + sweepPlatform())
}
