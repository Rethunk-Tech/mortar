/** The Nexus site segment of the game Mortar manages. */
export const NEXUS_DOMAIN = 'stardewvalley'

export const nexusModsUrl = (domain = NEXUS_DOMAIN) =>
  `https://www.nexusmods.com/${domain === '' ? NEXUS_DOMAIN : domain}/mods`

export const nexusModUrl = (modId: number | string, domain = NEXUS_DOMAIN, tab = '') =>
  `${nexusModsUrl(domain)}/${modId}${tab === '' ? '' : `?tab=${tab}`}`
