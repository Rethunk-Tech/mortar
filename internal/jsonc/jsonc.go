// Package jsonc turns the JSON Newtonsoft.Json accepts (SMAPI's reader) into strict JSON for encoding/json: it
// decodes a UTF-8 or UTF-16 BOM and strips comments and trailing commas; accepts single-quoted strings, unquoted
// property names and NaN/Infinity/undefined; and escapes control characters inside strings.
package jsonc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"unicode/utf16"
)

// Clean rewrites b as strict JSON where it can. Text that is not JSON in any form Newtonsoft reads is left to fail
// in encoding/json; string contents are never rewritten beyond escaping.
func Clean(b []byte) []byte {
	return dropTrailingCommas(normalise(decode(b)))
}

// decode returns b as UTF-8 text without its BOM; File.ReadAllText in SMAPI picks the encoding from the BOM the same way.
func decode(b []byte) []byte {
	switch {
	case bytes.HasPrefix(b, []byte("\xef\xbb\xbf")):
		return b[3:]
	case bytes.HasPrefix(b, []byte("\xff\xfe")):
		return utf16Bytes(b[2:], binary.LittleEndian)
	case bytes.HasPrefix(b, []byte("\xfe\xff")):
		return utf16Bytes(b[2:], binary.BigEndian)
	}
	return b
}

func utf16Bytes(b []byte, order binary.ByteOrder) []byte {
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = order.Uint16(b[2*i:])
	}
	return []byte(string(utf16.Decode(units)))
}

func isIdent(c byte) bool {
	return c == '_' || c == '$' || c >= 0x80 || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// appendString copies the string literal opening at b[i] as a double-quoted JSON string and returns the index of its
// closing quote, or the last byte when it never closes.
func appendString(out, b []byte, i int) ([]byte, int) {
	quote := b[i]
	out = append(out, '"')
	j := i + 1
	for j < len(b) && b[j] != quote {
		switch c := b[j]; {
		case c == '\\' && j+1 < len(b):
			j++
			switch n := b[j]; {
			case n == '\'':
				out = append(out, '\'')
			case n < 0x20:
				out = appendControl(out, n)
			default:
				out = append(out, '\\', n)
			}
		case c == '"':
			out = append(out, '\\', '"')
		case c < 0x20:
			out = appendControl(out, c)
		default:
			out = append(out, c)
		}
		j++
	}
	if j >= len(b) {
		return out, len(b) - 1
	}
	return append(out, '"'), j
}

func appendControl(out []byte, c byte) []byte {
	switch c {
	case '\n':
		return append(out, '\\', 'n')
	case '\r':
		return append(out, '\\', 'r')
	case '\t':
		return append(out, '\\', 't')
	}
	return fmt.Appendf(out, "\\u%04x", c)
}

// appendWord writes the bare word b[i:j]: a property name is quoted, NaN, Infinity and undefined become null, and
// anything else is left for encoding/json to accept (true, false, null, a number's exponent) or reject.
func appendWord(out, b []byte, i, j int) []byte {
	word := b[i:j]
	k := j
	for k < len(b) && slices.Contains([]byte(" \t\r\n"), b[k]) {
		k++
	}
	switch {
	case k < len(b) && b[k] == ':':
		out = append(out, '"')
		out = append(out, word...)
		return append(out, '"')
	case slices.Contains([]string{"NaN", "Infinity", "undefined"}, string(word)):
		out = bytes.TrimSuffix(out, []byte("-"))
		return append(out, "null"...)
	}
	return append(out, word...)
}

// normalise drops comments and rewrites every non-strict token outside strings.
func normalise(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		switch c := b[i]; {
		case c == '"' || c == '\'':
			out, i = appendString(out, b, i)
		case c == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
			out = append(out, '\n')
		case c == '/' && i+1 < len(b) && b[i+1] == '*':
			end := bytes.Index(b[i+2:], []byte("*/"))
			if end < 0 {
				return out
			}
			// Newlines survive so a parse error's line number still matches the file.
			out = append(out, bytes.Repeat([]byte("\n"), bytes.Count(b[i:i+end+4], []byte("\n")))...)
			out = append(out, ' ')
			i += end + 3
		case isIdent(c) && (c < '0' || c > '9'):
			j := i
			for j < len(b) && isIdent(b[j]) {
				j++
			}
			out = appendWord(out, b, i, j)
			i = j - 1
		default:
			out = append(out, c)
		}
	}
	return out
}

// stringEnd is the index of the quote closing the string literal that opens at b[i], or the last byte when it never closes.
func stringEnd(b []byte, i int) int {
	j := i + 1
	for j < len(b) && b[j] != '"' {
		if b[j] == '\\' {
			j++
		}
		j++
	}
	return min(j, len(b)-1)
}

func dropTrailingCommas(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '"':
			j := stringEnd(b, i)
			out = append(out, b[i:j+1]...)
			i = j
		case ',':
			j := i + 1
			for j < len(b) && slices.Contains([]byte(" \t\r\n"), b[j]) {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				continue
			}
			out = append(out, ',')
		default:
			out = append(out, b[i])
		}
	}
	return out
}
