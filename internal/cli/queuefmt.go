package cli

import (
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

func queueHumanState(state string) string {
	switch {
	case state == queue.StateQueued:
		return "waiting"
	case state == queue.StateDownloading || state == queue.StateInstalling:
		return "downloading"
	case queue.NeedsPlayer(state):
		return "needs a click"
	case state == queue.StatePaused:
		return "paused"
	case state == queue.StateFailed:
		return "failed"
	default:
		return state
	}
}

func queueErrorLine(raw string) string {
	if raw == "" {
		return ""
	}
	kind, _ := usererr.Parse(raw)
	return Sentence(kind)
}
