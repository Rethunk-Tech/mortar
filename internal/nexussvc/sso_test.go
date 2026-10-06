package nexussvc

import "testing"

func TestSignInOpensOnlyAWebAddress(t *testing.T) {
	s := &Service{}

	for _, bad := range []string{"steam://run/1", "file:///etc/passwd", "javascript:alert(1)"} {
		if err := s.openBrowser(bad); err == nil || err.Error() == "no browser to open" {
			t.Errorf("openBrowser(%q) = %v, want the opener's refusal", bad, err)
		}
	}
	if err := s.openBrowser("https://users.nexusmods.com/oauth"); err == nil || err.Error() != "no browser to open" {
		t.Fatalf("a web address passes the guard and only lacks a browser here: %v", err)
	}
}
