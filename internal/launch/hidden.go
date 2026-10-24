package launch

import (
	"regexp"
	"strings"
	"sync"
)

// The Console and the bridge mask numbers the same way (frontend console/logSource.ts mask, the bridge's LogTag.Mask).
var number = regexp.MustCompile(`-?[0-9]+(?:[.,:][0-9]+)*`)

// Mask is a message's first line with every number as "#": what one call keeps from line to line.
func Mask(message string) string {
	first, _, _ := strings.Cut(message, "\n")
	return strings.TrimSpace(number.ReplaceAllString(first, "#"))
}

// ModMessagePrefix starts the source of a hidden message a mod's log line names by its mod column (an untagged line,
// as SMAPI writes them): "mod:<Mod>", with the message's masked first line as its text.
const ModMessagePrefix = "mod:"

var hidden struct {
	mu     sync.RWMutex
	byGame map[string]map[ModMessage]bool
}

// ModMessage is a mod's message by its mod column and masked first line.
type ModMessage struct{ Mod, Text string }

// SetHiddenMessages sets the mod messages of a game's log no reader of a parsed log sees, with their continuation
// lines. A parsed log does not name its game, so every game's set applies to every log.
func SetHiddenMessages(game string, msgs []ModMessage) {
	set := make(map[ModMessage]bool, len(msgs))
	for _, m := range msgs {
		set[m] = true
	}
	hidden.mu.Lock()
	defer hidden.mu.Unlock()
	if hidden.byGame == nil {
		hidden.byGame = map[string]map[ModMessage]bool{}
	}
	hidden.byGame[game] = set
}

func isHidden(mod, message string) bool {
	hidden.mu.RLock()
	defer hidden.mu.RUnlock()
	key := ModMessage{mod, Mask(message)}
	for _, set := range hidden.byGame {
		if set[key] {
			return true
		}
	}
	return false
}
