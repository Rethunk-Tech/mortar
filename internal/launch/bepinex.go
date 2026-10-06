package launch

import (
	"fmt"
	"regexp"
	"strings"
)

// BepInEx writes `[Level  :    Source] message` to LogOutput.log, padding the level to seven characters and the
// source to ten; a message's further lines carry no header.
var bepinexHeader = regexp.MustCompile(`^\[(Fatal|Error|Warning|Message|Info|Debug) *: *([^\]]*)\] ?(.*)$`)

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
	m := bepinexHeader.FindStringSubmatch(line)
	if m == nil {
		return Entry{}, false
	}
	return Entry{Level: bepinexLevels[m[1]], Mod: strings.TrimSpace(m[2]), Message: m[3]}, true
}

func formatBepInEx(e Entry) string {
	return fmt.Sprintf("[%-7s:%10s] %s", bepinexNames[e.Level], e.Mod, e.Message)
}
