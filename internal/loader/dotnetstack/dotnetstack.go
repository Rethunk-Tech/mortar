// Package dotnetstack reads the .NET exception stacks a loader's or the game's log holds and names the mod frame
// nearest each throw. SMAPI, BepInEx, MelonLoader and Vintage Story print them in the runtime's formats, Unity's own
// among them.
package dotnetstack

import (
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

var (
	// exceptionLine opens a stack: "System.NullReferenceException: message", or an inner one after "--->". Unity's log
	// drops the namespace.
	exceptionLine = regexp.MustCompile(`\b((?:\w+\.)*\w*Exception): (.*)$`)
	// frameLine is a frame of a .NET stack: "   at Namespace.Type.Method(args) in file:line N". Mono and Unity print a
	// space before the argument list.
	frameLine = regexp.MustCompile(`^\s+at (?:\(wrapper [^)]*\) )?([\w.<>` + "`" + `$+]+?)\.([^.\s(]+)(?:\[[^\]]*\])? ?\(`)
	// unityFrame is a frame Unity's log prints after "Stack trace:", with no "at": "Namespace.Type.Method (args) (at <module>:0)".
	unityFrame = regexp.MustCompile(`^([\w.<>` + "`" + `$+]+?)\.([^.\s(:]+)(?:\[[^\]]*\])? ?\(.*\(at .*\)\s*$`)
	// patchFrame is a Harmony-generated method: DMD<...> wrapper frames, or a patched method renamed "..._Patch1".
	patchFrame = regexp.MustCompile(`^\s+at (?:\(wrapper [^)]*\) )?(?:DMD<|[\w.<>+]*_Patch\d*\()`)
	// dmdTarget is the method a Harmony wrapper stands in for: "DMD<Namespace.Type::Method>".
	dmdTarget = regexp.MustCompile(`DMD<([^>]+::[^>]+)>`)
	harmonyID = regexp.MustCompile(`(?i)(?:harmony ?id|patch ?owner|patched by)\W+([A-Za-z0-9_][\w.\-]*)`)
	stackEnd  = regexp.MustCompile(`^\s*--- End of (?:inner exception )?stack trace`)
	logPrefix = regexp.MustCompile(`^\[\d\d:\d\d:\d\d `)
	// inFile is what closes a frame: a file and line, or Mono's offset and module, or Unity's "(at module:0)".
	inFile = regexp.MustCompile(` (?:in .*:line \d+|\[0x[0-9a-f]+\] in .*:\d+|\(at .*\))$`)
)

// SMAPI is the root namespaces of code that is not a SMAPI mod's: SMAPI, its libraries, the game and the runtime.
var SMAPI = Set("stardewmoddingapi", "harmonylib", "monomod", "stardewvalley", "xtile", "system", "microsoft", "mono",
	"netcode", "newtonsoft", "monogame", "steamworks", "dmd", "lidgren", "sdl2", "mscorlib")

// Unity is the root namespaces of a Unity game's loaders, engine and runtime, for BepInEx and MelonLoader logs.
var Unity = Set("system", "microsoft", "mono", "unityengine", "unity", "unityeditor", "tmpro", "harmonylib", "harmonyx",
	"monomod", "dmd", "bepinex", "melonloader", "il2cppinterop", "il2cppsystem", "il2cpp", "steamworks", "netcode",
	"newtonsoft", "mscorlib", "dissonance", "gamenetcodestuff", "cecil")

// VintageStory is the root namespaces of the game's own code and the runtime, so a frame in one is not a mod's.
var VintageStory = Set("system", "microsoft", "mono", "vintagestory", "vintagestoryapi", "vsurvival", "vsessentials",
	"vscreativemod", "harmonylib", "monomod", "dmd", "cairo", "opentk", "newtonsoft", "protobuf", "sqlite", "mscorlib",
	"tarsos", "csscriptlibrary", "neo")

// Set lists lowercase root namespaces.
func Set(names ...string) map[string]bool {
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m
}

// Blames reads the exception stacks of a log and names, for each, the mod frame nearest the throw. A stack's frames
// run from the throw outwards, an inner exception's before its "End of inner exception" marker, so the first mod frame
// is the innermost. notMod are the root namespaces of code that is not a mod's. A stack that runs only through Harmony
// patches blames the Harmony id the log shows beside it, when it shows one.
func Blames(log string, notMod map[string]bool) []loader.Blame {
	var out []loader.Blame
	lines := strings.Split(strings.ReplaceAll(log, "\r", ""), "\n")
	for i := 0; i < len(lines); i++ {
		head := lines[i]
		if _, inner, ok := strings.Cut(head, "---> "); ok {
			head = inner
		}
		m := exceptionLine.FindStringSubmatch(head)
		if m == nil || i+1 >= len(lines) {
			continue
		}
		start := i + 1
		unity := strings.TrimSpace(lines[start]) == "Stack trace:"
		if unity {
			start++
		}
		end := start
		for end < len(lines) && (frameLine.MatchString(lines[end]) || unity && unityFrame.MatchString(lines[end]) || stackEnd.MatchString(lines[end]) || patchFrame.MatchString(lines[end]) || exceptionLine.MatchString(lines[end]) && !logPrefix.MatchString(lines[end])) {
			end++
		}
		if end == start {
			continue
		}
		if b, ok := blameOf(m[1]+": "+m[2], lines[start:end], lines[max(0, i-6):i], notMod); ok {
			out = append(out, b)
		}
		i = end - 1
	}
	return out
}

func blameOf(exception string, stack, before []string, notMod map[string]bool) (loader.Blame, bool) {
	patched := false
	for _, l := range stack {
		if patchFrame.MatchString(l) {
			patched = true
			continue
		}
		m := frameLine.FindStringSubmatch(l)
		if m == nil {
			m = unityFrame.FindStringSubmatch(l)
		}
		if m == nil {
			continue
		}
		root, _, _ := strings.Cut(m[1], ".")
		if !strings.Contains(m[1], ".") || notMod[strings.ToLower(root)] {
			continue
		}
		ns := m[1][:strings.LastIndex(m[1], ".")]
		return loader.Blame{Exception: exception, Frame: inFile.ReplaceAllString(strings.TrimSpace(l), ""), Namespace: ns}, true
	}
	if patched {
		for _, l := range append(append([]string{}, stack...), before...) {
			if m := harmonyID.FindStringSubmatch(l); m != nil {
				return loader.Blame{Exception: exception, Frame: strings.TrimSpace(firstPatch(stack)), PatchOwner: m[1]}, true
			}
		}
	}
	return loader.Blame{}, false
}

func firstPatch(stack []string) string {
	for _, l := range stack {
		if patchFrame.MatchString(l) {
			return inFile.ReplaceAllString(l, "")
		}
	}
	return ""
}

// Methods names the methods the stack below an exception line runs through, innermost first, as Harmony targets name
// them ("Namespace.Type::Method"): each frame's method, and for a Harmony wrapper the method it stands in for. lines
// start at the first line after the exception; the stack ends at the first line that is no frame.
func Methods(lines []string) []string {
	var out []string
	for i, l := range lines {
		l = strings.TrimRight(l, "\r")
		if i == 0 && strings.TrimSpace(l) == "Stack trace:" {
			continue
		}
		if m := dmdTarget.FindStringSubmatch(l); m != nil {
			out = append(out, m[1])
			continue
		}
		m := frameLine.FindStringSubmatch(l)
		if m == nil {
			m = unityFrame.FindStringSubmatch(l)
		}
		if m == nil {
			if stackEnd.MatchString(l) {
				continue
			}
			break
		}
		out = append(out, m[1]+"::"+m[2])
	}
	return out
}
