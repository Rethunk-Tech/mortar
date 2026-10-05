const LABELS: Record<string, string> = {
  nexus: 'Nexus',
  github: 'GitHub',
  thunderstore: 'Thunderstore',
  modrinth: 'Modrinth',
  itch: 'itch.io',
}

export const sourceLabel = (id: string): string => LABELS[id] ?? id
