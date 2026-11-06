package nexussvc

import (
	"context"
	"errors"
	"testing"

	"github.com/Rethunk-Tech/go-nexusmods/sso"
	"github.com/Rethunk-Tech/mortar/internal/opener"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
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

// A second sign-in is refused while one waits, CancelSSO ends the first with the cancelled state, and the
// refusal clears so the next sign-in can start. The callback listener is real, on a free loopback port; no
// browser is opened.
func TestSignInRefusesASecondAndCancelEndsTheFirst(t *testing.T) {
	var emitted []any
	opened := make(chan struct{})
	s := &Service{App: inPageHost{emitted: &emitted}}
	s.oauth = sso.OAuth{ClientID: "test", Addr: "127.0.0.1:0", OpenBrowser: func(string) error {
		close(opened)
		return nil
	}}
	first := make(chan error, 1)
	go func() {
		_, err := s.StartSSO(t.Context())
		first <- err
	}()
	<-opened

	if _, err := s.StartSSO(t.Context()); usererr.KindOf(err) != usererr.Busy {
		t.Fatalf("a second sign-in while one waits: kind %v, err %v", usererr.KindOf(err), err)
	}
	s.CancelSSO()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("the cancelled sign-in ended with %v", err)
	}
	last, ok := emitted[len(emitted)-1].(SSOState)
	if !ok || last.State != SSOFailed || !last.Cancelled {
		t.Fatalf("last state %#v, want failed and cancelled", emitted[len(emitted)-1])
	}
	s.CancelSSO()
	if s.run.cancel != nil {
		t.Fatal("the finished sign-in left its cancel behind")
	}
}
