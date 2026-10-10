// Package iniedit sets `key = value` lines in an ini-style file's text, leaving every other byte as it was.
package iniedit

import "strings"

// Set gives key the value: the first line holding key (compared without case, outside comments) is rewritten, and a
// file without it gets a new line at its end. Line endings are kept.
func Set(content, key, value string) string {
	lines := strings.SplitAfter(content, "\n")
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || line[0] == ';' || line[0] == '#' || line[0] == '[' {
			continue
		}
		k, _, ok := strings.Cut(line, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		ending := raw[len(strings.TrimRight(raw, "\r\n")):]
		lines[i] = strings.TrimSpace(k) + " = " + value + ending
		return strings.Join(lines, "")
	}
	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += eol
	}
	return content + key + " = " + value + eol
}
