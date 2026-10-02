package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/Rethunk-AI/mortar/internal/store"
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
	path := filepath.Join(dir, FileName)
	waitFile(t, path, true)
	if info, err := os.Stat(path); err == nil && info.Mode().Perm()&0o077 != 0 {
		t.Errorf("discovery file mode %v is readable by others", info.Mode().Perm())
	}

	var got map[string]string
	if err := CallDir(dir, "echo", Params{Game: "stardew"}, &got, time.Second); err != nil {
		t.Fatal(err)
	}
	if got["method"] != "echo" || got["game"] != "stardew" {
		t.Fatalf("got %v", got)
	}
	if err := CallDir(dir, "fail", Params{}, nil, time.Second); err == nil || err.Error() != "boom" {
		t.Fatalf("handler error not passed back: %v", err)
	}

	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	b, err := root.ReadFile(FileName)
	_ = root.Close()
	if err != nil {
		t.Fatal(err)
	}
	var d discovery
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	bad := t.TempDir()
	d.Token = strings.Repeat("0", len(d.Token))
	forged, _ := json.Marshal(d)
	if err := os.WriteFile(filepath.Join(bad, FileName), forged, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CallDir(bad, "echo", Params{}, nil, time.Second); err == nil || err.Error() != "unauthorized" {
		t.Fatalf("a wrong token must be refused, got %v", err)
	}

	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("Serve must end with the context's error, got %v", err)
	}
	waitFile(t, path, false)
	if err := CallDir(dir, "echo", Params{}, nil, time.Second); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("after shutdown: %v", err)
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
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
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
	if _, err := s.Handle(ctx, "mods.disable", Params{Game: "stardew", Profile: p.ID, UniqueIDs: []string{"Some.Mod"}}); err == nil {
		t.Error("disabling a mod the profile lacks must be an error")
	}
	if _, err := s.Handle(ctx, "nope", Params{Game: "stardew", Profile: p.ID}); err == nil {
		t.Error("an unknown method must be an error")
	}
}
