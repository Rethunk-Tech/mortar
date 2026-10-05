package support

import (
	"log"
	"strings"
	"unicode/utf8"
)

// maxRenderLog bounds one frontend report so a runaway message cannot flood mortar.log.
const maxRenderLog = 4000

// LogRenderError writes a frontend render failure (React's message and component stack) to mortar.log, so a loop
// that only shows in a user's build names its component. The window de-duplicates and caps the calls.
func (s *Service) LogRenderError(message, componentStack string) {
	msg := strings.TrimSpace(message + "\n" + componentStack)
	log.Printf("frontend render error: %s", strings.ReplaceAll(clip(msg, maxRenderLog), "\n", " | "))
}

// clip is s cut to at most n bytes on a character boundary.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
