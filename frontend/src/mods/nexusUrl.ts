export const nexusModsUrl = (domain: string) => `https://www.nexusmods.com/${domain}/mods`

export const nexusModUrl = (modId: number | string, domain: string, tab = '') =>
  `${nexusModsUrl(domain)}/${modId}${tab === '' ? '' : `?tab=${tab}`}`
