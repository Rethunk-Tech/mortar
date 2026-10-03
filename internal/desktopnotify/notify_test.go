package desktopnotify

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

type fakeNotifier struct {
	n     int
	title string
}

func (f *fakeNotifier) SendNotification(opts notifications.NotificationOptions) error {
	f.n++
	f.title = opts.Title
	return nil
}

func TestSendIfPrefOffSkips(t *testing.T) {
	f := &fakeNotifier{}
	Setup(f, func() string { return "" }, nil)
	t.Cleanup(func() { Setup(nil, nil, nil) })
	SendIf(false, "Download finished", "mod", nil)
	if f.n != 0 {
		t.Fatalf("pref off sent %d", f.n)
	}
	SendIf(true, "Download finished", "mod", nil)
	if f.n != 1 || f.title != "Download finished" {
		t.Fatalf("pref on sent %d %q", f.n, f.title)
	}
}
