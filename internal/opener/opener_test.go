package opener

import (
	"strings"
	"testing"
)

func TestWebOpensOnlyHTTPAndHTTPSWithAHost(t *testing.T) {
	for raw, ok := range map[string]bool{
		"https://www.nexusmods.com/stardewvalley/mods/1915?tab=files": true,
		"http://example.com/page#frag":                                true,
		"HTTPS://Example.com/":                                        true,
		"":                                                            false,
		"javascript:alert(1)":                                         false,
		"file:///etc/passwd":                                          false,
		"steam://run/1":                                               false,
		"nxm://stardewvalley/mods/1/files/2?key=k":                    false,
		"ms-msdt:/id PCWDiagnostic":                                   false,
		`\\server\share\x.exe`:                                        false,
		"//example.com/x":                                             false,
		"https:///nohost":                                             false,
		"https://user:pw@example.com/":                                false,
		"https://example.com/a b":                                     false,
		"https://example.com/\n":                                      false,
		"example.com/page":                                            false,
		"https://example.com/" + strings.Repeat("a", maxURL):          false,
	} {
		got, err := Web(raw)
		if (err == nil) != ok || (ok && got != raw) {
			t.Errorf("Web(%q) = %q, %v; want ok=%v", raw, got, err, ok)
		}
	}
}

func TestServiceRefusesWithoutOpeningAndOpensWebAddresses(t *testing.T) {
	var opened []string
	s := &Service{Open: func(u string) error { opened = append(opened, u); return nil }}

	if err := s.OpenWeb("steam://run/1"); err == nil {
		t.Fatal("a steam link was opened")
	}
	if err := s.OpenWeb("https://example.com/"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "https://example.com/" {
		t.Fatalf("opened = %v", opened)
	}
}

func TestSteamValidateTakesOnlyDigits(t *testing.T) {
	var opened []string
	s := &Service{Open: func(u string) error { opened = append(opened, u); return nil }}

	for _, bad := range []string{"", "12a", "../1", "1/../2", "1 2", strings.Repeat("1", 13)} {
		if err := s.OpenSteamValidate(bad); err == nil {
			t.Errorf("app id %q was accepted", bad)
		}
	}
	if err := s.OpenSteamValidate("413150"); err != nil || len(opened) != 1 || opened[0] != "steam://validate/413150" {
		t.Fatalf("opened = %v, %v", opened, err)
	}
}

func FuzzWeb(f *testing.F) {
	for _, s := range []string{"https://example.com/", "javascript:1", "file:///x", "https://u@h/", "\x00", "https://[::1]/"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		if got, err := Web(raw); err == nil && !strings.HasPrefix(strings.ToLower(got), "http") {
			t.Fatalf("Web accepted %q", raw)
		}
	})
}
