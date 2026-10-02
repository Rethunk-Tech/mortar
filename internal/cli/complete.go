package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/control"
	"github.com/Rethunk-AI/mortar/internal/profile"
)

func absolute(p string) (string, error) { return filepath.Abs(p) }

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
	"profile":    {"create", "rename", "copy", "delete", "compare", "match", "history", "revert"},
	"mods":       {"enable", "disable", "pin", "unpin", "remove"},
	"bundles":    {"apply"},
	"nexus":      {"untrack"},
	"trash":      {"list", "restore", "delete", "empty"},
	"completion": {"bash", "zsh", "fish"},
	"launchers":  {"add", "remove"},
	"tools":      {"run"},
	"problems":   {"dismissed", "dismiss", "restore"},
	"queue":      {"retry", "skip", "pause", "resume", "clear"},
	"backups":    {"list", "restore"},
}

// gameAt and profileAt give the positions (1-based after the verb) where a verb takes a game and a profile.
func positions(words []string) (gameAt, profileAt, modAt int) {
	switch words[0] {
	case "games", "doctor", "queue", "backups", "version", "help", "open", "completion", "launchers":
		return 0, 0, 0
	case "bundles":
		return 1, 0, 0
	case "nexus":
		return 2, 0, 0
	case "trash":
		return 0, 0, 0
	case "tools":
		if len(words) > 1 && words[1] == "run" {
			return 2, 3, 4
		}
		return 1, 0, 0
	case "profiles", "status", "stop":
		return 1, 0, 0
	case "profile":
		if len(words) > 1 && words[1] == "create" {
			return 2, 0, 0
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
			if !strings.HasPrefix(v, "-") && !strings.HasPrefix(v, "_") {
				cands = append(cands, v)
			}
		}
	} else if subs, ok := subverbs[words[0]]; ok && pos == 1 && words[0] != "mods" && words[0] != "problems" {
		cands = subs
	} else if words[0] == "problems" && pos == 1 {
		cands = append(cands, subverbs["problems"]...)
	} else if words[0] == "trash" && pos == 2 && len(words) > 1 && (words[1] == "restore" || words[1] == "delete") {
		game := "stardew"
		for i := 0; i+1 < len(words); i++ {
			if words[i] == "--game" {
				game = words[i+1]
			}
		}
		var items []profile.TrashItem
		if c.ask("trash.list", control.Params{Game: game}, &items, readTimeout) == nil {
			for _, item := range items {
				cands = append(cands, item.Name)
			}
		}
	} else {
		if words[0] == "mods" && pos == 1 {
			cands = append(cands, subverbs["mods"]...)
		}
		gameAt, profileAt, modAt := positions(words)
		switch pos {
		case gameAt:
			var rows []control.GameRow
			if c.ask("games", control.Params{}, &rows, readTimeout) == nil {
				for _, g := range rows {
					cands = append(cands, g.ID)
				}
			}
		case profileAt:
			var list []profile.Profile
			if c.ask("profiles", control.Params{Game: words[gameAt]}, &list, readTimeout) == nil {
				for _, p := range list {
					cands = append(cands, p.Name)
				}
			}
		case modAt:
			var rows []control.ModRow
			if c.ask("mods", control.Params{Game: words[gameAt], Profile: words[profileAt]}, &rows, readTimeout) == nil {
				for _, m := range rows {
					cands = append(cands, m.UniqueID)
				}
			}
		case 4:
			var rows []struct {
				ID string `json:"id"`
			}
			if c.ask("tools", control.Params{Game: words[gameAt]}, &rows, readTimeout) == nil {
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
