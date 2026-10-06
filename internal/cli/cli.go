// Package cli is Mortar's command line: `mortar <verb> ...` asks the running app over internal/control and prints
// the answer as a table, or as JSON with --json. It never opens a window or touches the data folder itself.
package cli

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Rethunk-Tech/mortar/internal/backup"
	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/controlwire"
	"github.com/Rethunk-Tech/mortar/internal/datadir"
	"github.com/Rethunk-Tech/mortar/internal/datasvc"
	"github.com/Rethunk-Tech/mortar/internal/doctor"
	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/game"
	"github.com/Rethunk-Tech/mortar/internal/launchsvc"
	"github.com/Rethunk-Tech/mortar/internal/loadorder"
	"github.com/Rethunk-Tech/mortar/internal/mod"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/problems"
	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/queue"
	"github.com/Rethunk-Tech/mortar/internal/saves"
	"github.com/Rethunk-Tech/mortar/internal/savessvc"
	"github.com/Rethunk-Tech/mortar/internal/selfexe"
	"github.com/Rethunk-Tech/mortar/internal/sharesvc"
	"github.com/Rethunk-Tech/mortar/internal/tools"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	readTimeout    = 2 * time.Minute
	installTimeout = 10 * time.Minute
	launchTimeout  = 4 * time.Minute
)

// verbs are the first words that make an invocation a command-line call rather than a window launch.
var verbs = map[string]bool{
	"games": true, "game": true, "profiles": true, "profile": true, "history": true, "mods": true, "mod": true, "install": true,
	"conflicts": true, "problems": true, "who": true, "updates": true, "share": true, "export": true, "open": true, "play": true,
	"runs": true, "logs": true, "saves": true, "launch": true, "perf": true, "stop": true, "status": true, "queue": true,
	"templates": true, "library": true, "archive": true,
	"browse":  true,
	"bundles": true, "source": true, "trash": true, "cache": true, "data": true, "store": true, "bisect": true, "lan": true, "app": true, "support": true, "links": true,
	"update": true, "backups": true, "doctor": true, "launchers": true, "tools": true, "settings": true, "loader": true, "sweep": true, "uninstall-cleanup": true, "quit": true, "version": true, "--version": true, "completion": true, "help": true, "--help": true, "-h": true, "__complete": true,
}

// Is reports whether args (without the program name) are a command-line call: a known verb, or a bare word that
// cannot be the link or file path a window launch takes, which then fails as an unknown command.
func Is(args []string) bool {
	if len(args) == 0 {
		return false
	}
	a := args[0]
	return verbs[a] || (a != "" && !strings.HasPrefix(a, "-") && !strings.ContainsAny(a, `/\:.`))
}

// caller sends one request to the running app; tests replace it.
type caller func(method string, p control.Params, out any, timeout time.Duration) error

type cmd struct {
	version       string
	call          caller
	out           io.Writer
	errOut        io.Writer
	json          bool
	verbose       bool
	all           bool
	unused        bool
	missing       bool
	yesFlag       bool
	force         bool
	presetFlag    string
	wait          bool
	vanilla       bool
	byMod         bool
	check         bool
	format        string
	run           string
	game          string
	profileFlag   string
	nameFlag      string
	noConfigs     bool
	updateFlag    bool
	unlinkFlag    bool
	reasonFlag    string
	undoFlag      bool
	changelogFlag bool
	everywhere    bool
	removeFlag    bool
	setFlag       bool
	clearFlag     bool
	testFlag      bool
	filter        string
	item          string
	mark          bool
	restore       bool
	sourceFlag    string
	fromFlag      string
	codeFlag      string
	peerFlag      string
	loaderFlag    string
	previewFlag   bool
	fileFlag      string
	versionFlag   string
	installFlag   string
	pageFlag      int
	keepFlag      string
	deleteFlag    string
	dismissFlag   string
	moveFlag      []string
	args          []string
}

// refusedError is a destructive action blocked until the user passes --yes: exit 2 with the message only.
type refusedError struct{ msg string }

func (e refusedError) Error() string { return e.msg }

// usageError is a malformed command line: exit 2 with the usage text.
type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

type offlineDoctorError struct {
	result map[string]any
	code   int
}

func (e offlineDoctorError) Error() string { return "doctor ran offline" }

// Run executes one command and returns the process exit code.
func Run(version string, args []string, stdout, stderr io.Writer) int {
	attachConsole()
	return run(version, control.Call, args, stdout, stderr)
}

func run(version string, call caller, args []string, stdout, stderr io.Writer) int {
	c := &cmd{version: version, call: call, out: stdout, errOut: stderr}
	if err := c.parse(args); err != nil {
		return c.fail(err)
	}
	if err := c.dispatch(); err != nil {
		return c.fail(err)
	}
	return 0
}

func (c *cmd) fail(err error) int {
	if _, ok := errors.AsType[playCheckError](err); ok {
		return 3
	}
	if offline, ok := errors.AsType[offlineDoctorError](err); ok {
		if c.json {
			_ = json.NewEncoder(c.out).Encode(offline.result)
		} else {
			fmt.Fprintln(c.out, offline.result["summary"])
			findings, _ := offline.result["findings"].([]string)
			for _, f := range findings {
				fmt.Fprintln(c.out, f)
			}
		}
		return offline.code
	}
	code := 1
	if _, ok := errors.AsType[usageError](err); ok {
		code = 2
	}
	if _, ok := errors.AsType[refusedError](err); ok {
		code = 2
	}
	if errors.Is(err, controlwire.ErrNotRunning) {
		code = 3
	}
	kind, raw := usererr.Parse(err.Error())
	if kind == usererr.Unknown {
		kind = usererr.KindOf(err)
		raw = err.Error()
	}
	if c.json {
		_ = json.NewEncoder(c.errOut).Encode(map[string]any{"error": raw, "kind": string(kind), "code": code})
		return code
	}
	shown := err.Error()
	if !c.verbose && code != 2 {
		shown = Sentence(kind)
		if kind == usererr.Unknown {
			shown = raw
		}
	} else if c.verbose {
		shown = raw
	}
	if code == 2 {
		fmt.Fprintln(c.errOut, "mortar:", err)
		if _, ok := errors.AsType[usageError](err); ok {
			fmt.Fprint(c.errOut, usage)
		}
		return code
	}
	fmt.Fprintln(c.errOut, "mortar:", shown)
	return code
}

func (c *cmd) parse(args []string) error {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			c.json = true
		case a == "-v", a == "--verbose":
			c.verbose = true
		case a == "--all":
			c.all = true
		case a == "--filter":
			if i+1 >= len(args) {
				return usageError{"--filter needs a value"}
			}
			i++
			c.filter = args[i]
		case strings.HasPrefix(a, "--filter="):
			c.filter = strings.TrimPrefix(a, "--filter=")
		case a == "--unused":
			c.unused = true
		case a == "--missing":
			c.missing = true
		case a == "--by-mod":
			c.byMod = true
		case a == "--yes":
			c.yesFlag = true
		case a == "--force":
			c.force = true
		case a == "--preset":
			if i+1 >= len(args) {
				return usageError{"--preset needs a name"}
			}
			i++
			c.presetFlag = args[i]
		case strings.HasPrefix(a, "--preset="):
			c.presetFlag = strings.TrimPrefix(a, "--preset=")
		case a == "--item":
			if i+1 >= len(args) {
				return usageError{"--item needs a mod"}
			}
			i++
			c.item = args[i]
		case strings.HasPrefix(a, "--item="):
			c.item = strings.TrimPrefix(a, "--item=")
		case a == "--mark":
			c.mark = true
		case a == "--restore":
			c.restore = true
		case a == "--check":
			c.check = true
		case a == "--wait":
			c.wait = true
		case a == "--vanilla":
			c.vanilla = true
		case a == "--update":
			c.updateFlag = true
		case a == "--unlink":
			c.unlinkFlag = true
		case a == "--reason":
			if i+1 >= len(args) {
				return usageError{"--reason needs a value"}
			}
			i++
			c.reasonFlag = args[i]
		case strings.HasPrefix(a, "--reason="):
			c.reasonFlag = strings.TrimPrefix(a, "--reason=")
		case a == "--undo":
			c.undoFlag = true
		case a == "--changelog":
			c.changelogFlag = true
		case a == "--everywhere":
			c.everywhere = true
		case a == "--remove":
			c.removeFlag = true
		case a == "--set":
			c.setFlag = true
		case a == "--clear":
			c.clearFlag = true
		case a == "--test":
			c.testFlag = true
		case a == "--format":
			if i+1 >= len(args) {
				return usageError{"--format needs md or text"}
			}
			i++
			c.format = args[i]
		case strings.HasPrefix(a, "--format="):
			c.format = strings.TrimPrefix(a, "--format=")
		case a == "--run":
			if i+1 >= len(args) {
				return usageError{"--run needs a run id"}
			}
			i++
			c.run = args[i]
		case strings.HasPrefix(a, "--run="):
			c.run = strings.TrimPrefix(a, "--run=")
		case a == "--profile":
			if i+1 >= len(args) {
				return usageError{"--profile needs a profile name"}
			}
			i++
			c.profileFlag = args[i]
		case strings.HasPrefix(a, "--profile="):
			c.profileFlag = strings.TrimPrefix(a, "--profile=")
		case a == "--no-configs":
			c.noConfigs = true
		case a == "--name":
			if i+1 >= len(args) {
				return usageError{"--name needs a name"}
			}
			i++
			c.nameFlag = args[i]
		case strings.HasPrefix(a, "--name="):
			c.nameFlag = strings.TrimPrefix(a, "--name=")
		case a == "--game":
			if i+1 >= len(args) {
				return usageError{"--game needs a game id"}
			}
			i++
			c.game = args[i]
		case strings.HasPrefix(a, "--game="):
			c.game = strings.TrimPrefix(a, "--game=")
		case a == "--loader":
			if i+1 >= len(args) {
				return usageError{"--loader needs a loader id"}
			}
			i++
			c.loaderFlag = args[i]
		case strings.HasPrefix(a, "--loader="):
			c.loaderFlag = strings.TrimPrefix(a, "--loader=")
		case a == "--install":
			if i+1 >= len(args) {
				return usageError{"--install needs an install id"}
			}
			i++
			c.installFlag = args[i]
		case strings.HasPrefix(a, "--install="):
			c.installFlag = strings.TrimPrefix(a, "--install=")
		case a == "--preview":
			c.previewFlag = true
		case a == "--file":
			if i+1 >= len(args) {
				return usageError{"--file needs a value"}
			}
			i++
			c.fileFlag = args[i]
		case strings.HasPrefix(a, "--file="):
			c.fileFlag = strings.TrimPrefix(a, "--file=")
		case a == "--version" && i == 0:
			// Leading, it is the usual spelling of the version command; later, it picks a mod version.
			c.args = append(c.args, "version")
		case a == "--version":
			if i+1 >= len(args) {
				return usageError{"--version needs a value"}
			}
			i++
			c.versionFlag = args[i]
		case strings.HasPrefix(a, "--version="):
			c.versionFlag = strings.TrimPrefix(a, "--version=")
		case a == "--from":
			if i+1 >= len(args) {
				return usageError{"--from needs a mod manager (vortex or mo2)"}
			}
			i++
			c.fromFlag = args[i]
		case strings.HasPrefix(a, "--from="):
			c.fromFlag = strings.TrimPrefix(a, "--from=")
		case a == "--source":
			if i+1 >= len(args) {
				return usageError{"--source needs a source id"}
			}
			i++
			c.sourceFlag = args[i]
		case strings.HasPrefix(a, "--source="):
			c.sourceFlag = strings.TrimPrefix(a, "--source=")
		case a == "--code" || a == "--peer":
			if i+1 >= len(args) {
				return usageError{a + " needs a value"}
			}
			i++
			if a == "--code" {
				c.codeFlag = args[i]
			} else {
				c.peerFlag = args[i]
			}
		case strings.HasPrefix(a, "--code="):
			c.codeFlag = strings.TrimPrefix(a, "--code=")
		case strings.HasPrefix(a, "--peer="):
			c.peerFlag = strings.TrimPrefix(a, "--peer=")
		case a == "--page":
			if i+1 >= len(args) {
				return usageError{"--page needs a number"}
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return usageError{"--page needs a number"}
			}
			c.pageFlag = n
		case strings.HasPrefix(a, "--page="):
			n, err := strconv.Atoi(strings.TrimPrefix(a, "--page="))
			if err != nil {
				return usageError{"--page needs a number"}
			}
			c.pageFlag = n
		case a == "--keep" || a == "--delete" || a == "--dismiss" || a == "--move":
			if i+1 >= len(args) {
				return usageError{a + " needs a value"}
			}
			i++
			switch a {
			case "--keep":
				c.keepFlag = args[i]
			case "--delete":
				c.deleteFlag = args[i]
			case "--dismiss":
				c.dismissFlag = args[i]
			default:
				c.moveFlag = append(c.moveFlag, args[i])
			}
		case a == "--help" || a == "-h":
			c.args = append(c.args, "help")
		case strings.HasPrefix(a, "--"):
			return usageError{"unknown flag " + a}
		default:
			c.args = append(c.args, a)
		}
	}
	return nil
}

// need returns the n positional arguments after the verb words, naming them in the error.
func (c *cmd) need(skip int, names ...string) ([]string, error) {
	got := c.args[skip:]
	if len(got) < len(names) {
		return nil, usageError{fmt.Sprintf("%s needs %s", strings.Join(c.args[:skip], " "), strings.Join(names[len(got):], ", "))}
	}
	return got, nil
}

// show runs a read verb and prints the decoded result as JSON or through human.
func show[T any](c *cmd, method string, p control.Params, human func(T)) error {
	var v T
	if err := c.call(method, p, &v, readTimeout); err != nil {
		return err
	}
	return c.emit(v, func() { human(v) })
}

// emit prints v as JSON with --json, otherwise calls human.
func (c *cmd) emit(v any, human func()) error {
	if c.json {
		enc := json.NewEncoder(c.out)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
	human()
	return nil
}

func (c *cmd) table(header string, rows [][]string) {
	tw := tabwriter.NewWriter(c.out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, header)
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	_ = tw.Flush()
}

func (c *cmd) dispatch() error {
	verb := c.args[0]
	if verbs[verb] {
		if handled, err := c.more(); handled {
			return err
		}
	}
	if !verbs[verb] {
		return usageError{"unknown command " + verb}
	}
	switch verb {
	case "help", "--help", "-h":
		fmt.Fprint(c.out, usage)
		return nil
	case "version":
		return c.emit(map[string]string{"version": c.version}, func() { fmt.Fprintln(c.out, "mortar", c.version) })
	case "completion":
		a, err := c.need(1, "a shell (bash, zsh or fish)")
		if err != nil {
			return err
		}
		return completion(c.out, a[0])
	case "__complete":
		return c.complete(c.args[1:])
	case "open":
		a, err := c.need(1, "a share link or .mortar file")
		if err != nil {
			return err
		}
		return open(a[0])
	case "games":
		return c.games()
	case "game":
		return c.gameCmd()
	case "doctor":
		return c.doctor()
	case "uninstall-cleanup":
		return c.uninstallCleanup()
	case "quit":
		return c.quit()
	case "sweep":
		return c.sweep()
	case "launchers":
		return c.launchers()
	case "queue":
		return c.queue()
	case "templates":
		return c.templatesCmd()
	case "library":
		return c.libraryCmd()
	case "archive":
		return c.archiveCmd()
	case "browse":
		return c.browse()
	case "update":
		return c.update()
	case "backups":
		return c.backups()
	case "loader":
		return c.loader()
	case "settings":
		return c.settings()
	case "bundles":
		return c.bundles()
	case "source":
		return c.source()
	case "bisect":
		return c.bisectCmd()
	case "trash":
		return c.trash()
	case "cache":
		return c.cacheCmd()
	case "store":
		return c.storeCmd()
	case "data":
		return c.dataCmd()
	case "profiles":
		a, err := c.need(1, "a game")
		if err != nil {
			return err
		}
		return c.profiles(a[0])
	case "history":
		return c.historyCmd()
	case "tools":
		return c.tools()
	case "profile":
		if len(c.args) > 1 {
			switch c.args[1] {
			case "set":
				return c.profileSet()
			case "shortcut":
				return c.profileShortcut()
			case "steam":
				return c.profileSteam()
			case "changes":
				return c.profileChanges()
			case "good":
				return c.profileGood()
			case "import":
				return c.profileImport()
			case "export":
				return c.profileExport()
			case "farm":
				return c.profileFarm()
			case "backup":
				return c.profileBackup()
			case "restore":
				return c.profileRestore()
			}
		}
		return c.profile()
	case "status", "stop":
		a, err := c.need(1, "a game")
		if err != nil {
			return err
		}
		return c.status(verb, a[0])
	case "play":
		return c.play()
	case "mods":
		if len(c.args) > 1 {
			switch c.args[1] {
			case "enable", "disable", "remove", "pin", "unpin", "tag", "untag", "category", "note", "skip-version", "split", "combine":
				return c.modsChange(c.args[1])
			case "group":
				return c.modsGroup()
			case "channel":
				return c.modsChannel()
			case "win":
				return c.modsWin()
			case "files":
				return c.modsFiles()
			case "config":
				return c.modsConfig()
			case "preset":
				return c.modsPreset()
			case "menu":
				return c.modsMenu()
			case "compat":
				return c.modsCompat()
			case "report":
				return c.modsReport()
			case "by-author":
				return c.modsByAuthor()
			}
		}
	}
	if verb == "logs" && len(c.args) >= 2 && c.args[1] == "share" {
		return c.shareLog()
	}
	if verb == "logs" && len(c.args) >= 2 && c.args[1] == "fixes" {
		return c.logFixes()
	}
	if verb == "logs" && len(c.args) >= 3 && c.args[1] == "search" {
		return c.searchLogs(c.args[2])
	}
	if verb == "saves" && len(c.args) > 1 && c.args[1] == "check" {
		return c.savesCheck()
	}
	if verb == "problems" && len(c.args) > 1 {
		switch c.args[1] {
		case "dismissed":
			return c.problemsDismissed()
		case "dismiss":
			return c.problemsDismiss()
		case "restore":
			return c.problemsRestore()
		}
	}
	if verb == "who" {
		return c.who()
	}
	if verb == "launch" && c.vanilla {
		a, err := c.need(1, "a game")
		if err != nil {
			return err
		}
		var st launchsvc.Status
		if err := c.call("launch.vanilla", control.Params{Game: a[0]}, &st, launchTimeout); err != nil {
			return err
		}
		return c.emit(st, func() { fmt.Fprintf(c.out, "%s is %s without mods.\n", a[0], st.State) })
	}
	if verb == "perf" {
		return c.perfReports()
	}
	if verb == "conflicts" && len(c.args) > 1 && c.args[1] == "map" {
		return c.conflictsMap()
	}
	if verb == "updates" && len(c.args) > 1 && c.args[1] == "apply" {
		return c.updatesApply()
	}
	a, err := c.need(1, "a game", "a profile")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], All: c.all, Run: c.run, Force: c.force, Preset: c.presetFlag, Install: c.installFlag}
	switch verb {
	case "mods":
		return c.mods(p)
	case "mod":
		if len(a) < 3 {
			return usageError{"mod needs a mod id (SMAPI id)"}
		}
		p.IDs = a[2:3]
		return c.mod(p)
	case "install":
		if len(a) < 3 {
			return usageError{"install needs an archive path"}
		}
		p.Path = absPath(a[2])
		return c.install(p)
	case "export":
		if len(a) < 3 {
			return usageError{"export needs the .mortar file to write"}
		}
		p.Path = absPath(a[2])
		return c.export(p)
	case "conflicts":
		return c.conflicts(p)
	case "problems":
		return c.problems(p)
	case "updates":
		return c.updates(p)
	case "share":
		return c.share(p)
	case "runs":
		return c.runs(p)
	case "logs":
		return c.logs(p)
	case "saves":
		return c.saves(p)
	case "launch":
		return c.launch(p)
	}
	return usageError{"unknown command " + verb}
}

func absPath(p string) string {
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

// open hands a share link or .mortar file to the app the way a browser or file manager would: a second instance
// forwards it to the running window, or starts Mortar when none runs.
func open(target string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if !strings.Contains(target, "://") {
		target = absPath(target)
	}
	exe = selfexe.Launchable(exe)
	proc, err := os.StartProcess(exe, []string{exe, target}, &os.ProcAttr{Files: []*os.File{nil, nil, nil}})
	if err != nil {
		return err
	}
	return proc.Release()
}

// missingName names a missing requirement by its page when known, with the minimum version it needs.
func missingName(m problems.Missing) string {
	name := m.ID.Local()
	if m.Where != nil && m.Where.PageName != "" {
		name = m.Where.PageName + " (" + m.ID.Local() + ")"
	}
	if m.MinimumVersion != "" {
		name += " " + m.MinimumVersion + "+"
	}
	if m.Optional {
		name += ", optional"
	}
	return name
}

func yes(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func gameSupportedLabel(ok bool) string {
	if ok {
		return "Supported"
	}
	return "Not supported"
}

func gameConfiguredLabel(ok bool) string {
	if ok {
		return "Installed"
	}
	return "Not set up"
}

// historySummary words a history event in English, as the window words it in the user's language.
func historySummary(ev profile.HistoryEvent) string {
	mods := func(n int) string {
		if n == 1 {
			return "1 mod"
		}
		return fmt.Sprintf("%d mods", n)
	}
	switch ev.Change {
	case profile.ChangeAdded:
		return "Added " + ev.Name
	case profile.ChangeRemoved:
		return "Removed " + ev.Name
	case profile.ChangeUpdated:
		return fmt.Sprintf("Updated %s from %s to %s", ev.Name, ev.From, ev.To)
	case profile.ChangeEnabled:
		return "Enabled " + ev.Name
	case profile.ChangeDisabled:
		return "Disabled " + ev.Name
	case profile.ChangePinned:
		return "Pinned " + ev.Name + " at its version"
	case profile.ChangeUnpinned:
		return "Unpinned " + ev.Name
	case profile.ChangeTagged:
		return fmt.Sprintf("Tagged %s with %s", ev.Name, ev.Detail)
	case profile.ChangeUntagged:
		return fmt.Sprintf("Removed the tag %s from %s", ev.Detail, ev.Name)
	case profile.ChangeTags:
		return "Changed the tags of " + ev.Name
	case profile.ChangeNote:
		return "Edited the note of " + ev.Name
	case profile.ChangeMods:
		if ev.Count > 1 {
			return "Changed " + mods(ev.Count)
		}
		return "Changed mods"
	case profile.ChangeImported:
		return "Imported " + mods(ev.Count)
	case profile.ChangeMoved:
		return "Moved " + mods(ev.Count) + " from the game's Mods folder"
	case profile.ChangeRestored:
		return "Restored " + mods(ev.Count)
	case profile.ChangeRestoredFromStore:
		if ev.Name != "" {
			return "Restored " + ev.Name + " from the store"
		}
		return "Restored " + mods(ev.Count) + " from the store"
	case profile.ChangeBeforeEdit:
		return "Before this change"
	case profile.ChangeReverted:
		return "Went back to " + ev.Target.Local().Format("2006-01-02 15:04")
	case profile.ChangeRestoredFile:
		return fmt.Sprintf("Restored %s of %s", ev.Detail, ev.Name)
	case profile.ChangeConfigEdited:
		return "Edited the settings of " + ev.Name
	case profile.ChangeConfigReset:
		return "Reset the settings of " + ev.Name
	case profile.ChangePresetApplied:
		return fmt.Sprintf("Applied the preset %s to %s", ev.Detail, ev.Name)
	case profile.ChangeOptionSet:
		return fmt.Sprintf("Set %s of %s for the next start", ev.Detail, ev.Name)
	case profile.ChangeCategoryRemoved:
		return "Deleted a custom category"
	case profile.ChangeChannel:
		return fmt.Sprintf("Switched %s to the %s update channel", ev.Name, ev.Detail)
	case profile.ChangeCollectionUnlinked:
		if ev.Name != "" {
			return "Unlinked the collection " + ev.Name
		}
		return "Unlinked the collection"
	case profile.ChangeTrimmed:
		return fmt.Sprintf("Trimmed history, dropped %d older changes", ev.Count)
	case profile.ChangeKnownGood:
		return "Known good"
	case profile.ChangeGroups:
		return "Changed groups"
	case profile.ChangeLoader:
		return "Changed loader"
	case profile.ChangeInstall:
		return "Changed game install"
	case profile.ChangeSaves:
		return "Changed separate saves"
	case profile.ChangeLaunch:
		return "Changed launch settings"
	case profile.ChangeSettings:
		return "Changed profile settings"
	}
	return "Changed the profile"
}

func modNamesForIDs(ids []string, mods []control.ModRow) []string {
	names := make(map[string]string, len(mods))
	for _, m := range mods {
		if m.ID != "" && m.Name != "" {
			names[m.ID.Local()] = m.Name
		}
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if n := names[id]; n != "" {
			out = append(out, n)
			continue
		}
		out = append(out, id)
	}
	return out
}

func countedMissing(r problems.Result) int {
	n := 0
	for _, x := range r.Missing {
		if x.Optional || x.Listed {
			continue
		}
		n++
	}
	return n
}

func harmlessMissing(r problems.Result) int {
	n := 0
	for _, x := range r.Missing {
		if x.Optional || x.Listed {
			n++
		}
	}
	return n
}

// problemsHumanCount is Count without the listed requirements, which the CLI reports as harmless. Count already
// leaves out optional and outside requirements, so those are not subtracted again.
func problemsHumanCount(r problems.Result) int {
	listed := 0
	for _, x := range r.Missing {
		if x.Listed && !x.Optional && !x.External {
			listed++
		}
	}
	return r.Count() - listed
}

// loaderColumn heads the runs' loader version column with the name of the loader they used, LOADER when they used
// different ones or it is unknown. A run that recorded no loader used the game's only one.
func loaderColumn(game string, list []launchsvc.Run) string {
	g, _ := components.Game(game)
	name := ""
	for _, r := range list {
		id := r.Loader
		if id == "" && len(g.Loaders) == 1 {
			id = g.Loaders[0].ID
		}
		i := slices.IndexFunc(g.Loaders, func(l components.GameLoader) bool { return l.ID == id })
		if i < 0 || (name != "" && name != g.Loaders[i].Name) {
			return "LOADER"
		}
		name = g.Loaders[i].Name
	}
	return strings.ToUpper(cmp.Or(name, "Loader"))
}

func runStartedLabel(started string) string {
	if started == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04"} {
		if t, err := time.Parse(layout, started); err == nil {
			return relativeDeleted(t)
		}
		if t, err := time.ParseInLocation(layout, started, time.Local); err == nil {
			return relativeDeleted(t)
		}
	}
	return started
}

func runOutcomeLabel(outcome string) string {
	switch strings.ToLower(outcome) {
	case "ran", "ok", "success", "succeeded", "exited":
		return "Ran"
	case "crashed", "crash":
		return "Crashed"
	case "failed", "fail", "error":
		return "Failed"
	default:
		if outcome == "" {
			return "Ran"
		}
		return outcome
	}
}

func runCauseLine(r launchsvc.Run) string {
	if r.Cause == nil {
		return ""
	}
	kind := usererr.Unknown
	if r.Cause.Reason != "" {
		kind, _ = usererr.Parse(r.Cause.Reason)
	}
	if kind == usererr.Unknown && r.Cause.Detail != "" {
		kind, _ = usererr.Parse(r.Cause.Detail)
	}
	line := Sentence(kind)
	if r.Cause.ModName != "" {
		return r.Cause.ModName + ": " + line
	}
	return line
}

func saveLastPlayed(s savessvc.Fit, profileNames map[string]string) string {
	name := profileNames[s.LastProfileID]
	var when time.Time
	if s.LastProfileAt > 0 {
		when = time.UnixMilli(s.LastProfileAt)
	}
	if name == "" && when.IsZero() {
		return ""
	}
	if name == "" {
		return relativeDeleted(when)
	}
	if when.IsZero() {
		return name
	}
	return name + " · " + relativeDeleted(when)
}

func (c *cmd) games() error {
	return show(c, "games", control.Params{}, func(rows []control.GameRow) {
		t := [][]string{}
		for _, g := range rows {
			t = append(t, []string{g.ID, gameSupportedLabel(g.Available), gameConfiguredLabel(g.Configured), fmt.Sprint(g.Profiles), g.Store, g.InstallDir})
		}
		c.table("GAME\tSUPPORTED\tCONFIGURED\tPROFILES\tSTORE\tFOLDER", t)
	})
}

func (c *cmd) profiles(gameID string) error {
	return show(c, "profiles", control.Params{Game: gameID}, func(list []profile.Profile) {
		t := [][]string{}
		for _, p := range list {
			if p.Error != "" {
				t = append(t, []string{p.Name, "Could not read this profile", "", ""})
				continue
			}
			on, total := 0, 0
			for _, e := range p.Entries {
				total += len(e.Mods)
				on += len(e.Mods) - len(e.Disabled)
			}
			t = append(t, []string{p.ID, p.Name, fmt.Sprintf("%d/%d", on, total), p.Updated.Local().Format("2006-01-02 15:04")})
		}
		c.table("ID\tNAME\tENABLED\tUPDATED", t)
	})
}

func (c *cmd) historyAll() error {
	if !c.all {
		return usageError{"history needs --all"}
	}
	a, err := c.need(1, "a game")
	if err != nil {
		return err
	}
	return show(c, "history.all", control.Params{Game: a[0]}, func(rows []profile.RecentEvent) {
		t := [][]string{}
		for _, row := range rows {
			t = append(t, []string{
				row.ProfileName, row.At.Local().Format("2006-01-02 15:04"), historySummary(row.HistoryEvent),
			})
		}
		c.table("PROFILE\tTIME\tSUMMARY", t)
	})
}

func (c *cmd) bundles() error {
	if len(c.args) > 1 && c.args[1] == "apply" {
		a, err := c.need(2, "a game", "a bundle", "a profile")
		if err != nil {
			return err
		}
		return show(c, "bundles.apply", control.Params{Game: a[0], Name: a[1], Profile: a[2]}, func(result control.BundleApply) {
			fmt.Fprintf(c.out, "Added %d mods.\n", result.Added)
			if len(result.Missing) > 0 {
				fmt.Fprintf(c.out, "Not in Mortar's store: %s\n", strings.Join(result.Missing, ", "))
			}
		})
	}
	a, err := c.need(1, "a game")
	if err != nil {
		return err
	}
	return show(c, "bundles", control.Params{Game: a[0]}, func(list []control.BundleRow) {
		rows := make([][]string, 0, len(list))
		for _, b := range list {
			rows = append(rows, []string{b.ID, b.Name, fmt.Sprint(len(b.Mods)), strings.Join(b.Profiles, ", ")})
		}
		c.table("ID\tNAME\tMODS\tPROFILES WITH ALL", rows)
	})
}

// gameOrDefault is --game, else the only installed game Mortar implements; with none or several it asks for --game.
func (c *cmd) gameOrDefault() (string, error) {
	if c.game != "" {
		return c.game, nil
	}
	var games []game.GameInfo
	if err := c.call("games", control.Params{}, &games, readTimeout); err != nil {
		return "", err
	}
	var ids []string
	for _, g := range games {
		if g.Installed && game.Find(g.ID) != nil {
			ids = append(ids, g.ID)
		}
	}
	if len(ids) != 1 {
		return "", usageError{"more than one game, or none, is installed: pass --game <id>"}
	}
	return ids[0], nil
}

func relativeDeleted(when time.Time) string {
	s := time.Since(when)
	switch {
	case s < time.Minute:
		return "just now"
	case s < time.Hour:
		n := s / time.Minute
		if n == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", n)
	case s < 24*time.Hour:
		n := s / time.Hour
		if n == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", n)
	default:
		n := s / (24 * time.Hour)
		if n == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", n)
	}
}

func (c *cmd) settings() error {
	if len(c.args) < 2 {
		return usageError{"settings needs get, set, export, import or reset"}
	}
	switch c.args[1] {
	case "export":
		a, err := c.need(2, "a file")
		if err != nil {
			return err
		}
		path := absPath(a[0])
		return show(c, "settings.export", control.Params{Path: path}, func(written map[string]string) { fmt.Fprintf(c.out, "Wrote %s.\n", path) })
	case "import":
		a, err := c.need(2, "a file")
		if err != nil {
			return err
		}
		path := absPath(a[0])
		if err := c.call("settings.import", control.Params{Path: path}, nil, readTimeout); err != nil {
			return err
		}
		return c.emit(map[string]string{"path": path}, func() { fmt.Fprintf(c.out, "Imported %s.\n", path) })
	case "reset":
		key := ""
		if len(c.args) > 2 {
			key = c.args[2]
		}
		if err := c.call("settings.reset", control.Params{Key: key, Game: c.game}, nil, readTimeout); err != nil {
			return err
		}
		return c.emit(map[string]string{"key": key, "game": c.game}, func() {
			if key == "" {
				fmt.Fprintln(c.out, "Settings reset.")
				return
			}
			fmt.Fprintf(c.out, "Reset %s.\n", key)
		})
	case "get":
		key := ""
		if len(c.args) > 2 {
			key = c.args[2]
		}
		return show(c, "settings.get", control.Params{Key: key, Game: c.game}, func(rows [][2]string) {
			out := make([][]string, 0, len(rows))
			for _, r := range rows {
				out = append(out, []string{r[0], r[1]})
			}
			c.table("KEY\tVALUE", out)
		})
	case "set":
		if len(c.args) < 4 {
			return usageError{"settings set needs a key and a value"}
		}
		return c.call("settings.set", control.Params{Key: c.args[2], Value: strings.Join(c.args[3:], " "), Game: c.game}, nil, readTimeout)
	default:
		return usageError{"unknown settings command " + c.args[1]}
	}
}

func (c *cmd) trash() error {
	if len(c.args) < 2 {
		return usageError{"trash needs list, restore, delete or empty"}
	}
	game, err := c.gameOrDefault()
	if err != nil {
		return err
	}
	sub := c.args[1]
	switch sub {
	case "list":
		return show(c, "trash.list", control.Params{Game: game}, func(items []profile.TrashItem) {
			if len(items) == 0 {
				fmt.Fprintln(c.out, "Trash is empty.")
				return
			}
			rows := [][]string{}
			for _, item := range items {
				left := fmt.Sprintf("%d days left", item.DaysLeft)
				if item.DaysLeft == 1 {
					left = "1 day left"
				}
				rows = append(rows, []string{item.Name, relativeDeleted(item.DeletedAt), left})
			}
			c.table("NAME\tDELETED\tDAYS LEFT", rows)
		})
	case "restore":
		a, err := c.need(2, "a deleted profile name or id")
		if err != nil {
			return err
		}
		return show(c, "trash.restore", control.Params{Game: game, Profile: a[0]}, func(p profile.Profile) { fmt.Fprintf(c.out, "Restored %s (%s).\n", p.Name, p.ID) })
	case "delete":
		a, err := c.need(2, "a deleted profile name or id")
		if err != nil {
			return err
		}
		if !c.yesFlag {
			return refusedError{"permanently deleting a profile needs --yes"}
		}
		return show(c, "trash.delete", control.Params{Game: game, Profile: a[0]}, func(r control.Removed) {
			fmt.Fprintf(c.out, "Permanently deleted %s.\n", strings.Join(r.Mods, ", "))
		})
	case "empty":
		if !c.yesFlag {
			return refusedError{"emptying trash needs --yes"}
		}
		if err := c.call("trash.empty", control.Params{Game: game}, nil, readTimeout); err != nil {
			return err
		}
		return c.emit(map[string]bool{"emptied": true}, func() { fmt.Fprintln(c.out, "Trash emptied.") })
	default:
		return usageError{"unknown trash command " + sub}
	}
}

func (c *cmd) source() error {
	if len(c.args) < 2 {
		return usageError{"source needs untrack or tracked"}
	}
	if c.sourceFlag == "" {
		return usageError{"source " + c.args[1] + " needs --source <id>"}
	}
	switch c.args[1] {
	case "untrack":
		return c.sourceUntrack()
	case "tracked":
		return c.sourceTracked()
	default:
		return usageError{"unknown source command " + c.args[1]}
	}
}

func (c *cmd) profile() error {
	if len(c.args) < 2 {
		return usageError{"profile needs create, from-save, rename, copy, compare, match, collection, history, health, revert, load-order, repair, list, shortcut, steam, delete, changes, good, import, export, backup or restore"}
	}
	sub := c.args[1]
	var p profile.Profile
	switch sub {
	case "list":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		if c.format != "md" && c.format != "text" {
			return usageError{"profile list needs --format md or text"}
		}
		var rows []control.ModRow
		if err := c.call("mods", control.Params{Game: a[0], Profile: a[1]}, &rows, readTimeout); err != nil {
			return err
		}
		return c.emit(enabledMods(rows), func() { c.printEnabledMods(rows, a[0]) })
	case "compare":
		a, err := c.need(2, "a game", "a profile", "a second profile")
		if err != nil {
			return err
		}
		return show(c, "profile.compare", control.Params{Game: a[0], Profile: a[1], Name: a[2]}, func(diff profile.CLICompare) { c.compareTable(diff) })
	case "match":
		a, err := c.need(2, "a game", "a profile", "a link or .mortar file")
		if err != nil {
			return err
		}
		return show(c, "profile.match", control.Params{Game: a[0], Profile: a[1], Path: a[2]}, func(match control.ProfileMatch) {
			fmt.Fprintf(c.out, "%d mods already match %s.\n", match.Already, a[1])
			if len(match.Missing) > 0 {
				fmt.Fprintf(c.out, "Missing in %s: %s\n", a[1], strings.Join(match.Missing, ", "))
			}
			if len(match.Different) > 0 {
				fmt.Fprintf(c.out, "Different version: %s\n", strings.Join(match.Different, ", "))
			}
			if len(match.OnlyYours) > 0 {
				fmt.Fprintf(c.out, "Only in yours: %s\n", strings.Join(match.OnlyYours, ", "))
			}
		})
	case "collection":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		if c.unlinkFlag && c.updateFlag {
			return usageError{"profile collection: use either --update or --unlink"}
		}
		p := control.Params{Game: a[0], Profile: a[1], All: c.updateFlag, Unlink: c.unlinkFlag}
		if c.unlinkFlag {
			return show(c, "profile.collection", p, func(prof profile.Profile) { fmt.Fprintln(c.out, "unlinked collection") })
		}
		if c.updateFlag {
			var res sharesvc.Result
			if err := c.call("profile.collection", p, &res, installTimeout); err != nil {
				return err
			}
			return c.emit(res, func() { fmt.Fprintf(c.out, "Queued %d downloads.\n", res.Queued) })
		}
		return show(c, "profile.collection", p, func(st sharesvc.CollectionStatus) {
			if !st.Linked {
				fmt.Fprintln(c.out, "not from a collection")
				return
			}
			fmt.Fprintf(c.out, "%s\n%s\nrevision %d\nlatest %d\n", st.Name, st.URL, st.Revision, st.Latest)
		})
	case "load-order":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		return show(c, "profile.loadOrder", control.Params{Game: a[0], Profile: a[1]}, func(order []loadorder.Row) { c.printLoadOrder(order) })
	case "history":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		return show(c, "profile.history", control.Params{Game: a[0], Profile: a[1]}, func(rows []profile.HistoryEvent) {
			t := [][]string{}
			for _, row := range rows {
				t = append(t, []string{row.At.Local().Format("2006-01-02 15:04"), historySummary(row)})
			}
			c.table("TIME\tSUMMARY", t)
		})
	case "health":
		return c.profileHealth()
	case "revert":
		a, err := c.need(2, "a game", "a profile", "an event id")
		if err != nil {
			return err
		}
		if err := c.call("profile.revert", control.Params{Game: a[0], Profile: a[1], Name: a[2]}, &p, readTimeout); err != nil {
			return err
		}
	case "from-save":
		return c.profileFromSave()
	case "create":
		a, err := c.need(2, "a game", "a name")
		if err != nil {
			return err
		}
		if err := c.call("profile.create", control.Params{Game: a[0], Name: strings.Join(a[1:], " ")}, &p, readTimeout); err != nil {
			return err
		}
	case "rename":
		a, err := c.need(2, "a game", "a profile", "a new name")
		if err != nil {
			return err
		}
		if err := c.call("profile.rename", control.Params{Game: a[0], Profile: a[1], Name: strings.Join(a[2:], " ")}, &p, readTimeout); err != nil {
			return err
		}
	case "copy":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		if err := c.call("profile.copy", control.Params{Game: a[0], Profile: a[1], Name: strings.Join(a[2:], " ")}, &p, readTimeout); err != nil {
			return err
		}
	case "delete":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		return show(c, "profile.delete", control.Params{Game: a[0], Profile: a[1]}, func(r control.Removed) {
			fmt.Fprintf(c.out, "Moved %s to Mortar's trash.\n", strings.Join(r.Mods, ", "))
		})
	case "repair":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		if err := c.call("profile.repair", control.Params{Game: a[0], Profile: a[1]}, &p, readTimeout); err != nil {
			return err
		}
	default:
		return usageError{"unknown profile command " + sub}
	}
	return c.emit(p, func() { fmt.Fprintf(c.out, "%s\t%s\n", p.ID, p.Name) })
}

func (c *cmd) compareTable(diff profile.CLICompare) {
	rows := [][]string{}
	for _, side := range diff.OnlyA {
		rows = append(rows, []string{"only-in-A", side.Name, side.Version, side.Source.Kind, enabledLabel(side.Enabled)})
	}
	for _, side := range diff.OnlyB {
		rows = append(rows, []string{"only-in-B", side.Name, side.Version, side.Source.Kind, enabledLabel(side.Enabled)})
	}
	for _, pair := range diff.DifferentVersion {
		rows = append(rows, []string{"different-version", pair.Name, pair.A.Version + " -> " + pair.B.Version, pair.A.Source.Kind + " -> " + pair.B.Source.Kind, ""})
	}
	for _, pair := range diff.DifferentSource {
		rows = append(rows, []string{"different-source", pair.Name, pair.A.Version + " -> " + pair.B.Version, pair.A.Source.Kind + " -> " + pair.B.Source.Kind, ""})
	}
	for _, pair := range diff.DifferentEnabled {
		rows = append(rows, []string{"different-enabled", pair.Name, "", pair.A.Source.Kind, enabledLabel(pair.A.Enabled) + " -> " + enabledLabel(pair.B.Enabled)})
	}
	for _, pair := range diff.Identical {
		rows = append(rows, []string{"identical", pair.Name, pair.A.Version, pair.A.Source.Kind, enabledLabel(pair.A.Enabled)})
	}
	c.table("SECTION\tNAME\tVERSION\tSOURCE\tENABLED", rows)
}

func enabledMods(rows []control.ModRow) []control.ModRow {
	out := make([]control.ModRow, 0, len(rows))
	for _, m := range rows {
		if m.Enabled {
			out = append(out, m)
		}
	}
	return out
}

func nexusPage(game, source string) string {
	if !strings.HasPrefix(source, "nexus:") {
		return ""
	}
	rest := strings.TrimPrefix(source, "nexus:")
	modID, _, _ := strings.Cut(rest, "/")
	if modID == "" || modID == "0" {
		return ""
	}
	g, ok := components.Game(game)
	if !ok || g.NexusDomain() == "" {
		return ""
	}
	id, err := strconv.Atoi(modID)
	if err != nil {
		return ""
	}
	return nexus.ModURL(g.NexusDomain(), id)
}

func (c *cmd) printEnabledMods(rows []control.ModRow, game string) {
	for _, m := range rows {
		if !m.Enabled {
			continue
		}
		link := nexusPage(game, m.Source)
		switch c.format {
		case "md":
			if link != "" {
				fmt.Fprintf(c.out, "- %s %s %s\n", m.Name, m.Version, link)
			} else {
				fmt.Fprintf(c.out, "- %s %s\n", m.Name, m.Version)
			}
		default:
			if link != "" {
				fmt.Fprintf(c.out, "%s\t%s\t%s\n", m.Name, m.Version, link)
			} else {
				fmt.Fprintf(c.out, "%s\t%s\n", m.Name, m.Version)
			}
		}
	}
}

func (c *cmd) tools() error {
	if len(c.args) > 1 && c.args[1] == "run" {
		a, err := c.need(2, "a game", "a profile", "a tool")
		if err != nil {
			return err
		}
		if err := c.call("tools.run", control.Params{Game: a[0], Profile: a[1], Name: a[2]}, nil, launchTimeout); err != nil {
			return err
		}
		return c.emit(map[string]any{"started": true, "tool": a[2]}, func() {
			fmt.Fprintf(c.out, "Started %s.\n", a[2])
		})
	}
	a, err := c.need(1, "a game")
	if err != nil {
		return err
	}
	return show(c, "tools", control.Params{Game: a[0]}, func(list []tools.Tool) {
		rows := [][]string{}
		for _, tool := range list {
			rows = append(rows, []string{tool.ID, tool.Name, tool.Executable, tool.WorkingDir})
		}
		c.table("ID\tNAME\tEXECUTABLE\tWORKING DIR", rows)
	})
}

func enabledLabel(on bool) string {
	if on {
		return "enabled"
	}
	return "disabled"
}

func (c *cmd) mods(p control.Params) error {
	return show(c, "mods", p, func(rows []control.ModRow) { c.modTable(rows) })
}

func (c *cmd) modTable(rows []control.ModRow) {
	t := [][]string{}
	for _, m := range rows {
		state := enabledLabel(m.Enabled)
		if m.Pinned {
			state += ", pinned"
		}
		if c.verbose {
			t = append(t, []string{m.ID.Local(), m.Name, m.Version, state, m.Source})
			continue
		}
		t = append(t, []string{m.Name, m.Version, state, m.Source})
	}
	if c.verbose {
		c.table("MOD ID\tNAME\tVERSION\tSTATE\tSOURCE", t)
		return
	}
	c.table("NAME\tVERSION\tSTATE\tSOURCE", t)
}

func (c *cmd) modsChange(sub string) error {
	a, err := c.need(2, "a game", "a profile", "one or more mod ids")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], IDs: a[2:]}
	switch sub {
	case "split", "combine":
		need := "a file"
		if sub == "combine" {
			need = "the entry to combine it into"
		}
		if len(a) < 4 {
			return usageError{"mods " + sub + " needs a mod and " + need}
		}
		p.IDs = a[2:4]
		var rows []control.ModRow
		if err := c.call("mods."+sub, p, &rows, readTimeout); err != nil {
			return err
		}
		return c.emit(rows, func() {
			if sub == "split" {
				fmt.Fprintln(c.out, "Installed that file as its own mod.")
				return
			}
			fmt.Fprintln(c.out, "Combined those mods into one entry.")
		})
	case "tag", "untag", "category", "note", "skip-version":
		if sub != "note" && sub != "skip-version" && len(a) < 4 {
			return usageError{"mods " + sub + " needs a value"}
		}
		if len(a) >= 4 {
			p.IDs = a[2 : len(a)-1]
			p.Value = a[len(a)-1]
			if len(p.IDs) == 0 {
				return usageError{"mods " + sub + " needs a mod id (SMAPI id)"}
			}
		} else {
			p.IDs = a[2:]
		}
		var rows []control.ModRow
		if err := c.call("mods."+sub, p, &rows, readTimeout); err != nil {
			return err
		}
		return c.emit(rows, func() { c.modTable(rows) })
	case "enable", "disable":
		var res profile.EnableResult
		if err := c.call("mods."+sub, p, &res, readTimeout); err != nil {
			return err
		}
		return c.emit(res, func() {
			fmt.Fprintf(c.out, "%sd %s.\n", strings.ToUpper(sub[:1])+sub[1:], strings.Join(p.IDs, ", "))
			if len(res.AlsoEnabled) > 0 {
				var mods []control.ModRow
				_ = c.call("mods", control.Params{Game: p.Game, Profile: p.Profile}, &mods, readTimeout)
				fmt.Fprintf(c.out, "Also enabled, as required: %s.\n", strings.Join(modNamesForIDs(res.AlsoEnabled, mods), ", "))
			}
		})
	case "remove":
		return show(c, "mods.remove", p, func(r control.Removed) { fmt.Fprintf(c.out, "Removed %s.\n", strings.Join(r.Mods, ", ")) })
	case "pin", "unpin":
		if sub == "pin" {
			p.Value = c.reasonFlag
		}
	}
	var rows []control.ModRow
	if err := c.call("mods."+sub, p, &rows, readTimeout); err != nil {
		return err
	}
	return c.emit(rows, func() { c.modTable(rows) })
}

func (c *cmd) mod(p control.Params) error {
	return show(c, "mod", p, func(m control.ModInfo) {
		state := enabledLabel(m.Enabled)
		fmt.Fprintf(c.out, "%s %s by %s, %s, from %s\n", m.Name, m.Version, m.Author, state, m.Source)
		if c.verbose {
			fmt.Fprintf(c.out, "Mod id: %s\n", m.ID)
		}
		list := func(label string, xs []string) {
			if len(xs) > 0 {
				fmt.Fprintf(c.out, "%s: %s\n", label, strings.Join(xs, ", "))
			}
		}
		list("Needs", mod.Locals(m.Needs))
		list("Optional", mod.Locals(m.Optional))
		list("Needed by", mod.Locals(m.Dependents))
		list("Optional for", mod.Locals(m.OptionalFor))
		for _, x := range m.Missing {
			fmt.Fprintf(c.out, "Missing: %s\n", missingName(x))
		}
		for _, x := range m.Conflicts {
			fmt.Fprintf(c.out, "Conflict: %s %s with %s\n", x.Kind, x.Target, strings.Join(x.Names, ", "))
		}
		for _, x := range m.Settings {
			fmt.Fprintf(c.out, "Setting: %s is %s, suggested %s\n", x.Field, x.Current, strings.Join(x.Suggested, " or "))
		}
	})
}

func (c *cmd) install(p control.Params) error {
	return c.installMethod("install", p)
}

func (c *cmd) installMethod(method string, p control.Params) error {
	var res control.InstallOutcome
	if err := c.call(method, p, &res, installTimeout); err != nil {
		return err
	}
	if res.Needs != "" {
		if c.json {
			_ = c.emit(res, func() {})
		}
		what := "its installer options"
		if res.Needs == "folder" {
			what = "which folder is the mod"
		}
		return fmt.Errorf("this archive needs you to choose %s: Mortar's window is asking now", what)
	}
	return c.emit(res, func() {
		verb := "Installed"
		if res.Updated {
			verb = "Updated"
		}
		fmt.Fprintf(c.out, "%s %s.\n", verb, strings.Join(res.Added, ", "))
	})
}

func (c *cmd) conflicts(p control.Params) error {
	return show(c, "conflicts", p, func(list []framework.AssetConflict) {
		if len(list) == 0 {
			fmt.Fprintln(c.out, "No conflicts.")
			return
		}
		t := [][]string{}
		for _, x := range list {
			kind := x.Kind
			if x.Cosmetic {
				kind += " (cosmetic)"
			}
			winner := x.WinnerName
			if winner == "" {
				winner = "unclear"
			}
			fixes := []string{}
			for _, f := range x.Fixes {
				fixes = append(fixes, fmt.Sprintf("%s %s=%s", f.Name, f.Field, f.Value))
			}
			t = append(t, []string{kind, x.Target, strings.Join(x.Names, ", "), winner, strings.Join(fixes, "; ")})
		}
		c.table("KIND\tTARGET\tMODS\tWINNER\tFIXES", t)
	})
}

func (c *cmd) problems(p control.Params) error {
	var r problems.Result
	if err := c.call("problems", p, &r, readTimeout); err != nil {
		return err
	}
	if c.format != "" && c.format != "text" {
		return usageError{"problems --format supports text"}
	}
	return c.emit(r, func() {
		if c.format == "text" {
			c.printProblemsText(r)
			return
		}
		c.printProblems(r)
	})
}

func (c *cmd) printProblemsText(r problems.Result) {
	conflicts, harmless := 0, 0
	for _, x := range r.AssetConflicts {
		if x.Cosmetic {
			harmless++
		} else {
			conflicts++
		}
	}
	harmless += harmlessMissing(r)
	fmt.Fprintf(c.out, "missing: %d\n", countedMissing(r))
	fmt.Fprintf(c.out, "duplicates: %d\n", len(r.Duplicates))
	fmt.Fprintf(c.out, "broken: %d\n", len(r.Broken))
	fmt.Fprintf(c.out, "conflicts: %d\n", conflicts)
	fmt.Fprintf(c.out, "settings: %d\n", len(r.Settings))
	fmt.Fprintf(c.out, "last-run errors: %d\n", len(r.RunErrors))
	fmt.Fprintf(c.out, "outside edits: %d\n", len(r.Drift))
	fmt.Fprintf(c.out, "counted: %d\n", problemsHumanCount(r))
	fmt.Fprintf(c.out, "harmless: %d\n", harmless)
}

func (c *cmd) printProblems(r problems.Result) {
	conflicts := problems.ConflictRows(r.AssetConflicts)
	fmt.Fprintf(c.out, "%d problems: %d missing, %d duplicates, %d broken, %d conflict groups, %d settings, %d last-run errors, %d outside edits\n",
		problemsHumanCount(r), countedMissing(r), len(r.Duplicates), len(r.Broken), conflicts, len(r.Settings), len(r.RunErrors), len(r.Drift))
	fmt.Fprintln(c.out, "Active")
	n := 0
	next := func() string {
		n++
		return fmt.Sprintf("%2d", n)
	}
	for _, x := range r.Missing {
		if x.Listed {
			fmt.Fprintf(c.out, "%s  missing    %s needs %s\n", next(), x.DependentName, missingName(x))
		} else {
			fmt.Fprintf(c.out, "missing    %s needs %s\n", x.DependentName, missingName(x))
		}
	}
	for _, x := range r.Duplicates {
		fmt.Fprintf(c.out, "duplicate  %s (%s)\n", x.Name, x.ID)
	}
	for _, x := range r.Broken {
		if x.Status == "abandoned" {
			fmt.Fprintf(c.out, "%s  broken     %s: %s %s\n", next(), x.Name, x.Status, x.Summary)
		} else {
			fmt.Fprintf(c.out, "broken     %s: %s %s\n", x.Name, x.Status, x.Summary)
		}
	}
	for _, x := range r.AssetConflicts {
		if !x.Cosmetic {
			if x.Kind != "" {
				fmt.Fprintf(c.out, "%s  conflict   %s %s: %s\n", next(), x.Kind, x.Target, strings.Join(x.Names, ", "))
			} else {
				fmt.Fprintf(c.out, "conflict   %s %s: %s\n", x.Kind, x.Target, strings.Join(x.Names, ", "))
			}
		}
	}
	for _, x := range r.Settings {
		fmt.Fprintf(c.out, "%s  setting    %s\n", next(), settingText(x))
	}
	for _, x := range r.RunErrors {
		fmt.Fprintf(c.out, "run error  %s (%s)\n", x.Name, x.ID)
	}
	for _, x := range r.Drift {
		fmt.Fprintf(c.out, "edited     %s %s\n", x.Kind, x.Folder)
	}
	fmt.Fprintf(c.out, "Dismissed (%d)\n", len(r.Dismissed))
}

func (c *cmd) problemsDismissed() error {
	profile, err := c.problemsProfile()
	if err != nil {
		return err
	}
	game, err := c.gameOrDefault()
	if err != nil {
		return err
	}
	return show(c, "problems.dismissed", control.Params{Game: game, Profile: profile}, func(list []problems.DismissedProblem) {
		for i, d := range list {
			kind, text := dismissedKindText(d)
			fmt.Fprintf(c.out, "%d\t%s\t%s\n", i+1, kind, text)
		}
	})
}

func (c *cmd) problemsDismiss() error {
	a, err := c.need(2, "a problem index")
	if err != nil {
		return err
	}
	index, err := parseProblemIndex(a[0])
	if err != nil {
		return err
	}
	profile, err := c.problemsProfile()
	if err != nil {
		return err
	}
	game, err := c.gameOrDefault()
	if err != nil {
		return err
	}
	var r problems.Result
	if err := c.call("problems", control.Params{Game: game, Profile: profile}, &r, readTimeout); err != nil {
		return err
	}
	rows := problems.DismissableRows(r)
	if index > len(rows) {
		return refusedError{fmt.Sprintf("problem index %d is not dismissable (%d dismissable rows; run mortar problems %s %s)", index, len(rows), game, profile)}
	}
	if err := c.call("problems.dismiss", control.Params{Game: game, Profile: profile, Index: index}, nil, readTimeout); err != nil {
		return err
	}
	return c.emit(map[string]bool{"dismissed": true}, func() { fmt.Fprintln(c.out, "Dismissed.") })
}

func (c *cmd) problemsRestore() error {
	a, err := c.need(2, "a dismissal token or index")
	if err != nil {
		return err
	}
	profile, err := c.problemsProfile()
	if err != nil {
		return err
	}
	game, err := c.gameOrDefault()
	if err != nil {
		return err
	}
	arg := a[0]
	p := control.Params{Game: game, Profile: profile}
	if idx, err := strconv.Atoi(strings.TrimSpace(arg)); err == nil && idx > 0 {
		p.Index = idx
	} else {
		p.Name = arg
	}
	if err := c.call("problems.restore", p, nil, readTimeout); err != nil {
		return err
	}
	return c.emit(map[string]bool{"restored": true}, func() { fmt.Fprintln(c.out, "Restored.") })
}

func (c *cmd) updates(p control.Params) error {
	var r problems.UpdatesResult
	if err := c.call("updates", p, &r, readTimeout); err != nil {
		return err
	}
	changelogs := map[int][]nexus.Changelog{}
	if c.changelogFlag {
		for i, u := range r.Updates {
			src, id := "nexus", strconv.Itoa(u.NexusID)
			switch {
			case u.GitHubRepo != "":
				src, id = "github", u.GitHubRepo
			case u.NexusID < 1:
				continue
			}
			var logs []nexus.Changelog
			if err := c.call("changelog", control.Params{Game: p.Game, Source: src, ID: id, Name: u.Installed, Value: u.Version}, &logs, readTimeout); err != nil {
				return err
			}
			changelogs[i] = logs
		}
	}
	return c.emit(r, func() {
		if len(r.Updates) == 0 {
			fmt.Fprintln(c.out, "Everything is up to date.")
			return
		}
		t := [][]string{}
		for _, u := range r.Updates {
			t = append(t, []string{u.Name, u.Installed, u.Version, u.Source, u.URL})
		}
		c.table("NAME\tINSTALLED\tNEWEST\tSOURCE\tPAGE", t)
		if !c.changelogFlag {
			return
		}
		for i, u := range r.Updates {
			logs := changelogs[i]
			if len(logs) == 0 {
				continue
			}
			fmt.Fprintf(c.out, "%s\n", u.Name)
			for _, e := range logs {
				fmt.Fprintf(c.out, "  %s\n", e.Version)
				for _, line := range e.Notes {
					fmt.Fprintf(c.out, "    - %s\n", line)
				}
				if len(e.Notes) == 0 && e.Body != "" {
					for line := range strings.SplitSeq(e.Body, "\n") {
						fmt.Fprintf(c.out, "    %s\n", strings.TrimRight(line, " \r"))
					}
				}
			}
		}
	})
}

func (c *cmd) share(p control.Params) error {
	var l control.ShareLink
	if err := c.call("share", p, &l, readTimeout); err != nil {
		return err
	}
	if l.TooLarge && !c.json {
		return errors.New("this profile is too large for a link; use mortar export to write a .mortar file")
	}
	return c.emit(l, func() { fmt.Fprintln(c.out, l.Web) })
}

func (c *cmd) export(p control.Params) error {
	var e control.Exported
	if err := c.call("export", p, &e, installTimeout); err != nil {
		return err
	}
	return c.emit(e, func() {
		fmt.Fprintf(c.out, "Wrote %s.\n", e.Path)
		if n := len(e.Skipped); n > 0 {
			shown := e.Skipped[:min(n, 5)]
			more := ""
			if n > len(shown) {
				more = fmt.Sprintf(" and %d more (--json lists all)", n-len(shown))
			}
			fmt.Fprintf(c.out, "Left out %d settings files: %s%s\n", n, strings.Join(shown, ", "), more)
		}
	})
}

func (c *cmd) runs(p control.Params) error {
	return show(c, "runs", p, func(list []launchsvc.Run) {
		t := [][]string{}
		for _, r := range list {
			t = append(t, []string{
				runStartedLabel(r.Started), (time.Duration(r.DurationMs) * time.Millisecond).Round(time.Second).String(),
				runOutcomeLabel(string(r.Outcome)), runEndedLabel(r.Exit), r.Preset, fmt.Sprint(r.Errors), fmt.Sprint(r.Warnings), r.LoaderVersion, r.GameVersion,
			})
		}
		c.table("STARTED\tDURATION\tOUTCOME\tENDED\tPRESET\tERRORS\tWARNINGS\t"+loaderColumn(p.Game, list)+"\tGAME", t)
		for _, r := range list {
			if r.Error != "" {
				fmt.Fprintf(c.out, "%s: %s\n", runStartedLabel(r.Started), r.Error)
			}
			if r.Cause != nil {
				fmt.Fprintf(c.out, "%s\n", runCauseLine(r))
			}
		}
	})
}

func (c *cmd) perfReports() error {
	if len(c.args) < 2 || c.args[1] != "reports" {
		return usageError{"perf needs reports <game> <profile>"}
	}
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	return show(c, "perf.reports", control.Params{Game: a[0], Profile: a[1]}, func(list []launchsvc.SavedReport) {
		if len(list) == 0 {
			fmt.Fprintln(c.out, "No performance reports.")
		}
		for _, r := range list {
			fmt.Fprintf(c.out, "%s  %s\n", r.At, r.ID)
			if f := r.Frame; f != nil {
				fmt.Fprintf(c.out, "%.0f fps over %.0f s; frame ms avg %.1f, p95 %.1f, p99 %.1f, max %.1f; Mono heap %d MiB used of %d MiB; %d GCs\n",
					f.FPS, f.Seconds, f.AvgMs, f.P95Ms, f.P99Ms, f.MaxMs, f.MonoUsed>>20, f.MonoHeap>>20, f.GCCollections)
			}
			rows := [][]string{}
			for _, row := range r.Rows {
				rows = append(rows, []string{row.Name, fmt.Sprintf("%.1f", row.AverageMs), fmt.Sprintf("%.1f", row.PeakMs), fmt.Sprintf("%.0f", row.Calls)})
			}
			c.table("MOD\tAVG MS\tPEAK MS\tCALLS", rows)
		}
	})
}

func (c *cmd) logs(p control.Params) error {
	return show(c, "logs", p, func(l control.RunLog) { fmt.Fprint(c.out, l.Text) })
}

func (c *cmd) shareLog() error {
	a, err := c.need(2, "a game", "a profile")
	if err != nil {
		return err
	}
	run := c.run
	if run == "" && len(c.args) > 4 {
		run = c.args[4]
	}
	p := control.Params{Game: a[0], Profile: a[1], Run: run}
	var l control.RunLog
	if err := c.call("logs", p, &l, readTimeout); err != nil {
		return err
	}
	if !c.yesFlag {
		if info, err := os.Stdin.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return refusedError{"sharing a log needs --yes when stdin is not a terminal"}
		}
		fmt.Fprintf(c.errOut, "This log is %s. Sharing uploads it to smapi.io, where it becomes public at a link. Continue? [y/N] ", humanBytes(int64(len(l.Text))))
		var answer string
		if _, err := fmt.Fscan(os.Stdin, &answer); err != nil {
			return err
		}
		if strings.ToLower(answer) != "y" && strings.ToLower(answer) != "yes" {
			return errors.New("cancelled")
		}
	}
	return show(c, "logs.share", p, func(result map[string]string) { fmt.Fprintln(c.out, result["url"]) })
}

func (c *cmd) searchLogs(query string) error {
	game, err := c.gameOrDefault()
	if err != nil {
		return err
	}
	return show(c, "logs.search", control.Params{Game: game, Profile: c.profileFlag, Query: query}, func(result launchsvc.RunSearch) {
		if len(result.Hits) == 0 {
			fmt.Fprintln(c.out, "No matches.")
			return
		}
		for _, hit := range result.Hits {
			fmt.Fprintf(c.out, "%s  %s  L%d: %s\n", hit.Started, hit.Outcome, hit.LineNumber, hit.Line)
		}
		if result.Truncated {
			fmt.Fprintln(c.out, "Showing the first 500 matches.")
		}
	})
}

func (c *cmd) saves(p control.Params) error {
	var list []savessvc.Fit
	if err := c.call("saves", p, &list, readTimeout); err != nil {
		return err
	}
	seasons := []string{"Spring", "Summer", "Fall", "Winter"}
	profileNames := map[string]string{}
	var profiles []profile.Profile
	if err := c.call("profiles", control.Params{Game: p.Game}, &profiles, readTimeout); err == nil {
		for _, pr := range profiles {
			if pr.ID != "" && pr.Name != "" {
				profileNames[pr.ID] = pr.Name
			}
		}
	}
	// Only Stardew Valley's saves have a farm and a farmer; other games' saves are listed by the name the app shows.
	farms := slices.ContainsFunc(list, func(s savessvc.Fit) bool { return s.Farm != "" || s.Farmer != "" })
	return c.emit(list, func() {
		t := [][]string{}
		for _, s := range list {
			season := fmt.Sprint(s.Season)
			if s.Season >= 0 && s.Season < len(seasons) {
				season = seasons[s.Season]
			}
			missing := []string{}
			for _, m := range append(s.Missing, s.LastMissing...) {
				if !slices.Contains(missing, m.Name) {
					missing = append(missing, m.Name)
				}
			}
			// A save that is a bare file (Lethal Company's) has no in-game date.
			date := ""
			if s.Year > 0 {
				date = fmt.Sprintf("%s %d, Year %d", season, s.Day, s.Year)
			}
			rest := []string{s.Folder, date, saveLastPlayed(s, profileNames), strings.Join(missing, ", ")}
			if farms {
				t = append(t, append([]string{s.Farm, s.Farmer}, rest...))
			} else {
				t = append(t, append([]string{saves.DisplayName(s.Farm, s.Folder)}, rest...))
			}
		}
		head := "SAVE\tFOLDER\tDATE\tLAST PLAYED WITH\tMISSING MODS"
		if farms {
			head = "FARM\tFARMER\tFOLDER\tDATE\tLAST PLAYED WITH\tMISSING MODS"
		}
		c.table(head, t)
	})
}

func (c *cmd) launch(p control.Params) error {
	var st launchsvc.Status
	if err := c.call("launch", p, &st, launchTimeout); err != nil {
		return err
	}
	if !c.wait {
		return c.emit(st, func() { fmt.Fprintf(c.out, "%s is %s.\n", p.Game, st.State) })
	}
	if !c.json {
		fmt.Fprintf(c.out, "%s is %s; waiting for it to close.\n", p.Game, st.State)
	}
	for st.State.Active() {
		time.Sleep(2 * time.Second)
		if err := c.call("status", control.Params{Game: p.Game, Install: p.Install}, &st, readTimeout); err != nil {
			return err
		}
	}
	return c.runs(control.Params{Game: p.Game, Profile: p.Profile})
}

func (c *cmd) status(verb, gameID string) error {
	var st launchsvc.Status
	if err := c.call(verb, control.Params{Game: gameID, Install: c.installFlag}, &st, launchTimeout); err != nil {
		return err
	}
	return c.emit(st, func() {
		if verb == "stop" {
			fmt.Fprintf(c.out, "%s stopped.\n", gameID)
			return
		}
		line := fmt.Sprintf("%s is %s", gameID, st.State)
		if st.Profile != "" {
			line += " (profile " + st.Profile + ")"
		}
		fmt.Fprintln(c.out, line+".")
	})
}

func (c *cmd) queue() error {
	if len(c.args) > 1 {
		sub := c.args[1]
		method := "queue." + sub
		switch sub {
		case "retry", "skip":
			var id string
			if len(c.args) > 2 {
				id = c.args[2]
			}
			var before queue.State
			if err := c.call("queue", control.Params{}, &before, readTimeout); err != nil {
				return err
			}
			n := queueActionCount(before, sub, id)
			var st queue.State
			if err := c.call(method, control.Params{Name: id}, &st, readTimeout); err != nil {
				return err
			}
			return c.emit(st, func() {
				if sub == "retry" {
					fmt.Fprintf(c.out, "Retried %d failed downloads.\n", n)
					return
				}
				fmt.Fprintf(c.out, "Skipped %d downloads.\n", n)
			})
		case "add":
			return c.queueAdd()
		case "retry-failed":
			return c.queueRetryFailed()
		case "pause", "resume", "clear":
			var st queue.State
			if err := c.call(method, control.Params{}, &st, readTimeout); err != nil {
				return err
			}
			return c.emit(st, func() { fmt.Fprintf(c.out, "Queue %s.\n", sub) })
		default:
			return usageError{"unknown queue command " + sub}
		}
	}
	return show(c, "queue", control.Params{}, func(st queue.State) {
		if len(st.Items) == 0 {
			fmt.Fprintln(c.out, "The download queue is empty.")
			return
		}
		t := [][]string{}
		for _, it := range st.Items {
			t = append(t, []string{it.ID, it.Name, it.Version, queueHumanState(it.State), it.Profile, queueErrorLine(it.Error)})
		}
		c.table("ID\tNAME\tVERSION\tSTATE\tPROFILE\tERROR", t)
		if st.Paused {
			fmt.Fprintln(c.out, "Paused.")
		}
	})
}

func (c *cmd) cacheCmd() error {
	if len(c.args) < 2 {
		return usageError{"cache needs size or clear"}
	}
	switch c.args[1] {
	case "size":
		return show(c, "cache.size", control.Params{}, func(info datasvc.CacheInfo) {
			fmt.Fprintf(c.out, "%s at %s\n", humanBytes(info.Size), info.Path)
		})
	case "clear":
		if err := c.call("cache.clear", control.Params{}, nil, readTimeout); err != nil {
			return err
		}
		return c.emit(map[string]bool{"cleared": true}, func() { fmt.Fprintln(c.out, "Cache cleared.") })
	default:
		return usageError{"unknown cache command " + c.args[1]}
	}
}

func (c *cmd) dataCmd() error {
	if len(c.args) > 1 && c.args[1] == "location" {
		return c.dataLocation()
	}
	if len(c.args) < 2 || c.args[1] != "usage" {
		return usageError{"data needs usage --by-mod or location"}
	}
	if !c.byMod {
		return usageError{"data usage needs --by-mod"}
	}
	return show(c, "data.usageByMod", control.Params{}, func(u datasvc.ModUsage) {
		rows := make([][]string, 0, len(u.Items))
		for _, it := range u.Items {
			rows = append(rows, []string{
				it.Game, it.Key, it.Name, strconv.FormatInt(it.Size, 10),
				strconv.Itoa(it.Profiles), strconv.FormatInt(it.ProfileSize, 10), it.LastUsed,
			})
		}
		c.table("GAME\tKEY\tNAME\tSIZE\tPROFILES\tCOPIES\tLAST USED", rows)
		fmt.Fprintf(c.out, "Total %d\n", u.Total)
	})
}

func (c *cmd) update() error {
	a, err := c.need(1, "a game", "a profile")
	if err != nil {
		return err
	}
	if len(a) == 2 && !c.all {
		return usageError{"update needs one or more mod ids, or --all"}
	}
	var res control.QueuedUpdates
	if err := c.call("updates.queue", control.Params{Game: a[0], Profile: a[1], IDs: a[2:], All: c.all}, &res, installTimeout); err != nil {
		return err
	}
	return c.emit(res, func() {
		fmt.Fprintf(c.out, "Queued %d updates.\n", res.Queued)
		if res.Before != "" {
			fmt.Fprintf(c.out, "Undo all: mortar profile revert %s %q %s\n", a[0], a[1], res.Before)
		}
	})
}

func (c *cmd) backups() error {
	game, err := c.gameOrDefault()
	if err != nil {
		return err
	}
	if len(c.args) > 1 {
		switch c.args[1] {
		case "restore":
			a, err := c.need(2, "a backup name")
			if err != nil {
				return err
			}
			if err := c.call("backups.restore", control.Params{Game: game, Name: a[0], IDs: a[1:], Profile: c.profileFlag}, nil, installTimeout); err != nil {
				return err
			}
			return c.emit(map[string]any{"restored": a[0], "saves": a[1:]}, func() {
				fmt.Fprintf(c.out, "Restored %s.\n", a[0])
			})
		case "keep", "unkeep":
			a, err := c.need(2, "a backup name")
			if err != nil {
				return err
			}
			method := "backups." + c.args[1]
			if err := c.call(method, control.Params{Game: game, Name: a[0]}, nil, readTimeout); err != nil {
				return err
			}
			pinned := c.args[1] == "keep"
			return c.emit(map[string]any{"name": a[0], "pinned": pinned}, func() {
				if pinned {
					fmt.Fprintf(c.out, "Kept %s.\n", a[0])
				} else {
					fmt.Fprintf(c.out, "Unkept %s.\n", a[0])
				}
			})
		case "create":
			a, err := c.need(2, "a save")
			if err != nil {
				return err
			}
			if err := c.call("backups.create", control.Params{Game: game, Name: a[0], Profile: c.profileFlag}, nil, readTimeout); err != nil {
				return err
			}
			return c.emit(map[string]any{"save": a[0]}, func() {
				fmt.Fprintf(c.out, "Backed up %s.\n", a[0])
			})
		case "usage":
			return c.backupsUsage(game)
		case "trim":
			return c.backupsTrim(game)
		case "list":
			break
		default:
			return usageError{"unknown backups command " + c.args[1]}
		}
	}
	return show(c, "backups", control.Params{Game: game}, func(list []backup.Backup) {
		if len(list) == 0 {
			fmt.Fprintln(c.out, "No backups.")
			return
		}
		rows := make([][]string, 0, len(list))
		for _, b := range list {
			var names []string
			for _, sn := range b.Saves {
				names = append(names, saves.DisplayName(sn.Farm, sn.Folder))
			}
			kept := ""
			if b.Pinned {
				kept = "kept"
			}
			rows = append(rows, []string{b.Name, relativeDeleted(time.UnixMilli(b.At)), b.Kind, strings.Join(names, ", "), humanBytes(b.Size), kept})
		}
		c.table("NAME\tTAKEN\tKIND\tSAVES\tSIZE\tKEPT", rows)
	})
}

func (c *cmd) launchers() error {
	method, p := "launchers", control.Params{}
	if len(c.args) > 1 {
		switch sub := c.args[1]; sub {
		case "add", "remove":
			a, err := c.need(2, "a launcher id", "a folder")
			if err != nil {
				return err
			}
			method, p = "launchers."+sub, control.Params{Name: a[0], Path: absPath(a[1])}
		default:
			return usageError{"unknown launchers command " + sub}
		}
	}
	var list []game.StoreApp
	if err := c.call(method, p, &list, readTimeout); err != nil {
		return err
	}
	return c.emit(list, func() {
		t := [][]string{}
		for _, l := range list {
			games := []string{}
			for _, g := range l.Games {
				games = append(games, g.Name)
			}
			where := strings.Join(l.Roots, ", ")
			if !l.Found {
				where = "not found"
			}
			t = append(t, []string{l.ID, l.Name, yes(l.Found), strings.Join(games, ", "), where})
		}
		c.table("ID\tNAME\tFOUND\tGAMES\tFOLDERS", t)
	})
}

func (c *cmd) doctor() error {
	var d control.Doctor
	if err := c.call("doctor", control.Params{}, &d, readTimeout); err != nil {
		if errors.Is(err, controlwire.ErrNotRunning) {
			return offlineDoctor()
		}
		return err
	}
	rep := liveDoctorReport(d, c.version)
	return c.emit(rep, func() {
		fmt.Fprint(c.out, doctor.PlainText(rep))
	})
}

func offlineDoctor() error {
	dir, err := datadir.Dir()
	if err != nil {
		return err
	}
	rep := doctor.Scan(dir)
	findings, fixes := doctor.Findings(rep)
	free, _ := datadir.FreeBytes(dir)
	cacheBytes, _ := datadir.Size(filepath.Join(dir, "cache"))
	result := map[string]any{
		"offline": true, "dataDir": dir, "freeBytes": free, "cacheBytes": cacheBytes,
		"findings": findings, "fixes": fixes, "summary": fmt.Sprintf("Offline doctor: %d findings.", len(findings)),
	}
	if len(findings) > 0 {
		if len(fixes) > 0 {
			result["summary"] = fmt.Sprintf("Offline doctor: %d findings; see fixes.", len(findings))
		}
	}
	code := 3
	if len(findings) > 0 {
		code = 1
	}
	return offlineDoctorError{result: result, code: code}
}

const usage = `Usage: mortar <command> [arguments] [--json]

Mortar must be running; these commands ask the open app. <profile> is an id or a name. A verb without <game>
takes --game <id>, which may be left out when exactly one game is installed.

  settings get [--game <id>] [key]     list settings, or one key
  settings set [--game <id>] <key> <value>  change a setting
  settings export <file>                  write portable settings JSON
  settings import <file>                  apply a portable settings JSON
  settings reset [key] [--game id]        restore defaults
  loader versions <game> [--loader id]    loader versions in the store and upstream
  loader install <game> <version> [--loader id]  install that loader version
  loader pin <game> <version|latest> [--loader id]  pin the loader, or follow latest
  games                                   supported games, whether each is configured
  profiles <game>                         profiles of a game
  profile list <game> <profile> --format md|text  enabled mods (name, version, Nexus link)
  profile create <game> <name>            new empty profile
  profile from-save <game> <save>         profile named after the farm, with that save's last mods
  profile rename <game> <profile> <name>
  profile copy <game> <profile> [name]
  profile match <game> <profile> <link-or-file> preview a friend's profile
  profile collection <game> <profile> [--update|--unlink]  collection link, revision, latest
  bundles <game>                         list saved bundles
  bundles apply <game> <bundle> <profile> apply a bundle
  bundles create <game> <name> <profile> [mod id...]  save a bundle of a profile's mods (all when none named)
  bundles delete <game> <bundle> | rename <game> <bundle> <name>
  bundles add <game> <bundle> <profile> <mod id>... | remove <game> <bundle> <mod id>...
  source untrack <game> --source nexus --all|--unused  bulk untrack a source's tracked mods
  source tracked <game> <profile> --source nexus --missing  tracked at the source but not in profile
  profile delete <game> <profile>         moves it to Mortar's trash
  profile repair <game> <profile>         rebuild damaged profile.json from history
  trash list [--game <id>]             recently deleted profiles
  trash restore <name|id> [--game <id>]
  trash delete <name|id> --yes            permanently delete one
  trash empty --yes [--game <id>]      purge all deleted profiles
  profile compare <game> <profileA> <profileB>
  profile history <game> <profile>       restore points
  profile health <game> <profile>        problem-check history
  history <game> --all                   recent changes across profiles
  history diff <game> <profile> <a> <b>  compare two snapshots
  history revert <game> <profile> <eventId> --item <mod>
  profile changes <game> <profile>       changes since last run
  profile good <game> <profile> [--mark|--restore]
  profile revert <game> <profile> <eventId>
  profile import <code|key|file> [--game <id>] [--name N | --profile P] [--preview]
  profile import --from vortex|mo2 [--game <id>] [<their profile>] [--preview]
                                          list another manager's profiles, or import one as a new profile
                                          import an r2modman code, .r2z or modpack and queue its downloads
  profile export <game> <profile> <file.zip> --format modpack [--no-configs]
                                          write a Thunderstore modpack of the profile's Thunderstore packages
  profile farm export <game> <profile>   the mods a Stardew guest must match, as JSON to share
  profile farm check|fix <game> <profile> <list.json|json>
                                          what differs from the host's list; fix queues the downloads
  profile backup <game> <profile> <file.zip>  everything but the mod files: settings, history, configs
  profile restore <file.zip> [--game <id>]    new profile from a backup; downloads its mods again
  profile load-order <game> <profile>    enabled mods in the loader's load order
  profile shortcut <game> <profile> [--remove]  desktop shortcut that plays this profile
  profile set <game> <profile> <field> <value>  notes|color|icon|description|install|launchOptions|launchPrefix|launchEnv|loader|
                                          defaultLaunchPreset|skipPlayCheck|cover, or a per-profile game setting
  profile steam <game> <profile>         add this profile to Steam as a non-Steam game
  game steam-launch-option <game> [--set|--clear]  read or change Steam's loader launch options
  game launch-preset-templates <game> [add|remove <name> [options [prefix [env]]]|use <name> <profile>]  game-wide launch preset templates (use copies one into a profile)
  mods <game> <profile>                   mods with version, state and source
  mods enable|disable|pin|unpin|remove <game> <profile> <mod id>...  (pin accepts --reason)
  mods tag|untag|category|note|skip-version <game> <profile> <mod> [value]
  mods channel <game> <profile> <mod> main|optional|beta
  mods group <game> <profile> list|create|delete|add|remove|on|off …
  mods split <game> <profile> <mod> <file>
  mods combine <game> <profile> <mod> <into-mod>
  mods win <game> <profile> <winner> <loser> [--undo]
  mods files <game> <profile> <mod>       linked extra files (keys for mods split)
  mods config <game> <profile> <mod> [<field> <value>]  print or set one config field
  mods preset <game> <profile> <mod> list|save|apply|delete [name]  config.json presets
  mods menu <game> <profile> <mod> [--set <page>/<index>=<value>]  captured GMCM menu
  sweep <game> [--install <id>] [--json]                   patch-day check after a game or SMAPI change
  mods compat <game> <profile>            non-ok SMAPI compatibility-list rows
  mods report <game> <profile> <mod> [--run ID]  report text and author URL for a mod's log errors
  mods by-author <game> <author>            mods installed in any profile for this author
  mod <game> <profile> <mod id>           one mod: dependencies, dependents, conflicts, settings
                                          (mod id is the SMAPI id)
  install <game> <profile> <archive>      install a local archive
  conflicts <game> <profile> [--all]      asset conflicts (--all includes cosmetic ones)
  conflicts map <game> <profile> [--filter x]
                                          every touched asset, with who writes it
  who <game> <profile> <query>            which mods change an asset
  problems <game> <profile> [--format text]  everything the Problems tab lists
  problems dismissed [--game <id>] [--profile <name>]  dismissed problems (index, kind, text, token)
  problems dismiss <index> [--game <id>] [--profile <name>]
  problems restore <token|index> [--game <id>] [--profile <name>]
  updates <game> <profile> [--changelog]  mods with a newer version
  updates apply --everywhere <game> [mod]  same update in every eligible profile
  saves <game> <profile>                  saves and the mods each one lacks
  saves check <game> <save> [<profile>]   mods that save last used that this profile lacks
  share <game> <profile>                  share link
  export <game> <profile> <file.mortar>   write a .mortar file
  open <link|file>                        hand a share link or .mortar file to Mortar
  play <game> <profile> --check           pre-Play summary; exits 3 when anything is wrong
  play <game> <profile> --test            launch, wait for the title screen, and stop
  launch <game> <profile> [--install <id>] [--preset NAME] [--wait] [--force]
                                       play (default launch preset unless --preset); --force skips Play warnings
  launch <game> --vanilla               start the game without mods
  perf reports <game> <profile>         saved performance reports
  status <game> | stop <game> [--install <id>]
  runs <game> <profile>                   recent launches
	logs <game> <profile> [--run <id>]      a stored SMAPI log (latest by default)
  logs share <game> <profile> [run] [--yes]  upload that run's log to smapi.io (prompts unless --yes)
  logs search <query> [--game <id>] [--profile <name>]  search all stored run logs
  logs fixes <game> <profile> [run]       recognised SMAPI errors in that run and their fixes
  queue                                   the download queue
  queue retry|skip [<id>]                 retry or skip queued downloads
  queue add <game> <profile> <id> --source nexus|github|thunderstore [--file <nexus file id|asset>] [--version <tag|version>]
                                          queue one mod (GitHub id is owner/repo, Thunderstore Namespace-Name)
  queue retry-failed                      requeue every retryable failed download in the history
  queue pause|resume|clear                control the download queue
  browse <game> <text> [--source <id>|all] [--page N]  search a source, or all of them (default)
  update <game> <profile> <mod id>...|--all
                                          queue available mod updates
  backups list [--game <id>] [--json]     list save backups
  backups create <save>                   pin a Manual backup of one save (its folder or its name); --profile P for P's own saves
  backups usage                           disk used by save backups, per save
  backups trim --keep N                   delete all but the newest N backups of each save (kept ones stay)
  cache size                              analysis cache size
  cache clear                             delete the analysis cache
  data usage --by-mod                     store items with size on disk
  data location                           the data folder in use, and whether it is portable
  history usage <game>                    events and disk used by each profile's history
  history trim <game> <profile> --keep N  keep a profile's newest N history events
  templates list <game> | delete <game> <name>
  templates save <game> <profile> <name>  capture a profile as a template
  templates apply <game> <template> <profile> [--preview]  merge it into a profile; --json holds the undo object
  templates undo <game> <profile> <file>  undo an apply from its saved undo object
  templates rename <game> <template> <name> | restore <game> <file>  restore takes one entry of list --json
  templates new <game> <template> <profile name>  create a profile from a template
  library extra <game>                    mods in the extra mods folder
  library hidden <game> <profile>         dot-hidden mods inside the profile's mods
  library old-files <game> <profile> [--keep KEY | --delete KEY]  files updates set aside
  library strays <game> [<profile> --move FOLDER...] [--dismiss FOLDER]  mods in the game's own Mods folder
  archive preview <path>                  what an archive holds, without extracting it
  archive downloads <game>                archives in the download folder no profile or store item accounts for
  store report [--game G]                 unused and duplicate store items
  store remove <game> <key>...            delete store items no profile uses
  store check <game>                      verify store files; lists damaged items
  store repair <game> <profile> <key>     fetch a damaged item again through the queue
  bisect start <game> <profile>           crash check on a copy; prints its id
  bisect status|stop <id>                 follow or cancel a crash check
  backups keep <name>                     keep a save backup during rotation
  backups unkeep <name>                   stop keeping a save backup
  backups restore <name> [save...]        restore a save backup; --profile P into P's own saves
  tools <game>                            configured external tools
  tools run <game> <profile> <tool>       start an external tool
  tools add <game> <name> <executable> [arg...] | remove <game> <id>
  tools update <game> <id> <name> <executable> [arg...]
  lan peers | inbox                       nearby Mortars; shares waiting for you
  lan send <game> <profile> <peer>        send a profile to a peer
  lan accept|decline <id>                 import a waiting share as a new profile, or refuse it
  lan pair                                show a code and wait for another computer to enter it
  lan pair --code <code> [--peer host:port]  enter the code another computer shows
  lan paired | unpair <id>                computers you paired; forget one
  data move <dir> [--preview]             move the data folder
  data cleanup [--preview]                remove unused store items and caches
  support diagnostics save <path.zip> [--game id] [--profile name]  write a redacted diagnostics zip
  app update check|install                Mortar's own updates
  links register                          claim mortar:// links and .mortar files
  links enable|disable --source nexus|thunderstore  take over or hand back a source's links
  game reset-install <game> --yes         delete the game's install folder
  game steam-status                       whether a usable Steam was found
  problems check-updates <game> <profile>  look for mod updates now
  launchers [add|remove <id> <folder>]   launchers, the games in each, and your added folders
  doctor                                  versions, folders and link handling
  quit [--force]                          close the running app and wait until it has exited; --force
                                          quits even with downloads or a game running
  completion bash|zsh|fish                shell completion script
  version | help
`
