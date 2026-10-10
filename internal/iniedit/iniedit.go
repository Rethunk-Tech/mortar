// Package iniedit sets `key = value` lines in an ini-style file's text, leaving every other byte as it was.
package iniedit

import (
	"errors"
	"strings"
)

const utf8BOM = "\xef\xbb\xbf"

// ErrUTF16 refuses a UTF-16 file: its bytes are not lines of text, and an edit would mangle it.
var ErrUTF16 = errors.New("the file is UTF-16, which Mortar does not edit")

// Check reports why content cannot be edited: ErrUTF16 for a UTF-16 file, nil otherwise.
func Check(content string) error {
	if strings.HasPrefix(content, "\xff\xfe") || strings.HasPrefix(content, "\xfe\xff") || strings.ContainsRune(content, 0) {
		return ErrUTF16
	}
	return nil
}

// Set gives key the value inside section ("" for any section, taking the first match). The first line holding key in
// the section (names compared without case, comments ignored) is rewritten; a missing key is added at the end of its
// section, and a missing section is created at the end of the file. A leading UTF-8 byte order mark and every
// line ending are kept.
func Set(content, section, key, value string) (string, error) {
	if err := Check(content); err != nil {
		return content, err
	}
	bom := ""
	if strings.HasPrefix(content, utf8BOM) {
		bom, content = utf8BOM, strings.TrimPrefix(content, utf8BOM)
	}
	eol := "\n"
	if strings.Contains(content, "\r\n") {
		eol = "\r\n"
	}
	lines := strings.SplitAfter(content, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	cur, inWanted := "", section == ""
	// last is the index after the final non-blank line of the wanted section, where a missing key goes.
	last, sectionSeen := -1, section == ""
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		switch {
		case line == "" || line[0] == ';' || line[0] == '#':
			continue
		case line[0] == '[':
			cur = strings.TrimSpace(strings.Trim(line, "[]"))
			inWanted = section == "" || strings.EqualFold(cur, section)
			sectionSeen = sectionSeen || inWanted
			if inWanted {
				last = i + 1
			}
			continue
		}
		if !inWanted {
			continue
		}
		last = i + 1
		k, _, ok := strings.Cut(line, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		ending := raw[len(strings.TrimRight(raw, "\r\n")):]
		lines[i] = strings.TrimSpace(k) + " = " + value + ending
		return bom + strings.Join(lines, ""), nil
	}
	if len(lines) > 0 && !strings.HasSuffix(lines[len(lines)-1], "\n") {
		lines[len(lines)-1] += eol
	}
	add := key + " = " + value + eol
	switch {
	case section == "" || (sectionSeen && last < 0):
		lines = append(lines, add)
	case sectionSeen:
		lines = append(lines[:last], append([]string{add}, lines[last:]...)...)
	default:
		lines = append(lines, "["+section+"]"+eol, add)
	}
	return bom + strings.Join(lines, ""), nil
}
