package smapi

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/loader"
)

func TestRunningStateReadsTheBridgeStatusWithoutSceneOrLocation(t *testing.T) {
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		r := bufio.NewReader(c)
		_, _ = r.ReadString('\n')
		cmd, _ := r.ReadString('\n')
		if cmd == "status\n" {
			_, _ = fmt.Fprint(c, `ok {"gameVersion":"1.6.15","gameVersionSource":"game","plugins":[{"guid":"Pathoschild.ContentPatcher","version":"2.5.0"}]}`+"\n")
		}
	}()
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v", ln.Addr())
	}
	folder := t.TempDir()
	state := fmt.Sprintf(`{"port":%d,"token":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}`, addr.Port)
	if err := os.WriteFile(filepath.Join(folder, bridge.SMAPI.StateFile), []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}
	live, err := Loader{}.RunningState(t.Context(), loader.ProfileView{Companion: folder})
	if err != nil {
		t.Fatal(err)
	}
	if live.GameVersion != "1.6.15" || live.Scene != "" || len(live.Plugins) != 1 || live.Plugins[0] != (loader.LivePlugin{GUID: "Pathoschild.ContentPatcher", Version: "2.5.0"}) {
		t.Fatalf("live = %+v", live)
	}
	if _, err := (Loader{}).RunningState(t.Context(), loader.ProfileView{}); err == nil {
		t.Fatal("a profile with no bridge answered a status query")
	}
}
