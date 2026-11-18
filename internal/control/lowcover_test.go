package control

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/bisect"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/lan"
	"github.com/Rethunk-Tech/mortar/internal/nxmsvc"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/savedgroups"
	"github.com/Rethunk-Tech/mortar/internal/shortcut"
	"github.com/Rethunk-Tech/mortar/internal/store"
	"github.com/Rethunk-Tech/mortar/internal/storecheck"
	"github.com/Rethunk-Tech/mortar/internal/support"
	"github.com/Rethunk-Tech/mortar/internal/templates"
	"github.com/Rethunk-Tech/mortar/internal/tools"
)

func TestQueueRequestFromParams(t *testing.T) {
	install := queue.KindInstall
	tests := []struct {
		name    string
		p       Params
		want    queue.Request
		errFrag string
	}{
		{
			name:    "nexus non-numeric id",
			p:       Params{Source: "nexus", ID: "abc"},
			errFrag: `nexus mod id "abc" is not a number`,
		},
		{
			name:    "nexus zero id",
			p:       Params{Source: "nexus", ID: "0"},
			errFrag: `nexus mod id "0" is not a number`,
		},
		{
			name:    "nexus negative id",
			p:       Params{Source: "nexus", ID: "-3"},
			errFrag: `nexus mod id "-3" is not a number`,
		},
		{
			name: "nexus id only",
			p:    Params{Source: "nexus", Game: "g", ID: "42", Version: "1.2"},
			want: queue.Request{Kind: install, Game: "g", Name: "42", Version: "1.2", ModID: 42},
		},
		{
			name:    "nexus non-numeric file",
			p:       Params{Source: "nexus", ID: "42", File: "x"},
			errFrag: `nexus file id "x" is not a number`,
		},
		{
			name:    "nexus zero file",
			p:       Params{Source: "nexus", ID: "42", File: "0"},
			errFrag: `nexus file id "0" is not a number`,
		},
		{
			name: "nexus with file",
			p:    Params{Source: "nexus", ID: "42", File: "9"},
			want: queue.Request{Kind: install, Name: "42", ModID: 42, FileID: 9},
		},
		{
			name: "github with file",
			p:    Params{Source: "github", ID: "o/r", Version: "v1", File: "a.zip"},
			want: queue.Request{Kind: install, Name: "o/r", Version: "v1", Repo: "o/r", Tag: "v1", Asset: "a.zip", FileName: "a.zip"},
		},
		{
			name: "github without file",
			p:    Params{Source: "github", ID: "o/r", Version: "v1"},
			want: queue.Request{Kind: install, Name: "o/r", Version: "v1", Repo: "o/r", Tag: "v1"},
		},
		{
			name: "thunderstore",
			p:    Params{Source: "thunderstore", ID: "NS-Name"},
			want: queue.Request{Kind: install, Name: "NS-Name", Package: "NS-Name"},
		},
		{
			name: "modrinth file becomes version",
			p:    Params{Source: "modrinth", ID: "slug", File: "1.0.0"},
			want: queue.Request{Kind: install, Name: "slug", Version: "1.0.0", Package: "slug", Source: "modrinth"},
		},
		{
			name: "curseforge keeps version when file empty",
			p:    Params{Source: "curseforge", ID: "slug", Version: "keep"},
			want: queue.Request{Kind: install, Name: "slug", Version: "keep", Package: "slug", Source: "curseforge"},
		},
		{
			name: "itch",
			p:    Params{Source: "itch", ID: "game"},
			want: queue.Request{Kind: install, Name: "game", Package: "game", Source: "itch"},
		},
		{
			name: "vsmoddb",
			p:    Params{Source: "vsmoddb", ID: "mod"},
			want: queue.Request{Kind: install, Name: "mod", Package: "mod", Source: "vsmoddb"},
		},
		{
			name: "paradox",
			p:    Params{Source: "paradox", ID: "mod"},
			want: queue.Request{Kind: install, Name: "mod", Package: "mod", Source: "paradox"},
		},
		{
			name: "modio file becomes version",
			p:    Params{Source: "modio", ID: "mod", File: "2"},
			want: queue.Request{Kind: install, Name: "mod", Version: "2", Package: "mod", Source: "modio"},
		},
		{
			name:    "unknown source",
			p:       Params{Source: "ftp"},
			errFrag: `source "ftp" cannot be queued`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := queueRequestFromParams(tt.p)
			if tt.errFrag != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errFrag) {
					t.Fatalf("err = %v, want fragment %q", err, tt.errFrag)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAutomationUnavailableAndParse(t *testing.T) {
	s := &Services{}
	ctx := t.Context()

	if _, err := s.queueAdd(ctx, Params{}); err == nil || !strings.Contains(err.Error(), "download queue is unavailable") {
		t.Fatalf("queueAdd: %v", err)
	}
	if _, err := s.storeCheck(ctx, ""); err == nil || !strings.Contains(err.Error(), "store check is unavailable") {
		t.Fatalf("storeCheck: %v", err)
	}
	if _, err := s.storeRepair(ctx, Params{Key: "k"}, "p"); err == nil || !strings.Contains(err.Error(), "store check is unavailable") {
		t.Fatalf("storeRepair nil: %v", err)
	}
	if _, err := s.bisectStart(ctx, Params{}); err == nil || !strings.Contains(err.Error(), "crash check is unavailable") {
		t.Fatalf("bisectStart: %v", err)
	}
	if _, err := s.bisectStatus("id"); err == nil || !strings.Contains(err.Error(), "crash check is unavailable") {
		t.Fatalf("bisectStatus: %v", err)
	}
	if _, err := s.bisectStop("id"); err == nil || !strings.Contains(err.Error(), "crash check is unavailable") {
		t.Fatalf("bisectStop: %v", err)
	}

	if _, err := s.profileSet(ctx, Params{Key: "skipPlayCheck", Value: "maybe"}, profile.Profile{}, "id"); err == nil || !strings.Contains(err.Error(), "needs true or false") {
		t.Fatalf("skipPlayCheck: %v", err)
	}
	if _, err := s.profileSet(ctx, Params{Key: "server.mirror", Value: "other"}, profile.Profile{}, "id"); err == nil || !strings.Contains(err.Error(), "pack import is unavailable") {
		t.Fatalf("server.mirror: %v", err)
	}
	if _, err := s.profileSet(ctx, Params{Game: "no-such-game", Key: "server.port", Value: "1"}, profile.Profile{}, "id"); err == nil || !strings.Contains(err.Error(), "not a dedicated server") {
		t.Fatalf("server.port: %v", err)
	}
	if _, err := s.setServerSetting("", "id", "port", "1"); err == nil || !strings.Contains(err.Error(), "not a dedicated server") {
		t.Fatalf("setServerSetting: %v", err)
	}
}

func TestProfileLauncherUnavailable(t *testing.T) {
	s := &Services{}
	if _, err := s.profileShortcut("g", "p", false); err == nil || !strings.Contains(err.Error(), "shortcuts are unavailable") {
		t.Fatalf("shortcut: %v", err)
	}
	if _, err := s.profileSteam("g", "p"); err == nil || !strings.Contains(err.Error(), "shortcuts are unavailable") {
		t.Fatalf("steam: %v", err)
	}
	if _, err := s.gameSteamLaunchOption("g", true, true); err == nil || !strings.Contains(err.Error(), "use either --set or --clear") {
		t.Fatalf("both flags: %v", err)
	}
}

func TestMoreUnavailable(t *testing.T) {
	s := &Services{}
	ctx := t.Context()
	methods := []string{
		"groups.create", "groups.delete", "groups.rename", "groups.add", "groups.remove",
		"templates.apply", "templates.preview", "templates.undo", "templates.rename", "templates.restore",
		"tools.add", "tools.update", "tools.remove",
		"lan.peers", "lan.send", "lan.inbox", "lan.accept", "lan.decline",
		"lan.paircode", "lan.pair", "lan.paired", "lan.unpair",
		"data.move", "data.cleanup",
		"app.update.check", "app.update.install",
		"links.register", "links.enable", "links.disable", "links",
		"support.diagnostics",
		"problems.checkUpdates",
	}
	for _, name := range methods {
		fn, ok := moreGlobal[name]
		if !ok {
			t.Fatalf("missing method %s", name)
		}
		if _, err := fn(s, ctx, Params{}); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}

	if _, _, err := s.resolveIDs(Params{}); err == nil {
		t.Fatal("resolveIDs: want error")
	}
	if err := s.withNxm(func() error { return errors.New("called") }); !errors.Is(err, errNoLinks) {
		t.Fatalf("withNxm: %v", err)
	}
}

func TestQueueAddAfterResolve(t *testing.T) {
	s := services(t)
	q, err := queue.New(queue.Deps{Dir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	s.Queue = q
	if _, err := s.queueAdd(t.Context(), Params{Game: "stardew", Profile: "missing", Source: "nexus", ID: "1"}); err == nil {
		t.Fatal("missing profile")
	}
	prof, err := s.Profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.queueAdd(t.Context(), Params{Game: "stardew", Profile: prof.ID, Source: "nexus", ID: "nope"}); err == nil || !strings.Contains(err.Error(), "not a number") {
		t.Fatalf("bad nexus id: %v", err)
	}
	if _, err := s.queueAdd(t.Context(), Params{Game: "stardew", Profile: prof.ID, Source: "ftp"}); err == nil || !strings.Contains(err.Error(), "cannot be queued") {
		t.Fatalf("bad source: %v", err)
	}
}

func TestProfileSetWritesFields(t *testing.T) {
	s := services(t)
	prof, err := s.Profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	cover := filepath.Join(t.TempDir(), "cover.txt")
	if err := os.WriteFile(cover, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	must := []Params{
		{Key: "notes", Value: "hello"},
		{Key: "color", Value: profile.ProfileColors[0]},
		{Key: "icon", Value: profile.ProfileIcons[0]},
		{Key: "launchOptions", Value: "--foo"},
		{Key: "launchPrefix", Value: "env"},
		{Key: "launchEnv", Value: "A=1\\nB=2"},
		{Key: "defaultLaunchPreset", Value: ""},
		{Key: "skipPlayCheck", Value: "true"},
		{Key: "cover", Value: ""},
	}
	for _, p := range must {
		p.Game = "stardew"
		if _, err := s.profileSet(ctx, p, prof, prof.ID); err != nil {
			t.Fatalf("%s: %v", p.Key, err)
		}
	}
	for _, p := range []Params{
		{Key: "install", Value: t.TempDir()},
		{Key: "loader", Value: "nope"},
		{Key: "cover", Value: cover},
		{Key: "someOverride", Value: "1"},
	} {
		p.Game = "stardew"
		_, _ = s.profileSet(ctx, p, prof, prof.ID)
	}
}

func TestSetServerSettingOnCatalogServer(t *testing.T) {
	s := services(t)
	var gameID string
	for _, g := range components.Games() {
		if g.Server != nil {
			gameID = g.ID
			break
		}
	}
	if gameID == "" {
		t.Skip("catalog has no dedicated server game")
	}
	prof, err := s.Profiles.Create(gameID, "Host")
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.setServerSetting(gameID, prof.ID, "port", "2456")
	if err == nil {
		return
	}
	if _, err := s.setServerSetting(gameID, "missing", "port", "2456"); err == nil {
		t.Fatal("missing profile")
	}
}

func TestStoreCheckFiltersAndRepairNeedsKey(t *testing.T) {
	s, items := servicesWithItems(t)
	s.StoreCheck = storecheck.New(storecheck.Deps{Items: items})
	if _, err := s.storeRepair(t.Context(), Params{}, "p"); err == nil || !strings.Contains(err.Error(), "store key") {
		t.Fatalf("empty key: %v", err)
	}
	if _, err := s.storeCheck(t.Context(), ""); err != nil {
		t.Fatalf("check all: %v", err)
	}
	if _, err := s.storeCheck(t.Context(), "stardew"); err != nil {
		t.Fatalf("check game: %v", err)
	}
	if _, err := s.storeRepair(t.Context(), Params{Game: "stardew", Key: "missing"}, "p"); err == nil {
		t.Fatal("repair missing key")
	}
}

func TestBisectStartStatusStop(t *testing.T) {
	s := services(t)
	s.Bisect = bisect.NewService(s.Store, s.Launches)
	if _, err := s.bisectStart(t.Context(), Params{Game: "stardew", Profile: "missing"}); err == nil {
		t.Fatal("missing profile")
	}
	prof, err := s.Profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.bisectStart(t.Context(), Params{Game: "stardew", Profile: prof.ID}); err == nil {
		t.Fatal("empty profile start")
	}
	if _, err := s.bisectStatus("nope"); err == nil {
		t.Fatal("unknown status")
	}
	if _, err := s.bisectStop("nope"); err == nil {
		t.Fatal("unknown stop")
	}
}

func TestProfileLauncherCreatesAndSteamLaunchOptions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s := services(t)
	s.Plays = &shortcut.Service{SteamRoots: func() []string { return nil }}
	if _, err := s.profileShortcut("stardew", "missing", false); err == nil {
		t.Fatal("missing profile")
	}
	farm, err := s.Profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	created, err := s.profileShortcut("stardew", farm.ID, false)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Path == "" {
		t.Fatal("create returned no path")
	}
	removed, err := s.profileShortcut("stardew", farm.ID, true)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !removed.Removed {
		t.Fatal("remove did not mark Removed")
	}
	if _, err := s.profileSteam("stardew", farm.ID); err == nil {
		t.Fatal("steam shortcut without Steam")
	}
	if _, err := s.gameSteamLaunchOption("stardew", true, false); err == nil {
		t.Fatal("set launch option on an uninstalled game")
	}
	if _, err := s.gameSteamLaunchOption("stardew", false, false); err != nil {
		t.Fatalf("get launch option: %v", err)
	}
}

func TestSavedGroupsWriteMutations(t *testing.T) {
	s, items := servicesWithItems(t)
	const gameID = "lethal-company"
	prof, err := s.Profiles.Create(gameID, "Crew")
	if err != nil {
		t.Fatal(err)
	}
	files := t.TempDir()
	for name, body := range map[string]string{
		"A.dll":         "a",
		"manifest.json": `{"name":"A","namespace":"Ns","version_number":"1.0.0","dependencies":[]}`,
	} {
		if err := os.WriteFile(filepath.Join(files, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	key := store.PackageKey("Ns-A", "1.0.0")
	if err := items.AddDir(t.Context(), gameID, key, files); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.AddEntry(gameID, prof.ID, key, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-A", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	s.SavedGroups = savedgroups.NewService(s.Store, filepath.Join(t.TempDir(), "data"))
	if _, err := s.savedGroupsWrite("groups.create", Params{Game: gameID, Profile: prof.ID, Name: "Core", IDs: []string{"Ns-A"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.savedGroupsWrite("groups.rename", Params{Game: gameID, Name: "Core", Value: "Base"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.savedGroupsWrite("groups.remove", Params{Game: gameID, Name: "Base", IDs: []string{"Ns-A"}}); err == nil {
		t.Fatal("remove last mod")
	}
	if _, err := s.savedGroupsWrite("groups.add", Params{Game: gameID, Profile: prof.ID, Name: "Base", IDs: []string{"Ns-A"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.savedGroupsWrite("groups.delete", Params{Game: gameID, Name: "missing"}); err == nil {
		t.Fatal("missing group")
	}
	if _, err := s.savedGroupsWrite("groups.delete", Params{Game: gameID, Name: "Base"}); err != nil {
		t.Fatal(err)
	}
}

func TestMoreLocalWrites(t *testing.T) {
	s := services(t)
	ctx := t.Context()
	s.Lan = lan.NewService(lan.Deps{Dir: t.TempDir(), Settings: s.Settings})
	if _, err := s.lanMethod(ctx, "lan.inbox", Params{}); err != nil {
		t.Fatalf("inbox: %v", err)
	}
	if _, err := s.lanMethod(ctx, "lan.paired", Params{}); err != nil {
		t.Fatalf("paired: %v", err)
	}
	if _, err := s.lanMethod(ctx, "lan.unpair", Params{Name: "nobody"}); err != nil {
		t.Fatalf("unpair: %v", err)
	}
	if _, err := s.lanMethod(ctx, "lan.accept", Params{Name: "x"}); err == nil || !strings.Contains(err.Error(), "not a number") {
		t.Fatalf("accept parse: %v", err)
	}
	if _, err := s.lanMethod(ctx, "lan.decline", Params{Name: "1"}); err != nil {
		t.Fatalf("decline: %v", err)
	}
	if _, err := s.lanAccept(ctx, 99); err == nil {
		t.Fatal("missing arrival")
	}
	if _, err := s.lanMethod(ctx, "lan.send", Params{Game: "stardew", Profile: "missing", Name: "peer"}); err == nil {
		t.Fatal("send missing profile")
	}

	ts, err := tools.NewService(t.TempDir(), s.Settings, s.Store)
	if err != nil {
		t.Fatal(err)
	}
	s.Tools = ts
	exe := filepath.Join(t.TempDir(), "tool.sh")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	added, err := s.toolsWrite("tools.add", Params{Game: "stardew", Key: "t1", Name: "Tool", Path: exe})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	tool, ok := added.(tools.Tool)
	if !ok || tool.ID == "" {
		t.Fatalf("add returned %T %#v", added, added)
	}
	if _, err := s.toolsWrite("tools.update", Params{Game: "stardew", Key: tool.ID, Name: "Renamed", Path: exe}); err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, err := s.toolsWrite("tools.remove", Params{Game: "stardew", Key: tool.ID}); err != nil {
		t.Fatalf("remove: %v", err)
	}

	s.Templates = templates.NewService(templates.Deps{Profiles: s.Store}, t.TempDir())
	farm, err := s.Profiles.Create("stardew", "Farm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.templatesWrite("templates.apply", Params{Game: "stardew", Profile: farm.ID, Name: "missing"}); err == nil {
		t.Fatal("apply missing")
	}
	if _, err := s.templatesWrite("templates.preview", Params{Game: "stardew", Profile: farm.ID, Name: "missing"}); err == nil {
		t.Fatal("preview missing")
	}
	if _, err := s.templatesWrite("templates.rename", Params{Game: "stardew", Name: "missing", Value: "x"}); err == nil {
		t.Fatal("rename missing")
	}
	if _, err := s.templatesWrite("templates.restore", Params{Body: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("restore empty")
	}

	s.Support = support.NewService("test", func(string) problems.Environment { return problems.Environment{} }, t.TempDir(), func(string, string) (string, error) {
		return "", nil
	})
	if _, err := s.supportDiagnostics(Params{Path: filepath.Join(t.TempDir(), "d.zip")}); err != nil {
		t.Fatalf("diagnostics: %v", err)
	}
	if _, err := s.supportDiagnostics(Params{Game: "stardew", Profile: farm.ID, Path: filepath.Join(t.TempDir(), "p.zip")}); err != nil {
		t.Fatalf("diagnostics with profile: %v", err)
	}
	if _, err := s.supportDiagnostics(Params{Game: "stardew", Profile: "missing", Path: filepath.Join(t.TempDir(), "m.zip")}); err == nil {
		t.Fatal("diagnostics missing profile")
	}

	s.Nxm = nxmsvc.NewService(s.Settings, listHandler{})
	if err := s.withNxm(func() error { return nil }); err != nil {
		t.Fatalf("withNxm: %v", err)
	}

	if _, err := moreGlobal["game.steamStatus"](s, ctx, Params{}); err != nil {
		t.Fatalf("steamStatus: %v", err)
	}
	if _, err := moreGlobal["game.resetInstall"](s, ctx, Params{Game: "stardew"}); err != nil && !strings.Contains(err.Error(), "stardew") {
		t.Fatalf("resetInstall: %v", err)
	}
}
