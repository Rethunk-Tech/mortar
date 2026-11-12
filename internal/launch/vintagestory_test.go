package launch

import "testing"

func TestParseVintageStoryMapsTheGamesLevelWords(t *testing.T) {
	for _, tc := range []struct {
		line  string
		level Level
	}{
		{"1.1.2026 10:00:00 [Fatal] dead", Alert},
		{"1.1.2026 10:00:00 [Error] bad", Error},
		{"1.1.2026 10:00:00 [Warning] hmm", Warn},
		{"1.1.2026 10:00:00 [Debug] trace", Debug},
		{"1.1.2026 10:00:00 [VerboseDebug] trace", Trace},
		{"1.1.2026 10:00:00 [Notification] hi", Info},
		{"1.1.2026 10:00:00 [StoryEvent] hi", Info},
	} {
		e, ok := parseVintageStory(tc.line)
		if !ok || e.Level != tc.level || e.Time != "10:00:00" {
			t.Errorf("%q gave %+v, %v", tc.line, e, ok)
		}
	}
	if _, ok := parseVintageStory("   at Something.Start()"); ok {
		t.Fatal("a stack trace line parsed as a log line")
	}
}
