package configsvc

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/testenv/testfs"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// fakeBridge answers every command with reply on loopback and writes the state file naming it into dir; "" writes
// none. The commands it was sent arrive on the returned channel.
func fakeBridge(t *testing.T, dir, reply string) <-chan string {
	t.Helper()
	got := make(chan string, 8)
	if reply == "" {
		return got
	}
	port := 1
	if !strings.HasPrefix(reply, "gone") {
		ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = ln.Close() })
		addr, ok := ln.Addr().(*net.TCPAddr)
		if !ok {
			t.Fatalf("listener address %v is not TCP", ln.Addr())
		}
		port = addr.Port
		go func() {
			for {
				c, err := ln.Accept()
				if err != nil {
					return
				}
				r := bufio.NewReader(c)
				_, _ = r.ReadString('\n')
				command, _ := r.ReadString('\n')
				got <- strings.TrimSpace(command)
				_, _ = fmt.Fprintln(c, reply)
				_ = c.Close()
			}
		}()
	}
	state := fmt.Sprintf(`{"port":%d,"token":"%s"}`, port, strings.Repeat("a", 64))
	path := filepath.Join(dir, filepath.FromSlash(bridge.BepInEx.StateFile))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestHostValuesAnswerEmptyUnlessTheBridgeAnswers(t *testing.T) {
	const rows = `ok [{"mod":"g","section":"S","key":"A","value":"1","inUse":true},{"mod":"g","section":"S","key":"B","value":"2","inUse":true},{"mod":"x","section":"S","key":"C","value":"3","inUse":true}]`
	for _, c := range []struct {
		name    string
		running func(string, string) bool
		file    string
		reply   string
		want    int
		wantErr bool
	}{
		{"no Running func", nil, "g.cfg", rows, 0, false},
		{"game not running", func(string, string) bool { return false }, "g.cfg", rows, 0, false},
		{"not a cfg", func(string, string) bool { return true }, "g.json", rows, 0, false},
		{"bridge not ready", func(string, string) bool { return true }, "g.cfg", "", 0, false},
		{"nothing listens", func(string, string) bool { return true }, "g.cfg", "gone", 0, false},
		{"bridge rejects", func(string, string) bool { return true }, "g.cfg", "error: old bridge", 0, false},
		{"unreadable reply", func(string, string) bool { return true }, "g.cfg", "ok {", 0, true},
		{"rows of the file's plugin", func(string, string) bool { return true }, "g.cfg", rows, 2, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			fakeBridge(t, dir, c.reply)
			s := &Service{Profiles: fakeProfiles{dir: dir}, Running: c.running}
			got, err := s.HostValues("g", "p", c.file)
			if (err != nil) != c.wantErr || (err == nil && (got == nil || len(got) != c.want)) {
				t.Fatalf("got %v, %v", got, err)
			}
		})
	}
}

func TestSaveHostValuesKeepsTheRowsAndNothingForNone(t *testing.T) {
	testfs.DataHome(t)
	dir := t.TempDir()
	fakeBridge(t, dir, `ok [{"mod":"g","section":"S","key":"A","value":"1","inUse":true},{"mod":"g","section":"S","key":"B","value":"2","inUse":true}]`)
	s := &Service{Profiles: fakeProfiles{dir: dir}, Running: func(string, string) bool { return true }}
	if n, err := s.SaveHostValues("lc", "p", "m", "g.cfg"); err != nil || n != 2 {
		t.Fatalf("saved %d, %v", n, err)
	}
	fresh := &Service{}
	fresh.loadKept()
	if got := fresh.hostKept[hostKey{"lc", "p", "m", "g.cfg"}]; len(got) != 2 || got[1] != (edit{"S", "B", "2"}) {
		t.Fatalf("kept %+v", got)
	}

	none := &Service{Profiles: fakeProfiles{dir: t.TempDir()}, Running: func(string, string) bool { return false }}
	if n, err := none.SaveHostValues("lc", "q", "m", "g.cfg"); err != nil || n != 0 || len(none.hostKept) != 0 {
		t.Fatalf("saved %d, %v, kept %v", n, err, none.hostKept)
	}
}

func TestWriteMelonSetsAValueAndMelonFileNeedsAMatchingCategory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "UserData", melonName)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(melonSample), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Service{Profiles: fakeProfiles{dir: dir}}
	if err := s.writeMelon("g", "p", []edit{{"BetterHands", "Count", "9"}}); err != nil {
		t.Fatal(err)
	}
	if b, _ := safefs.ReadFile(path); !strings.Contains(string(b), "Count = 9") {
		t.Fatalf("file:\n%s", b)
	}
	err := s.writeMelon("g", "p", []edit{{"BetterHands", "Count", "{ x = "}})
	if usererr.KindOf(err) != usererr.Invalid {
		t.Fatalf("invalid value: %v", err)
	}
	if got := s.melonFile("g", "p", "m", []string{"thunderstore:Ada-Better_Hands"}); len(got) != 1 {
		t.Fatalf("matching category: %v", got)
	}
	if got := s.melonFile("g", "p", "m", []string{"thunderstore:Ada-Other"}); got != nil {
		t.Fatalf("no matching category: %v", got)
	}
}
