package cli

import (
	"fmt"
	"io"
)

func writeModsMenu(w io.Writer, lines []string) {
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}

func modsMenuArgs(rest []string) (game, profile, mod, set string, err error) {
	var pos []string
	for len(rest) > 0 {
		arg := rest[0]
		rest = rest[1:]
		if arg != "--set" {
			pos = append(pos, arg)
			continue
		}
		if len(rest) == 0 {
			return "", "", "", "", fmt.Errorf("mods menu --set needs page/index=value")
		}
		set, rest = rest[0], rest[1:]
	}
	if len(pos) < 3 {
		return "", "", "", "", fmt.Errorf("mods menu <game> <profile> <mod>")
	}
	return pos[0], pos[1], pos[2], set, nil
}
