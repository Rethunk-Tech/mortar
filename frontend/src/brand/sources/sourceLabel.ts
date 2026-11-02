const LABELS: Record<string, string> = {
  nexus: 'Nexus',
  github: 'GitHub',
  thunderstore: 'Thunderstore',
  modrinth: 'Modrinth',
  curseforge: 'CurseForge',
  itch: 'itch.io',
  patreon: 'Patreon',
  modio: 'mod.io',
  vsmoddb: 'VS Mod DB',
  paradox: 'Paradox Mods',
  smapi: 'smapi.io',
  discord: 'Discord',
}

export const sourceLabel = (id: string): string => LABELS[id] ?? id
