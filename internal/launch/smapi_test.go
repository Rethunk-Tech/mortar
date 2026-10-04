package launch

import "testing"

func TestParserLevelsModsAndContinuations(t *testing.T) {
	var p Parser
	cases := []struct {
		line string
		want Entry
	}{
		{"[19:43:46 TRACE SMAPI] loading", Entry{Time: "19:43:46", Level: Trace, Mod: "SMAPI", Message: "loading"}},
		{"[19:43:46 DEBUG SMAPI] d", Entry{Time: "19:43:46", Level: Debug, Mod: "SMAPI", Message: "d"}},
		{"[19:43:46 INFO  Love of Cooking] Loaded 42 mods", Entry{Time: "19:43:46", Level: Info, Mod: "Love of Cooking", Message: "Loaded 42 mods"}},
		{"[19:43:49 WARN  SMAPI] obsolete", Entry{Time: "19:43:49", Level: Warn, Mod: "SMAPI", Message: "obsolete"}},
		{"[19:43:50 ERROR SMAPI] Failed:", Entry{Time: "19:43:50", Level: Error, Mod: "SMAPI", Message: "Failed:"}},
		{"   at Foo.Bar()", Entry{Time: "19:43:50", Level: Error, Mod: "SMAPI", Message: "   at Foo.Bar()", Cont: true}},
		{"", Entry{Time: "19:43:50", Level: Error, Mod: "SMAPI", Message: "", Cont: true}},
		{"[19:44:02 ALERT SMAPI] alert!", Entry{Time: "19:44:02", Level: Alert, Mod: "SMAPI", Message: "alert!"}},
		{"[19:44:03 INFO ] no mod", Entry{Time: "19:44:03", Level: Info, Message: "no mod"}},
		{"[19:44:04 INFO SMAPI]", Entry{Time: "19:44:04", Level: Info, Mod: "SMAPI"}},
		{"[19:44:05 LOUD SMAPI] not a level", Entry{Time: "19:44:04", Level: Info, Mod: "SMAPI", Message: "[19:44:05 LOUD SMAPI] not a level", Cont: true}},
	}
	for _, c := range cases {
		if got, shown := p.Parse(c.line); !shown || got != c.want {
			t.Errorf("Parse(%q) = %+v, want %+v", c.line, got, c.want)
		}
	}
}

func TestParserHeaderlessFirstLine(t *testing.T) {
	var p Parser
	got, _ := p.Parse("SMAPI 4.5.2 with Stardew Valley 1.6.15")
	if got.Level != Info || got.Cont || got.Mod != "" || got.Message != "SMAPI 4.5.2 with Stardew Valley 1.6.15" {
		t.Fatalf("got %+v", got)
	}
}

func TestBufferKeepsNewestMaxLines(t *testing.T) {
	var b Buffer
	total := MaxLines*2 + 7
	for i := 1; i <= total; i++ {
		b.Add(Entry{Seq: int64(i)})
		if i%1000 == 0 && len(b.Lines()) > MaxLines {
			t.Fatalf("after %d adds Lines() has %d entries", i, len(b.Lines()))
		}
	}
	lines := b.Lines()
	if len(lines) != MaxLines || lines[0].Seq != int64(total-MaxLines+1) || lines[len(lines)-1].Seq != int64(total) {
		t.Fatalf("got %d lines, %d..%d", len(lines), lines[0].Seq, lines[len(lines)-1].Seq)
	}
	lines[0].Seq = -1
	if b.Lines()[0].Seq == -1 {
		t.Fatal("Lines must return a copy")
	}
}

func TestBufferKeepsSMAPIStartupAndTail(t *testing.T) {
	var b Buffer
	b.Add(Entry{Seq: 1, Mod: "SMAPI", Message: "SMAPI startup"})
	b.Add(Entry{Seq: 2, Mod: "SMAPI", Message: "Loaded 2 mods"})
	for i := 3; i <= MaxLines+10; i++ {
		b.Add(Entry{Seq: int64(i), Mod: "game", Message: "tail"})
	}
	lines := b.Lines()
	if len(lines) != MaxLines || lines[0].Seq != 1 || lines[1].Message != "Loaded 2 mods" ||
		lines[2].Message != omittedStartupLog || lines[len(lines)-1].Seq != MaxLines+10 {
		t.Fatalf("startup and tail were not retained: len=%d first=%+v last=%+v", len(lines), lines[:min(3, len(lines))], lines[len(lines)-1])
	}
}

func TestParserHidesSuppressedMessageAndItsContinuations(t *testing.T) {
	var p Parser
	lines := []struct {
		line  string
		shown bool
	}{
		{"[19:43:46 INFO  SMAPI] Writing to the terminal is disabled because the --no-terminal argument was received. This usually means launching the terminal failed.", false},
		{"   continued", false},
		{"[19:43:47 INFO  SMAPI] Loaded 3 mods", true},
		{"   continued", true},
		{"[04:00:28 ERROR game] Error initializing the Galaxy API.", false},
		{"TypeInitializationException: The type initializer for 'Galaxy.Api.GalaxyInstancePINVOKE' threw an exception.", false},
		{"[04:00:28 TRACE game] Signing into GalaxySDK", true},
		{"[04:00:28 ERROR game] Galaxy SignInSteam failed with an exception:", false},
		{"   at Galaxy.Api.GalaxyInstance.User()", false},
		{"[04:00:29 ERROR SomeMod] A real error", true},
	}
	for _, c := range lines {
		if _, shown := p.Parse(c.line); shown != c.shown {
			t.Errorf("Parse(%q) shown = %v, want %v", c.line, shown, c.shown)
		}
	}
}

func TestModsPathReadsSMAPIsIntroLine(t *testing.T) {
	cases := map[string]struct {
		log  string
		want string
		ok   bool
	}{
		"home":     {"SMAPI 4.5.2 with Stardew Valley 1.6.15 on Unix\n[05:00:01 INFO  SMAPI] Mods go here: ~/p/mods\r\n", "~/p/mods", true},
		"absolute": {"[05:00:01 INFO  SMAPI] Mods go here: /srv/p/mods\n", "/srv/p/mods", true},
		"windows":  {"[05:00:01 INFO  SMAPI] Mods go here: ~\\AppData\\Roaming\\p\\mods\r\n", "~\\AppData\\Roaming\\p\\mods", true},
		"a mod":    {"[05:00:01 INFO  Other] Mods go here: /x\n", "", false},
		"missing":  {"[05:00:01 INFO  SMAPI] hello\n", "", false},
	}
	for name, c := range cases {
		if got, ok := ModsPath(c.log); got != c.want || ok != c.ok {
			t.Errorf("%s: got %q, %v", name, got, ok)
		}
	}
}

func TestLogOwnedByExpandsHome(t *testing.T) {
	log := "[05:00:01 INFO  SMAPI] Mods go here: ~/p/a/mods/\n"
	if !LogOwnedBy(log, "/home/u", "/home/u/p/a/mods") {
		t.Error("profile a wrote the log")
	}
	if LogOwnedBy(log, "/home/u", "/home/u/p/b/mods") || LogOwnedBy(log, "/home/v", "/home/u/p/a/mods") {
		t.Error("only profile a under /home/u wrote the log")
	}
	if !LogOwnedBy("[05:00:01 INFO  SMAPI] Mods go here: /srv/a\n", "/home/u", "/srv/a") {
		t.Error("an absolute path compares as written")
	}
	if LogOwnedBy("[05:00:01 INFO  SMAPI] hello\n", "/home/u", "/home/u/p/a/mods") {
		t.Error("a log without the line belongs to no profile")
	}
}
