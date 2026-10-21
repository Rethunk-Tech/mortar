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
	frameLine = regexp.MustCompile(`^\s+at (?:\(wrapper [^)]*\) )?([\w.<>` + "`" + `$+@\[\]-]+?)\.([^.\s(]+)(?:\[[^\]]*\])? ?\(`)
	// unityFrame is a frame Unity's log prints after "Stack trace:", with no "at": "Namespace.Type.Method (args) (at <module>:0)".
	unityFrame = regexp.MustCompile(`^([\w.<>` + "`" + `$+@\[\]-]+?)\.([^.\s(:]+)(?:\[[^\]]*\])? ?\(.*\(at .*\)\s*$`)
	// patchFrame is a Harmony-generated method: DMD<...> wrapper frames, or a patched method renamed "..._Patch1".
	patchFrame = regexp.MustCompile(`^\s+at (?:\(wrapper [^)]*\) )?(?:DMD<|[\w.<>+]*_Patch\d*\()`)
	// dmdTarget is the method a Harmony wrapper stands in for: "DMD<Namespace.Type::Method>".
	dmdTarget = regexp.MustCompile(`DMD<([^>]+::[^>]+)>`)
	harmonyID = regexp.MustCompile(`(?i)(?:harmony ?id|patch ?owner|patched by)\W+([A-Za-z0-9_][\w.\-]*)`)
	stackEnd  = regexp.MustCompile(`^\s*--- End of (?:inner exception )?stack trace`)
	logPrefix = regexp.MustCompile(`^\[\d\d:\d\d:\d\d `)
	// inFile is what closes a frame: a file and line, or Mono's offset and module, or Unity's "(at module:0)". Mono
	// writes "<mvid>:IL_003F" for a method it has no symbols for, and a space after it.
	inFile = regexp.MustCompile(` (?:in .*:line \d+|\[0x[0-9a-f]+\] in .*:(?:\d+|IL_[0-9A-Fa-f]+)|\(at .*\))\s*$`)
	// wrapperFrame is a runtime-made method in Unity's format, which has no "at" and no location: MonoMod's
	// "(wrapper dynamic-method) MonoMod.Utils.DynamicMethodDefinition.Trampoline<T::M>?123(args)".
	wrapperFrame = regexp.MustCompile(`^\(wrapper [^)]*\) ([\w.<>` + "`" + `$+@\[\]-]+?)\.([^.\s(]+) ?\(`)
	// colonFrame is Unity's own trace format for a Debug.Log call: "Namespace.Type:Method(args)".
	colonFrame = regexp.MustCompile(`^([\w.<>` + "`" + `$+@\[\]-]+):([^.\s(:]+) ?\(`)
	// location is where a frame says its code is: .NET's "in path:line N", Mono's "[0x2e] in path:180" or
	// "in <mvid>:IL_002E", Unity's "(at path:17)" or "(at <mvid>:IL_01D0)".
	location = regexp.MustCompile(`(?: in (.+):line (\d+)| \[0x[0-9a-f]+\] in (.+):(\d+|IL_[0-9A-Fa-f]+)| \(at (.+):(\d+|IL_[0-9A-Fa-f]+)\))$`)
	mvid     = regexp.MustCompile(`^<([0-9a-f]{32})>$`)
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

// Frame is one frame of a stack. Method is "Namespace.Type::Method", for a Harmony wrapper (DMD) the method it stands
// in for. Source is the file's base name and line when the runtime printed them; Module is the assembly's MVID, 32
// lowercase hex digits, when it printed that instead because it had no symbols for the method.
type Frame struct {
	Method string
	DMD    bool
	Source string
	Module string
}

// Frames reads the stack below an exception line, innermost first. lines start at the first line after the exception;
// the stack ends at the first line that is no frame.
func Frames(lines []string) []Frame {
	var out []Frame
	for i, l := range lines {
		l = strings.TrimRight(l, "\r ")
		if i == 0 && strings.TrimSpace(l) == "Stack trace:" {
			continue
		}
		f, ok := frameOf(l)
		if !ok {
			if stackEnd.MatchString(l) {
				continue
			}
			break
		}
		out = append(out, f)
	}
	return out
}

func frameOf(l string) (Frame, bool) {
	if m := dmdTarget.FindStringSubmatch(l); m != nil {
		return Frame{Method: m[1], DMD: true}, true
	}
	m := frameLine.FindStringSubmatch(l)
	for _, re := range []*regexp.Regexp{unityFrame, wrapperFrame, colonFrame} {
		if m != nil {
			break
		}
		m = re.FindStringSubmatch(l)
	}
	if m == nil {
		return Frame{}, false
	}
	f := Frame{Method: m[1] + "::" + m[2]}
	if loc := location.FindStringSubmatch(l); loc != nil {
		path, line := loc[1]+loc[3]+loc[5], loc[2]+loc[4]+loc[6]
		if id := mvid.FindStringSubmatch(path); id != nil {
			f.Module = id[1]
		} else if !strings.HasPrefix(path, "<") && !strings.HasPrefix(line, "IL_") && line != "0" {
			f.Source = path[strings.LastIndexAny(path, `/\`)+1:] + ":" + line
		}
	}
	return f, true
}

// Methods names the methods the stack below an exception line runs through, innermost first, as Harmony targets name
// them ("Namespace.Type::Method"): each frame's method, and for a Harmony wrapper the method it stands in for.
func Methods(lines []string) []string {
	frames := Frames(lines)
	out := make([]string, len(frames))
	for i, f := range frames {
		out[i] = f.Method
	}
	return out
}
