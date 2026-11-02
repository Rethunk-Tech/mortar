package devsvc

import (
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

// batchCommandLine is the command line that runs a .cmd or .bat file with args as cmd.exe reads them: /s strips only
// the outer pair of quotes, so a tool path with a space survives. Quoting keeps & | < > ( ) literal, but % still expands
// inside quotes and a quote would end the argument, so an argument holding either is refused: type names and file names
// reach here from the game's own files.
func batchCommandLine(comspec, tool string, args []string) (string, error) {
	var b strings.Builder
	b.WriteString(`"` + comspec + `" /d /s /c ""` + tool + `"`)
	for _, a := range append([]string{tool}, args...) {
		if strings.ContainsAny(a, "\"%\r\n") {
			return "", usererr.Wrap(usererr.Invalid, errors.New(a+" has a character a Windows command file cannot be given"))
		}
	}
	for _, a := range args {
		b.WriteString(` "` + a + `"`)
	}
	b.WriteString(`"`)
	return b.String(), nil
}
