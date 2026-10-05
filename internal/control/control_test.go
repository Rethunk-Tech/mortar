package control

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/mod"

	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/testenv"
)

func waitFile(t *testing.T, path string, present bool) {
	t.Helper()
	for range 200 {
		_, err := os.Stat(path)
		if (err == nil) == present {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s present=%v never happened", path, !present)
}

func TestModProblemsFiltersByNexusModAndOmitsCosmeticConflicts(t *testing.T) {
	p := profile.Profile{Entries: []profile.Entry{
		{Source: profile.Source{Kind: profile.KindNexus, ModID: 42}, Mods: []profile.Component{{ID: "smapi:Pack.Target", Name: "Target"}}},
		{Source: profile.Source{Kind: profile.KindNexus, ModID: 99}, Mods: []profile.Component{{ID: "smapi:Pack.Other", Name: "Other"}}},
	}}
	result := problems.Result{
		Missing: []problems.Missing{{DependentID: "smapi:pack.target", DependentName: "Target", ID: "smapi:Core.Required", Reason: "absent"}},
		Broken:  []problems.Broken{{ID: "smapi:Pack.Target", Name: "Target"}},
		AssetConflicts: []problems.AssetConflict{
			{PackIDs: []mod.ID{"smapi:Pack.Target", "smapi:Pack.Other"}, Names: []string{"Target", "Other"}},
			{PackIDs: []mod.ID{"smapi:Pack.Target", "smapi:Pack.Other"}, Names: []string{"Target", "Other"}, Cosmetic: true},
		},
	}
	got := modProblems(p, result, 42)
	if len(got) != 3 || got[0].Kind != "missing" || got[1].Kind != "broken" || got[2].Kind != "conflict" {
		t.Fatalf("modProblems = %+v", got)
	}
	if got[2].Text != "conflicts with Other" {
		t.Fatalf("conflict text = %q", got[2].Text)
	}
}

func TestServeAnswersOnlyTokenHoldersAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, dir, "1.2.3", func(_ context.Context, method string, p Params) (any, error) {
			if method == "fail" {
				return nil, errors.New("boom")
			}
			return map[string]string{"method": method, "game": p.Game}, nil
		})
	}()
	path := filepath.Join(dir, controlwire.FileName)
	waitFile(t, path, true)
	if info, err := os.Stat(path); err == nil && info.Mode().Perm()&0o077 != 0 {
		t.Errorf("controlwire.Discovery file mode %v is readable by others", info.Mode().Perm())
	}

	var got map[string]string
	if err := controlwire.CallDir(dir, "echo", Params{Game: "stardew"}, &got, time.Second); err != nil {
		t.Fatal(err)
	}
	if got["method"] != "echo" || got["game"] != "stardew" {
		t.Fatalf("got %v", got)
	}
	if err := controlwire.CallDir(dir, "fail", Params{}, nil, time.Second); err == nil || err.Error() != "boom" {
		t.Fatalf("handler error not passed back: %v", err)
	}

	var hello controlwire.Hello
	if err := controlwire.CallDir(dir, "hello", nil, &hello, time.Second); err != nil || hello != (controlwire.Hello{Version: "1.2.3", Protocol: controlwire.Protocol}) {
		t.Fatalf("hello = %+v, %v", hello, err)
	}
	if err := controlwire.CheckProtocol(dir, time.Second); err != nil {
		t.Fatal(err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := root.ReadFile(controlwire.FileName)
	_ = root.Close()
	if err != nil {
		t.Fatal(err)
	}
	var d controlwire.Discovery
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	bad := t.TempDir()
	d.Token = strings.Repeat("0", len(d.Token))
	forged, _ := json.Marshal(d)
	if err := os.WriteFile(filepath.Join(bad, controlwire.FileName), forged, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := controlwire.CallDir(bad, "echo", Params{}, nil, time.Second); err == nil || err.Error() != "unauthorized" {
		t.Fatalf("a wrong token must be refused, got %v", err)
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Serve must end with the context's error, got %v", err)
	}
	waitFile(t, path, false)
	if err := controlwire.CallDir(dir, "echo", Params{}, nil, time.Second); !errors.Is(err, controlwire.ErrNotRunning) {
		t.Fatalf("after shutdown: %v", err)
	}
}

func TestCheckProtocolRefusesAnotherProtocol(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = bufio.NewReader(conn).ReadBytes('\n')
		_, _ = fmt.Fprintf(conn, `{"result":{"version":"9.9.9","protocol":%d}}`+"\n", controlwire.Protocol+1)
	}()
	dir := t.TempDir()
	tcp, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatal("no TCP address")
	}
	d, _ := json.Marshal(controlwire.Discovery{Port: tcp.Port, Token: "t", Protocol: controlwire.Protocol + 1})
	if err := os.WriteFile(filepath.Join(dir, controlwire.FileName), d, 0o600); err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("this mortar CLI speaks control protocol %d, the app speaks %d", controlwire.Protocol, controlwire.Protocol+1)
	if err := controlwire.CheckProtocol(dir, time.Second); err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
}

func services(t *testing.T) *Services {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("LOCALAPPDATA", filepath.Join(tmp, "data"))
	home := filepath.Join(tmp, "home")
	st, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	_, profiles := testenv.Stores(t)
	return &Services{
		Version: "test", Settings: st, Games: game.NewService(home, st), Store: profiles,
		Profiles: profile.NewService(profiles, home, st),
		Problems: problems.NewService(home, st, profiles, &meta.Client{CacheDir: filepath.Join(tmp, "cache")}),
	}
}

func TestHandleProfilesByNameAndID(t *testing.T) {
	s := services(t)
	var events []string
	s.Emit = func(name string, data any) { events = append(events, fmt.Sprintf("%s:%v", name, data)) }
	ctx := context.Background()

	created, err := s.Handle(ctx, "profile.create", Params{Game: "stardew", Name: "Spring Farm"})
	if err != nil {
		t.Fatal(err)
	}
	p, ok := created.(profile.Profile)
	if !ok {
		t.Fatalf("create returned %T", created)
	}
	if len(events) != 1 || events[0] != ChangedEvent+":stardew" {
		t.Fatalf("a change must tell the window, got %v", events)
	}
	if _, err := s.Handle(ctx, "profile.create", Params{Game: "stardew", Name: " "}); err == nil {
		t.Error("a blank name must be refused")
	}

	for _, sel := range []string{p.ID, "spring farm"} {
		rows, err := s.Handle(ctx, "mods", Params{Game: "stardew", Profile: sel})
		if err != nil {
			t.Fatalf("resolve %q: %v", sel, err)
		}
		if got, ok := rows.([]ModRow); !ok || len(got) != 0 {
			t.Fatalf("new profile has mods %v", got)
		}
	}
	if _, err := s.Handle(ctx, "mods", Params{Game: "stardew", Profile: "nope"}); err == nil {
		t.Error("an unknown profile must be an error")
	}

	if _, err := s.Handle(ctx, "profile.create", Params{Game: "stardew", Name: "SPRING FARM"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Handle(ctx, "mods", Params{Game: "stardew", Profile: "Spring Farm"}); err == nil || !strings.Contains(err.Error(), "use an id") {
		t.Fatalf("an ambiguous name must list ids, got %v", err)
	}

	res, err := s.Handle(ctx, "conflicts", Params{Game: "stardew", Profile: p.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := res.([]problems.AssetConflict); !ok || len(got) != 0 {
		t.Fatalf("empty profile has conflicts %v", got)
	}
	if _, err := s.Handle(ctx, "mods.disable", Params{Game: "stardew", Profile: p.ID, IDs: []string{"Some.Mod"}}); err == nil {
		t.Error("disabling a mod the profile lacks must be an error")
	}
	if _, err := s.Handle(ctx, "nope", Params{Game: "stardew", Profile: p.ID}); err == nil {
		t.Error("an unknown method must be an error")
	}
}

func TestHandleTrash(t *testing.T) {
	s := services(t)
	var events []string
	s.Emit = func(name string, data any) { events = append(events, fmt.Sprintf("%s:%v", name, data)) }
	ctx := context.Background()

	p, err := s.Profiles.Create("stardew", "Gone")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Profiles.Delete("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	events = nil

	res, err := s.Handle(ctx, "trash.list", Params{Game: "stardew"})
	if err != nil {
		t.Fatal(err)
	}
	items, ok := res.([]profile.TrashItem)
	if !ok || len(items) != 1 || items[0].ID != p.ID {
		t.Fatalf("trash.list: %#v", res)
	}

	restored, err := s.Handle(ctx, "trash.restore", Params{Game: "stardew", Profile: p.Name})
	if err != nil {
		t.Fatal(err)
	}
	if prof, ok := restored.(profile.Profile); !ok || prof.ID != p.ID {
		t.Fatalf("restore: %#v", restored)
	}
	if len(events) != 1 || events[0] != ChangedEvent+":stardew" {
		t.Fatalf("restore must notify the window, got %v", events)
	}

	if err := s.Profiles.Delete("stardew", p.ID); err != nil {
		t.Fatal(err)
	}
	events = nil
	if _, err := s.Handle(ctx, "trash.delete", Params{Game: "stardew", Profile: p.ID}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("delete must notify the window, got %v", events)
	}
	if res, err := s.Handle(ctx, "trash.list", Params{Game: "stardew"}); err != nil {
		t.Fatal(err)
	} else if items, ok := res.([]profile.TrashItem); !ok || len(items) != 0 {
		t.Fatalf("after purge: %#v", res)
	}

	p2, err := s.Profiles.Create("stardew", "Also Gone")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Profiles.Delete("stardew", p2.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Handle(ctx, "trash.empty", Params{Game: "stardew"}); err != nil {
		t.Fatal(err)
	}
	if res, err := s.Handle(ctx, "trash.list", Params{Game: "stardew"}); err != nil {
		t.Fatal(err)
	} else if items, ok := res.([]profile.TrashItem); !ok || len(items) != 0 {
		t.Fatalf("after empty: %#v", res)
	}
}

func TestHandleProfileCompareAndHistory(t *testing.T) {
	s := services(t)
	ctx := context.Background()
	a, err := s.Profiles.Create("stardew", "A")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Profiles.Create("stardew", "B"); err != nil {
		t.Fatal(err)
	}
	res, err := s.Handle(ctx, "profile.compare", Params{Game: "stardew", Profile: a.ID, Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	diff, ok := res.(profile.CLICompare)
	if !ok || len(diff.OnlyA) != 0 || len(diff.OnlyB) != 0 || len(diff.Identical) != 0 {
		t.Fatalf("empty comparison: %#v", res)
	}
	res, err = s.Handle(ctx, "profile.history", Params{Game: "stardew", Profile: a.ID})
	if err != nil {
		t.Fatal(err)
	}
	if rows, ok := res.([]HistoryRow); !ok || len(rows) != 0 {
		t.Fatalf("empty history: %#v", res)
	}
}

func TestHandleLibraryMethods(t *testing.T) {
	s := services(t)
	ctx := context.Background()
	if _, err := s.Handle(ctx, "profile.create", Params{Game: "stardew", Name: "Farm"}); err != nil {
		t.Fatal(err)
	}
	usage, err := s.Handle(ctx, "history.usage", Params{Game: "stardew"})
	if err != nil {
		t.Fatal(err)
	}
	if rows, ok := usage.([]profile.HistoryUsage); !ok || len(rows) != 1 || rows[0].ProfileName != "Farm" {
		t.Fatalf("history.usage = %#v", usage)
	}
	if _, err := s.Handle(ctx, "library.hidden", Params{Game: "stardew", Profile: "Farm"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Handle(ctx, "library.hidden", Params{Game: "stardew", Profile: "nope"}); err == nil {
		t.Error("an unknown profile must be an error")
	}
	extra, err := s.Handle(ctx, "library.extra", Params{Game: "stardew"})
	if err != nil {
		t.Fatal(err)
	}
	if pv, ok := extra.(profile.GameModsPreview); !ok || len(pv.Mods) != 0 {
		t.Fatalf("library.extra without a folder = %#v", extra)
	}
	for _, m := range []string{"templates", "backups.usage", "data.location", "archive.preview"} {
		if _, err := s.Handle(ctx, m, Params{Game: "stardew"}); err == nil {
			t.Errorf("%s without its service must fail", m)
		}
	}
}
