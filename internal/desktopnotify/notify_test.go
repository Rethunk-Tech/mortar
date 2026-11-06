package desktopnotify

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-desktopenv/portal"
)

type fakeNotifier struct {
	n     int
	title string
}

func (f *fakeNotifier) Notify(opts Notice) error {
	f.n++
	f.title = opts.Title
	return nil
}

func TestSendIfPrefOffSkips(t *testing.T) {
	f := &fakeNotifier{}
	Setup(f, func() string { return "" }, nil)
	t.Cleanup(func() { Setup(nil, nil, nil) })
	SendIf(false, KindDownload, "Download finished", "mod", nil)
	if f.n != 0 {
		t.Fatalf("pref off sent %d", f.n)
	}
	SendIf(true, KindDownload, "Download finished", "mod", nil)
	if f.n != 1 || f.title != "Download finished" {
		t.Fatalf("pref on sent %d %q", f.n, f.title)
	}
}

func TestNoticePortalCarriesIconBytesAndDropsAMissingIcon(t *testing.T) {
	icon := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(icon, []byte{1, 2}, 0o600); err != nil {
		t.Fatal(err)
	}
	got := Notice{ID: "a", Title: "T", Body: "B", Icon: icon}.Portal()
	if got.ID != "a" || got.Title != "T" || got.Body != "B" || !bytes.Equal(got.IconPNG, []byte{1, 2}) {
		t.Fatalf("portal notice %+v", got)
	}
	if got := (Notice{ID: "a", Title: "T", Icon: filepath.Join(t.TempDir(), "none.png")}).Portal(); got.IconPNG != nil {
		t.Fatalf("missing icon kept: %v", got.IconPNG)
	}
}

func TestPortalDefaultActionCarriesData(t *testing.T) {
	got := Notice{ID: "run-end-p-1", Title: "T", Data: map[string]any{"game": "g", "profile": "p"}}.Portal()
	if got.DefaultAction != "open" || got.DefaultActionTarget != `{"game":"g","profile":"p"}` {
		t.Fatalf("default action %q target %q", got.DefaultAction, got.DefaultActionTarget)
	}
	if bare := (Notice{ID: "a", Title: "T"}).Portal(); bare.DefaultAction != "open" || bare.DefaultActionTarget != "" {
		t.Fatalf("bare notice %+v", bare)
	}
}

func TestRouteActionCallsTheHandler(t *testing.T) {
	var id string
	var data map[string]any
	h := func(i string, d map[string]any) { id, data = i, d }
	routeAction(portal.NotificationAction{ID: "x", Action: "other", Target: `{"a":"b"}`}, h)
	if id != "" {
		t.Fatal("foreign action routed")
	}
	routeAction(portal.NotificationAction{ID: "run-end-p-1", Action: "open", Target: `{"game":"g","profile":"p"}`}, h)
	if id != "run-end-p-1" || data["game"] != "g" || data["profile"] != "p" {
		t.Fatalf("routed %q %v", id, data)
	}
	routeAction(portal.NotificationAction{ID: "n", Action: "open"}, h)
	if id != "n" || data != nil {
		t.Fatalf("bare click %q %v", id, data)
	}
}
