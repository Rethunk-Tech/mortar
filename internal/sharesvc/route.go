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
