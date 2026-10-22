package i18n

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// parsePO reads the msgid and msgstr pairs of a gettext PO file. Untranslated and fuzzy entries and the header are
// left out, so a lookup that misses falls back to English. Contexts and plural forms are not used here and are errors.
func parsePO(r io.Reader) (catalog, error) {
	out := catalog{}
	var id, str strings.Builder
	var field *strings.Builder
	fuzzy := false
	flush := func() {
		if id.Len() > 0 && str.Len() > 0 && !fuzzy {
			out[id.String()] = str.String()
		}
		id.Reset()
		str.Reset()
		field, fuzzy = nil, false
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		keyword, rest, _ := strings.Cut(line, " ")
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "#~"):
		case strings.HasPrefix(line, "#"):
			if field == &str {
				flush()
			}
			fuzzy = fuzzy || strings.HasPrefix(line, "#,") && strings.Contains(line, "fuzzy")
		case keyword == "msgid" || keyword == "msgstr":
			if keyword == "msgid" {
				if field == &str {
					flush()
				}
				field = &id
			} else {
				field = &str
			}
			if err := appendQuoted(field, rest); err != nil {
				return nil, fmt.Errorf("line %d: %w", n, err)
			}
		case strings.HasPrefix(line, `"`) && field != nil:
			if err := appendQuoted(field, line); err != nil {
				return nil, fmt.Errorf("line %d: %w", n, err)
			}
		default:
			return nil, fmt.Errorf("line %d: unsupported %q", n, line)
		}
	}
	flush()
	return out, sc.Err()
}

func appendQuoted(b *strings.Builder, quoted string) error {
	s, err := strconv.Unquote(strings.TrimSpace(quoted))
	if err != nil {
		return err
	}
	b.WriteString(s)
	return nil
}
