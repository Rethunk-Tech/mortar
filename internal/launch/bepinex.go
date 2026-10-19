package launch

import (
	"fmt"
	"regexp"
	"strings"
)

// BepInEx writes `[Level  :    Source] message` to LogOutput.log, padding the level to seven characters and the
// source to ten; a message's further lines carry no header.
var bepinexHeader = regexp.MustCompile(`^\[(Fatal|Error|Warning|Message|Info|Debug) *: *([^\]]*)\] ?(.*)$`)

// bepinexStamp is the "[stamp] " mattymatty-AsyncLoggers puts before a header (ExtendedLogEventArgs.cs, ToTimestampedEventArgs):
// HH:mm:ss.fffffff by default, or a 16-digit tick, frame or counter, or a 16-character thread name.
var bepinexStamp = regexp.MustCompile(`^\[([^\]]*)\] (\[(?:Fatal|Error|Warning|Message|Info|Debug) *:.*)$`)

// SplitBepInExStamp separates a logger mod's leading "[stamp] " from a BepInEx line; every BepInEx reader matches on
// rest. A line without a stamp comes back whole.
func SplitBepInExStamp(line string) (stamp, rest string) {
	if m := bepinexStamp.FindStringSubmatch(line); m != nil {
		return m[1], m[2]
	}
	return "", line
}

// smapiTime is the clock SMAPI stamps its headers with; a stamp of any other shape came from a BepInEx logger mod.
var smapiTime = regexp.MustCompile(`^\d\d:\d\d:\d\d$`)

// bepinexLevels maps BepInEx's levels onto the console's. Message is BepInEx's level for a plugin's notable lines,
// which reads as plain information beside SMAPI's.
var bepinexLevels = map[string]Level{
	"Fatal":   Alert,
	"Error":   Error,
	"Warning": Warn,
	"Message": Info,
	"Info":    Info,
	"Debug":   Debug,
}

// bepinexNames writes a console level back the way BepInEx names it, so a formatted run log reads as LogOutput.log
// does for the analyzers.
var bepinexNames = map[Level]string{Alert: "Fatal", Error: "Error", Warn: "Warning", Info: "Info", Debug: "Debug", Trace: "Debug"}

func parseBepInEx(line string) (Entry, bool) {
	stamp, rest := SplitBepInExStamp(line)
	m := bepinexHeader.FindStringSubmatch(rest)
	if m == nil {
		return Entry{}, false
	}
	return Entry{Time: stamp, Level: bepinexLevels[m[1]], Mod: strings.TrimSpace(m[2]), Message: m[3]}, true
}

func formatBepInEx(e Entry) string {
	head := fmt.Sprintf("[%-7s:%10s] %s", bepinexNames[e.Level], e.Mod, e.Message)
	if e.Time != "" {
		return "[" + e.Time + "] " + head
	}
	return head
}
