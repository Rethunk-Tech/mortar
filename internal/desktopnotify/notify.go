// Package desktopnotify sends OS desktop notifications with the app icon.
package desktopnotify

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

// Notifier is the Wails notifications service.
type Notifier interface {
	SendNotification(notifications.NotificationOptions) error
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
func SendIf(enabled bool, title, body string, data map[string]any) {
	if !enabled {
		return
	}
	Send(title, body, data)
}

// Send posts a desktop notification with the app icon. It is a no-op when Setup has not run.
func Send(title, body string, data map[string]any) {
	if notifier == nil {
		return
	}
	opts := notifications.NotificationOptions{
		ID:    noticeID(title, data),
		Title: title,
		Body:  body,
		Data:  data,
	}
	if iconPath != nil {
		if icon := iconPath(); icon != "" {
			opts.Attachments = []notifications.NotificationAttachment{
				{ID: "icon", Path: icon, Type: "appLogoOverride"},
			}
		}
	}
	if err := notifier.SendNotification(opts); err != nil {
		log.Printf("desktop notification: %v", err)
	}
}

func noticeID(title string, data map[string]any) string {
	n := time.Now().UnixNano()
	profile := ""
	if data != nil {
		if p, ok := data["profile"].(string); ok {
			profile = p
		}
	}
	switch {
	case strings.HasPrefix(title, "Download "):
		return fmt.Sprintf("download-%d", n)
	case strings.Contains(title, "mod updates"):
		return fmt.Sprintf("mod-updates-%s-%d", profile, n)
	default:
		return fmt.Sprintf("run-end-%s-%d", profile, n)
	}
}
