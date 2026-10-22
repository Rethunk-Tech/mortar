//go:build !windows

package i18n

import "os"

// systemLanguage is the language tag of the first of LC_ALL, LC_MESSAGES and LANG that is set, as POSIX orders them.
func systemLanguage() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" {
			return posixTag(v)
		}
	}
	return ""
}
