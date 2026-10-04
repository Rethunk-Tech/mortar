package nexus

import "strconv"

// ModURL is a mod's page on Nexus; domain is the v1 site segment, Game for the configured game.
func ModURL(domain string, modID int) string {
	return "https://www.nexusmods.com/" + domain + "/mods/" + strconv.Itoa(modID)
}
