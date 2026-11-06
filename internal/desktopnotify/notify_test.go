package desktopnotify

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
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
