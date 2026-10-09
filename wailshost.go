package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/Rethunk-Tech/mortar/internal/desktopnotify"
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

func (h wailsHost) OpenURL(url string) error { return h.app.Browser.OpenURL(url) }

// wailsNotifier adapts the Wails notifications service to desktopnotify.Notifier.
type wailsNotifier struct {
	svc *notifications.NotificationService
}

func (n wailsNotifier) Notify(c desktopnotify.Notice) error {
	opts := notifications.NotificationOptions{ID: c.ID, Title: c.Title, Body: c.Body, Data: c.Data}
	if c.Icon != "" {
		opts.Attachments = []notifications.NotificationAttachment{{ID: "icon", Path: c.Icon, Type: "appLogoOverride"}}
	}
	return n.svc.SendNotification(opts)
}
