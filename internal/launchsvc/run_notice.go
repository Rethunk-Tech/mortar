package launchsvc

import (
	"fmt"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/launch"
)

// RunEndNotice is sent when a Mortar-started game run ends.
type RunEndNotice struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
	Title   string `json:"title"`
	Body    string `json:"body"`
}

// NoticeClickEvent is emitted when the user activates a Mortar desktop notification.
const NoticeClickEvent = "tray:notice"

// NoticeClick opens the notification's profile in the UI (console or mods updates).
type NoticeClick struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
	Tab     string `json:"tab"`
}

// RunEndNotificationText chooses the desktop notification title and body from the run log summary.
func RunEndNotificationText(gameName string, stats launch.Summary) (title, body string) {
	if stats.Crashed {
		title = gameName + " crashed"
		n := len(stats.Mods)
		if n == 1 {
			body = "1 mod logged errors"
		} else {
			body = fmt.Sprintf("%d mods logged errors", n)
		}
		if launch.ExitCrashed(stats.Exit) {
			desc := launch.DescribeExit(stats.Exit)
			if n == 0 {
				body = desc
			} else {
				body += "; " + desc
			}
		}
		return title, body
	}
	title = "Game closed"
	return title, body
}

// NoticeProfileFromResponse reads game, profile, and target tab from a notification activation.
func NoticeProfileFromResponse(id string, data map[string]any) (gameID, profileID, tab string) {
	switch {
	case strings.HasPrefix(id, "run-end-"):
		tab = "console"
	case strings.HasPrefix(id, "mod-updates-"):
		tab = "updates"
	default:
		return "", "", ""
	}
	if data == nil {
		return "", "", ""
	}
	gameID, _ = data["game"].(string)
	profileID, _ = data["profile"].(string)
	return gameID, profileID, tab
}
