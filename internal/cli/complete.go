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

// subverbs are the second words of the verbs that take one.
var subverbs = map[string][]string{
	"profile":    {"create", "from-save", "rename", "copy", "delete", "compare", "match", "collection", "history", "health", "revert", "load-order", "repair", "list", "shortcut", "steam", "changes", "good"},
	"history":    {"diff", "revert", "usage", "trim"},
	"mods":       {"enable", "disable", "pin", "unpin", "remove", "tag", "untag", "category", "channel", "note", "skip-version", "split", "combine", "files", "config", "preset", "menu", "compat", "report", "by-author", "win", "group"},
	"game":       {"steam-launch-option", "launch-preset-templates", "reset-install", "steam-status"},
	"bundles":    {"apply", "create", "delete", "rename", "add", "remove"},
	"lan":        {"peers", "send", "inbox", "accept", "decline"},
	"app":        {"update"},
	"links":      {"register", "enable", "disable"},
	"source":     {"untrack", "tracked"},
	"trash":      {"list", "restore", "delete", "empty"},
	"completion": {"bash", "zsh", "fish"},
	"launchers":  {"add", "remove"},
	"tools":      {"run", "add", "update", "remove"},
	"saves":      {"check"},
	"queue":      {"add", "retry", "retry-failed", "skip", "pause", "resume", "clear"},
	"backups":    {"list", "create", "keep", "unkeep", "restore", "usage", "trim"},
	"cache":      {"size", "clear"},
	"data":       {"usage", "location", "move", "cleanup"},
	"templates":  {"list", "save", "delete", "new", "apply", "undo", "rename", "restore"},
	"library":    {"extra", "hidden", "old-files", "strays"},
	"archive":    {"preview", "downloads"},
	"store":      {"report", "remove", "check", "repair"},
	"bisect":     {"start", "status", "stop"},
	"settings":   {"get", "set", "export", "import", "reset"},
	"loader":     {"versions", "install", "pin"},
	"logs":       {"search", "share", "fixes"},
	"updates":    {"apply"},
}

// gameAt and profileAt give the positions (1-based after the verb) where a verb takes a game and a profile.
func positions(words []string) (gameAt, profileAt, modAt int) {
	switch words[0] {
	case "games", "doctor", "queue", "backups", "cache", "data", "settings", "version", "help", "open", "completion", "launchers":
		return 0, 0, 0
	case "browse":
		return 1, 0, 0
	case "templates", "library", "archive":
		if len(words) > 1 && (words[1] == "save" || words[1] == "hidden" || words[1] == "old-files" || words[1] == "strays") {
			return 2, 3, 0
		}
		if words[0] == "archive" && len(words) > 1 && words[1] == "preview" {
			return 0, 0, 0
		}
		return 2, 0, 0
	case "store":
		switch {
		case len(words) > 1 && words[1] == "repair":
			return 2, 3, 0
		case len(words) > 1 && (words[1] == "remove" || words[1] == "check"):
			return 2, 0, 0
		}
		return 0, 0, 0
	case "bisect":
		if len(words) > 1 && words[1] == "start" {
			return 2, 3, 0
		}
		return 0, 0, 0
	case "game":
		return 2, 0, 0
	case "bundles":
		return 1, 0, 0
	case "source":
		if len(words) > 1 && words[1] == "tracked" {
			return 2, 3, 0
		}
		return 2, 0, 0
	case "trash":
		return 0, 0, 0
	case "tools":
		if len(words) > 1 && words[1] == "run" {
			return 2, 3, 4
		}
		return 1, 0, 0
	case "profiles", "status", "stop", "sweep":
		return 1, 0, 0
	case "loader":
		return 2, 0, 0
	case "history":
		if len(words) > 1 && (words[1] == "diff" || words[1] == "revert" || words[1] == "trim") {
			return 2, 3, 0
		}
		if len(words) > 1 && words[1] == "usage" {
			return 2, 0, 0
		}
		return 1, 0, 0
	case "profile":
		if len(words) > 1 && (words[1] == "create" || words[1] == "from-save") {
			return 2, 0, 0
		}
		if len(words) > 1 && (words[1] == "shortcut" || words[1] == "steam") {
			return 2, 3, 0
		}
		return 2, 3, 0
	case "mods":
		if len(words) > 1 && slices.Contains(subverbs["mods"], words[1]) {
			return 2, 3, 4
		}
		return 1, 2, 0
	case "mod":
		return 1, 2, 3
	case "problems":
		if len(words) > 1 && slices.Contains(subverbs["problems"], words[1]) {
			return 0, 0, 0
		}
		return 1, 2, 0
	case "updates":
		if len(words) > 1 && words[1] == "apply" {
			return 2, 0, 3
		}
		return 1, 2, 0
	case "conflicts":
		if len(words) > 1 && words[1] == "map" {
			return 2, 3, 0
		}
		return 1, 2, 0
	case "who":
		return 1, 2, 0
	case "logs":
		if len(words) > 1 && words[1] == "search" {
			return 0, 0, 0
		}
		if len(words) > 1 && words[1] == "share" {
			return 2, 3, 0
		}
		if len(words) > 1 && words[1] == "fixes" {
			return 2, 3, 0
		}
		return 1, 2, 0
	case "saves":
		if len(words) > 1 && words[1] == "check" {
			return 2, 4, 0
		}
		return 1, 2, 0
	}
	return 1, 2, 0
}

// complete prints candidates for the last word given the words before it; it stays quiet when Mortar is not running.
func (c *cmd) complete(words []string) error {
	if len(words) == 0 {
		words = []string{""}
	}
	cur := words[len(words)-1]
	pos := len(words) - 1
	var cands []string
	if pos == 0 {
		for v := range verbs {
			if !strings.HasPrefix(v, "-") && !strings.HasPrefix(v, "_") && v != "uninstall-cleanup" {
				cands = append(cands, v)
			}
		}
	} else if subs, ok := subverbs[words[0]]; ok && pos == 1 && words[0] != "mods" && words[0] != "problems" && words[0] != "logs" {
		cands = subs
	} else if words[0] == "problems" && pos == 1 {
		cands = append(cands, subverbs["problems"]...)
	} else if words[0] == "trash" && pos == 2 && len(words) > 1 && (words[1] == "restore" || words[1] == "delete") {
		game := c.game
		for i := 0; i+1 < len(words); i++ {
			if words[i] == "--game" {
				game = words[i+1]
			}
		}
		c.game = game
		var items []profile.TrashItem
		if game, err := c.gameOrDefault(); err == nil && c.call("trash.list", control.Params{Game: game}, &items, readTimeout) == nil {
			for _, item := range items {
				cands = append(cands, item.Name)
			}
		}
	} else {
		if words[0] == "mods" && pos == 1 {
			cands = append(cands, subverbs["mods"]...)
		}
		if words[0] == "logs" && pos == 1 {
			cands = append(cands, subverbs["logs"]...)
		}
		if words[0] == "conflicts" && pos == 1 {
			cands = append(cands, "map")
		}
		gameAt, profileAt, modAt := positions(words)
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
			var rows []struct {
				ID string `json:"id"`
			}
			if c.call("tools", control.Params{Game: words[gameAt]}, &rows, readTimeout) == nil {
				for _, tool := range rows {
					cands = append(cands, tool.ID)
				}
			}
		}
	}
	slices.Sort(cands)
	for _, s := range cands {
		if strings.HasPrefix(strings.ToLower(s), strings.ToLower(cur)) {
			fmt.Fprintln(c.out, s)
		}
	}
	return nil
}
