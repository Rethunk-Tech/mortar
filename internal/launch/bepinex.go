package launch

import (
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/go-bepinex/logline"
)

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
	l, ok := logline.Parse(line)
	if !ok {
		return Entry{}, false
	}
	e := Entry{Time: l.Stamp, Level: bepinexLevels[string(l.Level)], Mod: l.Source, Message: l.Message}
	e.Source, e.Object, e.Message = splitLogTag(e.Message)
	return e, true
}

// logTag writes the bridge's tag back, so a run log read again keeps each line's source.
func logTag(source, object string) string {
	var parts []string
	if source != "" {
		parts = append(parts, "src="+source)
	}
	if object != "" {
		parts = append(parts, "obj="+object)
	}
	return "⟦" + strings.Join(parts, ";") + "⟧ "
}

// splitLogTag takes the bridge's "⟦src=Type::Method;obj=path⟧ " off the front of a message.
func splitLogTag(message string) (source, object, rest string) {
	tag, rest, ok := strings.Cut(strings.TrimPrefix(message, "⟦"), "⟧ ")
	if !ok || !strings.HasPrefix(message, "⟦") {
		return "", "", message
	}
	for part := range strings.SplitSeq(tag, ";") {
		if v, ok := strings.CutPrefix(part, "src="); ok {
			source = v
		} else if v, ok := strings.CutPrefix(part, "obj="); ok {
			object = v
		}
	}
	return source, object, rest
}

func formatBepInEx(e Entry) string {
	message := e.Message
	if !e.Cont && (e.Source != "" || e.Object != "") {
		message = logTag(e.Source, e.Object) + message
	}
	return logline.Format(logline.Line{Stamp: e.Time, Level: logline.Level(bepinexNames[e.Level]), Source: e.Mod, Message: message})
}
