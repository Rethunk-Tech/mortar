package support

import (
	"log"
	"strings"
)

// maxRenderLog bounds one frontend report so a runaway message cannot flood mortar.log.
const maxRenderLog = 4000

// LogRenderError writes a frontend render failure (React's message and component stack) to mortar.log, so a loop
// that only shows in a user's build names its component. The window rate-limits and de-duplicates the calls.
func (s *Service) LogRenderError(message, componentStack string) {
	msg := strings.TrimSpace(message + "\n" + componentStack)
	if len(msg) > maxRenderLog {
		msg = msg[:maxRenderLog]
	}
	log.Printf("frontend render error: %s", strings.ReplaceAll(msg, "\n", " | "))
}
