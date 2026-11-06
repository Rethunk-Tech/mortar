package main

import (
	"context"
	"log"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
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
	if serverBuild {
		return opener.ErrOpenInPage
	}
	return h.app.Browser.OpenURL(url)
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
		log.Printf("portal notification: %v", err)
	}
	opts := notifications.NotificationOptions{ID: c.ID, Title: c.Title, Body: c.Body, Data: c.Data}
	if c.Icon != "" {
		opts.Attachments = []notifications.NotificationAttachment{{ID: "icon", Path: c.Icon, Type: "appLogoOverride"}}
	}
	return n.svc.SendNotification(opts)
}
