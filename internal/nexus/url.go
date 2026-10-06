package nexus

import (
	"net/url"
	"strconv"
	"strings"
)

// ModURL is a mod's page on Nexus.
func ModURL(domain string, modID int) string {
	return "https://www.nexusmods.com/" + domain + "/mods/" + strconv.Itoa(modID)
}

// ParseModURL reads a Nexus mod page link, in the old (/<domain>/mods/<id>) or new (/games/<domain>/mods/<id>)
// form; tabs, queries and fragments are ignored.
func ParseModURL(text string) (domain string, modID int, ok bool) {
	u, err := url.Parse(strings.TrimSpace(text))
	if err != nil || u.Scheme != "https" {
		return "", 0, false
	}
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if host != "nexusmods.com" && host != "next.nexusmods.com" {
		return "", 0, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 1 && parts[0] == "games" {
		parts = parts[1:]
	}
	if len(parts) != 3 || parts[0] == "" || parts[1] != "mods" {
		return "", 0, false
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil || id < 1 || strconv.Itoa(id) != parts[2] {
		return "", 0, false
	}
	return parts[0], id, true
}
