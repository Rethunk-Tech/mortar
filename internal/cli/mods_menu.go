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
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--set" {
			if i+1 >= len(rest) {
				return "", "", "", "", fmt.Errorf("mods menu --set needs page/index=value")
			}
			set = rest[i+1]
			i++
			continue
		}
		pos = append(pos, rest[i])
	}
	if len(pos) < 3 {
		return "", "", "", "", fmt.Errorf("mods menu <game> <profile> <mod>")
	}
	return pos[0], pos[1], pos[2], set, nil
}
