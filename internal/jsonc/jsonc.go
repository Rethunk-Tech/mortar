// Package jsonc strips a UTF-8 BOM, comments, and trailing commas so JSONC can be json.Unmarshal'd.
package jsonc

import (
	"bytes"
	"slices"
)

// Clean removes a UTF-8 BOM, // and /* */ comments outside strings, and trailing commas.
func Clean(b []byte) []byte {
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf"))
	return dropTrailingCommas(dropComments(b))
}

func dropComments(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		switch {
		case b[i] == '"':
			j := i + 1
			for j < len(b) && b[j] != '"' {
				if b[j] == '\\' {
					j++
				}
				j++
			}
			j = min(j, len(b)-1)
			out = append(out, b[i:j+1]...)
			i = j
		case b[i] == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
			out = append(out, '\n')
		case b[i] == '/' && i+1 < len(b) && b[i+1] == '*':
			end := bytes.Index(b[i+2:], []byte("*/"))
			if end < 0 {
				return out
			}
			i += end + 3
			out = append(out, ' ')
		default:
			out = append(out, b[i])
		}
	}
	return out
}

func dropTrailingCommas(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '"':
			j := i + 1
			for j < len(b) && b[j] != '"' {
				if b[j] == '\\' {
					j++
				}
				j++
			}
			j = min(j, len(b)-1)
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
