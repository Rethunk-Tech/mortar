package nexussvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/opener"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

// inPageHost is a server build's host: it refuses to open a browser and records what is emitted.
type inPageHost struct {
	winhost.Host
	emitted *[]any
}

func (inPageHost) OpenURL(string) error { return opener.ErrOpenInPage }

func (h inPageHost) Emit(_ string, data any) { *h.emitted = append(*h.emitted, data) }

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

func TestSignInInAServerBuildHandsTheAddressToThePage(t *testing.T) {
	var emitted []any
	s := &Service{App: inPageHost{emitted: &emitted}}
	const addr = "https://users.nexusmods.com/oauth/authorize?state=x"
	if err := s.openBrowser(addr); err != nil {
		t.Fatalf("the sign-in keeps waiting for the page's tab: %v", err)
	}
	if len(emitted) != 1 || emitted[0] != (SSOState{State: SSOOpenInPage, URL: addr}) {
		t.Fatalf("emitted %#v", emitted)
	}
}
