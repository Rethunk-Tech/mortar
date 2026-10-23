package thunderpack

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

// Kinds of Finding, spelled as BepInEx's so the Problems page words and fixes them alike.
const (
	kindMissingDependency = "missing-dependency"
	kindLoadException     = "load-exception"
	kindPatchException    = "patch-exception"
	kindChainloader       = "chainloader"
	kindDuplicate         = "duplicate-guid"
)

// LogSpec is where a loader writes its log and how its lines read. Pattern is a path below the profile folder that may
// end in a glob for a loader that names each run's log anew; Analyze reads one log's text.
type LogSpec struct {
	Pattern string
	Ready   *regexp.Regexp
	Analyze func(text string) []loader.Finding
}

// logged is a Loader whose Spec names a log: the console follows it and its analyzers feed the Problems page.
type logged struct{ Loader }

// LogGlob matches the loader's log files in the profile.
func (l logged) LogGlob(p loader.ProfileView) string {
	return filepath.Join(p.Dir, filepath.FromSlash(l.Log.Pattern))
}

// Path is the newest log file, or the pattern itself while there is none.
func (l logged) Path(p loader.ProfileView) (string, error) {
	glob := l.LogGlob(p)
	if newest := newestFile(glob); newest != "" {
		return newest, nil
	}
	return glob, nil
}

func (l logged) Ready(line string) bool { return l.Log.Ready != nil && l.Log.Ready.MatchString(line) }

// HasReady reports whether the loader's log has a line saying it is done; most thunderpack loaders' logs have none.
func (l logged) HasReady() bool { return l.Log.Ready != nil }

func (l logged) Analyzers() []loader.Analyzer {
	return []loader.Analyzer{analyzer{l.LoaderID, l.Log.Analyze}}
}

type analyzer struct {
	id      string
	analyze func(string) []loader.Finding
}

func (a analyzer) ID() string { return a.id }

func (a analyzer) Analyze(logs loader.Logs) []loader.Finding { return a.analyze(logs.Loader) }

// each calls f with every line of text and its 1-based number.
func each(text string, f func(n int, line string)) {
	for i, line := range strings.Split(text, "\n") {
		f(i+1, strings.TrimRight(line, "\r"))
	}
}

// modDir is the mod folder a path names below a mods or plugins folder, which is how a loader's log names a package.
var modDir = regexp.MustCompile(`[\\/](?:mods|plugins|Mods)[\\/]([^\\/'"]+)`)

func modOf(text string) string {
	if m := modDir.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	return ""
}

var (
	romFailed = regexp.MustCompile(`^\[[^\]]*\]\[(?:ERROR|WARNING)/[^\]]+\] (\S+) failed to load: (.*)$`)
	romBad    = regexp.MustCompile(`^\[[^\]]*\]\[ERROR/[^\]]+\] (Failed reading manifest\.json.*|Invalid (?:soft )?dependency string .*)$`)
)

// analyzeROM reads ReturnOfModding's and Hell2Modding's LogOutput.log: a Lua mod that failed to load names its
// "Owner-Name-Version" and the Lua error, and a bad manifest names its path.
func analyzeROM(text string) []loader.Finding {
	var out []loader.Finding
	each(text, func(n int, line string) {
		if m := romFailed.FindStringSubmatch(line); m != nil {
			out = append(out, loader.Finding{Kind: kindLoadException, Plugin: m[1], Message: m[2], Line: n, Source: "LogOutput.log"})
		} else if m := romBad.FindStringSubmatch(line); m != nil {
			out = append(out, loader.Finding{Kind: kindLoadException, Plugin: modOf(m[1]), Message: m[1], Line: n, Source: "LogOutput.log"})
		}
	})
	return out
}

var (
	gdLoadFail = regexp.MustCompile(`\[(?:WRN|ERR)\] Failed to load mod at (.+)$`)
	gdNoMani   = regexp.MustCompile(`\[WRN\] Mod at (.+) does not have a manifest\.json$`)
	gdMissing  = regexp.MustCompile(`\[WRN\] Mod (\S+) has missing/invalid dependencies: (.*)$`)
	gdCircular = regexp.MustCompile(`\[WRN\] Circular dependency detected: (.*)$`)
	gdAssembly = regexp.MustCompile(`\[WRN\] Failed to load assembly for mod (\S+)`)
	gdRun      = regexp.MustCompile(`\[ERR\] Failed to run mod on (.+)$`)
	gdDup      = regexp.MustCompile(`\[WRN\] Duplicate mod ID: (\S+)`)
	gdInit     = regexp.MustCompile(`\[ERR\] GDWeave failed to initialize`)
	quoted     = regexp.MustCompile(`"([^"]+)"`)
)

// analyzeGDWeave reads GDWeave.log, written by Serilog; the messages are GDWeave's ModLoader and Hooks strings.
func analyzeGDWeave(text string) []loader.Finding {
	var out []loader.Finding
	add := func(kind, plugin, msg string, n int, dep string) {
		out = append(out, loader.Finding{Kind: kind, Plugin: plugin, Message: msg, Dependency: dep, Line: n, Source: "GDWeave.log"})
	}
	each(text, func(n int, line string) {
		switch {
		case gdInit.MatchString(line):
			add(kindChainloader, "GDWeave", "GDWeave failed to initialize", n, "")
		case gdLoadFail.MatchString(line):
			dir := gdLoadFail.FindStringSubmatch(line)[1]
			add(kindLoadException, filepath.Base(strings.ReplaceAll(dir, `\`, "/")), "failed to load", n, "")
		case gdNoMani.MatchString(line):
			dir := gdNoMani.FindStringSubmatch(line)[1]
			add(kindLoadException, filepath.Base(strings.ReplaceAll(dir, `\`, "/")), "has no manifest.json", n, "")
		case gdMissing.MatchString(line):
			m := gdMissing.FindStringSubmatch(line)
			dep := ""
			if q := quoted.FindStringSubmatch(m[2]); q != nil {
				dep = q[1]
			}
			add(kindMissingDependency, m[1], "missing or invalid dependencies: "+m[2], n, dep)
		case gdCircular.MatchString(line):
			add(kindLoadException, "", "circular dependency: "+gdCircular.FindStringSubmatch(line)[1], n, "")
		case gdAssembly.MatchString(line):
			add(kindLoadException, gdAssembly.FindStringSubmatch(line)[1], "its assembly failed to load", n, "")
		case gdRun.MatchString(line):
			add(kindPatchException, "", "a script mod failed on "+gdRun.FindStringSubmatch(line)[1], n, "")
		case gdDup.MatchString(line):
			add(kindDuplicate, gdDup.FindStringSubmatch(line)[1], "another copy has the same mod id", n, "")
		}
	})
	return out
}

var (
	lovelyModule  = regexp.MustCompile(`^ERROR - \[♥\] (?:Failed to load|Evaluation of) module (\S+) .*patch from (.+):$`)
	lovelyNoMatch = regexp.MustCompile(`^WARN - \[♥\] Pattern .* on target '([^']+)' for pattern patch from (.+) resulted in no matches$`)
	lovelyRegex   = regexp.MustCompile(`^WARN - \[♥\] Regex .* on target '([^']+)' for regex patch from (.+) resulted in no matches$`)
	lovelyError   = regexp.MustCompile(`^ERROR - \[♥\] Error: (.*)$`)
)

// analyzeLovely reads a lovely-*.log: a Lua module patch that failed (the next line holds Lua's error) and a patch whose
// pattern does not match the game, which is how a game update breaks a Balatro mod. The mod is the folder of the
// patch file's path.
func analyzeLovely(text string) []loader.Finding {
	var out []loader.Finding
	lines := strings.Split(text, "\n")
	for i, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		switch {
		case lovelyModule.MatchString(line):
			m := lovelyModule.FindStringSubmatch(line)
			msg := "module " + m[1] + " failed"
			code, params := "moduleFailed", map[string]string{"name": m[1]}
			if i+1 < len(lines) {
				if e := lovelyError.FindStringSubmatch(strings.TrimRight(lines[i+1], "\r")); e != nil {
					msg += ": " + e[1]
					code, params["error"] = "moduleFailedWith", e[1]
				}
			}
			out = append(out, loader.Finding{Kind: kindLoadException, Plugin: modOf(m[2]), Message: msg, Line: i + 1, Source: "lovely.log", Code: code, Params: params})
		case lovelyNoMatch.MatchString(line), lovelyRegex.MatchString(line):
			m := lovelyNoMatch.FindStringSubmatch(line)
			if m == nil {
				m = lovelyRegex.FindStringSubmatch(line)
			}
			out = append(out, loader.Finding{
				Kind: kindPatchException, Plugin: modOf(m[2]), Message: "a patch for " + m[1] + " matched nothing in the game's code", Line: i + 1, Source: "lovely.log",
				Code: "patchMatchedNothing", Params: map[string]string{"name": m[1]},
			})
		}
	}
	return out
}

var (
	nsFailed  = regexp.MustCompile(`Mod file at '([^']+)' failed to load`)
	nsRead    = regexp.MustCompile(`Failed reading mod file (.+?): (.*)$`)
	nsNoFile  = regexp.MustCompile(`Mod file at '([^']+)' does not exist or could not be read`)
	nsRpak    = regexp.MustCompile(`Mod (.+?) has invalid rpak\.json`)
	nsVersion = regexp.MustCompile(`Mod '([^']+)' has several versions enabled`)
	nsCompile = regexp.MustCompile(`COMPILE ERROR (.*)$`)
	nsLevel   = regexp.MustCompile(`^\[\d{4}-\d\d-\d\d\] \[[\d:]+\] \[[^\]]+\] \[(warning|error|critical)\] `)
)

// analyzeNorthstar reads an nslog: Northstar's mod manager names a mod by its AUTHOR-MOD-VERSION folder or its name.
func analyzeNorthstar(text string) []loader.Finding {
	var out []loader.Finding
	each(text, func(n int, line string) {
		if !nsLevel.MatchString(line) {
			return
		}
		folder := func(path string) string { return filepath.Base(filepath.Dir(strings.ReplaceAll(path, `\`, "/"))) }
		add := func(kind, plugin, msg string) {
			out = append(out, loader.Finding{Kind: kind, Plugin: plugin, Message: msg, Line: n, Source: "nslog.txt"})
		}
		switch {
		case nsFailed.MatchString(line):
			add(kindLoadException, folder(nsFailed.FindStringSubmatch(line)[1]), "its mod.json failed to load")
		case nsRead.MatchString(line):
			m := nsRead.FindStringSubmatch(line)
			add(kindLoadException, folder(m[1]), m[2])
		case nsNoFile.MatchString(line):
			add(kindLoadException, folder(nsNoFile.FindStringSubmatch(line)[1]), "has no readable mod.json")
		case nsRpak.MatchString(line):
			add(kindLoadException, nsRpak.FindStringSubmatch(line)[1], "its rpak.json is invalid")
		case nsVersion.MatchString(line):
			add(kindDuplicate, nsVersion.FindStringSubmatch(line)[1], "several versions were enabled, so all were turned off")
		case nsCompile.MatchString(line):
			add(kindPatchException, modOf(line), "script compile error "+nsCompile.FindStringSubmatch(line)[1])
		}
	})
	return out
}
