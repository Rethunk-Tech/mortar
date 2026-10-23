// Package ghrepo validates a GitHub "owner/repo" name before it reaches an API URL.
package ghrepo

import (
	"regexp"
	"strings"
)

// repoPattern is a GitHub "owner/repo"; it also keeps anything but a name out of the API URL.
var repoPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// Valid reports a GitHub "owner/repo". It also refuses "." and "..", which the pattern admits but would
// move the API URL's path.
func Valid(repo string) bool {
	owner, name, _ := strings.Cut(repo, "/")
	return repoPattern.MatchString(repo) && strings.Trim(owner, ".") != "" && strings.Trim(name, ".") != ""
}
