//go:build !windows

package launchsvc

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/loader"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestABepInExGameReportsItsSceneOwnVersionAndWhichPackagesLoaded(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	set, err := settings.Open()
	if err != nil {
		t.Fatal(err)
	}
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ps, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	const lc = "lethal-company"
	p, err := ps.Create(lc, "A")
	if err != nil {
		t.Fatal(err)
	}
	dll, err := os.ReadFile(filepath.Join("..", "dotnet", "testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	withPlugin := tsPackage(t, "Fixture", map[string]string{"plugins/Mod.dll": string(dll)})
	if _, err := ps.InstallSource(t.Context(), lc, p.ID, withPlugin, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Fixture", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	assetsOnly := tsPackage(t, "Skins", map[string]string{"plugins/skins/a.png": "png"})
	if _, err := ps.InstallSource(t.Context(), lc, p.ID, assetsOnly, profile.Source{Kind: profile.KindThunderstore, Name: "Ns-Skins", Version: "1.0.0"}); err != nil {
		t.Fatal(err)
	}
	dir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	replies := make(chan string, 2)
	replies <- `ok {"gameVersion":"0.1","gameVersionSource":"unity","scene":"InitScene","plugins":[{"guid":"someone.else","version":"1.0.0"}]}`
	replies <- `ok {"gameVersion":"v73","gameVersionSource":"game","scene":"MainMenu","plugins":[{"guid":"COM.FIXTURE.PLUGIN","version":"1.2.3"}]}`
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			r := bufio.NewReader(c)
			_, _ = r.ReadString('\n')
			_, _ = r.ReadString('\n')
			_, _ = fmt.Fprintln(c, <-replies)
			_ = c.Close()
		}
	}()
	stateFile := filepath.Join(dir, filepath.FromSlash(bridge.BepInEx.StateFile))
	if err := os.MkdirAll(filepath.Dir(stateFile), 0o700); err != nil {
		t.Fatal(err)
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v", ln.Addr())
	}
	if err := os.WriteFile(stateFile, fmt.Appendf(nil, `{"port":%d,"token":"tok"}`, addr.Port), 0o600); err != nil {
		t.Fatal(err)
	}

	svc := NewService(t.TempDir(), set, ps)
	g := game.Find(lc)
	svc.logs[keyOf(g)] = session{buf: &launch.Buffer{}, profile: p.ID}
	st := Status{Game: lc, Install: installOf(g), State: Running, Profile: p.ID}
	l, _ := svc.loaderOf(lc, p.ID)
	live, ok := l.(loader.RunningState)
	if !ok {
		t.Fatal("BepInEx reports no running state")
	}
	packages := svc.livePackages(lc, p.ID)

	first, ok := svc.askLive(t.Context(), g, st, live, packages)
	if !ok || first.Scene != "InitScene" || len(first.Mods) != 1 || first.Mods[0].Loaded {
		t.Fatalf("before the plugin loaded: %+v", first)
	}
	if v := svc.logs[keyOf(g)].gameVersion; v != "" {
		t.Fatalf("Unity's Application.version was kept as the game's: %q", v)
	}
	if _, played := set.Get().LastPlayed[lc]; played {
		t.Fatalf("lastPlayed = %#v", set.Get().LastPlayed[lc])
	}
	second, ok := svc.askLive(t.Context(), g, st, live, packages)
	if !ok || second.Scene != "MainMenu" || len(second.Mods) != 1 || !second.Mods[0].Loaded || second.Mods[0].ID != first.Mods[0].ID {
		t.Fatalf("after the plugin loaded: %+v", second)
	}
	if v := svc.logs[keyOf(g)].gameVersion; v != "v73" {
		t.Fatalf("session version = %q", v)
	}
	if got := set.Get().LastPlayed[lc]; got.GameVersion != "v73" || got.Profile != p.ID {
		t.Fatalf("lastPlayed = %#v", got)
	}
}

func TestABepInExRunRecordsTheVersionItsBridgeReported(t *testing.T) {
	datadirtest.Use(t, t.TempDir())
	items, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	ps, err := profile.Open(items)
	if err != nil {
		t.Fatal(err)
	}
	const lc = "lethal-company"
	p, err := ps.Create(lc, "A")
	if err != nil {
		t.Fatal(err)
	}
	dir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "BepInEx"), 0o700); err != nil {
		t.Fatal(err)
	}
	const body = "[Info   :   BepInEx] BepInEx 5.4.22 - Lethal Company\n"
	if err := os.WriteFile(filepath.Join(dir, "BepInEx", "LogOutput.log"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := NewService(t.TempDir(), nil, ps)
	g := game.Find(lc)
	svc.logs[keyOf(g)] = session{buf: &launch.Buffer{}, profile: p.ID, gameVersion: "v73"}
	svc.record(g, p.ID, time.Now(), false)

	runs, err := svc.Runs(lc, p.ID)
	if err != nil || len(runs) != 1 || runs[0].GameVersion != "v73" {
		t.Fatalf("runs = %+v, %v", runs, err)
	}
}
