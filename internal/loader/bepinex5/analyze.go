package bepinex5

import (
	"regexp"
	"strings"
)

// Kinds of Finding.
const (
	KindMissingDependency = "missing-dependency"
	KindIncompatible      = "incompatible-version"
	KindLoadException     = "load-exception"
	KindPatchException    = "patch-exception"
	KindPreloader         = "preloader-patch"
	KindUnityException    = "unity-exception"
)

// Finding is one load failure read from a log.
type Finding struct {
	Kind string
	// Plugin is the plugin as the log names it: "Name Version" for BepInEx's own messages, the code's root namespace
	// for exceptions.
	Plugin  string
	Message string
	// Line is the 1-based line in the log the finding came from.
	Line int
	// Source is the log: "LogOutput.log" or "Player.log".
	Source string
}

// BepInEx's messages are written by BaseChainloader and AssemblyPatcher (BepInEx 5.4.x); the dependency messages have
// a trailing "Install the listed plugin(s)" sentence from 5.4.22 on.
var (
	errorLine       = regexp.MustCompile(`^\[(?:Error|Fatal)\s*:\s*([^\]]*)\] (.*)$`)
	missingDeps     = regexp.MustCompile(`^Could not load \[([^\]]+)\] because it has missing dependencies: (.*?)(?:\. Install the listed.*)?$`)
	incompatibleDep = regexp.MustCompile(`^Could not load \[([^\]]+)\] because (?:it has incompatible dependencies|the following dependencies are installed with an incompatible version): (.*?)(?:\. Update the listed.*)?$`)
	requiresBepInEx = regexp.MustCompile(`^(?:Skipping|Could not load) \[([^\]]+)\].*requires BepInEx( version)?.*$`)
	loadError       = regexp.MustCompile(`^Error loading \[([^\]]+)\]: (.*)$`)
	preloaderFail   = regexp.MustCompile(`^(Failed to load patcher \[[^\]]+\]|Failed to run \[[^\]]+\] when patching \[[^\]]+\]|Failed to run (?:Initializer|Finalizer) of \S+|Could not run preloader!)(.*)$`)
	harmonyError    = regexp.MustCompile(`^Error while running (?:static )?\S+ ([\w.]+)::\S+?\(.*\)\. Error: (.*)$`)

	unityException = regexp.MustCompile(`^((?:[\w]+\.)*\w*Exception): (.*)$`)
	stackFrame     = regexp.MustCompile(`^\s+at (?:\(wrapper [^)]*\) )?([\w.]+?)\.[^.\s(]+(?:<[^>]*>)? ?\(`)
)

// frameworkNamespaces are code that is not a mod, so a stack naming only them blames no plugin.
var frameworkNamespaces = []string{"System", "UnityEngine", "Unity", "GameNetcodeStuff", "Dissonance", "HarmonyLib", "MonoMod", "BepInEx", "Mono", "TMPro", "UnityEditor", "DMD", "Steamworks", "Netcode"}

// Analyze reads the load failures out of BepInEx's LogOutput.log and the Unity Player.log. Either may be empty. A
// failure repeated in a log is reported once, at its first line.
func Analyze(logOutput, playerLog string) []Finding {
	var out []Finding
	seen := map[string]bool{}
	add := func(f Finding) {
		if key := f.Source + "|" + f.Kind + "|" + f.Plugin + "|" + f.Message; !seen[key] {
			seen[key] = true
			out = append(out, f)
		}
	}
	for i, line := range strings.Split(logOutput, "\n") {
		m := errorLine.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		f := Finding{Line: i + 1, Source: "LogOutput.log"}
		switch msg := m[2]; {
		case missingDeps.MatchString(msg):
			g := missingDeps.FindStringSubmatch(msg)
			f.Kind, f.Plugin, f.Message = KindMissingDependency, g[1], "missing dependencies: "+g[2]
		case incompatibleDep.MatchString(msg):
			g := incompatibleDep.FindStringSubmatch(msg)
			f.Kind, f.Plugin, f.Message = KindIncompatible, g[1], "incompatible dependencies: "+g[2]
		case requiresBepInEx.MatchString(msg):
			g := requiresBepInEx.FindStringSubmatch(msg)
			f.Kind, f.Plugin, f.Message = KindIncompatible, g[1], msg
		case loadError.MatchString(msg):
			g := loadError.FindStringSubmatch(msg)
			f.Kind, f.Plugin, f.Message = KindLoadException, g[1], g[2]
		case preloaderFail.MatchString(msg):
			f.Kind, f.Plugin, f.Message = KindPreloader, "", msg
		case harmonyError.MatchString(msg):
			g := harmonyError.FindStringSubmatch(msg)
			f.Kind, f.Plugin, f.Message = KindPatchException, rootNamespace(g[1]), g[2]
		default:
			continue
		}
		add(f)
	}
	lines := strings.Split(playerLog, "\n")
	for i, line := range lines {
		m := unityException.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		if plugin := blamedPlugin(lines[i+1:]); plugin != "" {
			add(Finding{Kind: KindUnityException, Plugin: plugin, Message: m[1] + ": " + m[2], Line: i + 1, Source: "Player.log"})
		}
	}
	return out
}

// blamedPlugin is the root namespace of the first stack frame that is not game, Unity or loader code.
func blamedPlugin(stack []string) string {
	for _, l := range stack {
		m := stackFrame.FindStringSubmatch(strings.TrimRight(l, "\r"))
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
