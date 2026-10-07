package smapi

import (
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

var (
	// exceptionLine opens a stack: "System.NullReferenceException: message", or an inner one after "--->".
	exceptionLine = regexp.MustCompile(`\b((?:\w+\.)+\w*Exception): (.*)$`)
	// frameLine is a frame of a .NET stack: "   at Namespace.Type.Method(args) in file:line N".
	frameLine = regexp.MustCompile(`^\s+at (?:\(wrapper [^)]*\) )?([\w.<>` + "`" + `$+]+?)\.([^.\s(]+)(?:\[[^\]]*\])?\(`)
	// patchFrame is a Harmony-generated method: DMD<...> wrapper frames, or a patched method renamed "..._Patch1".
	patchFrame = regexp.MustCompile(`^\s+at (?:\(wrapper [^)]*\) )?(?:DMD<|[\w.<>+]*_Patch\d*\()`)
	harmonyID  = regexp.MustCompile(`(?i)(?:harmony ?id|patch ?owner|patched by)\W+([A-Za-z0-9_][\w.\-]*)`)
	stackEnd   = regexp.MustCompile(`^\s*--- End of (?:inner exception )?stack trace`)
	logPrefix  = regexp.MustCompile(`^\[\d\d:\d\d:\d\d `)
	inFile     = regexp.MustCompile(` in .*:line \d+$`)
)

// notMod are root namespaces of code that is not a mod's: SMAPI, its libraries, the game and the runtime.
var notMod = map[string]bool{
	"stardewmoddingapi": true, "harmonylib": true, "monomod": true, "stardewvalley": true, "xtile": true,
	"system": true, "microsoft": true, "mono": true, "netcode": true, "newtonsoft": true, "monogame": true,
	"steamworks": true, "dmd": true, "lidgren": true, "sdl2": true, "mscorlib": true,
}

// StackBlame reads the exception stacks of a SMAPI log and names, for each, the mod frame nearest the throw. A
// stack's frames run from the throw outwards, an inner exception's before its "End of inner exception" marker, so the
// first mod frame is the innermost. A stack that runs only through Harmony patches blames the Harmony id the log
// shows beside it, when it shows one.
func (Loader) StackBlame(log string) []loader.Blame {
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
		end := i + 1
		for end < len(lines) && (frameLine.MatchString(lines[end]) || stackEnd.MatchString(lines[end]) || patchFrame.MatchString(lines[end]) || exceptionLine.MatchString(lines[end]) && !logPrefix.MatchString(lines[end])) {
			end++
		}
		if end == i+1 {
			continue
		}
		if b, ok := blameOf(m[1]+": "+m[2], lines[i+1:end], lines[max(0, i-6):i]); ok {
			out = append(out, b)
		}
		i = end - 1
	}
	return out
}

func blameOf(exception string, stack, before []string) (loader.Blame, bool) {
	patched := false
	for _, l := range stack {
		if patchFrame.MatchString(l) {
			patched = true
			continue
		}
		m := frameLine.FindStringSubmatch(l)
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
