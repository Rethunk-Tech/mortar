package control

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func TestUpdatesSettled(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		items   []queue.Item
		ids     []string
		settled bool
		err     string
	}{
		{
			name:    "done is settled",
			items:   []queue.Item{{ID: "a", State: queue.StateDone}},
			ids:     []string{"a"},
			settled: true,
		},
		{
			name:    "waiting-click is settled",
			items:   []queue.Item{{ID: "a", State: queue.StateWaitingClick}},
			ids:     []string{"a"},
			settled: true,
		},
		{
			name:    "queued is not settled",
			items:   []queue.Item{{ID: "a", State: queue.StateQueued}},
			ids:     []string{"a"},
			settled: false,
		},
		{
			name:    "downloading is not settled",
			items:   []queue.Item{{ID: "a", State: queue.StateDownloading}},
			ids:     []string{"a"},
			settled: false,
		},
		{
			name:    "installing is not settled",
			items:   []queue.Item{{ID: "a", State: queue.StateInstalling}},
			ids:     []string{"a"},
			settled: false,
		},
		{
			name:    "failed uses item error",
			items:   []queue.Item{{ID: "a", Name: "Mod A", State: queue.StateFailed, Error: "disk full"}},
			ids:     []string{"a"},
			settled: true,
			err:     "disk full",
		},
		{
			name:    "other state without error uses name",
			items:   []queue.Item{{ID: "a", Name: "Mod A", State: queue.StateFailed}},
			ids:     []string{"a"},
			settled: true,
			err:     "the update for Mod A did not finish (failed)",
		},
		{
			name:    "other state without name uses a mod",
			items:   []queue.Item{{ID: "a", State: queue.StateFailed}},
			ids:     []string{"a"},
			settled: true,
			err:     "the update for a mod did not finish (failed)",
		},
		{
			name:    "missing id is ignored",
			items:   []queue.Item{{ID: "b", State: queue.StateQueued}},
			ids:     []string{"a"},
			settled: true,
		},
		{
			name:    "empty ids are settled",
			items:   []queue.Item{{ID: "a", State: queue.StateQueued}},
			ids:     nil,
			settled: true,
		},
		{
			name: "queued among done is not settled",
			items: []queue.Item{
				{ID: "a", State: queue.StateDone},
				{ID: "b", State: queue.StateQueued},
			},
			ids:     []string{"a", "b"},
			settled: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := updatesSettled(tc.items, tc.ids)
			if got != tc.settled {
				t.Fatalf("settled: got %v want %v", got, tc.settled)
			}
			if tc.err == "" {
				if err != nil {
					t.Fatalf("err: got %v want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tc.err {
				t.Fatalf("err: got %v want %s", err, tc.err)
			}
		})
	}
}

func TestQueueableUpdate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		u     problems.Update
		named bool
		want  bool
	}{
		{name: "plain", want: true},
		{name: "plain named", named: true, want: true},
		{name: "unofficial", u: problems.Update{Unofficial: true}},
		{name: "pick file", u: problems.Update{PickFile: true}},
		{name: "not distributable", u: problems.Update{NotDistributable: true}},
		{name: "switch unnamed", u: problems.Update{Switch: true}},
		{name: "switch named", u: problems.Update{Switch: true}, named: true, want: true},
		{
			name:  "unofficial switch named still skipped",
			u:     problems.Update{Unofficial: true, Switch: true},
			named: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := queueableUpdate(tc.u, tc.named); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestLaunchWarningError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  launchWarningError
		want string
	}{
		{
			name: "force only",
			want: "Play anyway with --force.",
		},
		{
			name: "game updated none broken",
			err: launchWarningError{
				game: "Stardew Valley",
				update: problems.UpdateWarning{
					Changed:   true,
					Installed: "1.6.15",
					Recorded:  "1.6.14",
				},
			},
			want: "The game was updated: Stardew Valley is now 1.6.15; this profile last launched on 1.6.14. None of this profile's mods are marked broken for the new version.\nPlay anyway with --force.",
		},
		{
			name: "game updated with broken mods",
			err: launchWarningError{
				game: "Stardew Valley",
				update: problems.UpdateWarning{
					Changed:   true,
					Installed: "1.6.15",
					Recorded:  "1.6.14",
					Broken:    []problems.Broken{{Name: "Automate"}, {Name: "Lookup Anything"}},
				},
			},
			want: "The game was updated: Stardew Valley is now 1.6.15; this profile last launched on 1.6.14. Mods marked broken: Automate, Lookup Anything.\nPlay anyway with --force.",
		},
		{
			name: "save gap with farmer",
			err: launchWarningError{
				gap:  true,
				save: savessvc.Fit{Info: saves.Info{Farmer: "Alex", Folder: "Alex_123"}},
			},
			want: "Your last save needs other mods: Alex's farm (Alex_123) was last played with mods this profile does not have on.\nPlay anyway with --force.",
		},
		{
			name: "save gap without farmer",
			err: launchWarningError{
				gap:  true,
				save: savessvc.Fit{Info: saves.Info{Folder: "Farm_1"}},
			},
			want: "Your last save needs other mods: The farm (Farm_1) was last played with mods this profile does not have on.\nPlay anyway with --force.",
		},
		{
			name: "update and save gap",
			err: launchWarningError{
				game: "Stardew Valley",
				update: problems.UpdateWarning{
					Changed:   true,
					Installed: "1.6.15",
					Recorded:  "1.6.14",
				},
				gap:  true,
				save: savessvc.Fit{Info: saves.Info{Farmer: "Alex", Folder: "Alex_123"}},
			},
			want: "The game was updated: Stardew Valley is now 1.6.15; this profile last launched on 1.6.14. None of this profile's mods are marked broken for the new version.\nYour last save needs other mods: Alex's farm (Alex_123) was last played with mods this profile does not have on.\nPlay anyway with --force.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestLaunchRefusesGameUpdateUnlessForced(t *testing.T) {
	s, p := launchHarness(t)
	markGameUpdated(t, s, p.ID)

	started := false
	s.Launches.EnsureLoader = func(context.Context, string, string, bool) error {
		started = true
		return errors.New("stop")
	}

	_, err := s.launch(t.Context(), "stardew", p.ID, "", "", false)
	var warn launchWarningError
	if !errors.As(err, &warn) || !warn.update.Changed {
		t.Fatalf("unforced launch: %v", err)
	}
	if started {
		t.Fatal("StartPreset ran despite the game-update warning")
	}

	_, err = s.launch(t.Context(), "stardew", p.ID, "", "", true)
	if errors.As(err, &warn) {
		t.Fatalf("force still refused: %v", err)
	}
	if !started {
		t.Fatal("force did not call StartPreset")
	}
}

func TestLaunchRefusesSaveGapUnlessForced(t *testing.T) {
	s, p := launchHarness(t)
	if !markSaveGap(t, s, p.ID) {
		t.Fatal("the fixture save did not register as a gap")
	}

	started := false
	s.Launches.EnsureLoader = func(context.Context, string, string, bool) error {
		started = true
		return errors.New("stop")
	}

	_, err := s.launch(t.Context(), "stardew", p.ID, "", "", false)
	var warn launchWarningError
	if !errors.As(err, &warn) || !warn.gap {
		t.Fatalf("unforced launch: %v", err)
	}
	if started {
		t.Fatal("StartPreset ran despite the save-gap warning")
	}

	_, err = s.launch(t.Context(), "stardew", p.ID, "", "", true)
	if errors.As(err, &warn) {
		t.Fatalf("force still refused: %v", err)
	}
	if !started {
		t.Fatal("force did not call StartPreset")
	}
}

func TestLaunchSkipsPlayCheck(t *testing.T) {
	s, p := launchHarness(t)
	markGameUpdated(t, s, p.ID)
	setGamePref(t, s, func(g *settings.GameSettings) { g.SkipPlayCheck = true })

	started := false
	s.Launches.EnsureLoader = func(context.Context, string, string, bool) error {
		started = true
		return errors.New("stop")
	}
	_, err := s.launch(t.Context(), "stardew", p.ID, "", "", false)
	if _, ok := errors.AsType[launchWarningError](err); ok {
		t.Fatalf("skipPlayCheck still refused: %v", err)
	}
	if !started {
		t.Fatal("skipPlayCheck did not call StartPreset")
	}
}

func TestAwaitStartReturnsLaunchFailureAndTimeout(t *testing.T) {
	s, p := launchHarness(t)
	s.Launches.EnsureLoader = func(context.Context, string, string, bool) error {
		return errors.New("loader missing")
	}
	if err := s.Launches.StartPreset(t.Context(), "stardew", p.ID, "", "", true); err != nil {
		t.Fatal(err)
	}
	_, err := s.awaitStart(t.Context(), "stardew", "")
	if err == nil || err.Error() == "the game did not start within 3 minutes" {
		t.Fatalf("launch failure: %v", err)
	}

	block := make(chan struct{})
	s.Launches.EnsureLoader = func(context.Context, string, string, bool) error {
		<-block
		return errors.New("stopped")
	}
	if err := s.Launches.StartPreset(t.Context(), "stardew", p.ID, "", "", true); err != nil {
		close(block)
		t.Fatal(err)
	}
	prev := launchWait
	launchWait = time.Millisecond
	defer func() { launchWait = prev }()
	_, err = s.awaitStart(t.Context(), "stardew", "")
	if err == nil || err.Error() != "the game did not start within 3 minutes" {
		close(block)
		t.Fatalf("timeout: %v", err)
	}
	close(block)
	waitIdle(t, s)
}

func TestQueueUpdatesNotFoundSkippingAndBatching(t *testing.T) {
	s, p := launchHarness(t)

	_, _, err := s.queueUpdates(t.Context(), "stardew", p.ID, nil, true, false)
	wantNotFound(t, err, "no updates available")
	_, _, err = s.queueUpdates(t.Context(), "stardew", p.ID, []string{"Test.Mod"}, false, false)
	wantNotFound(t, err, "no update available for Test.Mod")
	out, items, err := s.queueUpdates(t.Context(), "stardew", p.ID, nil, true, true)
	if err != nil || len(items) != 0 || out.Queued != 0 {
		t.Fatalf("beforePlay empty: %+v %v %v", out, items, err)
	}
}

func TestPickUpdatesHonoursNamesAllAndCautionBeforePlay(t *testing.T) {
	t.Parallel()
	plain := problems.Update{Key: "k", ID: "smapi:Test.Mod", Name: "Test Mod", Version: "2.0.0", GitHubRepo: "owner/mod"}
	caution := problems.Update{Key: "c", ID: "smapi:Caution.Mod", Name: "Caution", Version: "2.0.0", GitHubRepo: "owner/caution", CautionMessage: "read me"}
	sw := problems.Update{Key: "s", ID: "smapi:Switch.Mod", Name: "Switch", Version: "2.0.0", GitHubRepo: "owner/switch", Switch: true}
	all := []problems.Update{plain, caution, sw}
	keys := func(us []problems.Update) string {
		var out []string
		for _, u := range us {
			out = append(out, u.Key)
		}
		return strings.Join(out, ",")
	}
	for name, tc := range map[string]struct {
		ids             []string
		all, beforePlay bool
		want            string
	}{
		"all skips a switch":             {all: true, want: "k,c"},
		"before Play skips a caution":    {all: true, beforePlay: true, want: "k"},
		"a named switch is queued":       {ids: []string{"Switch.Mod"}, want: "s"},
		"names narrow the set":           {ids: []string{"Test.Mod"}, want: "k"},
		"all with a name takes the rest": {ids: []string{"Test.Mod"}, all: true, want: "k,c"},
	} {
		if got := keys(pickUpdates(all, tc.ids, tc.all, tc.beforePlay)); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

func TestUpdateBeforePlayNoUpdates(t *testing.T) {
	s, p := launchHarness(t)
	if err := s.updateBeforePlay(t.Context(), "stardew", p.ID); err != nil {
		t.Fatal(err)
	}
}

func TestLaunchUpdatesBeforePlayThenRefuses(t *testing.T) {
	s, p := launchHarness(t)
	markGameUpdated(t, s, p.ID)
	setGamePref(t, s, func(g *settings.GameSettings) { g.UpdateModsBeforePlayDefault = true })
	_, err := s.launch(t.Context(), "stardew", p.ID, "", "", false)
	var warn launchWarningError
	if !errors.As(err, &warn) || !warn.update.Changed {
		t.Fatalf("after an empty pre-Play update: %v", err)
	}
}

func launchHarness(t *testing.T) (*Services, profile.Profile) {
	t.Helper()
	s := services(t)
	s.Launches = launchsvc.NewService(t.TempDir(), s.Settings, s.Store)
	sv, err := savessvc.NewService(t.TempDir(), s.Store, s.Settings, &meta.Client{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	s.Saves = sv
	q, err := queue.New(queue.Deps{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	s.Queue = q
	lib := t.TempDir()
	folder := filepath.Join(lib, "steamapps", "common", "Stardew Valley")
	if err := os.MkdirAll(folder, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, "Stardew Valley.dll"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeSteamBuild(t, lib, "100")
	if _, err := s.Settings.Update(func(v *settings.Settings) { v.GameFolders["stardew"] = folder }); err != nil {
		t.Fatal(err)
	}
	p, err := s.Profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	return s, p
}

func writeSteamBuild(t *testing.T, lib, n string) {
	t.Helper()
	manifest := "\"AppState\"\n{\n\t\"buildid\"\t\t\"" + n + "\"\n}\n"
	if err := os.WriteFile(filepath.Join(lib, "steamapps", "appmanifest_413150.acf"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
}

func markGameUpdated(t *testing.T, s *Services, profileID string) {
	t.Helper()
	if _, err := s.Settings.RecordLastPlayed("stardew", profileID, time.Now(), ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Settings.RecordBuild("stardew", "100"); err != nil {
		t.Fatal(err)
	}
	folder := s.Settings.Get().GameFolders["stardew"]
	writeSteamBuild(t, filepath.Dir(filepath.Dir(filepath.Dir(folder))), "101")
	w, err := s.Problems.UpdateWarning(t.Context(), "stardew", profileID)
	if err != nil || !w.Changed {
		t.Fatalf("game update fixture: %+v, %v", w, err)
	}
}

func markSaveGap(t *testing.T, s *Services, profileID string) bool {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "StardewValley", "Saves", "Farm_1")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	body := "<SaveGame><modData><item><key><string>A.Mod/x</string></key><value><string>1</string></value></item></modData></SaveGame>"
	if err := os.WriteFile(filepath.Join(dir, "Farm_1"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	info := `<?xml version="1.0"?><Farmer><name>Alex</name><farmName>Gap</farmName></Farmer>`
	if err := os.WriteFile(filepath.Join(dir, "SaveGameInfo"), []byte(info), 0o600); err != nil {
		t.Fatal(err)
	}
	idx := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"A.Mod": ["Nexus:5"]}`))
	}))
	t.Cleanup(idx.Close)
	sv, err := savessvc.NewService(home, s.Store, s.Settings, &meta.Client{
		HTTP: idx.Client(), CacheDir: t.TempDir(), IndexURL: idx.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	s.Saves = sv
	fit, gap, err := s.Saves.LastSaveGap(t.Context(), "stardew", profileID)
	if err != nil {
		t.Fatalf("LastSaveGap: %v", err)
	}
	if !gap {
		t.Fatalf("gap fixture: %+v", fit)
	}
	return true
}

func setGamePref(t *testing.T, s *Services, edit func(*settings.GameSettings)) {
	t.Helper()
	if _, err := s.Settings.Update(func(v *settings.Settings) {
		g := v.GamePrefs("stardew")
		edit(&g)
		if v.Games == nil {
			v.Games = map[string]*settings.GameSettings{}
		}
		v.Games["stardew"] = &g
	}); err != nil {
		t.Fatal(err)
	}
}

func wantNotFound(t *testing.T, err error, want string) {
	t.Helper()
	if usererr.KindOf(err) != usererr.NotFound {
		t.Fatalf("kind: %v", err)
	}
	if _, raw := usererr.Parse(err.Error()); raw != want {
		t.Fatalf("msg: %v want %s", err, want)
	}
}

func waitIdle(t *testing.T, s *Services) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		st, err := s.Launches.Status("stardew")
		if err == nil && st.State == launchsvc.Idle {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}
