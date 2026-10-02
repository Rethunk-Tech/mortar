// Package cli is Mortar's command line: `mortar <verb> ...` asks the running app over internal/control and prints
// the answer as a table, or as JSON with --json. It never opens a window or touches the data folder itself.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/game"
	"github.com/Rethunk-AI/mortar/internal/launchsvc"
	"github.com/Rethunk-AI/mortar/internal/problems"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/queue"
	"github.com/Rethunk-AI/mortar/internal/savessvc"
	"github.com/Rethunk-AI/mortar/internal/tools"
)

const (
	readTimeout    = 2 * time.Minute
	installTimeout = 10 * time.Minute
	launchTimeout  = 4 * time.Minute
)

// verbs are the first words that make an invocation a command-line call rather than a window launch.
var verbs = map[string]bool{
	"games": true, "profiles": true, "profile": true, "mods": true, "mod": true, "install": true,
	"conflicts": true, "problems": true, "updates": true, "share": true, "export": true, "open": true,
	"runs": true, "logs": true, "saves": true, "launch": true, "stop": true, "status": true, "queue": true,
	"bundles": true, "nexus": true, "trash": true,
	"update": true, "backups": true, "doctor": true, "launchers": true, "tools": true, "version": true, "completion": true, "help": true, "--help": true, "-h": true, "__complete": true,
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
	version     string
	call        caller
	out         io.Writer
	errOut      io.Writer
	json        bool
	all         bool
	unused      bool
	yesFlag     bool
	force       bool
	wait        bool
	run         string
	game        string
	profileFlag string
	args        []string
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
	var offline offlineDoctorError
	if errors.As(err, &offline) {
		if c.json {
			_ = json.NewEncoder(c.out).Encode(offline.result)
		} else {
			fmt.Fprintln(c.out, offline.result["summary"])
			for _, f := range offline.result["findings"].([]string) {
				fmt.Fprintln(c.out, "Finding:", f)
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
	if errors.Is(err, control.ErrNotRunning) {
		code = 3
	}
	if c.json {
		_ = json.NewEncoder(c.errOut).Encode(map[string]any{"error": err.Error(), "code": code})
		return code
	}
	if code == 2 {
		fmt.Fprintln(c.errOut, "mortar:", err)
		if _, ok := errors.AsType[usageError](err); ok {
			fmt.Fprint(c.errOut, usage)
		}
		return code
	}
	fmt.Fprintln(c.errOut, "mortar:", err)
	return code
}

func (c *cmd) parse(args []string) error {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			c.json = true
		case a == "--all":
			c.all = true
		case a == "--unused":
			c.unused = true
		case a == "--yes":
			c.yesFlag = true
		case a == "--force":
			c.force = true
		case a == "--wait":
			c.wait = true
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
		case a == "--game":
			if i+1 >= len(args) {
				return usageError{"--game needs a game id"}
			}
			i++
			c.game = args[i]
		case strings.HasPrefix(a, "--game="):
			c.game = strings.TrimPrefix(a, "--game=")
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

func (c *cmd) ask(method string, p control.Params, out any, timeout time.Duration) error {
	return c.call(method, p, out, timeout)
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
	case "doctor":
		return c.doctor()
	case "launchers":
		return c.launchers()
	case "queue":
		return c.queue()
	case "update":
		return c.update()
	case "backups":
		return c.backups()
	case "bundles":
		return c.bundles()
	case "nexus":
		return c.nexus()
	case "trash":
		return c.trash()
	case "profiles":
		a, err := c.need(1, "a game")
		if err != nil {
			return err
		}
		return c.profiles(a[0])
	case "tools":
		return c.tools()
	case "profile":
		return c.profile()
	case "status", "stop":
		a, err := c.need(1, "a game")
		if err != nil {
			return err
		}
		return c.status(verb, a[0])
	case "mods":
		if len(c.args) > 1 {
			switch c.args[1] {
			case "enable", "disable", "remove", "pin", "unpin":
				return c.modsChange(c.args[1])
			}
		}
	}
	if verb == "logs" && len(c.args) >= 3 && c.args[1] == "search" {
		return c.searchLogs(c.args[2])
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
	a, err := c.need(1, "a game", "a profile")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], All: c.all, Run: c.run, Force: c.force}
	switch verb {
	case "mods":
		return c.mods(p)
	case "mod":
		if len(a) < 3 {
			return usageError{"mod needs a UniqueID"}
		}
		p.UniqueIDs = a[2:3]
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
	if a, err := absolute(p); err == nil {
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
	proc, err := os.StartProcess(exe, []string{exe, target}, &os.ProcAttr{Files: []*os.File{nil, nil, nil}})
	if err != nil {
		return err
	}
	return proc.Release()
}

// missingName names a missing requirement by its page when known, with the minimum version it needs.
func missingName(m problems.Missing) string {
	name := m.UniqueID
	if m.Where != nil && m.Where.PageName != "" {
		name = m.Where.PageName + " (" + m.UniqueID + ")"
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

func (c *cmd) games() error {
	var rows []control.GameRow
	if err := c.ask("games", control.Params{}, &rows, readTimeout); err != nil {
		return err
	}
	return c.emit(rows, func() {
		t := [][]string{}
		for _, g := range rows {
			t = append(t, []string{g.ID, g.Name, yes(g.Available), yes(g.Configured), fmt.Sprint(g.Profiles), g.Store, g.InstallDir})
		}
		c.table("ID\tNAME\tSUPPORTED\tCONFIGURED\tPROFILES\tSTORE\tFOLDER", t)
	})
}

func (c *cmd) profiles(gameID string) error {
	var list []profile.Profile
	if err := c.ask("profiles", control.Params{Game: gameID}, &list, readTimeout); err != nil {
		return err
	}
	return c.emit(list, func() {
		t := [][]string{}
		for _, p := range list {
			on, total := 0, 0
			for _, e := range p.Entries {
				total += len(e.Mods)
				on += len(e.Mods) - len(e.Disabled)
			}
			t = append(t, []string{p.ID, p.Name, fmt.Sprintf("%d/%d", on, total), p.Updated.Local().Format("2006-01-02 15:04")})
		}
		c.table("ID\tNAME\tMODS ON\tUPDATED", t)
	})
}

func (c *cmd) bundles() error {
	if len(c.args) > 1 && c.args[1] == "apply" {
		a, err := c.need(2, "a game", "a bundle", "a profile")
		if err != nil {
			return err
		}
		var result control.BundleApply
		if err := c.ask("bundles.apply", control.Params{Game: a[0], Name: a[1], Profile: a[2]}, &result, readTimeout); err != nil {
			return err
		}
		return c.emit(result, func() {
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
	var list []control.BundleRow
	if err := c.ask("bundles", control.Params{Game: a[0]}, &list, readTimeout); err != nil {
		return err
	}
	return c.emit(list, func() {
		rows := make([][]string, 0, len(list))
		for _, b := range list {
			rows = append(rows, []string{b.ID, b.Name, fmt.Sprint(len(b.Mods)), strings.Join(b.Profiles, ", ")})
		}
		c.table("ID\tNAME\tMODS\tPROFILES WITH ALL", rows)
	})
}

func (c *cmd) trashGame() string {
	if c.game != "" {
		return c.game
	}
	return "stardew"
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

func (c *cmd) trash() error {
	if len(c.args) < 2 {
		return usageError{"trash needs list, restore, delete or empty"}
	}
	game := c.trashGame()
	sub := c.args[1]
	switch sub {
	case "list":
		var items []profile.TrashItem
		if err := c.ask("trash.list", control.Params{Game: game}, &items, readTimeout); err != nil {
			return err
		}
		return c.emit(items, func() {
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
		var p profile.Profile
		if err := c.ask("trash.restore", control.Params{Game: game, Profile: a[0]}, &p, readTimeout); err != nil {
			return err
		}
		return c.emit(p, func() { fmt.Fprintf(c.out, "Restored %s (%s).\n", p.Name, p.ID) })
	case "delete":
		a, err := c.need(2, "a deleted profile name or id")
		if err != nil {
			return err
		}
		if !c.yesFlag {
			return refusedError{"permanently deleting a profile needs --yes"}
		}
		var r control.Removed
		if err := c.ask("trash.delete", control.Params{Game: game, Profile: a[0]}, &r, readTimeout); err != nil {
			return err
		}
		return c.emit(r, func() {
			fmt.Fprintf(c.out, "Permanently deleted %s.\n", strings.Join(r.Mods, ", "))
		})
	case "empty":
		if !c.yesFlag {
			return refusedError{"emptying trash needs --yes"}
		}
		if err := c.ask("trash.empty", control.Params{Game: game}, nil, readTimeout); err != nil {
			return err
		}
		return c.emit(map[string]bool{"emptied": true}, func() { fmt.Fprintln(c.out, "Trash emptied.") })
	default:
		return usageError{"unknown trash command " + sub}
	}
}

func (c *cmd) nexus() error {
	if len(c.args) < 2 || c.args[1] != "untrack" {
		return usageError{"unknown nexus command"}
	}
	a, err := c.need(2, "a game")
	if err != nil {
		return err
	}
	if c.all == c.unused {
		return usageError{"nexus untrack needs exactly one of --all or --unused"}
	}
	var count int
	if err := c.ask("nexus.tracked", control.Params{Game: a[0]}, &count, readTimeout); err != nil {
		return err
	}
	if !c.yesFlag {
		if info, err := os.Stdin.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return refusedError{"nexus untrack needs --yes when stdin is not a terminal"}
		}
		fmt.Fprintf(c.errOut, "Untrack %d mods? [y/N] ", count)
		var answer string
		if _, err := fmt.Fscan(os.Stdin, &answer); err != nil {
			return err
		}
		if strings.ToLower(answer) != "y" && strings.ToLower(answer) != "yes" {
			return errors.New("cancelled")
		}
	}
	var result control.NexusUntrack
	if err := c.ask("nexus.untrack", control.Params{Game: a[0], Unused: c.unused}, &result, readTimeout); err != nil {
		return err
	}
	return c.emit(result, func() {
		fmt.Fprintf(c.out, "Untracked %d mods; %d remaining.\n", result.Untracked, result.Remaining)
		if result.StoppedForLimit {
			fmt.Fprintln(c.out, "Stopped at the Nexus API limit.")
		}
	})
}

func (c *cmd) profile() error {
	if len(c.args) < 2 {
		return usageError{"profile needs create, rename, copy, compare, match, history, revert or delete"}
	}
	sub := c.args[1]
	var p profile.Profile
	switch sub {
	case "compare":
		a, err := c.need(2, "a game", "a profile", "a second profile")
		if err != nil {
			return err
		}
		var diff profile.CLICompare
		if err := c.ask("profile.compare", control.Params{Game: a[0], Profile: a[1], Name: a[2]}, &diff, readTimeout); err != nil {
			return err
		}
		return c.emit(diff, func() { c.compareTable(diff) })
	case "match":
		a, err := c.need(2, "a game", "a profile", "a link or .mortar file")
		if err != nil {
			return err
		}
		var match control.ProfileMatch
		if err := c.ask("profile.match", control.Params{Game: a[0], Profile: a[1], Path: a[2]}, &match, readTimeout); err != nil {
			return err
		}
		return c.emit(match, func() {
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
	case "history":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		var rows []control.HistoryRow
		if err := c.ask("profile.history", control.Params{Game: a[0], Profile: a[1]}, &rows, readTimeout); err != nil {
			return err
		}
		return c.emit(rows, func() {
			t := [][]string{}
			for _, row := range rows {
				t = append(t, []string{row.ID, row.At.Local().Format("2006-01-02 15:04"), row.Kind, fmt.Sprint(row.Count), row.Summary})
			}
			c.table("ID\tTIME\tKIND\tCOUNT\tSUMMARY", t)
		})
	case "revert":
		a, err := c.need(2, "a game", "a profile", "an event id")
		if err != nil {
			return err
		}
		if err := c.ask("profile.revert", control.Params{Game: a[0], Profile: a[1], Name: a[2]}, &p, readTimeout); err != nil {
			return err
		}
	case "create":
		a, err := c.need(2, "a game", "a name")
		if err != nil {
			return err
		}
		if err := c.ask("profile.create", control.Params{Game: a[0], Name: strings.Join(a[1:], " ")}, &p, readTimeout); err != nil {
			return err
		}
	case "rename":
		a, err := c.need(2, "a game", "a profile", "a new name")
		if err != nil {
			return err
		}
		if err := c.ask("profile.rename", control.Params{Game: a[0], Profile: a[1], Name: strings.Join(a[2:], " ")}, &p, readTimeout); err != nil {
			return err
		}
	case "copy":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		if err := c.ask("profile.copy", control.Params{Game: a[0], Profile: a[1], Name: strings.Join(a[2:], " ")}, &p, readTimeout); err != nil {
			return err
		}
	case "delete":
		a, err := c.need(2, "a game", "a profile")
		if err != nil {
			return err
		}
		var r control.Removed
		if err := c.ask("profile.delete", control.Params{Game: a[0], Profile: a[1]}, &r, readTimeout); err != nil {
			return err
		}
		return c.emit(r, func() { fmt.Fprintf(c.out, "Moved %s to Mortar's trash.\n", strings.Join(r.Mods, ", ")) })
	default:
		return usageError{"unknown profile command " + sub}
	}
	return c.emit(p, func() { fmt.Fprintf(c.out, "%s\t%s\n", p.ID, p.Name) })
}

func (c *cmd) compareTable(diff profile.CLICompare) {
	rows := [][]string{}
	for _, side := range diff.OnlyA {
		rows = append(rows, []string{"only-in-A", side.Name, side.UniqueID, side.Version, yes(side.Enabled)})
	}
	for _, side := range diff.OnlyB {
		rows = append(rows, []string{"only-in-B", side.Name, side.UniqueID, side.Version, yes(side.Enabled)})
	}
	for _, pair := range diff.DifferentVersion {
		rows = append(rows, []string{"different-version", pair.Name, pair.UniqueID, pair.A.Version + " -> " + pair.B.Version, ""})
	}
	for _, pair := range diff.DifferentEnabled {
		rows = append(rows, []string{"different-enabled", pair.Name, pair.UniqueID, "", yes(pair.A.Enabled) + " -> " + yes(pair.B.Enabled)})
	}
	for _, pair := range diff.Identical {
		rows = append(rows, []string{"identical", pair.Name, pair.UniqueID, pair.A.Version, yes(pair.A.Enabled)})
	}
	c.table("SECTION\tNAME\tUNIQUEID\tVERSION\tENABLED", rows)
}

func (c *cmd) tools() error {
	if len(c.args) > 1 && c.args[1] == "run" {
		a, err := c.need(2, "a game", "a profile", "a tool")
		if err != nil {
			return err
		}
		if err := c.ask("tools.run", control.Params{Game: a[0], Profile: a[1], Name: a[2]}, nil, launchTimeout); err != nil {
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
	var list []tools.Tool
	if err := c.ask("tools", control.Params{Game: a[0]}, &list, readTimeout); err != nil {
		return err
	}
	return c.emit(list, func() {
		rows := [][]string{}
		for _, tool := range list {
			rows = append(rows, []string{tool.ID, tool.Name, tool.Executable, tool.WorkingDir})
		}
		c.table("ID\tNAME\tEXECUTABLE\tWORKING DIR", rows)
	})
}

func (c *cmd) mods(p control.Params) error {
	var rows []control.ModRow
	if err := c.ask("mods", p, &rows, readTimeout); err != nil {
		return err
	}
	return c.emit(rows, func() { c.modTable(rows) })
}

func (c *cmd) modTable(rows []control.ModRow) {
	t := [][]string{}
	for _, m := range rows {
		state := "on"
		if !m.Enabled {
			state = "off"
		}
		if m.Pinned {
			state += ", pinned"
		}
		t = append(t, []string{m.UniqueID, m.Name, m.Version, state, m.Source})
	}
	c.table("UNIQUEID\tNAME\tVERSION\tSTATE\tSOURCE", t)
}

func (c *cmd) modsChange(sub string) error {
	a, err := c.need(2, "a game", "a profile", "one or more UniqueIDs")
	if err != nil {
		return err
	}
	p := control.Params{Game: a[0], Profile: a[1], UniqueIDs: a[2:]}
	switch sub {
	case "enable", "disable":
		var res profile.EnableResult
		if err := c.ask("mods."+sub, p, &res, readTimeout); err != nil {
			return err
		}
		return c.emit(res, func() {
			fmt.Fprintf(c.out, "%sd %s.\n", strings.ToUpper(sub[:1])+sub[1:], strings.Join(p.UniqueIDs, ", "))
			if len(res.AlsoEnabled) > 0 {
				fmt.Fprintf(c.out, "Also enabled, as required: %s.\n", strings.Join(res.AlsoEnabled, ", "))
			}
		})
	case "remove":
		var r control.Removed
		if err := c.ask("mods.remove", p, &r, readTimeout); err != nil {
			return err
		}
		return c.emit(r, func() { fmt.Fprintf(c.out, "Removed %s.\n", strings.Join(r.Mods, ", ")) })
	}
	var rows []control.ModRow
	if err := c.ask("mods."+sub, p, &rows, readTimeout); err != nil {
		return err
	}
	return c.emit(rows, func() { c.modTable(rows) })
}

func (c *cmd) mod(p control.Params) error {
	var m control.ModInfo
	if err := c.ask("mod", p, &m, readTimeout); err != nil {
		return err
	}
	return c.emit(m, func() {
		state := "on"
		if !m.Enabled {
			state = "off"
		}
		fmt.Fprintf(c.out, "%s (%s) %s by %s, %s, from %s\n", m.Name, m.UniqueID, m.Version, m.Author, state, m.Source)
		list := func(label string, xs []string) {
			if len(xs) > 0 {
				fmt.Fprintf(c.out, "%s: %s\n", label, strings.Join(xs, ", "))
			}
		}
		list("Needs", m.Needs)
		list("Optional", m.Optional)
		list("Needed by", m.Dependents)
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
	var res control.InstallOutcome
	if err := c.ask("install", p, &res, installTimeout); err != nil {
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
		return fmt.Errorf("this archive needs you to choose %s: install it from Mortar's window", what)
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
	var list []problems.AssetConflict
	if err := c.ask("conflicts", p, &list, readTimeout); err != nil {
		return err
	}
	return c.emit(list, func() {
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
	if err := c.ask("problems", p, &r, readTimeout); err != nil {
		return err
	}
	return c.emit(r, func() { c.printProblems(r) })
}

func (c *cmd) printProblems(r problems.Result) {
	conflicts := 0
	for _, x := range r.AssetConflicts {
		if !x.Cosmetic {
			conflicts++
		}
	}
	fmt.Fprintf(c.out, "%d problems: %d missing, %d duplicates, %d broken, %d conflicts, %d settings, %d last-run errors, %d outside edits\n",
		r.Count(), len(r.Missing), len(r.Duplicates), len(r.Broken), conflicts, len(r.Settings), len(r.RunErrors), len(r.Drift))
	fmt.Fprintf(c.out, "Dismissed (%d)\n", len(r.Dismissed))
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
		fmt.Fprintf(c.out, "duplicate  %s (%s)\n", x.Name, x.UniqueID)
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
		fmt.Fprintf(c.out, "%s  setting    %s %s=%s for %s\n", next(), x.Name, x.Field, x.Current, strings.Join(x.ForNames, ", "))
	}
	for _, x := range r.RunErrors {
		fmt.Fprintf(c.out, "run error  %s (%s)\n", x.Name, x.UniqueID)
	}
	for _, x := range r.Drift {
		fmt.Fprintf(c.out, "edited     %s %s\n", x.Kind, x.Folder)
	}
}

func (c *cmd) problemsDismissed() error {
	profile, err := c.problemsProfile()
	if err != nil {
		return err
	}
	game := c.problemsGame()
	var list []problems.DismissedProblem
	if err := c.ask("problems.dismissed", control.Params{Game: game, Profile: profile}, &list, readTimeout); err != nil {
		return err
	}
	return c.emit(list, func() {
		for i, d := range list {
			kind, text := dismissedKindText(d)
			fmt.Fprintf(c.out, "%d\t%s\t%s\t%s\n", i+1, kind, text, d.Token)
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
	game := c.problemsGame()
	var r problems.Result
	if err := c.ask("problems", control.Params{Game: game, Profile: profile}, &r, readTimeout); err != nil {
		return err
	}
	rows := dismissableRows(r)
	if index > len(rows) {
		return refusedError{fmt.Sprintf("problem index %d is not dismissable (%d dismissable rows; run mortar problems %s %s)", index, len(rows), game, profile)}
	}
	if err := c.ask("problems.dismiss", control.Params{Game: game, Profile: profile, ModID: index}, nil, readTimeout); err != nil {
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
	game := c.problemsGame()
	arg := a[0]
	p := control.Params{Game: game, Profile: profile}
	if idx, err := strconv.Atoi(strings.TrimSpace(arg)); err == nil && idx > 0 {
		p.ModID = idx
	} else {
		p.Name = arg
	}
	if err := c.ask("problems.restore", p, nil, readTimeout); err != nil {
		return err
	}
	return c.emit(map[string]bool{"restored": true}, func() { fmt.Fprintln(c.out, "Restored.") })
}

func (c *cmd) updates(p control.Params) error {
	var r problems.UpdatesResult
	if err := c.ask("updates", p, &r, readTimeout); err != nil {
		return err
	}
	return c.emit(r, func() {
		if len(r.Updates) == 0 {
			fmt.Fprintln(c.out, "Everything is up to date.")
			return
		}
		t := [][]string{}
		for _, u := range r.Updates {
			t = append(t, []string{u.UniqueID, u.Name, u.Installed, u.Version, u.URL})
		}
		c.table("UNIQUEID\tNAME\tINSTALLED\tNEWEST\tPAGE", t)
	})
}

func (c *cmd) share(p control.Params) error {
	var l control.ShareLink
	if err := c.ask("share", p, &l, readTimeout); err != nil {
		return err
	}
	if l.TooLarge && !c.json {
		return errors.New("this profile is too large for a link; use mortar export to write a .mortar file")
	}
	return c.emit(l, func() { fmt.Fprintln(c.out, l.Web) })
}

func (c *cmd) export(p control.Params) error {
	var e control.Exported
	if err := c.ask("export", p, &e, installTimeout); err != nil {
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
	var list []launchsvc.Run
	if err := c.ask("runs", p, &list, readTimeout); err != nil {
		return err
	}
	return c.emit(list, func() {
		t := [][]string{}
		for _, r := range list {
			t = append(t, []string{
				r.ID, r.Started, (time.Duration(r.DurationMs) * time.Millisecond).Round(time.Second).String(),
				string(r.Outcome), fmt.Sprint(r.Errors), fmt.Sprint(r.Warnings), r.SMAPIVersion, r.GameVersion,
			})
		}
		c.table("ID\tSTARTED\tDURATION\tOUTCOME\tERRORS\tWARNINGS\tSMAPI\tGAME", t)
		for _, r := range list {
			if r.Cause != nil && r.Cause.ModName != "" {
				fmt.Fprintf(c.out, "%s: Caused by %s — %s\n", r.ID, r.Cause.ModName, r.Cause.Detail)
			}
		}
	})
}

func (c *cmd) logs(p control.Params) error {
	var l control.RunLog
	if err := c.ask("logs", p, &l, readTimeout); err != nil {
		return err
	}
	return c.emit(l, func() { fmt.Fprint(c.out, l.Text) })
}

func (c *cmd) searchLogs(query string) error {
	var result launchsvc.RunSearch
	if err := c.call("logs.search", control.Params{Game: "stardew", Profile: c.profileFlag, Query: query}, &result, readTimeout); err != nil {
		return err
	}
	return c.emit(result, func() {
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
	if err := c.ask("saves", p, &list, readTimeout); err != nil {
		return err
	}
	seasons := []string{"Spring", "Summer", "Fall", "Winter"}
	return c.emit(list, func() {
		t := [][]string{}
		for _, s := range list {
			season := fmt.Sprint(s.Season)
			if s.Season >= 0 && s.Season < len(seasons) {
				season = seasons[s.Season]
			}
			missing := []string{}
			for _, m := range s.Missing {
				missing = append(missing, m.Name)
			}
			t = append(t, []string{s.Folder, s.Farmer, s.Farm, fmt.Sprintf("%s %d, Year %d", season, s.Day, s.Year), strings.Join(missing, ", ")})
		}
		c.table("FOLDER\tFARMER\tFARM\tDATE\tMISSING MODS", t)
	})
}

func (c *cmd) launch(p control.Params) error {
	var st launchsvc.Status
	if err := c.ask("launch", p, &st, launchTimeout); err != nil {
		return err
	}
	if !c.wait {
		return c.emit(st, func() { fmt.Fprintf(c.out, "%s is %s.\n", p.Game, st.State) })
	}
	if !c.json {
		fmt.Fprintf(c.out, "%s is %s; waiting for it to close.\n", p.Game, st.State)
	}
	for st.State == launchsvc.Running || st.State == launchsvc.Launching {
		time.Sleep(2 * time.Second)
		if err := c.ask("status", control.Params{Game: p.Game}, &st, readTimeout); err != nil {
			return err
		}
	}
	return c.runs(control.Params{Game: p.Game, Profile: p.Profile})
}

func (c *cmd) status(verb, gameID string) error {
	var st launchsvc.Status
	if err := c.ask(verb, control.Params{Game: gameID}, &st, launchTimeout); err != nil {
		return err
	}
	return c.emit(st, func() {
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
			var st queue.State
			if err := c.ask(method, control.Params{Name: id}, &st, readTimeout); err != nil {
				return err
			}
			return c.emit(st, func() { fmt.Fprintf(c.out, "Queue %s.\n", sub) })
		case "pause", "resume", "clear":
			var st queue.State
			if err := c.ask(method, control.Params{}, &st, readTimeout); err != nil {
				return err
			}
			return c.emit(st, func() { fmt.Fprintf(c.out, "Queue %s.\n", sub) })
		default:
			return usageError{"unknown queue command " + sub}
		}
	}
	var st queue.State
	if err := c.ask("queue", control.Params{}, &st, readTimeout); err != nil {
		return err
	}
	return c.emit(st, func() {
		if len(st.Items) == 0 {
			fmt.Fprintln(c.out, "The download queue is empty.")
			return
		}
		t := [][]string{}
		for _, it := range st.Items {
			t = append(t, []string{it.ID, it.Name, it.Version, it.State, it.Profile, it.Error})
		}
		c.table("ID\tNAME\tVERSION\tSTATE\tPROFILE\tERROR", t)
		if st.Paused {
			fmt.Fprintln(c.out, "Paused.")
		}
	})
}

func (c *cmd) update() error {
	a, err := c.need(1, "a game", "a profile")
	if err != nil {
		return err
	}
	if len(a) == 2 && !c.all {
		return usageError{"update needs one or more UniqueIDs, or --all"}
	}
	var st queue.State
	if err := c.ask("updates.queue", control.Params{Game: a[0], Profile: a[1], UniqueIDs: a[2:], All: c.all}, &st, installTimeout); err != nil {
		return err
	}
	return c.emit(st, func() { fmt.Fprintf(c.out, "Queued updates.\n") })
}

func (c *cmd) backups() error {
	if len(c.args) > 1 && c.args[1] == "restore" {
		a, err := c.need(2, "a backup name")
		if err != nil {
			return err
		}
		if err := c.ask("backups.restore", control.Params{Name: a[0], UniqueIDs: a[1:]}, nil, installTimeout); err != nil {
			return err
		}
		return c.emit(map[string]any{"restored": a[0], "saves": a[1:]}, func() {
			fmt.Fprintf(c.out, "Restored %s.\n", a[0])
		})
	}
	if len(c.args) > 1 && c.args[1] != "list" {
		return usageError{"unknown backups command " + c.args[1]}
	}
	var list any
	if err := c.ask("backups", control.Params{}, &list, readTimeout); err != nil {
		return err
	}
	return c.emit(list, func() { fmt.Fprintln(c.out, list) })
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
	if err := c.ask(method, p, &list, readTimeout); err != nil {
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
	if err := c.ask("doctor", control.Params{}, &d, readTimeout); err != nil {
		if errors.Is(err, control.ErrNotRunning) {
			return offlineDoctor()
		}
		return err
	}
	return c.emit(d, func() {
		fmt.Fprintf(c.out, "Mortar %s (this command %s)\nData folder: %s\n", d.Version, c.version, d.DataDir)
		for _, g := range d.Games {
			env := d.Environment[g.ID]
			fmt.Fprintf(c.out, "%s: installed %s, folder %q, store %s, game %s, SMAPI %s, %s\n",
				g.Name, yes(g.Installed), g.InstallDir, g.Store, env.GameVersion, env.APIVersion, env.Platform)
		}
		handler := "off"
		if d.NxmHandled {
			handler = "Mortar"
			if d.NxmPrevious != "" {
				handler += " (other games go to " + d.NxmPrevious + ")"
			}
		}
		fmt.Fprintf(c.out, "nxm:// links: %s\n", handler)
	})
}

func offlineDoctor() error {
	dir, err := datadir.Dir()
	if err != nil {
		return err
	}
	findings := []string{}
	fixes := []string{}
	settingsPath := filepath.Join(dir, "settings.json")
	if b, err := os.ReadFile(settingsPath); err != nil {
		findings = append(findings, "settings.json is unreadable or missing")
		fixes = append(fixes, "restore settings.json from a known-good copy")
	} else if !json.Valid(b) {
		findings = append(findings, "settings.json is corrupt")
		fixes = append(fixes, "restore settings.json from a known-good copy")
	}
	if copies, _ := filepath.Glob(settingsPath + ".*"); len(copies) > 0 {
		findings = append(findings, fmt.Sprintf("%d settings temporary/copy files are present", len(copies)))
		fixes = append(fixes, "keep the newest valid settings.json and remove abandoned copies")
	}
	_ = filepath.WalkDir(filepath.Join(dir, "profiles"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr == nil && !entry.IsDir() && entry.Name() == "profile.json" {
			if b, readErr := os.ReadFile(path); readErr != nil || !json.Valid(b) {
				findings = append(findings, "damaged profile.json: "+path)
				fixes = append(fixes, "restore or remove the damaged profile")
			}
		}
		return nil
	})
	if info, statErr := os.Stat(filepath.Join(dir, control.FileName)); statErr == nil && time.Since(info.ModTime()) > 24*time.Hour {
		findings = append(findings, "control.json is stale")
		fixes = append(fixes, "start Mortar once to refresh control.json")
	}
	free, freeErr := freeSpace(dir)
	if freeErr == nil {
		if free == 0 {
			findings = append(findings, "data folder has no free space")
			fixes = append(fixes, "free disk space before starting Mortar")
		}
	}
	if b, err := os.ReadFile(filepath.Join(dir, "store", "index.json")); err != nil || !json.Valid(b) {
		findings = append(findings, "store index.json is unreadable or corrupt")
		fixes = append(fixes, "restore the store index or let Mortar rebuild it")
	}
	var cacheBytes int64
	_ = filepath.WalkDir(filepath.Join(dir, "cache"), func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr == nil && !entry.IsDir() {
			if info, statErr := entry.Info(); statErr == nil {
				cacheBytes += info.Size()
			}
		}
		return nil
	})
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

Mortar must be running; these commands ask the open app. <profile> is an id or a name.

  games                                   supported games, whether each is configured
  profiles <game>                         profiles of a game
  profile create <game> <name>            new empty profile
  profile rename <game> <profile> <name>
  profile copy <game> <profile> [name]
  profile match <game> <profile> <link-or-file> preview a friend's profile
  bundles <game>                         list saved bundles
  bundles apply <game> <bundle> <profile> apply a bundle
  nexus untrack <game> --all|--unused    bulk untrack Nexus mods
  profile delete <game> <profile>         moves it to Mortar's trash
  trash list [--game stardew]             recently deleted profiles
  trash restore <name|id> [--game stardew]
  trash delete <name|id> --yes            permanently delete one
  trash empty --yes [--game stardew]      purge all deleted profiles
  profile compare <game> <profileA> <profileB>
  profile history <game> <profile>       restore points
  profile revert <game> <profile> <eventId>
  mods <game> <profile>                   mods with version, state and source
  mods enable|disable <game> <profile> <UniqueID>...
  mods pin|unpin <game> <profile> <UniqueID>...
  mods remove <game> <profile> <UniqueID>...   removes each mod's whole download
  mod <game> <profile> <UniqueID>         one mod: dependencies, dependents, conflicts, settings
  install <game> <profile> <archive>      install a local archive
  conflicts <game> <profile> [--all]      asset conflicts (--all includes cosmetic ones)
  problems <game> <profile>               everything the Problems tab lists
  problems dismissed [--profile <name>]   dismissed problems (index, kind, text, token)
  problems dismiss <index> [--profile <name>]
  problems restore <token|index> [--profile <name>]
  updates <game> <profile>                mods with a newer version
  saves <game> <profile>                  saves and the mods each one lacks
  share <game> <profile>                  share link
  export <game> <profile> <file.mortar>   write a .mortar file
  open <link|file>                        hand a share link or .mortar file to Mortar
  launch <game> <profile> [--wait] [--force] play; --force skips Play warnings
  status <game> | stop <game>
  runs <game> <profile>                   recent launches
  logs <game> <profile> [--run <id>]      a stored SMAPI log (latest by default)
  logs search <query> [--profile <name>]  search all stored run logs
  queue                                   the download queue
  queue retry|skip [<id>]                 retry or skip queued downloads
  queue pause|resume|clear                control the download queue
  update <game> <profile> <UniqueID>...|--all
                                          queue available mod updates
  backups list                            list save backups
  backups restore <name> [save...]        restore a save backup
  tools <game>                            configured external tools
  tools run <game> <profile> <tool>       start an external tool
  launchers [add|remove <id> <folder>]   launchers, the games in each, and your added folders
  doctor                                  versions, folders and link handling
  completion bash|zsh|fish                shell completion script
  version | help
`
