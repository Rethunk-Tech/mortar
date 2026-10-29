package portal

import (
	"regexp"
	"testing"
)

func TestDesktopEntryTextEscapesWhatTheFormatReads(t *testing.T) {
	t.Parallel()
	got := desktopEntryText(`Mods\new` + "\nline\rend")
	if want := `Mods\\new line end`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// A portal handle token is a D-Bus object path element, and each call needs its own so two calls never share one
// Request path.
func TestNewTokenIsUniqueAndAPathElement(t *testing.T) {
	t.Parallel()
	element := regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	a, b := newToken("launcher"), newToken("launcher")
	if a == b || !element.MatchString(a) || !element.MatchString(b) {
		t.Fatalf("tokens %q, %q", a, b)
	}
}
