package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/control"
	"github.com/Rethunk-Tech/mortar/internal/profile"
)

// Each shell script asks `mortar __complete <words before the cursor> <word at the cursor>` for candidates.
var scripts = map[string]string{
	"bash": `_mortar() {
	local IFS=$'\n'
	COMPREPLY=($(mortar __complete "${COMP_WORDS[@]:1:COMP_CWORD}" 2>/dev/null))
}
complete -o default -F _mortar mortar
`,
	"zsh": `#compdef mortar
_mortar() {
	local -a c
	c=("${(@f)$(mortar __complete "${(@)words[2,CURRENT]}" 2>/dev/null)}")
	compadd -a c
}
compdef _mortar mortar
`,
	"fish": `complete -c mortar -f -a '(mortar __complete (commandline -opc)[2..-1] (commandline -ct) 2>/dev/null)'
`,
}

func completion(w io.Writer, shell string) error {
	s, ok := scripts[shell]
	if !ok {
		return usageError{"completion supports bash, zsh and fish"}
	}
	_, err := io.WriteString(w, s)
	return err
}

func subverbsFrom(m map[string]func(*cmd) error, rest ...string) []string {
	out := make([]string, 0, len(m)+len(rest))
	for k := range m {
		out = append(out, k)
	}
	out = append(out, rest...)
	slices.Sort(out)
	return out
}

// subverbs are the second words of the verbs that take one.
var subverbs = map[string][]string{
	"profile":    subverbsFrom(profileDispatchSubs, "create", "from-save", "rename", "copy", "delete", "compare", "match", "collection", "history", "health", "revert", "load-order", "repair", "list", "freeze", "unfreeze", "favourite", "unfavourite"),
	"history":    {"diff", "revert", "usage", "trim"},
	"mods":       subverbsFrom(modsDispatchSubs, "enable", "disable", "pin", "unpin", "remove", "tag", "untag", "category", "note", "skip-version", "split", "combine", "favourite", "unfavourite", "favourites"),
	"game":       {"folder", "steam-launch-option", "launch-preset-templates", "reset-install", "steam-status"},
	"groups":     {"apply", "create", "delete", "rename", "add", "remove"},
	"lan":        {"peers", "send", "inbox", "accept", "decline", "pair", "paired", "unpair"},
	"app":        {"update"},
	"support":    {"diagnostics"},
	"links":      {"register", "enable", "disable", "list"},
	"source":     {"untrack", "tracked"},
	"trash":      {"list", "restore", "delete", "empty"},
	"completion": {"bash", "zsh", "fish"},
	"launchers":  {"add", "remove"},
	"tools":      {"run", "add", "update", "remove"},
	"saves":      {"check"},
	"queue":      {"add", "retry", "retry-failed", "skip", "pause", "resume", "pause-item", "resume-item", "top", "up", "down", "clear"},
	"backups":    {"list", "create", "keep", "unkeep", "restore", "usage", "trim"},
	"cache":      {"size", "clear"},
	"data":       {"usage", "location", "move", "cleanup"},
	"templates":  {"list", "save", "delete", "new", "apply", "undo", "rename", "restore"},
	"library":    {"extra", "hidden", "old-files", "strays"},
	"archive":    {"preview", "downloads"},
	"store":      {"report", "remove", "check", "repair"},
	"bisect":     {"start", "status", "stop"},
	"antivirus":  {"status"},
	"problems":   {"dismissed", "dismiss", "restore", "check-updates"},
	"settings":   {"get", "set", "export", "import", "reset"},
	"loader":     {"versions", "install", "pin"},
	"dev":        {"add", "clone", "list", "remove"},
	"logs":       {"search", "share", "fixes"},
	"updates":    {"apply"},
}

// fixedPositions holds the verbs whose game/profile/mod positions do not depend on the second word.
var fixedPositions = map[string][3]int{
	"games": {}, "doctor": {}, "backups": {}, "cache": {}, "data": {}, "settings": {}, "antivirus": {}, "version": {}, "help": {}, "open": {}, "completion": {}, "launchers": {}, "app": {}, "support": {}, "links": {}, "trash": {},
	"browse":   {1, 0, 0},
	"game":     {2, 0, 0},
	"loader":   {2, 0, 0},
	"dev":      {2, 0, 0},
	"profiles": {1, 0, 0},
	"status":   {1, 0, 0},
	"stop":     {1, 0, 0},
	"sweep":    {1, 0, 0},
	"mod":      {1, 2, 3},
	"who":      {1, 2, 0},
}

func inList(s string, list ...string) bool { return slices.Contains(list, s) }

// subPositions holds the verbs whose positions depend on the second word.
var subPositions = map[string]func(sub string) (gameAt, profileAt, modAt int){
	"queue": func(s string) (int, int, int) {
		if s == "add" {
			return 2, 3, 0
		}
		return 0, 0, 0
	},
	"templates": templatesPositions,
	"library":   libraryPositions,
	"archive": func(s string) (int, int, int) {
		if s == "preview" {
			return 0, 0, 0
		}
		return libraryPositions(s)
	},
	"store": func(s string) (int, int, int) {
		switch s {
		case "repair":
			return 2, 3, 0
		case "remove", "check":
			return 2, 0, 0
		}
		return 0, 0, 0
	},
	"bisect": func(s string) (int, int, int) {
		if s == "start" {
			return 2, 3, 0
		}
		return 0, 0, 0
	},
	"groups": func(s string) (int, int, int) {
		switch {
		case !slices.Contains(subverbs["groups"], s):
			return 1, 0, 0
		case inList(s, "apply", "create", "add"):
			return 2, 4, 0
		}
		return 2, 0, 0
	},
	"source": func(s string) (int, int, int) {
		if s == "tracked" {
			return 2, 3, 0
		}
		return 2, 0, 0
	},
	"tools": func(s string) (int, int, int) {
		switch {
		case s == "run":
			// The tool is completed by name at position 4, not as a mod id.
			return 2, 3, 0
		case slices.Contains(subverbs["tools"], s):
			return 2, 0, 0
		}
		return 1, 0, 0
	},
	"lan": func(s string) (int, int, int) {
		if s == "send" {
			return 2, 3, 0
		}
		return 0, 0, 0
	},
	"history": func(s string) (int, int, int) {
		switch {
		case inList(s, "diff", "revert", "trim"):
			return 2, 3, 0
		case s == "usage":
			return 2, 0, 0
		}
		return 1, 0, 0
	},
	"profile": func(s string) (int, int, int) {
		switch {
		case s == "farm":
			return 3, 4, 0
		case inList(s, "create", "from-save"):
			return 2, 0, 0
		}
		return 2, 3, 0
	},
	"mods": func(s string) (int, int, int) {
		if slices.Contains(subverbs["mods"], s) {
			return 2, 3, 4
		}
		return 1, 2, 0
	},
	"problems": func(s string) (int, int, int) {
		if slices.Contains(subverbs["problems"], s) {
			return 0, 0, 0
		}
		return 1, 2, 0
	},
	"updates": func(s string) (int, int, int) {
		if s == "apply" {
			return 2, 0, 3
		}
		return 1, 2, 0
	},
	"conflicts": func(s string) (int, int, int) {
		if s == "map" {
			return 2, 3, 0
		}
		return 1, 2, 0
	},
	"logs": func(s string) (int, int, int) {
		switch s {
		case "search":
			return 0, 0, 0
		case "share", "fixes":
			return 2, 3, 0
		}
		return 1, 2, 0
	},
	"saves": func(s string) (int, int, int) {
		if s == "check" {
			return 2, 4, 0
		}
		return 1, 2, 0
	},
}

func libraryPositions(s string) (int, int, int) {
	if inList(s, "save", "hidden", "old-files", "strays") {
		return 2, 3, 0
	}
	return 2, 0, 0
}

// templatesPositions: apply is <game> <template> <profile> and undo is <game> <profile> <file>.
func templatesPositions(s string) (int, int, int) {
	switch s {
	case "apply":
		return 2, 4, 0
	case "undo":
		return 2, 3, 0
	}
	return libraryPositions(s)
}

// gameAt and profileAt give the positions (1-based after the verb) where a verb takes a game and a profile.
func positions(words []string) (gameAt, profileAt, modAt int) {
	if p, ok := fixedPositions[words[0]]; ok {
		return p[0], p[1], p[2]
	}
	sub := ""
	if len(words) > 1 {
		sub = words[1]
	}
	if f, ok := subPositions[words[0]]; ok {
		return f(sub)
	}
	return 1, 2, 0
}

// complete prints candidates for the last word given the words before it; it stays quiet when Mortar is not running.
func (c *cmd) complete(words []string) {
	if len(words) == 0 {
		words = []string{""}
	}
	cur := words[len(words)-1]
	cands := c.candidates(words, cur, len(words)-1)
	slices.Sort(cands)
	for _, s := range cands {
		if strings.HasPrefix(strings.ToLower(s), strings.ToLower(cur)) {
			fmt.Fprintln(c.out, s)
		}
	}
}

func (c *cmd) candidates(words []string, cur string, pos int) []string {
	if pos == 0 {
		var cands []string
		for v := range verbs {
			if !strings.HasPrefix(v, "-") && !strings.HasPrefix(v, "_") && v != "uninstall-cleanup" {
				cands = append(cands, v)
			}
		}
		return cands
	}
	if subs, ok := subverbs[words[0]]; ok && pos == 1 && !inList(words[0], "mods", "problems", "logs", "groups", "tools") {
		return subs
	}
	if words[0] == "trash" && pos == 2 && inList(words[1], "restore", "delete") {
		return c.trashCands(words)
	}
	cands := wordCands(words, pos)
	if pos == 2 && words[0] == "settings" && inList(words[1], "get", "set", "reset") {
		cands = append(cands, c.settingNames()...)
	}
	cands = append(cands, sourceFlagCands(words, cur)...)
	prev := ""
	if len(words) >= 2 {
		prev = words[len(words)-2]
	}
	if prev != "--source" && !strings.HasPrefix(cur, "-") {
		cands = append(cands, c.positionalCands(words, pos)...)
	}
	return cands
}

// wordCands lists the fixed words that can follow the verb at pos 1 or its subverb at pos 2.
func wordCands(words []string, pos int) []string {
	switch pos {
	case 1:
		switch words[0] {
		case "mods", "logs", "groups", "tools", "problems":
			return slices.Clone(subverbs[words[0]])
		case "conflicts":
			return []string{"map"}
		}
	case 2:
		switch words[0] + " " + words[1] {
		case "profile farm":
			return []string{"export", "check", "fix"}
		case "app update":
			return []string{"check", "install"}
		case "support diagnostics":
			return []string{"save"}
		}
	}
	return nil
}

func (c *cmd) trashCands(words []string) []string {
	for i := 0; i+1 < len(words); i++ {
		if words[i] == "--game" {
			c.game = words[i+1]
		}
	}
	var items []profile.TrashItem
	game, err := c.gameOrDefault()
	if err != nil || c.call("trash.list", control.Params{Game: game}, &items, readTimeout) != nil {
		return nil
	}
	var cands []string
	for _, item := range items {
		cands = append(cands, item.Name)
	}
	return cands
}

func (c *cmd) settingNames() []string {
	var rows [][2]string
	if c.call("settings.get", control.Params{Game: c.game}, &rows, readTimeout) != nil {
		return nil
	}
	var cands []string
	for _, r := range rows {
		cands = append(cands, r[0])
	}
	return cands
}

// positionalCands asks Mortar for the game, profile, mod or tool names that fit the word at pos.
func (c *cmd) positionalCands(words []string, pos int) []string {
	gameAt, profileAt, modAt := positions(words)
	var cands []string
	switch pos {
	case gameAt:
		var rows []control.GameRow
		if c.call("games", control.Params{}, &rows, readTimeout) == nil {
			for _, g := range rows {
				cands = append(cands, g.ID)
			}
		}
	case profileAt:
		var list []profile.Profile
		if c.call("profiles", control.Params{Game: words[gameAt]}, &list, readTimeout) == nil {
			for _, p := range list {
				if p.Error == "" && p.Name != "" {
					cands = append(cands, p.Name)
				}
			}
		}
	case modAt:
		var rows []control.ModRow
		if c.call("mods", control.Params{Game: words[gameAt], Profile: words[profileAt]}, &rows, readTimeout) == nil {
			for _, m := range rows {
				cands = append(cands, m.ID.Local())
			}
		}
	case 4:
		if words[0] != "tools" || len(words) < 2 || words[1] != "run" {
			return cands
		}
		var rows []struct {
			ID string `json:"id"`
		}
		if c.call("tools", control.Params{Game: words[gameAt]}, &rows, readTimeout) == nil {
			for _, tool := range rows {
				cands = append(cands, tool.ID)
			}
		}
	}
	return cands
}

func sourceFlagCands(words []string, cur string) []string {
	var ids []string
	switch {
	case words[0] == "queue" && len(words) > 1 && words[1] == "add":
		ids = []string{"nexus", "github", "thunderstore", "modrinth", "curseforge", "itch", "vsmoddb", "paradox", "modio"}
	case words[0] == "links" && len(words) > 1 && (words[1] == "enable" || words[1] == "disable"):
		ids = []string{"nexus", "thunderstore", "curseforge", "vsmoddb"}
	default:
		return nil
	}
	prev := ""
	if len(words) >= 2 {
		prev = words[len(words)-2]
	}
	if prev == "--source" {
		return ids
	}
	if strings.HasPrefix(cur, "--source=") {
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			out = append(out, "--source="+id)
		}
		return out
	}
	pos := len(words) - 1
	flagAt := words[0] == "links" && pos >= 2 || words[0] == "queue" && pos >= 5
	if flagAt && (cur == "" || len(cur) <= len("--source") && "--source"[:len(cur)] == cur) {
		return []string{"--source"}
	}
	return nil
}
