package steam

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/datadir"
	"github.com/Rethunk-AI/mortar/internal/fsx"
)

// vdfToken is one token of a text VDF file with its byte span, so an edit can change one value and leave every
// other byte of Steam's file as it was.
type vdfToken struct {
	kind       byte // 's' quoted string, '{', '}'
	text       string
	start, end int
}

func scanVDF(b []byte) ([]vdfToken, error) {
	var out []vdfToken
	for i := 0; i < len(b); {
		c := b[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c == '/' && i+1 < len(b) && b[i+1] == '/':
			for i < len(b) && b[i] != '\n' {
				i++
			}
		case c == '{' || c == '}':
			out = append(out, vdfToken{kind: c, start: i, end: i + 1})
			i++
		case c == '"':
			start := i
			var sb strings.Builder
			i++
			for i < len(b) && b[i] != '"' {
				if b[i] == '\\' && i+1 < len(b) {
					i++
					switch b[i] {
					case 'n':
						sb.WriteByte('\n')
					case 't':
						sb.WriteByte('\t')
					default:
						sb.WriteByte(b[i])
					}
				} else {
					sb.WriteByte(b[i])
				}
				i++
			}
			if i >= len(b) {
				return nil, errors.New("unterminated string in VDF")
			}
			i++
			out = append(out, vdfToken{kind: 's', text: sb.String(), start: start, end: i})
		default:
			return nil, fmt.Errorf("unexpected %q in VDF at byte %d", c, i)
		}
	}
	return out, nil
}

func quoteVDF(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// setLaunchOptions sets apps/<appID>/LaunchOptions under UserLocalConfigStore/Software/Valve/Steam to merge(old),
// creating the app's block or the key when missing; it returns the new file and the value it wrote.
func setLaunchOptions(b []byte, appID string, merge func(old string) string) ([]byte, string, error) {
	toks, err := scanVDF(b)
	if err != nil {
		return nil, "", err
	}
	want := []string{"userlocalconfigstore", "software", "valve", "steam", "apps"}
	var path []string
	var pending string
	appsClose, appOpen, appClose := -1, -1, -1
	var optVal *vdfToken
	inApp := func() bool { return len(path) == len(want)+1 && pathIs(path, want) && path[len(want)] == appID }
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		switch t.kind {
		case 's':
			if pending == "" {
				pending = t.text
				if i+1 < len(toks) && toks[i+1].kind == 's' {
					if inApp() && strings.EqualFold(t.text, "LaunchOptions") {
						optVal = &toks[i+1]
					}
					i++
					pending = ""
				}
			}
		case '{':
			path = append(path, strings.ToLower(pending))
			if pathIs(path, want) && len(path) == len(want)+1 && pending == appID {
				appOpen = t.end
			}
			pending = ""
		case '}':
			if len(path) == 0 {
				return nil, "", errors.New("unbalanced braces in VDF")
			}
			if inApp() {
				appClose = t.start
			}
			if len(path) == len(want) && pathIs(path, want) {
				appsClose = t.start
			}
			path = path[:len(path)-1]
		}
	}
	switch {
	case optVal != nil:
		value := merge(optVal.text)
		return splice(b, optVal.start, optVal.end, quoteVDF(value)), value, nil
	case appOpen >= 0 && appClose >= 0:
		value := merge("")
		return splice(b, appClose, appClose, "\t\"LaunchOptions\"\t\t"+quoteVDF(value)+"\n\t\t\t\t\t"), value, nil
	case appsClose >= 0:
		value := merge("")
		block := quoteVDF(appID) + "\n\t\t\t\t\t{\n\t\t\t\t\t\t\"LaunchOptions\"\t\t" + quoteVDF(value) + "\n\t\t\t\t\t}\n\t\t\t\t"
		return splice(b, appsClose, appsClose, block), value, nil
	}
	return nil, "", errors.New("localconfig.vdf has no apps section; launch the game from Steam once first")
}

func pathIs(path, want []string) bool {
	if len(path) < len(want) {
		return false
	}
	for i, w := range want {
		if path[i] != w {
			return false
		}
	}
	return true
}

func splice(b []byte, start, end int, s string) []byte {
	out := make([]byte, 0, len(b)+len(s))
	out = append(out, b[:start]...)
	out = append(out, s...)
	return append(out, b[end:]...)
}

// SetLaunchOptions changes appID's launch options for the MostRecent account to merge(current). Steam rewrites
// localconfig.vdf on exit, so the caller makes sure Steam is closed. It returns the value written.
func (s Steam) SetLaunchOptions(appID string, merge func(current string) string) (string, error) {
	dir, err := s.userConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "localconfig.vdf")
	b, err := fsx.ReadFile(path)
	if err != nil {
		return "", err
	}
	next, value, err := setLaunchOptions(b, appID, merge)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if err := backupBeforeEdit(path, next, info.Mode().Perm()); err != nil {
		return "", err
	}
	return value, datadir.WriteFile(path, next, info.Mode().Perm())
}
