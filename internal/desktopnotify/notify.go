// Package desktopnotify sends OS desktop notifications with the app icon.
package desktopnotify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/portal"
	"github.com/Rethunk-Tech/go-safefs"
)

// Kind says what a notification is about; it decides the id the OS hands back when the user activates it.
type Kind string

const (
	KindDownload   Kind = "download"
	KindModUpdates Kind = "mod-updates"
	KindRunEnd     Kind = "run-end"
	KindOther      Kind = "notice"
)

// Notice is one notification; Icon is the path of the app logo to show beside it, or "".
type Notice struct {
	ID, Title, Body, Icon string
	Data                  map[string]any
}

// Notifier delivers a Notice to the OS. The Wails notifications service sits behind it, because importing that
// service links GTK through cgo into every test binary that reaches this package.
type Notifier interface {
	Notify(Notice) error
}

var (
	notifier Notifier
	iconPath func() string
	prefOn   func(string) bool
)

// Setup injects the notifications service, app icon path, and preference lookup.
func Setup(n Notifier, icon func() string, prefs func(string) bool) {
	notifier = n
	iconPath = icon
	prefOn = prefs
}

// Pref reports whether the named desktop-notification preference is on.
func Pref(key string) bool {
	if prefOn == nil {
		return false
	}
	return prefOn(key)
}

// SendIf sends a desktop notification when enabled is true.
func SendIf(enabled bool, kind Kind, title, body string, data map[string]any) {
	if !enabled {
		return
	}
	Send(kind, title, body, data)
}

// Send posts a desktop notification with the app icon. It is a no-op when Setup has not run.
func Send(kind Kind, title, body string, data map[string]any) {
	if notifier == nil {
		return
	}
	opts := Notice{ID: noticeID(kind, data), Title: title, Body: body, Data: data}
	if iconPath != nil {
		opts.Icon = iconPath()
	}
	if err := notifier.Notify(opts); err != nil {
		log.Printf("desktop notification: %v", err)
	}
}

func noticeID(kind Kind, data map[string]any) string {
	n := time.Now().UnixNano()
	if kind == KindDownload || kind == KindOther {
		return fmt.Sprintf("%s-%d", kind, n)
	}
	profile, _ := data["profile"].(string)
	return fmt.Sprintf("%s-%s-%d", kind, profile, n)
}

// openAction is the portal action a click on a notification sends; its target carries the notice's Data.
const openAction = "open"

// Portal converts n for the Notification portal, whose click comes back through RouteClicks. An unreadable icon is
// dropped: the text is what matters.
func (n Notice) Portal() portal.Notification {
	out := portal.Notification{ID: n.ID, Title: n.Title, Body: n.Body, DefaultAction: openAction}
	if len(n.Data) > 0 {
		if raw, err := json.Marshal(n.Data); err == nil {
			out.DefaultActionTarget = string(raw)
		}
	}
	if n.Icon != "" {
		if png, err := safefs.ReadFile(n.Icon); err == nil {
			out.IconPNG = png
		}
	}
	return out
}

// ClickFunc receives a clicked notification's id and the Data it was sent with, the same pair the Wails service
// reports.
type ClickFunc func(id string, data map[string]any)

// routeAction hands a click on a default action to h; any other action is not ours.
func routeAction(a portal.NotificationAction, h ClickFunc) {
	if a.Action != openAction {
		return
	}
	var data map[string]any
	if a.Target != "" {
		if err := json.Unmarshal([]byte(a.Target), &data); err != nil {
			log.Printf("desktop notification click target: %v", err)
		}
	}
	h(a.ID, data)
}

var stopClicks = func() {}

// RouteClicks delivers portal notification clicks to h until StopClicks. The portal does not report the window
// activation token to Wails, so the handler raises the window itself.
func RouteClicks(ctx context.Context, h ClickFunc) error {
	stop, err := portal.OnNotificationAction(ctx, func(a portal.NotificationAction) { routeAction(a, h) })
	if err != nil {
		return err
	}
	stopClicks = stop
	return nil
}

// StopClicks ends the RouteClicks subscription.
func StopClicks() { stopClicks() }
