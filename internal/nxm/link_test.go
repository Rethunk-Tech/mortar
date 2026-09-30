package nxm

import (
	"errors"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	const ok = "nxm://stardewvalley/mods/1915/files/77?key=abc_-x&expires=1000600&user_id=42"
	got, err := Parse(ok, 42, now)
	if err != nil || got != (Link{ModID: 1915, FileID: 77, Key: "abc_-x", Expires: 1000600, UserID: 42}) {
		t.Fatalf("valid link: %+v, %v", got, err)
	}
	cases := map[string]struct {
		raw    string
		user   int
		reason string
	}{
		"expired":      {"nxm://stardewvalley/mods/1/files/2?key=k&expires=1000000&user_id=42", 42, ReasonExpired},
		"wrong user":   {"nxm://stardewvalley/mods/1/files/2?key=k&expires=1000600&user_id=7", 42, ReasonUser},
		"signed out":   {ok, 0, ReasonSignedIn},
		"wrong game":   {"nxm://skyrim/mods/1/files/2?key=k&expires=1000600&user_id=42", 42, ReasonGame},
		"other scheme": {"https://stardewvalley/mods/1/files/2?key=k&expires=1000600&user_id=42", 42, ReasonForm},
		"mod not num":  {"nxm://stardewvalley/mods/x/files/2?key=k&expires=1000600&user_id=42", 42, ReasonForm},
		"file signed":  {"nxm://stardewvalley/mods/1/files/-2?key=k&expires=1000600&user_id=42", 42, ReasonForm},
		"zero id":      {"nxm://stardewvalley/mods/0/files/2?key=k&expires=1000600&user_id=42", 42, ReasonForm},
		"leading zero": {"nxm://stardewvalley/mods/01/files/2?key=k&expires=1000600&user_id=42", 42, ReasonForm},
		"path junk":    {"nxm://stardewvalley/mods/1/files/2/extra?key=k&expires=1000600&user_id=42", 42, ReasonForm},
		"oauth":        {"nxm://oauth/callback?code=1", 42, ReasonGame},
		"extra query":  {ok + "&admin=1", 42, ReasonForm},
		"repeated key": {"nxm://stardewvalley/mods/1/files/2?key=a&key=b&user_id=42", 42, ReasonForm},
		"no key":       {"nxm://stardewvalley/mods/1/files/2?key=&expires=1000600&user_id=42", 42, ReasonForm},
		"fragment":     {ok + "#x", 42, ReasonForm},
		"userinfo":     {"nxm://me@stardewvalley/mods/1/files/2?key=k&expires=1000600&user_id=42", 42, ReasonForm},
	}
	for name, c := range cases {
		_, err := Parse(c.raw, c.user, now)
		var re *RejectError
		if !errors.As(err, &re) || re.Reason != c.reason {
			t.Errorf("%s: got %v, want reason %s", name, err, c.reason)
		}
	}
}

func TestIsLink(t *testing.T) {
	if !IsLink("NXM://a") || IsLink("mortar://a") || IsLink("--flag") {
		t.Error("IsLink")
	}
}
