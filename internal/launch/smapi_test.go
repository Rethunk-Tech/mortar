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
