package launchsvc

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/datadir/datadirtest"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launch"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/settings"
	"github.com/Rethunk-Tech/mortar/internal/store"
)

func TestARunningBepInExGameRecordsTheVersionItsBridgeReports(t *testing.T) {
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
	dir, err := ps.ProfileDir(lc, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	var asked atomic.Int32
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			asked.Add(1)
			r := bufio.NewReader(c)
			_, _ = r.ReadString('\n')
			_, _ = r.ReadString('\n')
			_, _ = fmt.Fprint(c, "ok {\"gameVersion\":\"v73\",\"scene\":\"MainMenu\",\"plugins\":[]}\n")
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
	body := fmt.Sprintf(`{"port":%d,"token":"tok"}`, addr.Port)
	if err := os.WriteFile(stateFile, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	svc := NewService(t.TempDir(), set, ps)
	g := game.Find(lc)
	svc.logs[keyOf(g)] = session{buf: &launch.Buffer{}, profile: p.ID}
	svc.set(Status{Game: lc, Install: installOf(g), State: Running, Profile: p.ID})
	for range 2 {
		if !svc.noteGameVersion(t.Context(), g) {
			t.Fatal("an answered version leaves nothing to ask")
		}
	}

	if v := svc.logs[keyOf(g)].gameVersion; v != "v73" {
		t.Fatalf("session version = %q", v)
	}
	if got := set.Get().LastPlayed[lc]; got.GameVersion != "v73" || got.Profile != p.ID {
		t.Fatalf("lastPlayed = %#v", got)
	}
	if n := asked.Load(); n != 1 {
		t.Fatalf("asked the bridge %d times; once it answered it is not asked again", n)
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
