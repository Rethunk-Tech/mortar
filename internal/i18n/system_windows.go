//go:build windows

package i18n

import "golang.org/x/sys/windows"

const muiLanguageName = 0x8

// systemLanguage is the user's preferred display language as a tag such as "de-DE".
func systemLanguage() string {
	langs, err := windows.GetUserPreferredUILanguages(muiLanguageName)
	if err != nil || len(langs) == 0 {
		return ""
	}
	return langs[0]
}
