package configsvc

import "strings"

// splitLines cuts text at "\n" into lines and the ending each had ("\r\n", "\n", or "" for an unterminated last line),
// so a write keeps every line's own ending in a file that mixes them.
func splitLines(text string) (lines, eols []string) {
	for _, l := range strings.SplitSeq(text, "\n") {
		if cut, ok := strings.CutSuffix(l, "\r"); ok {
			lines, eols = append(lines, cut), append(eols, "\r\n")
		} else {
			lines, eols = append(lines, l), append(eols, "\n")
		}
	}
	eols[len(eols)-1] = ""
	return lines, eols
}

func joinLines(lines, eols []string, bom bool) string {
	var b strings.Builder
	if bom {
		b.WriteRune(0xFEFF)
	}
	for i, l := range lines {
		b.WriteString(l)
		b.WriteString(eols[i])
	}
	return b.String()
}
