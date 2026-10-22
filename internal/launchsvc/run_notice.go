package launchsvc

import (
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/i18n"
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

// RunEndNotificationText chooses the desktop notification for a run's end from its log summary. Only a crash is
// worth a notification; ok is false for a run that closed normally.
func RunEndNotificationText(gameName string, stats launch.Summary) (title, body string, ok bool) {
	if stats.Crashed {
		title = i18n.T("{game} crashed", "game", gameName)
		n := len(stats.Mods)
		body = i18n.T("Mods with errors: {count}", "count", strconv.Itoa(n))
		if launch.ExitCrashed(stats.Exit) {
			desc := launch.DescribeExitLocalized(stats.Exit)
			if n == 0 {
				body = desc
			} else {
				body = i18n.T("{errors}. {exit}", "errors", body, "exit", desc)
			}
		}
		return title, body, true
	}
	return "", "", false
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
