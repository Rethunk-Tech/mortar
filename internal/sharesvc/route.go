package sharesvc

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/game"
)

type modRoute struct {
	game  string
	modID int
}

func parseModRoute(value string) (modRoute, bool) {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "mortar" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.RawPath != "" {
		return modRoute{}, false
	}
	id := u.Host
	if !game.Valid(id) {
		return modRoute{}, false
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "mod" || parts[1] == "" || parts[1][0] < '1' || parts[1][0] > '9' {
		return modRoute{}, false
	}
	for _, r := range parts[1][1:] {
		if r < '0' || r > '9' {
			return modRoute{}, false
		}
	}
	modID, err := strconv.Atoi(parts[1])
	if err != nil || strconv.Itoa(modID) != parts[1] {
		return modRoute{}, false
	}
	return modRoute{game: id, modID: modID}, true
}

type shortRoute struct {
	game    string
	code    string
	version int
}

const (
	shortCodeLen  = 10
	shareSiteHost = "mortar.rethunk.tech"
)

func parseShortRoute(value string) (shortRoute, bool) {
	u, err := url.Parse(value)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.RawPath != "" {
		return shortRoute{}, false
	}
	var id, rest string
	switch u.Scheme {
	case "mortar":
		if u.Host == "" {
			return shortRoute{}, false
		}
		id = u.Host
		rest = strings.TrimPrefix(u.Path, "/")
	case "https":
		if u.Host != shareSiteHost {
			return shortRoute{}, false
		}
		path := strings.TrimPrefix(u.Path, "/")
		slash := strings.IndexByte(path, '/')
		if slash < 1 {
			return shortRoute{}, false
		}
		id, rest = path[:slash], path[slash+1:]
	default:
		return shortRoute{}, false
	}
	if !game.Valid(id) {
		return shortRoute{}, false
	}
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || len(parts) > 3 || parts[0] != "s" || !shortCode(parts[1]) {
		return shortRoute{}, false
	}
	out := shortRoute{game: id, code: parts[1]}
	if len(parts) == 3 {
		n, ok := shortVersion(parts[2])
		if !ok {
			return shortRoute{}, false
		}
		out.version = n
	}
	return out, true
}

func shortCode(code string) bool {
	if len(code) != shortCodeLen {
		return false
	}
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			continue
		}
		return false
	}
	return true
}

func shortVersion(s string) (int, bool) {
	if s == "" || s[0] < '1' || s[0] > '9' {
		return 0, false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil || strconv.Itoa(n) != s {
		return 0, false
	}
	return n, true
}
