// Package desktopnotify sends OS desktop notifications with the app icon.
package desktopnotify

import (
	"fmt"
	"log"
	"time"
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
