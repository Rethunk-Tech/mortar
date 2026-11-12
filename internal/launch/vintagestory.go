package launch

import "github.com/Rethunk-Tech/go-vintagestory/vslog"

// vintageStoryLevels maps the game's levels onto the console's; its others (Notification, Event, Chat, Audit, Build,
// StoryEvent, Worldgen) read as information.
var vintageStoryLevels = map[string]Level{
	"Fatal":        Alert,
	"Error":        Error,
	"Warning":      Warn,
	"Debug":        Debug,
	"VerboseDebug": Trace,
}

func parseVintageStory(line string) (Entry, bool) {
	l, ok := vslog.ParseLine(line)
	if !ok {
		return Entry{}, false
	}
	level, ok := vintageStoryLevels[l.Level]
	if !ok {
		level = Info
	}
	return Entry{Time: l.Time, Level: level, Message: l.Message}, true
}
