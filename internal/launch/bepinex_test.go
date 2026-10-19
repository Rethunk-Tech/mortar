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

// mattymatty-AsyncLoggers prefixes a line with its time; the entry keeps it as Time and the run log writes it back.
func TestParserSplitsALoggerModsStampFromBepInExLines(t *testing.T) {
	log := "[Info   :   BepInEx] BepInEx 5.4.23.5\n[14:35:39.1344735] [Info   :SlendermanMod] Registered spawn rate for level type All to 2"
	got := ParseLog(log)
	if e := got[1]; e.Time != "14:35:39.1344735" || e.Level != Info || e.Mod != "SlendermanMod" || e.Message != "Registered spawn rate for level type All to 2" {
		t.Fatalf("stamped entry = %+v", e)
	}
	if e := got[0]; e.Time != "" || e.Mod != "BepInEx" {
		t.Fatalf("plain entry = %+v", e)
	}
	if out := FormatLog(got); out != log {
		t.Fatalf("FormatLog = %q", out)
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

func TestParserReadsVintageStoryLevels(t *testing.T) {
	var p Parser
	cases := []struct {
		line string
		want Entry
	}{
		{"16.10.2026 05:25:45 [Notification] Game Version: v1.22.7 (Stable)", Entry{Time: "05:25:45", Level: Info, Message: "Game Version: v1.22.7 (Stable)"}},
		{"16.10.2026 05:25:46 [VerboseDebug] Loaded 3 mods", Entry{Time: "05:25:46", Level: Trace, Message: "Loaded 3 mods"}},
		{"6.1.2026 05:25:47 [Error] Exception: boom", Entry{Time: "05:25:47", Level: Error, Message: "Exception: boom"}},
		{"   at Vintagestory.Foo()", Entry{Time: "05:25:47", Level: Error, Message: "   at Vintagestory.Foo()", Cont: true}},
	}
	for _, c := range cases {
		if got, shown := p.Parse(c.line); !shown || got != c.want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", c.line, got, shown, c.want)
		}
	}
}

func TestThunderpackLogLinesParse(t *testing.T) {
	var p Parser
	for line, want := range map[string]Entry{
		"WARN - [♥] Pattern did not match":                                {Level: Warn, Mod: "Lovely", Message: "Pattern did not match"},
		"INFO - [G] Game says hi":                                         {Level: Info, Mod: "Game", Message: "Game says hi"},
		"2024-05-01 12:00:01.123 +00:00 [WRN] Duplicate mod ID: a":        {Time: "12:00:01", Level: Warn, Mod: "GDWeave", Message: "Duplicate mod ID: a"},
		"[12:3:45.0000001][ERROR/lua_module.cpp:156] X failed to load: e": {Time: "12:3:45", Level: Error, Mod: "lua_module.cpp", Message: "X failed to load: e"},
		"[2024-05-01] [12:00:00] [NORTHSTAR] [warning] hello":             {Time: "12:00:00", Level: Warn, Mod: "NORTHSTAR", Message: "hello"},
	} {
		if got, _ := p.Parse(line); got != want {
			t.Errorf("%q: %+v, want %+v", line, got, want)
		}
	}
}
