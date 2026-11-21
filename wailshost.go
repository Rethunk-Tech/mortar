package main

import (
	"context"
	"log/slog"
	"os/exec"
	"runtime"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-desktopenv/hostenv"
	"github.com/Rethunk-Tech/go-desktopenv/portal"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/Rethunk-Tech/mortar/internal/desktopnotify"
	"github.com/Rethunk-Tech/mortar/internal/opener"
	"github.com/Rethunk-Tech/mortar/internal/winhost"
)

// wailsHost adapts the running Wails application to the services' winhost.Host.
type wailsHost struct{ app *application.App }

func (h wailsHost) Emit(name string, data any) { h.app.Event.Emit(name, data) }

func (h wailsHost) open(d winhost.Dialog) *application.OpenFileDialogStruct {
	o := h.app.Dialog.OpenFile().SetTitle(d.Title).CanChooseDirectories(d.Folder).CanChooseFiles(!d.Folder)
	for _, f := range d.Filters {
		o.AddFilter(f.Name, f.Pattern)
	}
	o.AttachToWindow(h.app.Window.Current())
	return o
}

func (h wailsHost) OpenFile(d winhost.Dialog) (string, error) {
	return h.open(d).PromptForSingleSelection()
}

func (h wailsHost) OpenFiles(d winhost.Dialog) ([]string, error) {
	return h.open(d).PromptForMultipleSelection()
}

func (h wailsHost) SaveFile(d winhost.Dialog) (string, error) {
	s := h.app.Dialog.SaveFile()
	s.SetOptions(&application.SaveFileDialogOptions{Title: d.Title, Filename: d.Filename})
	for _, f := range d.Filters {
		s.AddFilter(f.Name, f.Pattern)
	}
	s.AttachToWindow(h.app.Window.Current())
	return s.PromptForSingleSelection()
}

func (h wailsHost) ClipboardText() (string, bool) { return h.app.Clipboard.Text() }

// OpenURL refuses in a server build like the opener service does, so nothing reaches the host's own browser.
func (h wailsHost) OpenURL(url string) error {
	return openSystem(h.app, url)
}

// openSystem hands an address to the system handler. On Linux it runs xdg-open as Wails does, but with the host's
// environment: inside the AppImage the inherited LD_LIBRARY_PATH would load the bundle's GTK, GLib and SQLite into the
// browser xdg-open starts, which then fails to open (LibreWolf on KDE does).
func openSystem(app *application.App, url string) error {
	if serverBuild {
		return opener.ErrOpenInPage
	}
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return app.Browser.OpenURL(url)
	}
	cmd := exec.CommandContext(context.Background(), "xdg-open", url) //nolint:gosec // G204: every caller passes a checked address
	cmd.Env = hostenv.Environ()
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// wailsNotifier adapts the Wails notifications service to desktopnotify.Notifier.
type wailsNotifier struct {
	svc *notifications.NotificationService
}

// Notify uses the Notification portal inside Flatpak, where the notification service is not reachable; its clicks
// return through desktopnotify.RouteClicks. Everywhere else the Wails service shows the notification.
func (n wailsNotifier) Notify(c desktopnotify.Notice) error {
	if flatpak.Inside() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := portal.AddNotification(ctx, c.Portal())
		if err == nil {
			return nil
		}
		slog.Warn("portal notification", "err", err)
	}
	opts := notifications.NotificationOptions{ID: c.ID, Title: c.Title, Body: c.Body, Data: c.Data}
	if c.Icon != "" {
		opts.Attachments = []notifications.NotificationAttachment{{ID: "icon", Path: c.Icon, Type: "appLogoOverride"}}
	}
	return n.svc.SendNotification(opts)
}
