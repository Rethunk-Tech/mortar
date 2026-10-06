package launch

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

func TestParserReadsBepInExLevelsSourcesAndWrappedLines(t *testing.T) {
	var p Parser
	cases := []struct {
		line string
		want Entry
	}{
		{"[Message:   BepInEx] BepInEx 5.4.23.5 - Lethal Company", Entry{Level: Info, Mod: "BepInEx", Message: "BepInEx 5.4.23.5 - Lethal Company"}},
		{"[Info   :   BepInEx] Loading [CullFactory 2.0.11]", Entry{Level: Info, Mod: "BepInEx", Message: "Loading [CullFactory 2.0.11]"}},
		{"[Info   :Lethal Company Input Utils] Registered 3 keybinds", Entry{Level: Info, Mod: "Lethal Company Input Utils", Message: "Registered 3 keybinds"}},
		{"[Debug  :  HarmonyX] Patching void Foo::Bar()", Entry{Level: Debug, Mod: "HarmonyX", Message: "Patching void Foo::Bar()"}},
		{"[Warning:CullFactory] Portal has no renderer", Entry{Level: Warn, Mod: "CullFactory", Message: "Portal has no renderer"}},
		{"[Error  : Unity Log] NullReferenceException: Object reference not set", Entry{Level: Error, Mod: "Unity Log", Message: "NullReferenceException: Object reference not set"}},
		{"  at Foo.Bar () [0x00000] in <abc>:0", Entry{Level: Error, Mod: "Unity Log", Message: "  at Foo.Bar () [0x00000] in <abc>:0", Cont: true}},
		{"[Fatal  :   BepInEx] Error occurred starting the game", Entry{Level: Alert, Mod: "BepInEx", Message: "Error occurred starting the game"}},
	}
	for _, c := range cases {
		got, shown := p.Parse(c.line)
		if !shown || got != c.want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", c.line, got, shown, c.want)
		}
	}
}

// The run log is formatted from the console's entries, and BepInEx's analyzers read it back as LogOutput.log.
func TestFormatLogWritesBepInExLinesBackInItsFormat(t *testing.T) {
	log := "[Error  :   BepInEx] Could not load [A 1.0.0] because it has missing dependencies: B\n  at X.Y ()\n[Warning:CullFactory] Portal has no renderer"
	want := "[Error  :   BepInEx] Could not load [A 1.0.0] because it has missing dependencies: B\n  at X.Y ()\n[Warning:CullFactory] Portal has no renderer"
	if got := FormatLog(ParseLog(log)); got != want {
		t.Fatalf("FormatLog = %q\nwant %q", got, want)
	}
}

// fakeBepInEx writes LogOutput.log the way BepInEx does on a launch: it truncates the last run's log, then flushes its
// buffer every so often, which can end mid-line.
func fakeBepInEx(t *testing.T, path string, chunks []string) {
	t.Helper()
	f, err := fsx.Create(path)
	if err != nil {
		t.Error(err)
		return
	}
	defer func() { _ = f.Close() }()
	for _, c := range chunks {
		time.Sleep(10 * time.Millisecond)
		if _, err := f.WriteString(c); err != nil {
			t.Error(err)
			return
		}
	}
}

func TestRunFollowsBepInExLogIntoParsedEntries(t *testing.T) {
	log := filepath.Join(t.TempDir(), "LogOutput.log")
	if err := os.WriteFile(log, []byte("[Info   :   BepInEx] the last run\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(log, old, old); err != nil {
		t.Fatal(err)
	}
	chunks := []string{
		"[Message:   BepInEx] BepInEx 5.4.23.5 - Lethal Company\r\n[Info   :   BepInEx] Loading [CullFa",
		"ctory 2.0.11]\r\n[Info   :Lethal Company Input Utils] Registered 3 keybinds\r\n",
		"[Error  : Unity Log] NullReferenceException\r\n  at Foo.Bar ()\r\n",
	}
	done := make(chan struct{})
	run := func(string, string, ...string) (<-chan error, error) {
		go func() {
			defer close(done)
			fakeBepInEx(t, log, chunks)
		}()
		return make(chan error), nil
	}
	var mu sync.Mutex
	var p Parser
	var got []Entry
	onLines := func(lines []string) {
		mu.Lock()
		defer mu.Unlock()
		for _, l := range lines {
			if e, shown := p.Parse(l); shown {
				got = append(got, e)
			}
		}
	}
	if err := Run(t.Context(), run, Command{Name: "steam", LogFile: log, Relay: true}, fast, onLines); err != nil {
		t.Fatal(err)
	}
	<-done
	want := []Entry{
		{Level: Info, Mod: "BepInEx", Message: "BepInEx 5.4.23.5 - Lethal Company"},
		{Level: Info, Mod: "BepInEx", Message: "Loading [CullFactory 2.0.11]"},
		{Level: Info, Mod: "Lethal Company Input Utils", Message: "Registered 3 keybinds"},
		{Level: Error, Mod: "Unity Log", Message: "NullReferenceException"},
		{Level: Error, Mod: "Unity Log", Message: "  at Foo.Bar ()", Cont: true},
	}
	snapshot := func() []Entry {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(got)
	}
	deadline := time.Now().Add(2 * time.Second)
	for !slices.Equal(snapshot(), want) && time.Now().Before(deadline) {
		time.Sleep(fast.Poll)
	}
	if s := snapshot(); !slices.Equal(s, want) {
		t.Fatalf("entries = %+v\nwant %+v", s, want)
	}
}
