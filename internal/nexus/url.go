package nexus

import "strconv"

// ModURL is a mod's page on Nexus.
func ModURL(domain string, modID int) string {
	return "https://www.nexusmods.com/" + domain + "/mods/" + strconv.Itoa(modID)
}
