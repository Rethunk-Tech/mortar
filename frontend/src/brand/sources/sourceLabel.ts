const LABELS: Record<string, string> = {
  nexus: 'Nexus',
  github: 'GitHub',
  thunderstore: 'Thunderstore',
  modrinth: 'Modrinth',
  curseforge: 'CurseForge',
  itch: 'itch.io',
  patreon: 'Patreon',
  vsmoddb: 'VS Mod DB',
  paradox: 'Paradox Mods',
  smapi: 'smapi.io',
}

export const sourceLabel = (id: string): string => LABELS[id] ?? id
