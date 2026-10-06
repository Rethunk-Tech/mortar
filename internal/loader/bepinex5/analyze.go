package bepinex5

import (
	"regexp"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// Kinds of Finding.
const (
	KindMissingDependency = "missing-dependency"
	KindIncompatible      = "incompatible-version"
	KindLoadException     = "load-exception"
	KindPatchException    = "patch-exception"
	KindPreloader         = "preloader-patch"
	KindUnityException    = "unity-exception"
	// KindDependencyNotLoaded is a plugin skipped because a plugin it needs failed earlier in the same log.
	KindDependencyNotLoaded = "dependency-not-loaded"
	// KindDuplicateGUID is an older copy of a plugin skipped because a newer one with the same GUID loaded.
	KindDuplicateGUID = "duplicate-guid"
	// KindIncompatiblePlugin is a plugin that declares itself incompatible with another installed one.
	KindIncompatiblePlugin = "incompatible-plugin"
	// KindLoaderVersion is a plugin built for a newer BepInEx than the profile has.
	KindLoaderVersion = "loader-version"
	// KindChainloader is the chainloader itself failing, so no plugin loads.
	KindChainloader = "chainloader"
	// KindPluginError is an error a plugin logged under its own name.
	KindPluginError = "plugin-error"
)

// Finding is one load failure read from a log.
type Finding = loader.Finding

// The chainloader and preloader messages are BepInEx 5.4.x's Chainloader and AssemblyPatcher format strings; the
// dependency messages gained a trailing "Install the listed plugin(s)" sentence in some 5.4 builds.
var (
	logLine          = regexp.MustCompile(`^\[(Error|Fatal|Warning)\s*:\s*([^\]]*)\] (.*)$`)
	missingDeps      = regexp.MustCompile(`^Could not load \[([^\]]+)\] because it has missing dependencies: (.*?)(?:\. Install the listed.*)?$`)
	incompatibleDep  = regexp.MustCompile(`^Could not load \[([^\]]+)\] because (?:it has incompatible dependencies|the following dependencies are installed with an incompatible version): (.*?)(?:\. Update the listed.*)?$`)
	incompatibleWith = regexp.MustCompile(`^Could not load \[([^\]]+)\] because it is incompatible with: (.*)$`)
	wrongBepInEx     = regexp.MustCompile(`^Plugin \[([^\]]+)\] targets a wrong version of BepInEx \(([^)]*)\)`)
	requiresBepInEx  = regexp.MustCompile(`^(?:Skipping|Could not load) \[([^\]]+)\].*requires BepInEx( version)?.*$`)
	// Valheim's pack adds the GUID and both paths, and words an identical second copy as a duplicate.
	newerExists   = regexp.MustCompile(`^Skipping \[([^\]]+)\](?: \([^)]*\) at .*?)? because (?:a newer version exists \((.*)\)|a duplicate of it was already loaded from (.*))$`)
	depNotLoaded  = regexp.MustCompile(`^Skipping \[([^\]]+)\] because it has a dependency that was not loaded\.`)
	loadError     = regexp.MustCompile(`^Error loading \[([^\]]+)\] ?: (.*)$`)
	preloaderFail = regexp.MustCompile(`^(Failed to load patcher \[[^\]]+\]|Failed to run \[[^\]]+\] when patching \[[^\]]+\]|Failed to run (?:Initializer|Finalizer) of \S+|Could not run preloader!)(.*)$`)
	harmonyError  = regexp.MustCompile(`^Error while running (?:static )?\S+ ([\w.]+)::\S+?\(.*\)\. Error: (.*)$`)
	// Harmony rejecting a patch while a plugin loads reaches the log through Unity's logger, naming the patch method.
	patchTarget = regexp.MustCompile(`^\w+Exception: .*for patch method (?:static )?(?:\S+ )?([\w.+]+)::`)
	patchWords  = regexp.MustCompile(`(?i)transpiler|prefix|postfix|harmony|patch`)

	unityException = regexp.MustCompile(`^((?:[\w]+\.)*\w*Exception): (.*)$`)
	stackFrame     = regexp.MustCompile(`^\s+at (?:\(wrapper [^)]*\) )?([\w.]+?)\.[^.\s(]+(?:<[^>]*>)? ?\(`)
	// unityFrame is a frame of the stack BepInEx's Unity log listener writes after "Stack trace:": Unity's own format,
	// "Namespace.Type.Method (args) (at <module>:0)", or "Type:Method(args)" for engine frames.
	unityFrame = regexp.MustCompile(`^([\w.]+?)[.:][^.\s(:]+(?:<[^>]*>)? ?\(`)
	loading    = regexp.MustCompile(`^\[Info\s*:\s*BepInEx\] Loading \[([^\]]+)\]`)
	// chainloaderStart is the frame of an exception thrown while the chainloader adds a plugin, in Awake.
	chainloaderStart = regexp.MustCompile(`BepInEx\.Bootstrap\.Chainloader[.:]Start ?\(`)
)

// loaderSources are the log sources that are BepInEx, Harmony or Unity rather than a plugin speaking for itself.
var loaderSources = []string{"BepInEx", "Preloader", "HarmonyX", "Unity Log"}

// notices are loader warnings that need nothing from the player: a plugin limited to another game's process, a type
// a plugin DLL holds that is not a plugin, and Harmony's lookup miss that precedes the patch failure it causes.
var notices = regexp.MustCompile(`^Skipping \[[^\]]+\] because of process filters|^Skipping (?:over )?type \[|^AccessTools\.\w+: Could not find`)

// frameworkNamespaces are code that is not a mod, so a stack naming only them blames no plugin.
var frameworkNamespaces = []string{"System", "UnityEngine", "Unity", "GameNetcodeStuff", "Dissonance", "HarmonyLib", "MonoMod", "BepInEx", "Mono", "TMPro", "UnityEditor", "DMD", "Steamworks", "Netcode"}

// Analyze reads the load failures out of BepInEx's LogOutput.log and the Unity Player.log. Either may be empty. A
// failure repeated in a log is reported once, at its first line.
func Analyze(logOutput, playerLog string) []Finding {
	var out []Finding
	seen := map[string]bool{}
	add := func(f Finding) {
		key := f.Source + "|" + f.Kind + "|" + f.Plugin + "|" + f.Message
		if f.Kind == KindUnityException {
			// Unity writes an exception to Player.log and BepInEx copies it into LogOutput.log; it is one failure.
			key = f.Kind + "|" + f.Plugin + "|" + f.Message
		}
		if f.Kind == KindPluginError {
			key = f.Source + "|" + f.Kind + "|" + f.Plugin
		}
		if !seen[key] {
			seen[key] = true
			out = append(out, f)
		}
	}
	logged := map[string]bool{}
	outLines := strings.Split(logOutput, "\n")
	for i, line := range outLines {
		f, ok := classify(strings.TrimRight(line, "\r"))
		if !ok {
			f, ok = unityLogException(line, outLines[:i], outLines[i+1:])
		}
		if ok && f.Kind != "" {
			f.Line, f.Source = i+1, "LogOutput.log"
			if f.Kind == KindUnityException {
				logged[f.Message] = true
			}
			add(f)
		}
	}
	lines := strings.Split(playerLog, "\n")
	for i, line := range lines {
		m := unityException.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil || logged[m[1]+": "+m[2]] {
			continue
		}
		plugin := loadingPlugin(lines[:i], lines[i+1:])
		if plugin == "" {
			plugin = blamedPlugin(lines[i+1:], stackFrame)
		}
		if plugin != "" {
			add(Finding{Kind: KindUnityException, Plugin: plugin, Message: m[1] + ": " + m[2], Line: i + 1, Source: "Player.log"})
		}
	}
	return out
}

// classify reads one LogOutput.log line. ok is false for a warning or error no rule recognises; a recognised line
// that needs nothing from the player has an empty Kind.
func classify(line string) (f Finding, ok bool) {
	m := logLine.FindStringSubmatch(line)
	if m == nil {
		return Finding{}, true
	}
	level, source, msg := m[1], strings.TrimSpace(m[2]), m[3]
	g := func(re *regexp.Regexp) []string { return re.FindStringSubmatch(msg) }
	switch {
	case g(missingDeps) != nil:
		g := g(missingDeps)
		dep, _, _ := strings.Cut(g[2], ",")
		dep, _, _ = strings.Cut(strings.TrimSpace(dep), " ")
		return Finding{Kind: KindMissingDependency, Plugin: g[1], Message: "missing dependencies: " + g[2], Dependency: dep}, true
	case g(incompatibleDep) != nil:
		g := g(incompatibleDep)
		return Finding{Kind: KindIncompatible, Plugin: g[1], Message: "incompatible dependencies: " + g[2]}, true
	case g(incompatibleWith) != nil:
		g := g(incompatibleWith)
		return Finding{Kind: KindIncompatiblePlugin, Plugin: g[1], Message: "incompatible with: " + g[2]}, true
	case g(wrongBepInEx) != nil:
		g := g(wrongBepInEx)
		return Finding{Kind: KindLoaderVersion, Plugin: g[1], Message: msg}, true
	case g(requiresBepInEx) != nil:
		return Finding{Kind: KindIncompatible, Plugin: g(requiresBepInEx)[1], Message: msg}, true
	case g(newerExists) != nil:
		g := g(newerExists)
		if g[3] != "" {
			return Finding{Kind: KindDuplicateGUID, Plugin: g[1], Message: "the same copy loaded instead from " + g[3]}, true
		}
		return Finding{Kind: KindDuplicateGUID, Plugin: g[1], Message: "a newer copy loaded instead: " + g[2]}, true
	case g(depNotLoaded) != nil:
		return Finding{Kind: KindDependencyNotLoaded, Plugin: g(depNotLoaded)[1], Message: msg}, true
	case g(loadError) != nil:
		g := g(loadError)
		return Finding{Kind: KindLoadException, Plugin: g[1], Message: g[2]}, true
	case g(preloaderFail) != nil:
		return Finding{Kind: KindPreloader, Message: msg}, true
	case g(harmonyError) != nil:
		g := g(harmonyError)
		return Finding{Kind: KindPatchException, Plugin: rootNamespace(g[1]), Message: g[2]}, true
	case g(patchTarget) != nil:
		return Finding{Kind: KindPatchException, Plugin: rootNamespace(g(patchTarget)[1]), Message: msg}, true
	case level == "Fatal" && source == "BepInEx" && msg == "Error occurred starting the game":
		return Finding{Kind: KindChainloader, Message: msg}, true
	case notices.MatchString(msg):
		return Finding{}, true
	case level == "Error" && !slices.Contains(loaderSources, source):
		kind := KindPluginError
		if patchWords.MatchString(msg) {
			kind = KindPatchException
		}
		return Finding{Kind: kind, Plugin: source, Message: msg}, true
	}
	return Finding{}, false
}

// Unclassified counts the warnings and errors in a LogOutput.log that no rule recognises, so a run record shows how
// much of its log Mortar could not explain.
func Unclassified(logOutput string) int {
	n := 0
	for line := range strings.Lines(logOutput) {
		if _, ok := classify(strings.TrimRight(line, "\r\n")); !ok {
			n++
		}
	}
	return n
}

// unityLogException reads an exception Unity caught (one thrown in a plugin's Awake or Update) as BepInEx's Unity log
// listener writes it: the exception line, "Stack trace:", then Unity's frames, which blame the first mod namespace.
func unityLogException(line string, before, rest []string) (Finding, bool) {
	m := logLine.FindStringSubmatch(strings.TrimRight(line, "\r"))
	if m == nil || m[1] != "Error" || strings.TrimSpace(m[2]) != "Unity Log" || len(rest) == 0 ||
		strings.TrimSpace(rest[0]) != "Stack trace:" {
		return Finding{}, false
	}
	u := unityException.FindStringSubmatch(m[3])
	if u == nil {
		return Finding{}, false
	}
	plugin := loadingPlugin(before, rest[1:])
	if plugin == "" {
		plugin = blamedPlugin(rest[1:], unityFrame)
	}
	if plugin == "" {
		return Finding{}, false
	}
	return Finding{Kind: KindUnityException, Plugin: plugin, Message: u[1] + ": " + u[2]}, true
}

// loadingPlugin is the plugin, as "Name Version", the chainloader last logged loading, when the exception's stack
// runs through Chainloader.Start: the plugin threw in its own Awake, and packages can share a namespace, so the
// stack's namespace alone may not say which one.
func loadingPlugin(before, stack []string) string {
	inStart := false
	for _, l := range stack {
		if strings.TrimSpace(l) == "" {
			break
		}
		if chainloaderStart.MatchString(l) {
			inStart = true
			break
		}
	}
	if !inStart {
		return ""
	}
	for _, l := range slices.Backward(before) {
		if m := loading.FindStringSubmatch(strings.TrimRight(l, "\r")); m != nil {
			return m[1]
		}
	}
	return ""
}

// blamedPlugin is the root namespace of the first stack frame, read with frame, that is not game, Unity or loader code.
func blamedPlugin(stack []string, frame *regexp.Regexp) string {
	for _, l := range stack {
		m := frame.FindStringSubmatch(strings.TrimRight(l, "\r"))
		if m == nil {
			if strings.HasPrefix(l, "  at ") {
				continue
			}
			return ""
		}
		// The game's own code has no namespace, so a frame with none is not blamed.
		if ns := rootNamespace(m[1]); strings.Contains(m[1], ".") && !isFramework(ns) {
			return ns
		}
	}
	return ""
}

func rootNamespace(typeName string) string {
	root, _, _ := strings.Cut(typeName, ".")
	return root
}

func isFramework(ns string) bool {
	for _, f := range frameworkNamespaces {
		if strings.EqualFold(ns, f) {
			return true
		}
	}
	return false
}
