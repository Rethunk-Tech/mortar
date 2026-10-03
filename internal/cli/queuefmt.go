package cli

import (
	"github.com/Rethunk-AI/mortar/internal/queue"
	"github.com/Rethunk-AI/mortar/internal/usererr"
)

func queueHumanState(state string) string {
	switch state {
	case queue.StateQueued:
		return "waiting"
	case queue.StateDownloading, queue.StateInstalling:
		return "downloading"
	case queue.StateWaitingClick, queue.StateNeedsChoice, queue.StateNeedsConfirm, queue.StateNeedsFomod, queue.StateNeedsRoot, queue.StateNeedsMerge:
		return "needs a click"
	case queue.StateFailed:
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

func queueActionCount(st queue.State, sub, id string) int {
	n := 0
	for _, it := range st.Items {
		if id != "" && it.ID != id {
			continue
		}
		if sub == "retry" && (it.State == queue.StateFailed || it.State == queue.StateSkipped) {
			n++
			continue
		}
		if sub == "skip" {
			switch it.State {
			case queue.StateQueued, queue.StateWaitingClick, queue.StateNeedsChoice, queue.StateNeedsConfirm, queue.StateNeedsFomod, queue.StateNeedsRoot, queue.StateNeedsMerge:
				n++
			}
		}
	}
	return n
}
