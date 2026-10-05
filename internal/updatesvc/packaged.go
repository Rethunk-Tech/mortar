package updatesvc

import "strings"

// PackagedBy names the package manager that installed exe when it was not built as a package: Scoop and Homebrew
// install the release's own binaries, so only the install path tells them apart. It returns "" for a plain download.
func PackagedBy(exe string) string {
	p := strings.ToLower(strings.ReplaceAll(exe, `\`, "/"))
	switch {
	case strings.Contains(p, "/scoop/apps/"):
		return "scoop"
	case strings.Contains(p, "/cellar/mortar/"), strings.Contains(p, "/.linuxbrew/"):
		return "homebrew"
	}
	return ""
}
