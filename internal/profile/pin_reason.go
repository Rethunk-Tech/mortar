package profile

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// MaxPinReason caps why a version was pinned, in characters.
const MaxPinReason = 200

func cleanPinReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", nil
	}
	if n := utf8.RuneCountInString(reason); n > MaxPinReason {
		return "", fmt.Errorf("pin reason is longer than %d characters", MaxPinReason)
	}
	return reason, nil
}
