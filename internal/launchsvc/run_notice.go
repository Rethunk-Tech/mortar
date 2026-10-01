package launchsvc

import (
	"fmt"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/launch"
)

// RunEndNotice is sent when a Mortar-started game run ends.
type RunEndNotice struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
	Title   string `json:"title"`
	Body    string `json:"body"`
}

// NoticeClickEvent is emitted when the user activates a run-end desktop notification.
const NoticeClickEvent = "tray:notice"

// NoticeClick opens the ended run's profile console in the UI.
type NoticeClick struct {
	Game    string `json:"game"`
	Profile string `json:"profile"`
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
		return title, body
	}
	title = "Game closed"
	return title, body
}

// NoticeProfileFromResponse reads the profile id from a run-end notification activation.
func NoticeProfileFromResponse(id string, data map[string]any) (gameID, profileID string) {
	if !strings.HasPrefix(id, "run-end-") {
		return "", ""
	}
	if data == nil {
		return "", ""
	}
	gameID, _ = data["game"].(string)
	profileID, _ = data["profile"].(string)
	return gameID, profileID
}
